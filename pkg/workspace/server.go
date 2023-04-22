package workspace

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"gitlab.com/keibiengine/keibi-engine/pkg/source"
	"gitlab.com/keibiengine/keibi-engine/pkg/workspace/client/pipedrive"

	"github.com/go-redis/cache/v8"
	"gitlab.com/keibiengine/keibi-engine/pkg/onboard/client"

	client2 "gitlab.com/keibiengine/keibi-engine/pkg/inventory/client"

	v1 "k8s.io/api/apps/v1"

	"github.com/labstack/gommon/log"

	corev1 "k8s.io/api/core/v1"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"

	contourv1 "github.com/projectcontour/contour/apis/projectcontour/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	apimeta "github.com/fluxcd/pkg/apis/meta"
	"github.com/go-redis/redis/v8"
	"github.com/labstack/echo/v4"
	velerov1api "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	authapi "gitlab.com/keibiengine/keibi-engine/pkg/auth/api"
	authclient "gitlab.com/keibiengine/keibi-engine/pkg/auth/client"
	"gitlab.com/keibiengine/keibi-engine/pkg/internal/httpclient"
	"gitlab.com/keibiengine/keibi-engine/pkg/internal/httpserver"
	"gitlab.com/keibiengine/keibi-engine/pkg/workspace/api"
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
	e               *echo.Echo
	cfg             *Config
	db              *Database
	authClient      authclient.AuthServiceClient
	pipedriveClient pipedrive.PipedriveServiceClient
	kubeClient      k8sclient.Client // the kubernetes client
	rdb             *redis.Client
	cache           *cache.Cache
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

	err = velerov1api.AddToScheme(s.kubeClient.Scheme())
	if err != nil {
		return nil, fmt.Errorf("add velerov1api to scheme: %w", err)
	}

	err = v1.AddToScheme(s.kubeClient.Scheme())
	if err != nil {
		return nil, fmt.Errorf("add v1 to scheme: %w", err)
	}

	s.authClient = authclient.NewAuthServiceClient(cfg.AuthBaseUrl)
	s.pipedriveClient = pipedrive.NewPipedriveServiceClient(logger, cfg.PipedriveBaseUrl, cfg.PipedriveApiToken)

	s.rdb = redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddress,
		Password: "", // no password set
		DB:       0,  // use default DB
	})

	s.cache = cache.New(&cache.Options{
		Redis:      s.rdb,
		LocalCache: cache.NewTinyLFU(2000, 1*time.Minute),
	})
	return s, nil
}

