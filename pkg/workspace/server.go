package workspace

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	aws2 "github.com/kaytu-io/kaytu-aws-describer/aws"
	"github.com/kaytu-io/kaytu-engine/pkg/internal/httpclient"
	httpserver2 "github.com/kaytu-io/kaytu-engine/pkg/internal/httpserver"

	"github.com/kaytu-io/kaytu-util/pkg/source"

	"github.com/go-redis/cache/v8"
	"github.com/kaytu-io/kaytu-engine/pkg/onboard/client"

	client2 "github.com/kaytu-io/kaytu-engine/pkg/inventory/client"

	v1 "k8s.io/api/apps/v1"

	"github.com/labstack/gommon/log"

	corev1 "k8s.io/api/core/v1"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"

	contourv1 "github.com/projectcontour/contour/apis/projectcontour/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	apimeta "github.com/fluxcd/pkg/apis/meta"
	"github.com/go-redis/redis/v8"
	authapi "github.com/kaytu-io/kaytu-engine/pkg/auth/api"
	authclient "github.com/kaytu-io/kaytu-engine/pkg/auth/client"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/api"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"k8s.io/apimachinery/pkg/api/meta"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/sony/sonyflake"
)

const (
	reconcilerInterval = 30 * time.Second
)

var (
	ErrInternalServer = errors.New("internal server error")
)

type Server struct {
	logger               *zap.Logger
	e                    *echo.Echo
	cfg                  *Config
	db                   *Database
	authClient           authclient.AuthServiceClient
	kubeClient           k8sclient.Client // the kubernetes client
	rdb                  *redis.Client
	cache                *cache.Cache
	dockerRegistryConfig string
	awsConfig            aws.Config
}

func NewServer(cfg *Config) (*Server, error) {
	s := &Server{
		cfg: cfg,
	}

	logger, err := zap.NewProduction()
	if err != nil {
		return nil, fmt.Errorf("new zap logger: %s", err)
	}
	s.e = httpserver2.Register(logger, s)

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

	err = v1.AddToScheme(s.kubeClient.Scheme())
	if err != nil {
		return nil, fmt.Errorf("add v1 to scheme: %w", err)
	}

	s.authClient = authclient.NewAuthServiceClient(cfg.AuthBaseUrl)

	s.rdb = redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddress,
		Password: "", // no password set
		DB:       0,  // use default DB
	})

	s.cache = cache.New(&cache.Options{
		Redis:      s.rdb,
		LocalCache: cache.NewTinyLFU(2000, 1*time.Minute),
	})

	secretKey := types.NamespacedName{
		Name:      "registry",
		Namespace: s.cfg.KeibiOctopusNamespace,
	}
	var registrySecret corev1.Secret
	err = s.kubeClient.Get(context.Background(), secretKey, &registrySecret)
	if err != nil {
		return nil, err
	}
	s.dockerRegistryConfig = base64.StdEncoding.EncodeToString(registrySecret.Data[".dockerconfigjson"])

	s.awsConfig, err = aws2.GetConfig(context.Background(), cfg.S3AccessKey, cfg.S3SecretKey, "", "", nil)
	if err != nil {
		return nil, err
	}

	s.logger = logger

	return s, nil
}

