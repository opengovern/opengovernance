package workspace

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	apimeta "github.com/fluxcd/pkg/apis/meta"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
	"gitlab.com/keibiengine/keibi-engine/pkg/workspace/api"
	"gorm.io/gorm"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/util/validation"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	KeibiUserID = "X-Keibi-UserID"

	ReconcilerInterval = 30
)

var (
	ErrInternalServer = errors.New("internal server error")
)

type Server struct {
	e          *echo.Echo
	cfg        *Config
	db         *Database
	kubeClient client.Client // the kubernetes client
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

	kubeClient, err := s.newKubeClient()
	if err != nil {
		s.e.Logger.Fatalf("new kube client: %v", err)
	}
	s.kubeClient = kubeClient

	// init the http routers
	v1 := s.e.Group("/api/v1")
	v1.POST("/workspace", s.CreateWorkspace)
	v1.DELETE("/workspace/:workspace_id", s.DeleteWorkspace)
	v1.GET("/workspaces", s.ListWorkspaces)

	return s
}

func (s *Server) Start() error {
	go s.startReconciler()

	s.e.Logger.Infof("workspace service is started on %s", s.cfg.ServerAddr)
	return s.e.Start(s.cfg.ServerAddr)
}

func (s *Server) startReconciler() {
	ticker := time.NewTimer(time.Second * ReconcilerInterval)
	defer ticker.Stop()

	for range ticker.C {
		workspaces, err := s.db.ListWorkspaces()
		if err != nil {
			s.e.Logger.Errorf("list workspaces: %v", err)
		} else {
			for _, workspace := range workspaces {
				if err := s.handleWorkspace(&workspace); err != nil {
					s.e.Logger.Errorf("handle workspace %s: %v", workspace.ID.String(), err)
				}
			}
		}
		// reset the time ticker
		ticker.Reset(time.Second * ReconcilerInterval)
	}
}

func (s *Server) handleWorkspace(workspace *Workspace) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	status := WorkspaceStatus(workspace.Status)
	switch status {
	case StatusProvisioning:
		helmRelease, err := s.findHelmRelease(ctx, workspace)
		if err != nil {
			return fmt.Errorf("find helm release: %w", err)
		}
		if helmRelease == nil {
			if err := s.createHelmRelease(ctx, workspace); err != nil {
				return fmt.Errorf("create helm release: %w", err)
			}
			// update the workspace status next loop
			return nil
		}

		newStatus := status
		// check the status of helm release
		if meta.IsStatusConditionTrue(helmRelease.Status.Conditions, apimeta.ReadyCondition) {
			newStatus = StatusProvisioned
		} else if meta.IsStatusConditionTrue(helmRelease.Status.Conditions, apimeta.StalledCondition) {
			newStatus = StatusProvisioningFailed
		}
		if newStatus != status {
			if err := s.db.UpdateWorkspaceStatus(workspace.ID, newStatus); err != nil {
				return fmt.Errorf("update workspace status: %w", err)
			}
		}
	case StatusDeleting:
		helmRelease, err := s.findHelmRelease(ctx, workspace)
		if err != nil {
			return fmt.Errorf("find helm release: %w", err)
		}
		if helmRelease != nil {
			if err := s.deleteHelmRelease(ctx, workspace); err != nil {
				return fmt.Errorf("delete helm release: %w", err)
			}
			// update the workspace status next loop
			return nil
		}
		if err := s.db.UpdateWorkspaceStatus(workspace.ID, StatusDeleted); err != nil {
			return fmt.Errorf("update workspace status: %w", err)
		}
	}
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

	domain := strings.ToLower(request.Name + s.cfg.DomainSuffix)
	if errors := validation.IsQualifiedName(domain); len(errors) > 0 {
		c.Logger().Errorf("invalid domain: %v", errors)
		return echo.NewHTTPError(http.StatusBadRequest, errors)
	}

	ownerId := c.Request().Header.Get(KeibiUserID)
	if ownerId == "" {
		return echo.NewHTTPError(http.StatusUnauthorized, "user id is empty")
	}

	workspace := &Workspace{
		ID:          uuid.New(),
		Name:        strings.ToLower(request.Name),
		OwnerId:     ownerId,
		Domain:      domain,
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
		ID: workspace.ID.String(),
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
	id, err := uuid.Parse(value)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid workspace id")
	}

	ownerId := c.Request().Header.Get(KeibiUserID)
	if ownerId == "" {
		return echo.NewHTTPError(http.StatusUnauthorized, "user id is empty")
	}

	workspace, err := s.db.GetWorkspace(id)
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

	if err := s.db.UpdateWorkspaceStatus(id, StatusDeleting); err != nil {
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
	workspaces := make([]*api.WorkspaceResponse, len(dbWorkspaces))
	for _, workspace := range dbWorkspaces {
		workspaces = append(workspaces, &api.WorkspaceResponse{
			ID:          workspace.ID.String(),
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