func (s *Server) Register(e *echo.Echo) {
	v1 := e.Group("/api/v1")

	v1.POST("/workspace", httpserver.AuthorizeHandler(s.CreateWorkspace, authapi.EditorRole))
	v1.DELETE("/workspace/:workspace_id", httpserver.AuthorizeHandler(s.DeleteWorkspace, authapi.EditorRole))
	v1.POST("/workspace/:workspace_id/suspend", httpserver.AuthorizeHandler(s.SuspendWorkspace, authapi.EditorRole))
	v1.POST("/workspace/:workspace_id/resume", httpserver.AuthorizeHandler(s.ResumeWorkspace, authapi.EditorRole))
	v1.GET("/workspaces/limits/:workspace_name", httpserver.AuthorizeHandler(s.GetWorkspaceLimits, authapi.ViewerRole))
	v1.GET("/workspaces/limits/byid/:workspace_id", httpserver.AuthorizeHandler(s.GetWorkspaceLimitsByID, authapi.ViewerRole))
	v1.GET("/workspaces/byid/:workspace_id", httpserver.AuthorizeHandler(s.GetWorkspaceByID, authapi.ViewerRole))
	v1.GET("/workspaces", httpserver.AuthorizeHandler(s.ListWorkspaces, authapi.ViewerRole))
	v1.GET("/workspaces/:workspace_id", httpserver.AuthorizeHandler(s.GetWorkspace, authapi.ViewerRole))
	v1.POST("/workspace/:workspace_id/owner", httpserver.AuthorizeHandler(s.ChangeOwnership, authapi.EditorRole))
	v1.POST("/workspace/:workspace_id/name", httpserver.AuthorizeHandler(s.ChangeName, authapi.KeibiAdminRole))
	v1.POST("/workspace/:workspace_id/tier", httpserver.AuthorizeHandler(s.ChangeTier, authapi.KeibiAdminRole))
	v1.POST("/workspace/:workspace_id/organization", httpserver.AuthorizeHandler(s.ChangeOrganization, authapi.KeibiAdminRole))
	v1.GET("/workspace/:workspace_id/backup", httpserver.AuthorizeHandler(s.ListBackups, authapi.ViewerRole))
	v1.POST("/workspace/:workspace_id/backup", httpserver.AuthorizeHandler(s.PerformBackup, authapi.EditorRole))
	v1.GET("/workspace/:workspace_id/backup/restores", httpserver.AuthorizeHandler(s.ListRestore, authapi.ViewerRole))
	v1.POST("/workspace/:workspace_id/backup/:backup_name/restore", httpserver.AuthorizeHandler(s.PerformRestore, authapi.EditorRole))
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
	switch WorkspaceStatus(workspace.Status) {
	case StatusDeleting, StatusDeleted:
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
		if workspace.Status == string(StatusProvisioned) {
			fmt.Printf("suspending workspace %s\n", workspace.Name)
			if err := s.db.UpdateWorkspaceStatus(workspace.ID, StatusSuspending); err != nil {
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

	var includes []contourv1.Include
	for _, w := range workspaces {
		if w.Status != string(StatusProvisioned) {
			continue
		}
		includes = append(includes, contourv1.Include{
			Name:      "http-proxy-route",
			Namespace: w.ID,
			Conditions: []contourv1.MatchCondition{
				{
					Prefix: "/" + w.Name,
				},
			},
		})
	}

	key := types.NamespacedName{
		Name:      "http-proxy-route",
		Namespace: s.cfg.KeibiOctopusNamespace,
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
			Namespace: s.cfg.KeibiOctopusNamespace,
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
			s.e.Logger.Infof("create helm release %s with status %s", workspace.ID, workspace.Status)
			if err := s.createHelmRelease(ctx, workspace); err != nil {
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

			newStatus = StatusProvisioned
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
			newStatus = StatusProvisioning
		} else if meta.IsStatusConditionTrue(helmRelease.Status.Conditions, apimeta.StalledCondition) {
			newStatus = StatusProvisioningFailed
		}
		if newStatus != status {
			if err := s.db.UpdateWorkspaceStatus(workspace.ID, newStatus); err != nil {
				return fmt.Errorf("update workspace status: %w", err)
			}
		}
	case StatusProvisioningFailed:
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
			newStatus = StatusProvisioning
		} else if meta.IsStatusConditionFalse(helmRelease.Status.Conditions, apimeta.ReadyCondition) {
			newStatus = StatusProvisioning
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

		var list velerov1api.BackupList
		err = s.kubeClient.List(context.Background(), &list)
		if err != nil {
			return err
		}

		var backup *velerov1api.Backup
		for _, v := range list.Items {
			if !strings.HasPrefix(v.Name, workspace.Name+".") {
				// . is to prevent backups from other workspaces with similar name to
				// appear in the results
				continue
			}

			timeStr := strings.Split(v.Name, ".")
			tim, err := time.Parse("2006-02-01-15-04-05", timeStr[1])
			if err != nil {
				return err
			}
			if tim.After(time.Now().Add(-1 * time.Hour)) {
				backup = &v
			}
		}
		if backup == nil {
			return s.CreateBackup(*workspace)
		} /* else if backup.Status.Phase != velerov1api.BackupPhaseCompleted {
			return nil
		}*/

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
	case StatusSuspending:
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

		if err := s.db.UpdateWorkspaceStatus(workspace.ID, StatusSuspended); err != nil {
			return fmt.Errorf("update workspace status: %w", err)
		}
	}
	return nil
}

// CreateWorkspace godoc
//
//	@Summary		Create workspace for workspace service
//	@Description	Returns workspace created
//	@Tags			workspace
//	@Accept			json
//	@Produce		json
//	@Param			request	body		api.CreateWorkspaceRequest	true	"Create workspace request"
//	@Success		200		{object}	api.CreateWorkspaceResponse
//	@Router			/workspace/api/v1/workspace [post]
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
		Status:      StatusProvisioning.String(),
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
//	@Tags			workspace
//	@Accept			json
//	@Produce		json
//	@Param			workspace_id	path	string	true	"Workspace ID"
//	@Success		200
//	@Router			/workspace/api/v1/workspace/:workspace_id [delete]
func (s *Server) DeleteWorkspace(c echo.Context) error {
	userID := httpserver.GetUserID(c)

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

	if err := s.db.UpdateWorkspaceStatus(id, StatusDeleting); err != nil {
		c.Logger().Errorf("delete workspace: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, ErrInternalServer)
	}
	return c.JSON(http.StatusOK, map[string]string{"message": "success"})
}

// GetWorkspace godoc
//
//	@Summary		Get workspace for workspace service
//	@Description	Get workspace with workspace id
//	@Tags			workspace
//	@Accept			json
//	@Produce		json
//	@Param			workspace_id	path	string	true	"Workspace ID"
//	@Success		200
//	@Router			/workspace/api/v1/workspace/:workspace_id [get]
func (s *Server) GetWorkspace(c echo.Context) error {
	userId := httpserver.GetUserID(c)
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

	var organization *api.OrganizationResponse
	if workspace.OrganizationID != nil {
		org, err := s.pipedriveClient.GetPipedriveOrganization(c.Request().Context(), *workspace.OrganizationID)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		organization = &api.OrganizationResponse{
			ID:            *workspace.OrganizationID,
			CompanyName:   org.Name,
			Url:           org.URL,
			AddressLine1:  org.Address,
			City:          org.AddressLocality,
			State:         org.AddressAdminAreaLevel1,
			Country:       org.AddressCountry,
			ContactPhone:  pipedrive.GetPrimaryValue(org.Contact.Phones),
			ContactEmail:  pipedrive.GetPrimaryValue(org.Contact.Emails),
			ContactPerson: org.Contact.Name,
		}
	}

	return c.JSON(http.StatusOK, api.WorkspaceResponse{
		ID:           workspace.ID,
		OwnerId:      workspace.OwnerId,
		URI:          workspace.URI,
		Name:         workspace.Name,
		Tier:         string(workspace.Tier),
		Version:      version,
		Status:       workspace.Status,
		Description:  workspace.Description,
		CreatedAt:    workspace.CreatedAt,
		Organization: organization,
	})
}

// ResumeWorkspace godoc
//
//	@Summary	Resume workspace
//	@Tags		workspace
//	@Accept		json
//	@Produce	json
//	@Param		workspace_id	path	string	true	"Workspace ID"
//	@Success	200
//	@Router		/workspace/api/v1/workspace/:workspace_id/resume [post]
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

	if workspace.Status != string(StatusSuspended) {
		return echo.NewHTTPError(http.StatusBadRequest, "workspace is not suspended")
	}

	err = s.rdb.SetEX(context.Background(), "last_access_"+workspace.Name, time.Now().UnixMilli(),
		30*24*time.Hour).Err()
	if err != nil {
		return err
	}

	if err := s.db.UpdateWorkspaceStatus(workspace.ID, StatusProvisioning); err != nil {
		return fmt.Errorf("update workspace status: %w", err)
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "success"})
}

// SuspendWorkspace godoc
//
//	@Summary	Suspend workspace
//	@Tags		workspace
//	@Accept		json
//	@Produce	json
//	@Param		workspace_id	path	string	true	"Workspace ID"
//	@Success	200
//	@Router		/workspace/api/v1/workspace/:workspace_id/suspend [post]
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

	if workspace.Status != string(StatusProvisioned) {
		return echo.NewHTTPError(http.StatusBadRequest, "workspace is not provisioned")
	}

	err = s.rdb.Del(context.Background(), "last_access_"+workspace.Name).Err()
	if err != nil {
		return err
	}
	if err := s.db.UpdateWorkspaceStatus(workspace.ID, StatusSuspending); err != nil {
		return fmt.Errorf("update workspace status: %w", err)
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "success"})
}