func (s *Server) Register(e *echo.Echo) {
	v1Group := e.Group("/api/v1")

	workspaceGroup := v1Group.Group("/workspace")
	workspaceGroup.POST("", httpserver2.AuthorizeHandler(s.CreateWorkspace, authapi.EditorRole))
	workspaceGroup.DELETE("/:workspace_id", httpserver2.AuthorizeHandler(s.DeleteWorkspace, authapi.EditorRole))
	workspaceGroup.POST("/:workspace_id/suspend", httpserver2.AuthorizeHandler(s.SuspendWorkspace, authapi.EditorRole))
	workspaceGroup.POST("/:workspace_id/resume", httpserver2.AuthorizeHandler(s.ResumeWorkspace, authapi.EditorRole))
	workspaceGroup.GET("/current", httpserver2.AuthorizeHandler(s.GetCurrentWorkspace, authapi.ViewerRole))
	workspaceGroup.POST("/:workspace_id/owner", httpserver2.AuthorizeHandler(s.ChangeOwnership, authapi.EditorRole))
	workspaceGroup.POST("/:workspace_id/name", httpserver2.AuthorizeHandler(s.ChangeName, authapi.KeibiAdminRole))
	workspaceGroup.POST("/:workspace_id/tier", httpserver2.AuthorizeHandler(s.ChangeTier, authapi.KeibiAdminRole))
	workspaceGroup.POST("/:workspace_id/organization", httpserver2.AuthorizeHandler(s.ChangeOrganization, authapi.KeibiAdminRole))

	workspacesGroup := v1Group.Group("/workspaces")
	workspacesGroup.GET("/limits/:workspace_name", httpserver2.AuthorizeHandler(s.GetWorkspaceLimits, authapi.ViewerRole))
	workspacesGroup.GET("/limits/byid/:workspace_id", httpserver2.AuthorizeHandler(s.GetWorkspaceLimitsByID, authapi.ViewerRole))
	workspacesGroup.GET("/byid/:workspace_id", httpserver2.AuthorizeHandler(s.GetWorkspaceByID, authapi.ViewerRole))
	workspacesGroup.GET("", httpserver2.AuthorizeHandler(s.ListWorkspaces, authapi.ViewerRole))
	workspacesGroup.GET("/:workspace_id", httpserver2.AuthorizeHandler(s.GetWorkspace, authapi.ViewerRole))

	organizationGroup := v1Group.Group("/organization")
	organizationGroup.POST("", httpserver2.AuthorizeHandler(s.CreateOrganization, authapi.EditorRole))
	organizationGroup.DELETE("/:organizationId", httpserver2.AuthorizeHandler(s.DeleteOrganization, authapi.EditorRole))
}

