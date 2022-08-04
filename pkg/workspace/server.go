package workspace

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	contourv1 "github.com/projectcontour/contour/apis/projectcontour/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	apimeta "github.com/fluxcd/pkg/apis/meta"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	authapi "gitlab.com/keibiengine/keibi-engine/pkg/auth/api"
	"gitlab.com/keibiengine/keibi-engine/pkg/auth/client"
	"gitlab.com/keibiengine/keibi-engine/pkg/internal/httpclient"
	"gitlab.com/keibiengine/keibi-engine/pkg/internal/httpserver"
	"gitlab.com/keibiengine/keibi-engine/pkg/workspace/api"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/util/validation"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	reconcilerInterval = 30 * time.Second
)

var (
	ErrInternalServer = errors.New("internal server error")
)

type Server struct {
	e          *echo.Echo
	cfg        *Config
	db         *Database
	kubeClient k8sclient.Client // the kubernetes client
}

func NewServer(cfg *Config) (*Server, error) {
	s := &Server{
		cfg: cfg,
	}

	logger, err := zap.NewProduction()
	if err != nil {
		return nil, fmt.Errorf("new zap logger: %s", err)
	}
	s.e = httpserver.Register(logger, s)

	db, err := NewDatabase(cfg, logger)
	if err != nil {
		return nil, fmt.Errorf("new database: %w", err)
	}
	s.db = db

	kubeClient, err := s.newKubeClient()
	if err != nil {
		return nil, fmt.Errorf("new kube client: %w", err)
	}
	s.kubeClient = kubeClient

	err = contourv1.AddToScheme(s.kubeClient.Scheme())
	if err != nil {
		return nil, fmt.Errorf("add contourv1 to scheme: %w", err)
	}

	return s, nil
}

func (s *Server) Register(e *echo.Echo) {
	v1 := e.Group("/api/v1")

	v1.POST("/workspace", s.CreateWorkspace)
	v1.DELETE("/workspace/:workspace_id", s.DeleteWorkspace)
	v1.GET("/workspaces", s.ListWorkspaces)
}

func (s *Server) Start() error {
	go s.startReconciler()

	s.e.Logger.Infof("workspace service is started on %s", s.cfg.ServerAddr)
	return s.e.Start(s.cfg.ServerAddr)
}

func (s *Server) startReconciler() {
	ticker := time.NewTimer(reconcilerInterval)
	defer ticker.Stop()

	for range ticker.C {
		workspaces, err := s.db.ListWorkspaces()
		if err != nil {
			s.e.Logger.Errorf("list workspaces: %v", err)
		} else {
			for _, workspace := range workspaces {
				if err := s.handleWorkspace(workspace); err != nil {
					s.e.Logger.Errorf("handle workspace %s: %v", workspace.ID.String(), err)
				}
			}

			if err := s.syncHTTPProxy(workspaces); err != nil {
				s.e.Logger.Errorf("syncing http proxy: %v", err)
			}
		}
		// reset the time ticker
		ticker.Reset(reconcilerInterval)
	}
}