// ListWorkspaces godoc
//
//	@Summary		List all workspaces with owner id
//	@Description	Returns all workspaces with owner id
//	@Tags			workspace
//	@Accept			json
//	@Produce		json
//	@Success		200	{array}	[]api.WorkspaceResponse
//	@Router			/workspace/api/v1/workspaces [get]
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
			ID:          workspace.ID,
			OwnerId:     workspace.OwnerId,
			URI:         workspace.URI,
			Name:        workspace.Name,
			Tier:        string(workspace.Tier),
			Version:     version,
			Status:      workspace.Status,
			Description: workspace.Description,
			CreatedAt:   workspace.CreatedAt,
		})
	}
	return c.JSON(http.StatusOK, workspaces)
}

// ChangeOwnership godoc
//
//	@Summary	Change ownership of workspace
//	@Tags		workspace
//	@Accept		json
//	@Produce	json
//	@Param		request	body	api.ChangeWorkspaceOwnershipRequest	true	"Change ownership request"
//	@Router		/workspace/api/v1/workspace/{workspace_id}/owner [post]
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
//	@Tags		workspace
//	@Accept		json
//	@Produce	json
//	@Param		request	body	api.ChangeWorkspaceNameRequest	true	"Change name request"
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
//	@Tags		workspace
//	@Accept		json
//	@Produce	json
//	@Param		request	body	api.ChangeWorkspaceTierRequest	true	"Change tier request"
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
//	@Tags		workspace
//	@Accept		json
//	@Produce	json
//	@Param		request	body	api.ChangeWorkspaceOrganizationRequest	true	"Change organization request"
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

	err = s.db.UpdateWorkspaceOrganization(workspaceID, request.NewOrgID)
	if err != nil {
		return err
	}

	return c.NoContent(http.StatusOK)
}