func (s *Server) Start() error {
	go s.startReconciler()

	s.e.Logger.SetLevel(log.DEBUG)
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
					s.e.Logger.Errorf("handle workspace %s: %v", workspace.ID, err)
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
	if workspace.Tier != api.Tier_Free {
		return nil
	}
	switch api.WorkspaceStatus(workspace.Status) {
	case api.StatusDeleting, api.StatusDeleted:
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

	if time.Now().UnixMilli()-lastAccess > s.cfg.AutoSuspendDuration.Milliseconds() {
		if workspace.Status == api.StatusProvisioned {
			fmt.Printf("suspending workspace %s\n", workspace.Name)
			if err := s.db.UpdateWorkspaceStatus(workspace.ID, api.StatusSuspending); err != nil {
				return fmt.Errorf("update workspace status: %w", err)
			}
		}
	} /* else {
		if workspace.Status == string(StatusSuspended) {
			fmt.Printf("resuming workspace %s\n", workspace.Name)
			if err := s.db.UpdateWorkspaceStatus(workspace.ID, StatusProvisioning); err != nil {
				return fmt.Errorf("update workspace status: %w", err)
			}
		}
	}*/
	return nil
}

func (s *Server) syncHTTPProxy(workspaces []*Workspace) error {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	var httpIncludes []contourv1.Include
	var grpcIncludes []contourv1.Include
	for _, w := range workspaces {
		if w.Status != api.StatusProvisioned {
			continue
		}
		httpIncludes = append(httpIncludes, contourv1.Include{
			Name:      "http-proxy-route",
			Namespace: w.ID,
			Conditions: []contourv1.MatchCondition{
				{
					Prefix: "/" + w.Name,
				},
			},
		})
		grpcIncludes = append(grpcIncludes, contourv1.Include{
			Name:      "grpc-proxy-route",
			Namespace: w.ID,
			Conditions: []contourv1.MatchCondition{
				{
					Header: &contourv1.HeaderMatchCondition{
						Name:    "workspace-name",
						Present: true,
						Exact:   w.Name,
					},
				},
			},
		})
	}

	httpKey := types.NamespacedName{
		Name:      "http-proxy-route",
		Namespace: s.cfg.KeibiOctopusNamespace,
	}
	var httpProxy contourv1.HTTPProxy

	grpcKey := types.NamespacedName{
		Name:      "grpc-proxy-route",
		Namespace: s.cfg.KeibiOctopusNamespace,
	}
	var grpcProxy contourv1.HTTPProxy

	httpExists := true
	if err := s.kubeClient.Get(ctx, httpKey, &httpProxy); err != nil {
		if apierrors.IsNotFound(err) {
			httpExists = false
		} else {
			return err
		}
	}

	grpcExists := true
	if err := s.kubeClient.Get(ctx, grpcKey, &grpcProxy); err != nil {
		if apierrors.IsNotFound(err) {
			grpcExists = false
		} else {
			return err
		}
	}

	httpResourceVersion := httpProxy.GetResourceVersion()
	httpProxy = contourv1.HTTPProxy{
		TypeMeta: metav1.TypeMeta{
			Kind:       "HTTPProxy",
			APIVersion: "projectcontour.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "http-proxy-route",
			Namespace: s.cfg.KeibiOctopusNamespace,
		},
		Spec: contourv1.HTTPProxySpec{
			Includes: httpIncludes,
		},
		Status: contourv1.HTTPProxyStatus{},
	}

	grpcResourceVersion := grpcProxy.GetResourceVersion()
	grpcProxy = contourv1.HTTPProxy{
		TypeMeta: metav1.TypeMeta{
			Kind:       "HTTPProxy",
			APIVersion: "projectcontour.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "grpc-proxy-route",
			Namespace: s.cfg.KeibiOctopusNamespace,
		},
		Spec: contourv1.HTTPProxySpec{
			Includes: grpcIncludes,
		},
		Status: contourv1.HTTPProxyStatus{},
	}

	if httpExists {
		httpProxy.SetResourceVersion(httpResourceVersion)
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

	if grpcExists {
		grpcProxy.SetResourceVersion(grpcResourceVersion)
		err := s.kubeClient.Update(ctx, &grpcProxy)
		if err != nil {
			return err
		}
	} else {
		err := s.kubeClient.Create(ctx, &grpcProxy)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Server) handleWorkspace(workspace *Workspace) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	status := api.WorkspaceStatus(workspace.Status)
	switch status {
	case api.StatusProvisioning:
		helmRelease, err := s.findHelmRelease(ctx, workspace)
		if err != nil {
			return fmt.Errorf("find helm release: %w", err)
		}
		if helmRelease == nil {
			s.e.Logger.Infof("create helm release %s with status %s", workspace.ID, workspace.Status)
			if err := s.createHelmRelease(ctx, workspace, s.dockerRegistryConfig); err != nil {
				return fmt.Errorf("create helm release: %w", err)
			}
			// update the workspace status next loop
			return nil
		}

		values := helmRelease.GetValues()
		currentReplicaCount, err := getReplicaCount(values)
		if err != nil {
			return fmt.Errorf("getReplicaCount: %w", err)
		}

		if currentReplicaCount == 0 {
			values, err = updateValuesSetReplicaCount(values, 1)
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

		newStatus := status
		// check the status of helm release
		if meta.IsStatusConditionTrue(helmRelease.Status.Conditions, apimeta.ReadyCondition) {
			// when the helm release installed successfully, set the rolebinding
			limits := api.GetLimitsByTier(workspace.Tier)
			authCtx := &httpclient.Context{
				UserID:         workspace.OwnerId,
				UserRole:       authapi.AdminRole,
				WorkspaceName:  workspace.Name,
				WorkspaceID:    workspace.ID,
				MaxUsers:       limits.MaxUsers,
				MaxConnections: limits.MaxConnections,
				MaxResources:   limits.MaxResources,
			}

			if err := s.authClient.PutRoleBinding(authCtx, &authapi.PutRoleBindingRequest{
				UserID:   workspace.OwnerId,
				RoleName: authapi.AdminRole,
			}); err != nil {
				return fmt.Errorf("put role binding: %w", err)
			}

			err = s.rdb.SetEX(context.Background(), "last_access_"+workspace.Name, time.Now().UnixMilli(), s.cfg.AutoSuspendDuration).Err()
			if err != nil {
				return fmt.Errorf("set last access: %v", err)
			}

			newStatus = api.StatusProvisioned
		} else if meta.IsStatusConditionFalse(helmRelease.Status.Conditions, apimeta.ReadyCondition) {
			if !helmRelease.Spec.Suspend {
				helmRelease.Spec.Suspend = true
				err = s.kubeClient.Update(ctx, helmRelease)
				if err != nil {
					return fmt.Errorf("suspend helmrelease: %v", err)
				}
			} else {
				helmRelease.Spec.Suspend = false
				err = s.kubeClient.Update(ctx, helmRelease)
				if err != nil {
					return fmt.Errorf("suspend helmrelease: %v", err)
				}
			}
			newStatus = api.StatusProvisioning
		} else if meta.IsStatusConditionTrue(helmRelease.Status.Conditions, apimeta.StalledCondition) {
			newStatus = api.StatusProvisioningFailed
		}
		if newStatus != status {
			if err := s.db.UpdateWorkspaceStatus(workspace.ID, newStatus); err != nil {
				return fmt.Errorf("update workspace status: %w", err)
			}
		}
	case api.StatusProvisioningFailed:
		helmRelease, err := s.findHelmRelease(ctx, workspace)
		if err != nil {
			return fmt.Errorf("find helm release: %w", err)
		}
		if helmRelease == nil {
			return nil
		}

		newStatus := status
		// check the status of helm release
		if meta.IsStatusConditionTrue(helmRelease.Status.Conditions, apimeta.ReadyCondition) {
			newStatus = api.StatusProvisioning
		} else if meta.IsStatusConditionFalse(helmRelease.Status.Conditions, apimeta.ReadyCondition) {
			newStatus = api.StatusProvisioning
		} else if meta.IsStatusConditionTrue(helmRelease.Status.Conditions, apimeta.StalledCondition) {
			newStatus = api.StatusProvisioningFailed
		}
		if newStatus != status {
			if err := s.db.UpdateWorkspaceStatus(workspace.ID, newStatus); err != nil {
				return fmt.Errorf("update workspace status: %w", err)
			}
		}
	case api.StatusDeleting:
		helmRelease, err := s.findHelmRelease(ctx, workspace)
		if err != nil {
			return fmt.Errorf("find helm release: %w", err)
		}

		if helmRelease != nil {
			s.e.Logger.Infof("delete helm release %s with status %s", workspace.ID, workspace.Status)
			if err := s.deleteHelmRelease(ctx, workspace); err != nil {
				return fmt.Errorf("delete helm release: %w", err)
			}
			// update the workspace status next loop
			return nil
		}

		namespace, err := s.findTargetNamespace(ctx, workspace.ID)
		if err != nil {
			return fmt.Errorf("find target namespace: %w", err)
		}
		if namespace != nil {
			s.e.Logger.Infof("delete target namespace %s with status %s", workspace.ID, workspace.Status)
			if err := s.deleteTargetNamespace(ctx, workspace.ID); err != nil {
				return fmt.Errorf("delete target namespace: %w", err)
			}
			// update the workspace status next loop
			return nil
		}

		if err := s.db.DeleteWorkspace(workspace.ID); err != nil {
			return fmt.Errorf("update workspace status: %w", err)
		}
	case api.StatusSuspending:
		helmRelease, err := s.findHelmRelease(ctx, workspace)
		if err != nil {
			return fmt.Errorf("find helm release: %w", err)
		}
		if helmRelease == nil {
			return fmt.Errorf("cannot find helmrelease")
		}

		var pods corev1.PodList
		err = s.kubeClient.List(ctx, &pods, k8sclient.InNamespace(workspace.ID))
		if err != nil {
			return fmt.Errorf("fetching list of pods: %w", err)
		}

		for _, pod := range pods.Items {
			if strings.HasPrefix(pod.Name, "describe-connection-worker") {
				// waiting for describe jobs to finish
				return nil
			}
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

		if len(pods.Items) > 0 {
			// waiting for pods to go down
			return nil
		}

		if err := s.db.UpdateWorkspaceStatus(workspace.ID, api.StatusSuspended); err != nil {
			return fmt.Errorf("update workspace status: %w", err)
		}
	}
	return nil
}

// CreateWorkspace godoc
//
//	@Summary		Create workspace for workspace service
//	@Description	Returns workspace created
//	@Security		BearerToken
//	@Tags			workspace
//	@Accept			json
//	@Produce		json
//	@Param			request	body		api.CreateWorkspaceRequest	true	"Create workspace request"
//	@Success		200		{object}	api.CreateWorkspaceResponse
//	@Router			/workspace/api/v1/workspace [post]
func (s *Server) CreateWorkspace(c echo.Context) error {
	userID := httpserver2.GetUserID(c)

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
	if !regexp.MustCompile(`^[a-zA-Z0-9\-]*$`).MatchString(request.Name) {
		return echo.NewHTTPError(http.StatusBadRequest, "name is invalid")
	}
	if len(request.Name) > 150 {
		return echo.NewHTTPError(http.StatusBadRequest, "name over 150 characters")
	}

	switch request.Tier {
	case string(api.Tier_Free), string(api.Tier_Teams):
	default:
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid Tier")
	}
	uri := strings.ToLower("https://" + s.cfg.DomainSuffix + "/" + request.Name)
	sf := sonyflake.NewSonyflake(sonyflake.Settings{})
	id, err := sf.NextID()
	if err != nil {
		return err
	}

	workspace := &Workspace{
		ID:          fmt.Sprintf("ws-%d", id),
		Name:        strings.ToLower(request.Name),
		OwnerId:     userID,
		URI:         uri,
		Status:      api.StatusProvisioning,
		Description: request.Description,
		Tier:        api.Tier(request.Tier),
	}
	if err := s.db.CreateWorkspace(workspace); err != nil {
		if strings.Contains(err.Error(), "duplicate key value") {
			return echo.NewHTTPError(http.StatusFound, "workspace already exists")
		}
		c.Logger().Errorf("create workspace: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, ErrInternalServer)
	}
	return c.JSON(http.StatusOK, api.CreateWorkspaceResponse{
		ID: workspace.ID,
	})
}

// DeleteWorkspace godoc
//
//	@Summary		Delete workspace for workspace service
//	@Description	Delete workspace with workspace id
//	@Security		BearerToken
//	@Tags			workspace
//	@Accept			json
//	@Produce		json
//	@Param			workspace_id	path	string	true	"Workspace ID"
//	@Success		200
//	@Router			/workspace/api/v1/workspace/{workspace_id} [delete]
func (s *Server) DeleteWorkspace(c echo.Context) error {
	userID := httpserver2.GetUserID(c)

	id := c.Param("workspace_id")
	if id == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "workspace id is empty")
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

	if err := s.db.UpdateWorkspaceStatus(id, api.StatusDeleting); err != nil {
		c.Logger().Errorf("delete workspace: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, ErrInternalServer)
	}
	return c.JSON(http.StatusOK, map[string]string{"message": "success"})
}

// GetWorkspace godoc
//
//	@Summary		Get workspace for workspace service
//	@Description	Get workspace with workspace id
//	@Security		BearerToken
//	@Tags			workspace
//	@Accept			json
//	@Produce		json
//	@Param			workspace_id	path	string	true	"Workspace ID"
//	@Success		200
//	@Router			/workspace/api/v1/workspaces/{workspace_id} [get]
func (s *Server) GetWorkspace(c echo.Context) error {
	userId := httpserver2.GetUserID(c)
	resp, err := s.authClient.GetUserRoleBindings(httpclient.FromEchoContext(c))
	if err != nil {
		return fmt.Errorf("GetUserRoleBindings: %v", err)
	}

	id := c.Param("workspace_id")
	if id == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "workspace id is empty")
	}

	workspace, err := s.db.GetWorkspace(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "workspace not found")
		}
		c.Logger().Errorf("find workspace: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, ErrInternalServer)
	}

	hasRoleInWorkspace := false
	for _, roleBinding := range resp.RoleBindings {
		if roleBinding.WorkspaceID == workspace.ID {
			hasRoleInWorkspace = true
		}
	}
	if resp.GlobalRoles != nil {
		hasRoleInWorkspace = true
	}

	if workspace.OwnerId != userId && !hasRoleInWorkspace {
		return echo.NewHTTPError(http.StatusForbidden, "operation is forbidden")
	}

	version := "unspecified"
	var keibiVersionConfig corev1.ConfigMap
	err = s.kubeClient.Get(context.Background(), k8sclient.ObjectKey{
		Namespace: workspace.ID,
		Name:      "keibi-version",
	}, &keibiVersionConfig)
	if err == nil {
		version = keibiVersionConfig.Data["version"]
	} else {
		fmt.Printf("failed to load version due to %v\n", err)
	}

	return c.JSON(http.StatusOK, api.WorkspaceResponse{
		Workspace: workspace.ToAPI(),
		Version:   version,
	})
}

// ResumeWorkspace godoc
//
//	@Summary	Resume workspace
//	@Tags		workspace
//	@Security	BearerToken
//	@Accept		json
//	@Produce	json
//	@Param		workspace_id	path	string	true	"Workspace ID"
//	@Success	200
//	@Router		/workspace/api/v1/workspace/{workspace_id}/resume [post]
func (s *Server) ResumeWorkspace(c echo.Context) error {
	id := c.Param("workspace_id")
	if id == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "workspace id is empty")
	}

	workspace, err := s.db.GetWorkspace(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "workspace not found")
		}
		c.Logger().Errorf("find workspace: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, ErrInternalServer)
	}

	if workspace.Status != api.StatusSuspended {
		return echo.NewHTTPError(http.StatusBadRequest, "workspace is not suspended")
	}

	err = s.rdb.SetEX(context.Background(), "last_access_"+workspace.Name, time.Now().UnixMilli(),
		30*24*time.Hour).Err()
	if err != nil {
		return err
	}

	if err := s.db.UpdateWorkspaceStatus(workspace.ID, api.StatusProvisioning); err != nil {
		return fmt.Errorf("update workspace status: %w", err)
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "success"})
}