func (s *Server) syncHTTPProxy(workspaces []*Workspace) error {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	var includes []contourv1.Include
	for _, w := range workspaces {
		includes = append(includes, contourv1.Include{
			Name:      "http-proxy-route",
			Namespace: w.ID.String(),
			Conditions: []contourv1.MatchCondition{
				{
					Prefix: "/" + w.Name,
				},
			},
		})
	}

	err := s.kubeClient.Create(ctx, &contourv1.HTTPProxy{
		TypeMeta: metav1.TypeMeta{
			Kind:       "HTTPProxy",
			APIVersion: "projectcontour.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "http-proxy-route",
			Namespace: OctopusNamespace,
		},
		Spec: contourv1.HTTPProxySpec{
			Includes: includes,
		},
		Status: contourv1.HTTPProxyStatus{},
	})
	if err != nil {
		return err
	}

	return nil
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
			s.e.Logger.Infof("create helm release %s with status %s", workspace.ID.String(), workspace.Status)
			if err := s.createHelmRelease(ctx, workspace); err != nil {
				return fmt.Errorf("create helm release: %w", err)
			}
			// update the workspace status next loop
			return nil
		}

		newStatus := status
		// check the status of helm release
		if meta.IsStatusConditionTrue(helmRelease.Status.Conditions, apimeta.ReadyCondition) {
			// when the helm release installed successfully, set the rolebinding
			authClient := client.NewAuthServiceClient(s.cfg.AuthBaseUrl)
			authCtx := &httpclient.Context{
				UserID:        workspace.OwnerId.String(),
				UserRole:      authapi.AdminRole,
				WorkspaceName: workspace.Name,
			}
			if err := authClient.PutRoleBinding(authCtx, &authapi.PutRoleBindingRequest{
				UserID: workspace.OwnerId,
				Role:   authapi.AdminRole,
			}); err != nil {
				return fmt.Errorf("put role binding: %w", err)
			}

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
			s.e.Logger.Infof("delete helm release %s with status %s", workspace.ID.String(), workspace.Status)
			if err := s.deleteHelmRelease(ctx, workspace); err != nil {
				return fmt.Errorf("delete helm release: %w", err)
			}
			// update the workspace status next loop
			return nil
		}

		namespace, err := s.findTargetNamespace(ctx, workspace.ID.String())
		if err != nil {
			return fmt.Errorf("find target namespace: %w", err)
		}
		if namespace != nil {
			s.e.Logger.Infof("delete target namespace %s with status %s", workspace.ID.String(), workspace.Status)
			if err := s.deleteTargetNamespace(ctx, workspace.ID.String()); err != nil {
				return fmt.Errorf("delete target namespace: %w", err)
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
// @Param        request  body      api.CreateWorkspaceRequest  true  "Create workspace request"
// @Success      200      {object}  api.CreateWorkspaceResponse
// @Router       /workspace/api/v1/workspace [post]
func (s *Server) CreateWorkspace(c echo.Context) error {
	userID := httpserver.GetUserID(c)

	var request api.CreateWorkspaceRequest
	if err := c.Bind(&request); err != nil {
		c.Logger().Errorf("bind the request: %v", err)
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}
	if request.Name == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "name is empty")
	}
	if request.Name == "keibi" {
		return echo.NewHTTPError(http.StatusBadRequest, "name cannot be keibi")
	}
	if strings.Contains(request.Name, ".") {
		return echo.NewHTTPError(http.StatusBadRequest, "name is invalid")
	}

	domain := strings.ToLower(request.Name + s.cfg.DomainSuffix)
	if errors := validation.IsQualifiedName(domain); len(errors) > 0 {
		c.Logger().Errorf("invalid domain: %v", errors)
		return echo.NewHTTPError(http.StatusBadRequest, errors)
	}

	workspace := &Workspace{
		ID:          uuid.New(),
		Name:        strings.ToLower(request.Name),
		OwnerId:     userID,
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
	userID := httpserver.GetUserID(c)

	value := c.Param("workspace_id")
	if value == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "workspace id is empty")
	}
	id, err := uuid.Parse(value)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid workspace id")
	}

	workspace, err := s.db.GetWorkspace(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "workspace not found")
		}
		c.Logger().Errorf("find workspace: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, ErrInternalServer)
	}
	if workspace.OwnerId != userID {
		return echo.NewHTTPError(http.StatusForbidden, "operation is forbidden")
	}

	if err := s.db.UpdateWorkspaceStatus(id, StatusDeleting); err != nil {
		c.Logger().Errorf("delete workspace: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, ErrInternalServer)
	}
	return c.JSON(http.StatusOK, map[string]string{"message": "success"})
}

// ListWorkspaces godoc
// @Summary      List all workspaces with owner id
// @Description  Returns all workspaces with owner id
// @Tags         workspace
// @Accept       json
// @Produce      json
// @Success      200  {array}  []api.WorkspaceResponse
// @Router       /workspace/api/v1/workspaces [get]
func (s *Server) ListWorkspaces(c echo.Context) error {
	userID := httpserver.GetUserID(c)

	dbWorkspaces, err := s.db.ListWorkspacesByOwner(userID)
	if err != nil {
		c.Logger().Errorf("list workspaces: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, ErrInternalServer)
	}
	workspaces := make([]*api.WorkspaceResponse, 0)
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