// PerformBackup godoc
//
//	@Summary	perform backup of workspace
//	@Tags		workspace
//	@Accept		json
//	@Produce	json
//	@Router		/workspace/api/v1/workspace/{workspace_id}/backup [post]
func (s *Server) PerformBackup(c echo.Context) error {
	userID := httpserver.GetUserID(c)
	workspaceID := c.Param("workspace_id")

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

	if err := s.CreateBackup(*w); err != nil {
		return err
	}

	return c.NoContent(http.StatusOK)
}

func (s *Server) CreateBackup(w Workspace) error {
	backupName := fmt.Sprintf("%s.%s.backup", w.Name, time.Now().Format("2006-02-01-15-04-05"))
	// performing backup using velero
	backupSpec := velerov1api.Backup{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Backup",
			APIVersion: "velero.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      backupName,
			Namespace: "velero",
		},
		Spec: velerov1api.BackupSpec{
			IncludedNamespaces: []string{w.ID},
			OrLabelSelectors: []*metav1.LabelSelector{
				{
					MatchLabels: map[string]string{
						"app": "vault",
					},
				},
				{
					MatchLabels: map[string]string{
						"app": "postgres-cluster",
					},
				},
			},
		},
	}

	err := s.kubeClient.Create(context.Background(), &backupSpec)
	if err != nil {
		return err
	}
	return nil
}

// ListBackups godoc
//
//	@Summary	lists backup of workspace
//	@Tags		workspace
//	@Accept		json
//	@Produce	json
//	@Router		/workspace/api/v1/workspace/{workspace_id}/backup [get]
func (s *Server) ListBackups(c echo.Context) error {
	userID := httpserver.GetUserID(c)
	workspaceID := c.Param("workspace_id")

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

	var list velerov1api.BackupList
	err = s.kubeClient.List(context.Background(), &list)
	if err != nil {
		return err
	}

	backupList := map[string]api.BackupStatus{}
	for _, v := range list.Items {
		if !strings.HasPrefix(v.Name, w.Name+".") {
			// . is to prevent backups from other workspaces with similar name to
			// appear in the results
			continue
		}
		backupList[v.Name] = api.BackupStatus{
			Phase:               v.Status.Phase,
			Progress:            v.Status.Progress,
			Expiration:          v.Status.Expiration,
			StartTimestamp:      v.Status.StartTimestamp,
			CompletionTimestamp: v.Status.CompletionTimestamp,
			TotalAttempted:      v.Status.CSIVolumeSnapshotsAttempted + v.Status.VolumeSnapshotsAttempted,
			TotalCompleted:      v.Status.CSIVolumeSnapshotsCompleted + v.Status.VolumeSnapshotsCompleted,
			Warnings:            v.Status.Warnings,
			Errors:              v.Status.Errors,
			ValidationErrors:    v.Status.ValidationErrors,
			FailureReason:       v.Status.FailureReason,
		}
	}

	return c.JSON(http.StatusOK, backupList)
}