// SuspendWorkspace godoc
//
//	@Summary	Suspend workspace
//	@Tags		workspace
//	@Security	BearerToken
//	@Accept		json
//	@Produce	json
//	@Param		workspace_id	path	string	true	"Workspace ID"
//	@Success	200
//	@Router		/workspace/api/v1/workspace/{workspace_id}/suspend [post]
func (s *Server) SuspendWorkspace(c echo.Context) error {
	id := c.Param("workspace_id")
	if id == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "workspace id is empty")
	}

	workspace, err := s.db.GetWorkspace(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "workspace not found")
		}
		c.Logger().Errorf("find workspace: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, ErrInternalServer)
	}

	if workspace.Status != api.StatusProvisioned {
		return echo.NewHTTPError(http.StatusBadRequest, "workspace is not provisioned")
	}

	err = s.rdb.Del(context.Background(), "last_access_"+workspace.Name).Err()
	if err != nil {
		return err
	}
	if err := s.db.UpdateWorkspaceStatus(workspace.ID, api.StatusSuspending); err != nil {
		return fmt.Errorf("update workspace status: %w", err)
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "success"})
}

// ListWorkspaces godoc
//
//	@Summary		List all workspaces with owner id
//	@Description	Returns all workspaces with owner id
//	@Security		BearerToken
//	@Tags			workspace
//	@Accept			json
//	@Produce		json
//	@Success		200	{array}	api.WorkspaceResponse
//	@Router			/workspace/api/v1/workspaces [get]
func (s *Server) ListWorkspaces(c echo.Context) error {
	userId := httpserver2.GetUserID(c)
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
		if workspace.Status == api.StatusDeleted {
			continue
		}

		hasRoleInWorkspace := false
		for _, rb := range resp.RoleBindings {
			if rb.WorkspaceID == workspace.ID {
				hasRoleInWorkspace = true
			}
		}
		if resp.GlobalRoles != nil {
			hasRoleInWorkspace = true
		}

		if workspace.OwnerId != userId && !hasRoleInWorkspace {
			continue
		}

		version := "unspecified"
		var keibiVersionConfig corev1.ConfigMap
		err = s.kubeClient.Get(context.Background(), k8sclient.ObjectKey{
			Namespace: workspace.ID,
			Name:      "keibi-version",
		}, &keibiVersionConfig)
		if err == nil {
			version = keibiVersionConfig.Data["version"]
		} else {
			fmt.Printf("failed to load version due to %v\n", err)
		}

		workspaces = append(workspaces, &api.WorkspaceResponse{
			Workspace: workspace.ToAPI(),
			Version:   version,
		})
	}
	return c.JSON(http.StatusOK, workspaces)
}

