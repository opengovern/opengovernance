package workspace

import (
	"errors"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/lithammer/shortuuid/v4"
	"gorm.io/gorm"
)

const (
	KeibiUserID           = "X-Keibi-UserID"
	WorkspaceDomainSuffix = ".app.keibi.io"

	StatusProvisioning       = "PROVISIONING"
	StatusProvisioned        = "PROVISIONED"
	StatusProvisioningFailed = "PROVISIONING_FAILED"
	StatusDeleting           = "DELETING"
	StatusDeleted            = "DELETED"
)

var (
	ErrInternalServer = errors.New("internal server error")
)

type HTTPServer struct {
	Address string
	DB      *Database
}

func NewServer(address string, db *Database) *HTTPServer {
	return &HTTPServer{
		Address: address,
		DB:      db,
	}
}

func (s *HTTPServer) Start() error {
	e := echo.New()

	e.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
		Format: "method=${method}, uri=${uri}, status=${status}\n",
	}))

	v1 := e.Group("/api/v1")

	v1.POST("/workspace", s.CreateWorkspace)
	v1.DELETE("/workspace/:workspace_id", s.DeleteWorkspace)
	v1.GET("/workspaces", s.ListWorkspaces)

	return e.Start(s.Address)
}

type CreateWorkspaceRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type CreateWorkspaceResponse struct {
	WorkspaceId string `json:"workspace_id"`
}

// CreateWorkspace godoc
// @Summary      Create workspace for workspace service
// @Description  Returns workspace created
// @Tags         workspace
// @Accept       json
// @Produce      json
// @Param        request  body  CreateWorkspaceRequest  true  "Create workspace request"
// @Success      200      {object}
// @Router       /workspace/api/v1/workspace [post]
func (s *HTTPServer) CreateWorkspace(c echo.Context) error {
	var request CreateWorkspaceRequest
	if err := c.Bind(&request); err != nil {
		c.Logger().Errorf("bind the request: %v", err)
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}
	if request.Name == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "name is empty")
	}
	name, err := uuid.Parse(request.Name)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid name")
	}

	ownerId := c.Request().Header.Get(KeibiUserID)
	if ownerId == "" {
		return echo.NewHTTPError(http.StatusUnauthorized, "user id is empty")
	}

	workspace := &Workspace{
		WorkspaceId: shortuuid.New(),
		Name:        name,
		OwnerId:     ownerId,
		Domain:      name.String() + WorkspaceDomainSuffix,
		Status:      StatusProvisioning,
		Description: request.Description,
	}
	if err := s.DB.CreateWorkspace(workspace); err != nil {
		c.Logger().Errorf("create workspace: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, ErrInternalServer)
	}
	return c.JSON(http.StatusOK, CreateWorkspaceResponse{
		WorkspaceId: workspace.WorkspaceId,
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
func (s *HTTPServer) DeleteWorkspace(c echo.Context) error {
	workspaceId := c.Param("workspace_id")
	if workspaceId == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "workspace id is empty")
	}

	ownerId := c.Request().Header.Get(KeibiUserID)
	if ownerId == "" {
		return echo.NewHTTPError(http.StatusUnauthorized, "user id is empty")
	}

	workspace, err := s.DB.GetWorkspace(workspaceId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return echo.NewHTTPError(http.StatusFound, "workspace not found")
		}
		c.Logger().Errorf("find workspace: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, ErrInternalServer)
	}
	if workspace.OwnerId != ownerId {
		return echo.NewHTTPError(http.StatusForbidden, "operation is forbidden")
	}

	if err := s.DB.UpdateWorkspaceStatus(workspaceId, StatusDeleting); err != nil {
		c.Logger().Errorf("delete workspace: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, ErrInternalServer)
	}
	return c.JSON(http.StatusOK, "success")
}

type WorkspaceResponse struct {
	WorkspaceId string    `json:"workspace_id"`
	Name        string    `json:"name"`
	OwnerId     string    `json:"owner_id"`
	Domain      string    `json:"domain"`
	Status      string    `json:"status"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

// ListWorkspaces godoc
// @Summary      List all workspaces with owner id
// @Description  Returns all workspaces with owner id
// @Tags         workspace
// @Accept       json
// @Produce      json
// @Success      200  {object}  []WorkspaceResponse
// @Router       /workspace/api/v1/workspaces [get]
func (s *HTTPServer) ListWorkspaces(c echo.Context) error {
	ownerId := c.Request().Header.Get(KeibiUserID)
	if ownerId == "" {
		return echo.NewHTTPError(http.StatusUnauthorized, "user id is empty")
	}

	dbWorkspaces, err := s.DB.ListWorkspacesByOwner(ownerId)
	if err != nil {
		c.Logger().Errorf("list workspaces: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, ErrInternalServer)
	}
	workspaces := make([]*WorkspaceResponse, 0)
	for _, workspace := range dbWorkspaces {
		workspaces = append(workspaces, &WorkspaceResponse{
			WorkspaceId: workspace.WorkspaceId,
			OwnerId:     workspace.OwnerId,
			Domain:      workspace.Domain,
			Name:        workspace.Name.String(),
			Status:      workspace.Status,
			Description: workspace.Description,
			CreatedAt:   workspace.CreatedAt,
		})
	}
	return c.JSON(http.StatusOK, workspaces)
}
