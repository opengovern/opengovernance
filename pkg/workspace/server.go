package workspace

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"

	contourv1 "github.com/projectcontour/contour/apis/projectcontour/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	apimeta "github.com/fluxcd/pkg/apis/meta"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	authapi "gitlab.com/keibiengine/keibi-engine/pkg/auth/api"
	authclient "gitlab.com/keibiengine/keibi-engine/pkg/auth/client"
	"gitlab.com/keibiengine/keibi-engine/pkg/internal/httpclient"
	"gitlab.com/keibiengine/keibi-engine/pkg/internal/httpserver"
	"gitlab.com/keibiengine/keibi-engine/pkg/workspace/api"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"k8s.io/apimachinery/pkg/api/meta"
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
	authClient authclient.AuthServiceClient
	kubeClient k8sclient.Client // the kubernetes client
	rdb        *redis.Client
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

	s.authClient = authclient.NewAuthServiceClient(cfg.AuthBaseUrl)

	s.rdb = redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddress,
		Password: "", // no password set
		DB:       0,  // use default DB
	})

	return s, nil
}

func (s *Server) Register(e *echo.Echo) {
	v1 := e.Group("/api/v1")

	v1.POST("/workspace", s.CreateWorkspace)
	v1.DELETE("/workspace/:workspace_id", s.DeleteWorkspace)
	v1.GET("/workspaces/limits/:workspace_name", s.GetWorkspaceLimits)
	v1.GET("/workspaces/limits/byid/:workspace_id", s.GetWorkspaceLimitsByID)
	v1.GET("/workspaces", s.ListWorkspaces)
}

func (s *Server) Start() error {
	go s.startReconciler()

	s.e.Logger.Infof("workspace service is started on %s", s.cfg.ServerAddr)
	return s.e.Start(s.cfg.ServerAddr)
}