// GetCurrentWorkspace godoc
//
//	@Summary		List all workspaces with owner id
//	@Description	Returns all workspaces with owner id
//	@Security		BearerToken
//	@Tags			workspace
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	api.WorkspaceResponse
//	@Router			/workspace/api/v1/workspace/current [get]
func (s *Server) GetCurrentWorkspace(c echo.Context) error {
	wsName := httpserver2.GetWorkspaceName(c)

	workspace, err := s.db.GetWorkspaceByName(wsName)
	if err != nil {
		return fmt.Errorf("ListWorkspaces: %v", err)
	}

	version := "unspecified"
	var keibiVersionConfig corev1.ConfigMap
	err = s.kubeClient.Get(context.Background(), k8sclient.ObjectKey{
		Namespace: workspace.ID,
		Name:      "keibi-version",
	}, &keibiVersionConfig)
	if err == nil {
		version = keibiVersionConfig.Data["version"]
	} else {
		fmt.Printf("failed to load version due to %v\n", err)
	}

	return c.JSON(http.StatusOK, api.WorkspaceResponse{
		Workspace: workspace.ToAPI(),
		Version:   version,
	})
}

// ChangeOwnership godoc
//
//	@Summary	Change ownership of workspace
//	@Tags		workspace
//	@Security	BearerToken
//	@Accept		json
//	@Produce	json
//	@Param		request			body	api.ChangeWorkspaceOwnershipRequest	true	"Change ownership request"
//	@Param		workspace_id	path	string								true	"WorkspaceID"
//	@Success	200
//	@Router		/workspace/api/v1/workspace/{workspace_id}/owner [post]
func (s *Server) ChangeOwnership(c echo.Context) error {
	userID := httpserver2.GetUserID(c)
	workspaceID := c.Param("workspace_id")

	var request api.ChangeWorkspaceOwnershipRequest
	if err := c.Bind(&request); err != nil {
		c.Logger().Errorf("bind the request: %v", err)
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}
	if workspaceID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "workspace id is empty")
	}

	w, err := s.db.GetWorkspace(workspaceID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "workspace not found")
		}
		return err
	}

	if w.OwnerId != userID {
		return echo.NewHTTPError(http.StatusForbidden, "operation is forbidden")
	}

	err = s.db.UpdateWorkspaceOwner(workspaceID, request.NewOwnerUserID)
	if err != nil {
		return err
	}

	return c.NoContent(http.StatusOK)
}