// PerformRestore godoc
//
//	@Summary	perform restore of backup of workspace
//	@Tags		workspace
//	@Accept		json
//	@Produce	json
//	@Router		/workspace/api/v1/workspace/{workspace_id}/backup/{backup_name}/restore [post]
func (s *Server) PerformRestore(c echo.Context) error {
	userID := httpserver.GetUserID(c)
	workspaceID := c.Param("workspace_id")
	backupName := c.Param("backup_name")

	if workspaceID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "workspace id is empty")
	}
	if backupName == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "backup name is empty")
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
	restoreName := fmt.Sprintf("%s.%s.restore", w.Name, time.Now().Format("2006-02-01-15-04-05"))

	// performing restore using velero
	restoreSpec := velerov1api.Restore{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Restore",
			APIVersion: "velero.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      restoreName,
			Namespace: "velero",
		},
		Spec: velerov1api.RestoreSpec{
			BackupName:         backupName,
			IncludedNamespaces: []string{workspaceID},
		},
	}

	err = s.kubeClient.Create(context.Background(), &restoreSpec)
	if err != nil {
		return err
	}

	return c.NoContent(http.StatusOK)
}

// ListRestore godoc
//
//	@Summary	lists restore of workspace
//	@Tags		workspace
//	@Accept		json
//	@Produce	json
//	@Router		/workspace/api/v1/workspace/{workspace_id}/backup/restores [get]
func (s *Server) ListRestore(c echo.Context) error {
	userID := httpserver.GetUserID(c)
	workspaceID := c.Param("workspace_id")

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

	var list velerov1api.RestoreList
	err = s.kubeClient.List(context.Background(), &list)
	if err != nil {
		return err
	}

	backupList := map[string]velerov1api.RestoreStatus{}
	for _, v := range list.Items {
		if !strings.HasPrefix(v.Name, w.Name+".") {
			// -20YY is to prevent backups from other workspaces with similar name to
			// appear in the results
			continue
		}

		backupList[v.Name] = v.Status
	}

	return c.JSON(http.StatusOK, backupList)
}

// GetWorkspaceLimits godoc
//
//	@Summary	Get workspace limits
//	@Tags		workspace
//	@Accept		json
//	@Produce	json
//	@Param		workspace_name	path	string	true	"Workspace Name"
//	@Param		ignore_usage	query	bool	false	"Ignore usage"
//	@Success	200				{array}	api.WorkspaceLimitsUsage
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
//	@Tags		workspace
//	@Accept		json
//	@Produce	json
//	@Param		workspace_id	path	string	true	"Workspace Name"
//	@Success	200				{array}	api.WorkspaceLimits
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
//	@Tags		workspace
//	@Accept		json
//	@Produce	json
//	@Param		workspace_id	path	string	true	"Workspace Name"
//	@Success	200				{array}	api.WorkspaceLimits
//	@Router		/workspace/api/v1/workspaces/byid/{workspace_id} [get]
func (s *Server) GetWorkspaceByID(c echo.Context) error {
	workspaceID := c.Param("workspace_id")

	dbWorkspace, err := s.db.GetWorkspace(workspaceID)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, api.Workspace{
		ID:             dbWorkspace.ID,
		Name:           dbWorkspace.Name,
		OwnerId:        dbWorkspace.OwnerId,
		URI:            dbWorkspace.URI,
		Status:         dbWorkspace.Status,
		Description:    dbWorkspace.Description,
		Tier:           dbWorkspace.Tier,
		OrganizationID: dbWorkspace.OrganizationID,
	})
}