func (s *Server) startReconciler() {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("reconciler crashed: %v, restarting ...\n", r)
			go s.startReconciler()
		}
	}()

	ticker := time.NewTimer(reconcilerInterval)
	defer ticker.Stop()

	for range ticker.C {
		fmt.Printf("reconsiler started\n")

		workspaces, err := s.db.ListWorkspaces()
		if err != nil {
			s.e.Logger.Errorf("list workspaces: %v", err)
		} else {
			for _, workspace := range workspaces {
				if err := s.handleWorkspace(workspace); err != nil {
					s.e.Logger.Errorf("handle workspace %s: %v", workspace.ID.String(), err)
				}

				if err := s.handleAutoSuspend(workspace); err != nil {
					s.e.Logger.Errorf("handleAutoSuspend: %v", err)
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

func (s *Server) handleAutoSuspend(workspace *Workspace) error {
	if workspace.Tier != Tier_Free {
		return nil
	}

	if workspace.Status != string(StatusProvisioned) {
		return nil
	}

	fmt.Printf("checking for auto-suspend %s\n", workspace.Name)

	res, err := s.rdb.Get(context.Background(), "last_access_"+workspace.Name).Result()
	if err != nil {
		if err != redis.Nil {
			return fmt.Errorf("get last access: %v", err)
		}
	}

	lastAccess, _ := strconv.ParseInt(res, 10, 64)
	fmt.Printf("last access: %d [%s]\n", lastAccess, res)
	if time.Now().Unix()-lastAccess > int64(s.cfg.AutoSuspendDuration) {
		fmt.Printf("suspending workspace %s\n", workspace.Name)
		if err := s.db.UpdateWorkspaceStatus(workspace.ID, StatusSuspending); err != nil {
			return fmt.Errorf("update workspace status: %w", err)
		}
	}
	return nil
}

func (s *Server) syncHTTPProxy(workspaces []*Workspace) error {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	var includes []contourv1.Include
	for _, w := range workspaces {
		if w.Status != string(StatusProvisioned) {
			continue
		}
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

	key := types.NamespacedName{
		Name:      "http-proxy-route",
		Namespace: OctopusNamespace,
	}
	var httpProxy contourv1.HTTPProxy

	exists := true
	if err := s.kubeClient.Get(ctx, key, &httpProxy); err != nil {
		if apierrors.IsNotFound(err) {
			exists = false
		} else {
			return err
		}
	}

	resourceVersion := httpProxy.GetResourceVersion()
	httpProxy = contourv1.HTTPProxy{
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
	}

	if exists {
		httpProxy.SetResourceVersion(resourceVersion)
		err := s.kubeClient.Update(ctx, &httpProxy)
		if err != nil {
			return err
		}
	} else {
		err := s.kubeClient.Create(ctx, &httpProxy)
		if err != nil {
			return err
		}
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
			limits := GetLimitsByTier(workspace.Tier)
			authCtx := &httpclient.Context{
				UserID:         workspace.OwnerId.String(),
				UserRole:       authapi.AdminRole,
				WorkspaceName:  workspace.Name,
				MaxUsers:       limits.MaxUsers,
				MaxConnections: limits.MaxConnections,
				MaxResources:   limits.MaxResources,
			}

			if err := s.authClient.PutRoleBinding(authCtx, &authapi.PutRoleBindingRequest{
				UserID: workspace.OwnerId,
				Role:   authapi.AdminRole,
			}); err != nil {
				return fmt.Errorf("put role binding: %w", err)
			}

			err = s.rdb.SetEX(context.Background(), "last_access_"+workspace.Name, time.Now().UnixMilli(), s.cfg.AutoSuspendDuration).Err()
			if err != nil {
				return fmt.Errorf("set last access: %v", err)
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
	case StatusSuspending:
		helmRelease, err := s.findHelmRelease(ctx, workspace)
		if err != nil {
			return fmt.Errorf("find helm release: %w", err)
		}
		if helmRelease == nil {
			return fmt.Errorf("cannot find helmrelease")
		}

		values := helmRelease.GetValues()
		currentReplicaCount, err := getReplicaCount(values)
		if err != nil {
			return fmt.Errorf("getReplicaCount: %w", err)
		}

		if currentReplicaCount != 0 {
			values, err = updateValuesSetReplicaCount(values, 0)
			if err != nil {
				return fmt.Errorf("updateValuesSetReplicaCount: %w", err)
			}

			b, err := json.Marshal(values)
			if err != nil {
				return fmt.Errorf("marshalling values: %w", err)
			}
			helmRelease.Spec.Values.Raw = b
			err = s.kubeClient.Update(ctx, helmRelease)
			if err != nil {
				return fmt.Errorf("updating replica count: %w", err)
			}
			return nil
		}

		var pods corev1.PodList
		err = s.kubeClient.List(ctx, &pods, k8sclient.InNamespace(workspace.ID.String()))
		if err != nil {
			return fmt.Errorf("fetching list of pods: %w", err)
		}

		if len(pods.Items) > 0 {
			// waiting for pods to go down
			return nil
		}

		if err := s.db.UpdateWorkspaceStatus(workspace.ID, StatusSuspended); err != nil {
			return fmt.Errorf("update workspace status: %w", err)
		}
	}
	return nil
}

// CreateWorkspace godoc
// @Summary      Create workspace for workspace service
// @Description  Returns workspace created
// @Tags     workspace
// @Accept   json
// @Produce  json
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
	if request.Name == "keibi" || request.Name == "workspaces" {
		return echo.NewHTTPError(http.StatusBadRequest, "name cannot be keibi or workspaces")
	}
	if strings.Contains(request.Name, ".") {
		return echo.NewHTTPError(http.StatusBadRequest, "name is invalid")
	}

	switch request.Tier {
	case string(Tier_Free), string(Tier_Teams):
	default:
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid Tier")
	}
	uri := strings.ToLower("https://" + s.cfg.DomainSuffix + "/" + request.Name)
	workspace := &Workspace{
		ID:          uuid.New(),
		Name:        strings.ToLower(request.Name),
		OwnerId:     userID,
		URI:         uri,
		Status:      StatusProvisioning.String(),
		Description: request.Description,
		Tier:        Tier(request.Tier),
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
	userId := httpserver.GetUserID(c)
	resp, err := s.authClient.GetUserRoleBindings(httpclient.FromEchoContext(c))
	if err != nil {
		return fmt.Errorf("GetUserRoleBindings: %v", err)
	}

	dbWorkspaces, err := s.db.ListWorkspaces()
	if err != nil {
		return fmt.Errorf("ListWorkspaces: %v", err)
	}

	workspaces := make([]*api.WorkspaceResponse, 0)
	for _, workspace := range dbWorkspaces {
		if workspace.Status == string(StatusDeleted) {
			continue
		}

		hasRoleInWorkspace := false
		for _, rb := range resp {
			if rb.WorkspaceName == workspace.Name {
				hasRoleInWorkspace = true
			}
		}

		if workspace.OwnerId != userId && !hasRoleInWorkspace {
			continue
		}

		workspaces = append(workspaces, &api.WorkspaceResponse{
			ID:          workspace.ID.String(),
			OwnerId:     workspace.OwnerId,
			URI:         workspace.URI,
			Name:        workspace.Name,
			Tier:        string(workspace.Tier),
			Status:      workspace.Status,
			Description: workspace.Description,
			CreatedAt:   workspace.CreatedAt,
		})
	}
	return c.JSON(http.StatusOK, workspaces)
}

// ChangeOwnership godoc
// @Summary  Change ownership of workspace
// @Tags     workspace
// @Accept   json
// @Produce  json
// @Param    request  body  api.ChangeWorkspaceOwnershipRequest  true  "Change ownership request"
// @Router   /workspace/api/v1/workspace/{workspace_id}/owner [post]
func (s *Server) ChangeOwnership(c echo.Context) error {
	userID := httpserver.GetUserID(c)
	workspaceID := c.Param("workspace_id")

	var request api.ChangeWorkspaceOwnershipRequest
	if err := c.Bind(&request); err != nil {
		c.Logger().Errorf("bind the request: %v", err)
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}
	if workspaceID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "workspace id is empty")
	}

	workspaceUUID, err := uuid.Parse(workspaceID)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid workspace id")
	}

	w, err := s.db.GetWorkspace(workspaceUUID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "workspace not found")
		}
		return err
	}

	if w.OwnerId != userID {
		return echo.NewHTTPError(http.StatusForbidden, "operation is forbidden")
	}

	err = s.db.UpdateWorkspaceOwner(workspaceUUID, request.NewOwnerUserID)
	if err != nil {
		return err
	}

	return c.NoContent(http.StatusOK)
}

// GetWorkspaceLimits godoc
// @Summary  Get workspace limits
// @Tags         workspace
// @Accept       json
// @Produce      json
// @Param    workspace_name  path     string  true  "Workspace Name"
// @Success  200             {array}  api.WorkspaceLimits
// @Router   /workspace/api/v1/workspaces/limits/{workspace_name} [get]
func (s *Server) GetWorkspaceLimits(c echo.Context) error {
	workspaceName := c.Param("workspace_name")
	dbWorkspace, err := s.db.GetWorkspaceByName(workspaceName)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, GetLimitsByTier(dbWorkspace.Tier))
}

// GetWorkspaceLimitsByID godoc
// @Summary  Get workspace limits
// @Tags         workspace
// @Accept       json
// @Produce      json
// @Param    workspace_id  path     string  true  "Workspace Name"
// @Success  200             {array}  api.WorkspaceLimits
// @Router   /workspace/api/v1/workspaces/limits/byid/{workspace_id} [get]
func (s *Server) GetWorkspaceLimitsByID(c echo.Context) error {
	workspaceID := c.Param("workspace_id")
	workspaceUUID, err := uuid.Parse(workspaceID)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid workspace id")
	}

	dbWorkspace, err := s.db.GetWorkspace(workspaceUUID)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, GetLimitsByTier(dbWorkspace.Tier))
}