// ChangeName godoc
//
//	@Summary	Change name of workspace
//	@Security	BearerToken
//	@Tags		workspace
//	@Accept		json
//	@Produce	json
//	@Param		request			body	api.ChangeWorkspaceNameRequest	true	"Change name request"
//	@Param		workspace_id	path	string							true	"WorkspaceID"
//	@Success	200
//	@Router		/workspace/api/v1/workspace/{workspace_id}/name [post]
func (s *Server) ChangeName(c echo.Context) error {
	workspaceID := c.Param("workspace_id")

	var request api.ChangeWorkspaceNameRequest
	if err := c.Bind(&request); err != nil {
		c.Logger().Errorf("bind the request: %v", err)
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}
	if workspaceID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "workspace id is empty")
	}

	_, err := s.db.GetWorkspace(workspaceID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "workspace not found")
		}
		return err
	}

	err = s.db.UpdateWorkspaceName(workspaceID, request.NewName)
	if err != nil {
		return err
	}

	return c.NoContent(http.StatusOK)
}

// ChangeTier godoc
//
//	@Summary	Change Tier of workspace
//	@Security	BearerToken
//	@Tags		workspace
//	@Accept		json
//	@Produce	json
//	@Param		request			body	api.ChangeWorkspaceTierRequest	true	"Change tier request"
//	@Param		workspace_id	path	string							true	"WorkspaceID"
//	@Success	200
//	@Router		/workspace/api/v1/workspace/{workspace_id}/tier [post]
func (s *Server) ChangeTier(c echo.Context) error {
	workspaceID := c.Param("workspace_id")

	var request api.ChangeWorkspaceTierRequest
	if err := c.Bind(&request); err != nil {
		c.Logger().Errorf("bind the request: %v", err)
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}
	if workspaceID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "workspace id is empty")
	}

	_, err := s.db.GetWorkspace(workspaceID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "workspace not found")
		}
		return err
	}

	err = s.db.UpdateWorkspaceTier(workspaceID, request.NewTier)
	if err != nil {
		return err
	}

	return c.NoContent(http.StatusOK)
}

