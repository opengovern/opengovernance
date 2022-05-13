package workspace

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
	"gitlab.com/keibiengine/keibi-engine/pkg/workspace/api"
	"gorm.io/gorm"
)

const (
	KeibiUserID           = "X-Keibi-UserID"
	WorkspaceNameLenLimit = 20
)

var (
	ErrInternalServer = errors.New("internal server error")
)

type Server struct {
	e   *echo.Echo
	cfg *Config
	db  *Database
}

func NewServer(cfg *Config) *Server {
	s := &Server{
		e:   echo.New(),
		cfg: cfg,
	}
	s.e.HideBanner = true
	s.e.HidePort = true

	s.e.Logger.SetHeader(`{"time":"${time_rfc3339}",` +
		`"level":"${level}",` +
		`"file":"${short_file}",` +
		`"line":"${line}"` +
		`}`)
	s.e.Logger.SetLevel(log.INFO)

	s.e.Use(middleware.Recover())
	s.e.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
		Skipper: middleware.DefaultSkipper,
		Format: `{"time":"${time_rfc3339}",` +
			`"remote_ip":"${remote_ip}",` +
			`"method":"${method}",` +
			`"uri":"${uri}",` +
			`"bytes_in":${bytes_in},` +
			`"bytes_out":${bytes_out},` +
			`"status":${status},` +
			`"error":"${error}",` +
			`"latency_human":"${latency_human}"` +
			`}` + "\n",
		CustomTimeFormat: "2006-01-02 15:04:05.000",
	}))

	db, err := NewDatabase(cfg)
	if err != nil {
		s.e.Logger.Fatalf("new database: %v", err)
	}
	s.db = db

	// init the http routers
	v1 := s.e.Group("/api/v1")
	v1.POST("/workspace", s.CreateWorkspace)
	v1.DELETE("/workspace/:workspace_id", s.DeleteWorkspace)
	v1.GET("/workspaces", s.ListWorkspaces)

	// init the cronjobs

	return s
}

func (s *Server) Start(ctx context.Context) {
	go func() {
		s.e.Logger.Infof("workspace service is started on %s", s.cfg.ServerAddr)
		if err := s.e.Start(s.cfg.ServerAddr); err != nil {
			if !errors.Is(err, http.ErrServerClosed) {
				s.e.Logger.Errorf("echo start: %v", err)
			}
		}
	}()
}

func (s *Server) Stop() error {
	if s.e != nil {
		if err := s.e.Shutdown(context.Background()); err != nil {
			s.e.Logger.Errorf("shutdown workspace service: %v", err)
		}
	}
	s.e.Logger.Info("workspace service is stopped")
	return nil
}

// CreateWorkspace godoc
// @Summary      Create workspace for workspace service
// @Description  Returns workspace created
// @Tags         workspace
// @Accept       json
// @Produce      json
// @Param        request  body  api.CreateWorkspaceRequest  true  "Create workspace request"
// @Success      200      {object}
// @Router       /workspace/api/v1/workspace [post]
func (s *Server) CreateWorkspace(c echo.Context) error {
	var request api.CreateWorkspaceRequest
	if err := c.Bind(&request); err != nil {
		c.Logger().Errorf("bind the request: %v", err)
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}
	if request.Name == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "name is empty")
	}
	if len(request.Name) > WorkspaceNameLenLimit {
		return echo.NewHTTPError(http.StatusBadRequest, "name length should be less than 20")
	}

	ownerId := c.Request().Header.Get(KeibiUserID)
	if ownerId == "" {
		return echo.NewHTTPError(http.StatusUnauthorized, "user id is empty")
	}

	workspace := &Workspace{
		WorkspaceId: uuid.New(),
		Name:        request.Name,
		OwnerId:     ownerId,
		Domain:      request.Name + s.cfg.DomainSuffix,
		Status:      StatusProvisioning.String(),
		Description: request.Description,
	}
	if err := s.db.CreateWorkspace(workspace); err != nil {
		if strings.Contains(err.Error(), "duplicate key value") {
			return echo.NewHTTPError(http.StatusFound, "workspace already exists")
		}
		c.Logger().Errorf("create workspace: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, ErrInternalServer)
	}
	return c.JSON(http.StatusOK, api.CreateWorkspaceResponse{
		WorkspaceId: workspace.WorkspaceId.String(),
	})
}

// DeleteWorkspace godoc
// @Summary      Delete workspace for workspace service
// @Description  Delete workspace with workspace id
// @Tags         workspace
// @Accept       json
// @Produce      json
// @Param        workspace_id  path  string  true  "Workspace ID"
// @Success      200
// @Router       /workspace/api/v1/workspace/:workspace_id [delete]
func (s *Server) DeleteWorkspace(c echo.Context) error {
	value := c.Param("workspace_id")
	if value == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "workspace id is empty")
	}
	workspaceId, err := uuid.Parse(value)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid workspace id")
	}

	ownerId := c.Request().Header.Get(KeibiUserID)
	if ownerId == "" {
		return echo.NewHTTPError(http.StatusUnauthorized, "user id is empty")
	}

	workspace, err := s.db.GetWorkspace(workspaceId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "workspace not found")
		}
		c.Logger().Errorf("find workspace: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, ErrInternalServer)
	}
	if workspace.OwnerId != ownerId {
		return echo.NewHTTPError(http.StatusForbidden, "operation is forbidden")
	}

	if err := s.db.UpdateWorkspaceStatus(workspaceId, StatusDeleting); err != nil {
		c.Logger().Errorf("delete workspace: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, ErrInternalServer)
	}
	return c.JSON(http.StatusOK, "success")
}

// ListWorkspaces godoc
// @Summary      List all workspaces with owner id
// @Description  Returns all workspaces with owner id
// @Tags         workspace
// @Accept       json
// @Produce      json
// @Success      200  {object}  []api.WorkspaceResponse
// @Router       /workspace/api/v1/workspaces [get]
func (s *Server) ListWorkspaces(c echo.Context) error {
	ownerId := c.Request().Header.Get(KeibiUserID)
	if ownerId == "" {
		return echo.NewHTTPError(http.StatusUnauthorized, "user id is empty")
	}

	dbWorkspaces, err := s.db.ListWorkspacesByOwner(ownerId)
	if err != nil {
		c.Logger().Errorf("list workspaces: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, ErrInternalServer)
	}
	workspaces := make([]*api.WorkspaceResponse, 0)
	for _, workspace := range dbWorkspaces {
		workspaces = append(workspaces, &api.WorkspaceResponse{
			WorkspaceId: workspace.WorkspaceId.String(),
			OwnerId:     workspace.OwnerId,
			Domain:      workspace.Domain,
			Name:        workspace.Name,
			Status:      workspace.Status,
			Description: workspace.Description,
			CreatedAt:   workspace.CreatedAt,
		})
	}
	return c.JSON(http.StatusOK, workspaces)
}