// ChangeOrganization godoc
//
//	@Summary	Change organization of workspace
//	@Security	BearerToken
//	@Tags		workspace
//	@Accept		json
//	@Produce	json
//	@Param		request			body	api.ChangeWorkspaceOrganizationRequest	true	"Change organization request"
//	@Param		workspace_id	path	string									true	"WorkspaceID"
//	@Success	200
//	@Router		/workspace/api/v1/workspace/{workspace_id}/organization [post]
func (s *Server) ChangeOrganization(c echo.Context) error {
	workspaceID := c.Param("workspace_id")

	var request api.ChangeWorkspaceOrganizationRequest
	if err := c.Bind(&request); err != nil {
		c.Logger().Errorf("bind the request: %v", err)
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}
	if workspaceID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "workspace id is empty")
	}

	_, err := s.db.GetWorkspace(workspaceID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "workspace not found")
		}
		return err
	}

	_, err = s.db.GetOrganization(request.NewOrgID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "organization not found")
		}
		return err
	}

	err = s.db.UpdateWorkspaceOrganization(workspaceID, request.NewOrgID)
	if err != nil {
		return err
	}

	return c.NoContent(http.StatusOK)
}

// GetWorkspaceLimits godoc
//
//	@Summary	Get workspace limits
//	@Security	BearerToken
//	@Tags		workspace
//	@Accept		json
//	@Produce	json
//	@Param		workspace_name	path		string	true	"Workspace Name"
//	@Param		ignore_usage	query		bool	false	"Ignore usage"
//	@Success	200				{object}	api.WorkspaceLimitsUsage
//	@Router		/workspace/api/v1/workspaces/limits/{workspace_name} [get]
func (s *Server) GetWorkspaceLimits(c echo.Context) error {
	var response api.WorkspaceLimitsUsage

	workspaceName := c.Param("workspace_name")
	ignoreUsage := c.QueryParam("ignore_usage")

	dbWorkspace, err := s.db.GetWorkspaceByName(workspaceName)
	if err != nil {
		return err
	}

	if ignoreUsage != "true" {
		ectx := httpclient.FromEchoContext(c)
		ectx.UserRole = authapi.AdminRole
		resp, err := s.authClient.GetWorkspaceRoleBindings(ectx, workspaceName, dbWorkspace.ID)
		if err != nil {
			return fmt.Errorf("GetWorkspaceRoleBindings: %v", err)
		}
		response.CurrentUsers = int64(len(resp))

		inventoryURL := strings.ReplaceAll(InventoryTemplate, "%NAMESPACE%", dbWorkspace.ID)
		inventoryClient := client2.NewInventoryServiceClient(inventoryURL)
		resourceCount, err := inventoryClient.CountResources(httpclient.FromEchoContext(c))
		response.CurrentResources = resourceCount

		onboardURL := strings.ReplaceAll(OnboardTemplate, "%NAMESPACE%", dbWorkspace.ID)
		onboardClient := client.NewOnboardServiceClient(onboardURL, s.cache)
		count, err := onboardClient.CountSources(httpclient.FromEchoContext(c), source.Nil)
		response.CurrentConnections = count
	}

	limits := api.GetLimitsByTier(dbWorkspace.Tier)
	response.MaxUsers = limits.MaxUsers
	response.MaxConnections = limits.MaxConnections
	response.MaxResources = limits.MaxResources
	response.ID = dbWorkspace.ID
	response.Name = dbWorkspace.Name
	return c.JSON(http.StatusOK, response)
}

// GetWorkspaceLimitsByID godoc
//
//	@Summary	Get workspace limits
//	@Security	BearerToken
//	@Tags		workspace
//	@Accept		json
//	@Produce	json
//	@Param		workspace_id	path		string	true	"Workspace ID"
//	@Success	200				{object}	api.WorkspaceLimits
//	@Router		/workspace/api/v1/workspaces/limits/byid/{workspace_id} [get]
func (s *Server) GetWorkspaceLimitsByID(c echo.Context) error {
	workspaceID := c.Param("workspace_id")

	dbWorkspace, err := s.db.GetWorkspace(workspaceID)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, api.GetLimitsByTier(dbWorkspace.Tier))
}

// GetWorkspaceByID godoc
//
//	@Summary	Get workspace
//	@Security	BearerToken
//	@Tags		workspace
//	@Accept		json
//	@Produce	json
//	@Param		workspace_id	path		string	true	"Workspace ID"
//	@Success	200				{object}	api.Workspace
//	@Router		/workspace/api/v1/workspaces/byid/{workspace_id} [get]
func (s *Server) GetWorkspaceByID(c echo.Context) error {
	workspaceID := c.Param("workspace_id")

	dbWorkspace, err := s.db.GetWorkspace(workspaceID)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, dbWorkspace.ToAPI())
}

// CreateOrganization godoc
//
//	@Summary	Create an organization
//	@Security	BearerToken
//	@Tags		workspace
//	@Accept		json
//	@Produce	json
//	@Param		request	body		api.Organization	true	"Organization"
//	@Success	201		{object}	api.Organization
//	@Router		/workspace/api/v1/organization [post]
func (s *Server) CreateOrganization(c echo.Context) error {
	var request api.Organization
	if err := c.Bind(&request); err != nil {
		return err
	}

	dbOrg := Organization{
		CompanyName:  request.CompanyName,
		Url:          request.Url,
		Address:      request.Address,
		City:         request.City,
		State:        request.State,
		Country:      request.Country,
		ContactPhone: request.ContactPhone,
		ContactEmail: request.ContactEmail,
		ContactName:  request.ContactName,
	}
	err := s.db.CreateOrganization(&dbOrg)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusCreated, dbOrg.ToAPI())
}

// DeleteOrganization godoc
//
//	@Summary	Create an organization
//	@Security	BearerToken
//	@Tags		workspace
//	@Accept		json
//	@Produce	json
//	@Param		organizationId	path	int	true	"Organization ID"
//	@Success	202
//	@Router		/workspace/api/v1/organization/{organizationId} [delete]
func (s *Server) DeleteOrganization(c echo.Context) error {
	organizationIDStr := c.Param("organizationId")
	organizationID, err := strconv.ParseInt(organizationIDStr, 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid organization ID")
	}
	_, err = s.db.GetOrganization(uint(organizationID))
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return echo.NewHTTPError(http.StatusNotFound, "Organization not found")
		}
		return err
	}

	err = s.db.DeleteOrganization(uint(organizationID))
	if err != nil {
		return err
	}

	return c.NoContent(http.StatusAccepted)
}
