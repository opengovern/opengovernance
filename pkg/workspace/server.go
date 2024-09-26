package workspace

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	dexApi "github.com/dexidp/dex/api/v2"
	api6 "github.com/hashicorp/vault/api"
	api2 "github.com/kaytu-io/kaytu-util/pkg/api"
	"github.com/kaytu-io/kaytu-util/pkg/httpclient"
	httpserver2 "github.com/kaytu-io/kaytu-util/pkg/httpserver"
	"github.com/kaytu-io/kaytu-util/pkg/postgres"
	"github.com/kaytu-io/kaytu-util/pkg/source"
	"github.com/kaytu-io/kaytu-util/pkg/vault"
	api5 "github.com/kaytu-io/open-governance/pkg/analytics/api"
	client4 "github.com/kaytu-io/open-governance/pkg/compliance/client"
	api3 "github.com/kaytu-io/open-governance/pkg/describe/api"
	client3 "github.com/kaytu-io/open-governance/pkg/describe/client"
	inventoryApi "github.com/kaytu-io/open-governance/pkg/inventory/api"
	client5 "github.com/kaytu-io/open-governance/pkg/metadata/client"
	"github.com/kaytu-io/open-governance/pkg/metadata/models"
	onboardApi "github.com/kaytu-io/open-governance/pkg/onboard/api"
	"github.com/kaytu-io/open-governance/pkg/workspace/config"
	"github.com/kaytu-io/open-governance/pkg/workspace/db"
	db2 "github.com/kaytu-io/open-governance/pkg/workspace/db"
	"github.com/kaytu-io/open-governance/pkg/workspace/statemanager"
	model2 "github.com/kaytu-io/open-governance/services/demo-importer/db/model"
	"github.com/kaytu-io/open-governance/services/migrator/db/model"
	"google.golang.org/grpc"
	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/kaytu-io/open-governance/pkg/onboard/client"

	client2 "github.com/kaytu-io/open-governance/pkg/inventory/client"

	v1 "k8s.io/api/apps/v1"

	"github.com/labstack/gommon/log"

	corev1 "k8s.io/api/core/v1"

	authapi "github.com/kaytu-io/open-governance/pkg/auth/api"
	authclient "github.com/kaytu-io/open-governance/pkg/auth/client"
	"github.com/kaytu-io/open-governance/pkg/workspace/api"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
	"gorm.io/gorm"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	ErrInternalServer = errors.New("internal server error")
)

type Server struct {
	logger             *zap.Logger
	e                  *echo.Echo
	cfg                config.Config
	db                 *db.Database
	migratorDb         *db.Database
	authClient         authclient.AuthServiceClient
	kubeClient         k8sclient.Client // the kubernetes client
	StateManager       *statemanager.Service
	vault              vault.VaultSourceConfig
	vaultSecretHandler vault.VaultSecretHandler
}

func NewServer(ctx context.Context, logger *zap.Logger, cfg config.Config) (*Server, error) {
	s := &Server{
		cfg: cfg,
	}

	s.e, _ = httpserver2.Register(logger, s)

	dbs, err := db.NewDatabase(cfg, logger)
	if err != nil {
		return nil, fmt.Errorf("new database: %w", err)
	}
	s.db = dbs

	migratorDbCfg := postgres.Config{
		Host:    cfg.Postgres.Host,
		Port:    cfg.Postgres.Port,
		User:    cfg.Postgres.Username,
		Passwd:  cfg.Postgres.Password,
		DB:      "migrator",
		SSLMode: cfg.Postgres.SSLMode,
	}
	migratorOrm, err := postgres.NewClient(&migratorDbCfg, logger)
	if err != nil {
		return nil, fmt.Errorf("new postgres client: %w", err)
	}
	if err := migratorOrm.AutoMigrate(&model.Migration{}); err != nil {
		return nil, fmt.Errorf("gorm migrate: %w", err)
	}
	s.migratorDb = &db.Database{Orm: migratorOrm}

	kubeClient, err := statemanager.NewKubeClient()
	if err != nil {
		return nil, fmt.Errorf("new kube client: %w", err)
	}
	s.kubeClient = kubeClient

	err = v1.AddToScheme(s.kubeClient.Scheme())
	if err != nil {
		return nil, fmt.Errorf("add v1 to scheme: %w", err)
	}

	s.authClient = authclient.NewAuthServiceClient(cfg.Auth.BaseURL)

	s.logger = logger

	switch cfg.Vault.Provider {
	case vault.AwsKMS:
		s.vault, err = vault.NewKMSVaultSourceConfig(ctx, cfg.Vault.Aws, cfg.Vault.KeyId)
		if err != nil {
			logger.Error("new kms vaultClient source config", zap.Error(err))
			return nil, fmt.Errorf("new kms vaultClient source config: %w", err)
		}
	case vault.AzureKeyVault:
		s.vault, err = vault.NewAzureVaultClient(ctx, logger, cfg.Vault.Azure, cfg.Vault.KeyId)
		if err != nil {
			logger.Error("new azure vaultClient source config", zap.Error(err))
			return nil, fmt.Errorf("new azure vaultClient source config: %w", err)
		}
		s.vaultSecretHandler, err = vault.NewAzureVaultSecretHandler(logger, cfg.Vault.Azure)
		if err != nil {
			logger.Error("new azure vaultClient secret handler", zap.Error(err))
			return nil, fmt.Errorf("new azure vaultClient secret handler: %w", err)
		}
	case vault.HashiCorpVault:
		s.vaultSecretHandler, err = vault.NewHashiCorpVaultSecretHandler(ctx, logger, cfg.Vault.HashiCorp)
		if err != nil {
			logger.Error("new hashicorp vaultClient secret handler", zap.Error(err))
			return nil, fmt.Errorf("new hashicorp vaultClient secret handler: %w", err)
		}

		s.vault, err = vault.NewHashiCorpVaultClient(ctx, logger, cfg.Vault.HashiCorp, cfg.Vault.KeyId)
		if err != nil {
			if strings.Contains(err.Error(), api6.ErrSecretNotFound.Error()) {
				b := make([]byte, 32)
				_, err := rand.Read(b)
				if err != nil {
					return nil, err
				}

				_, err = s.vaultSecretHandler.SetSecret(ctx, cfg.Vault.KeyId, b)
				if err != nil {
					return nil, err
				}

				s.vault, err = vault.NewHashiCorpVaultClient(ctx, logger, cfg.Vault.HashiCorp, cfg.Vault.KeyId)
				if err != nil {
					logger.Error("new hashicorp vaultClient source config after setSecret", zap.Error(err))
					return nil, fmt.Errorf("new hashicorp vaultClient source config after setSecret: %w", err)
				}
			} else {
				logger.Error("new hashicorp vaultClient source config", zap.Error(err))
				return nil, fmt.Errorf("new hashicorp vaultClient source config: %w", err)
			}
		}
	default:
		return nil, fmt.Errorf("unsupported vault provider: %s", cfg.Vault.Provider)
	}

	s.StateManager, err = statemanager.New(ctx, cfg, s.vault, s.vaultSecretHandler, s.db, s.kubeClient)
	if err != nil {
		return nil, fmt.Errorf("failed to load initiate state manager: %v", err)
	}

	return s, nil
}

func (s *Server) Register(e *echo.Echo) {
	v1Group := e.Group("/api/v1")

	workspaceGroup := v1Group.Group("/workspace")
	workspaceGroup.GET("/current", httpserver2.AuthorizeHandler(s.GetCurrentWorkspace, api2.ViewerRole))
	workspaceGroup.POST("/:workspace_id/owner", httpserver2.AuthorizeHandler(s.ChangeOwnership, api2.EditorRole))
	workspaceGroup.POST("/:workspace_id/organization", httpserver2.AuthorizeHandler(s.ChangeOrganization, api2.KaytuAdminRole))

	bootstrapGroup := v1Group.Group("/bootstrap")
	bootstrapGroup.GET("/:workspace_name", httpserver2.AuthorizeHandler(s.GetBootstrapStatus, api2.EditorRole))

	workspacesGroup := v1Group.Group("/workspaces")
	workspacesGroup.GET("/limits/:workspace_name", httpserver2.AuthorizeHandler(s.GetWorkspaceLimits, api2.ViewerRole))
	workspacesGroup.GET("/byid/:workspace_id", httpserver2.AuthorizeHandler(s.GetWorkspaceByID, api2.InternalRole))
	workspacesGroup.GET("", httpserver2.AuthorizeHandler(s.ListWorkspaces, api2.ViewerRole))
	workspacesGroup.GET("/:workspace_id", httpserver2.AuthorizeHandler(s.GetWorkspace, api2.ViewerRole))
	workspacesGroup.GET("/byname/:workspace_name", httpserver2.AuthorizeHandler(s.GetWorkspaceByName, api2.ViewerRole))

	organizationGroup := v1Group.Group("/organization")
	organizationGroup.GET("", httpserver2.AuthorizeHandler(s.ListOrganization, api2.KaytuAdminRole))
	organizationGroup.POST("", httpserver2.AuthorizeHandler(s.CreateOrganization, api2.KaytuAdminRole))
	organizationGroup.DELETE("/:organizationId", httpserver2.AuthorizeHandler(s.DeleteOrganization, api2.KaytuAdminRole))

	costEstimatorGroup := v1Group.Group("/costestimator")
	costEstimatorGroup.GET("/aws", httpserver2.AuthorizeHandler(s.GetAwsCost, api2.ViewerRole))
	costEstimatorGroup.GET("/azure", httpserver2.AuthorizeHandler(s.GetAzureCost, api2.ViewerRole))

	v3 := e.Group("/api/v3")
	v3.PUT("/sample/purge", httpserver2.AuthorizeHandler(s.PurgeSampleData, api2.ViewerRole))
	v3.PUT("/sample/sync", httpserver2.AuthorizeHandler(s.SyncDemo, api2.ViewerRole))
	v3.PUT("/sample/loaded", httpserver2.AuthorizeHandler(s.WorkspaceLoadedSampleData, api2.ViewerRole))
	v3.GET("/sample/sync/status", httpserver2.AuthorizeHandler(s.GetSampleSyncStatus, api2.ViewerRole))
	v3.GET("/migration/status", httpserver2.AuthorizeHandler(s.GetMigrationStatus, api2.ViewerRole))
	v3.GET("/configured/status", httpserver2.AuthorizeHandler(s.GetConfiguredStatus, api2.ViewerRole))
	v3.PUT("/configured/set", httpserver2.AuthorizeHandler(s.SetConfiguredStatus, api2.InternalRole))
	v3.PUT("/configured/unset", httpserver2.AuthorizeHandler(s.UnsetConfiguredStatus, api2.ViewerRole))
	v3.GET("/about", httpserver2.AuthorizeHandler(s.GetAbout, api2.ViewerRole))
}

func (s *Server) Start(ctx context.Context) error {
	go s.StateManager.StartReconciler(ctx)

	s.e.Logger.SetLevel(log.DEBUG)
	s.e.Logger.Infof("workspace service is started on %s", s.cfg.Http.Address)
	return s.e.Start(s.cfg.Http.Address)
}

func (s *Server) getBootstrapStatus(ws *db2.Workspace) (api.BootstrapStatusResponse, error) {
	resp := api.BootstrapStatusResponse{
		MinRequiredConnections: 3,
		WorkspaceCreationStatus: api.BootstrapProgress{
			Total: 2,
		},
		DiscoveryStatus: api.BootstrapProgress{
			Total: 4,
		},
		AnalyticsStatus: api.BootstrapProgress{
			Total: 4,
		},
		ComplianceStatus: api.BootstrapProgress{
			Total: int64(0),
		},
	}

	hctx := &httpclient.Context{UserRole: api2.InternalRole}
	schedulerURL := strings.ReplaceAll(s.cfg.Scheduler.BaseURL, "%NAMESPACE%", s.cfg.KaytuOctopusNamespace)
	schedulerClient := client3.NewSchedulerServiceClient(schedulerURL)

	if ws.Status == api.StateID_Provisioning {
		if !ws.IsBootstrapInputFinished {
			return resp, nil
		}
		resp.WorkspaceCreationStatus.Done = 1

		if !ws.IsCreated {
			return resp, nil
		}
		resp.WorkspaceCreationStatus.Done = 2

		status, err := schedulerClient.GetDescribeAllJobsStatus(hctx)
		if err != nil {
			return resp, err
		}

		if status != nil {
			switch *status {
			case api3.DescribeAllJobsStatusNoJobToRun:
				resp.DiscoveryStatus.Done = 1
			case api3.DescribeAllJobsStatusJobsRunning:
				resp.DiscoveryStatus.Done = 2
			case api3.DescribeAllJobsStatusJobsFinished:
				resp.DiscoveryStatus.Done = 3
			case api3.DescribeAllJobsStatusResourcesPublished:
				resp.DiscoveryStatus.Done = 4
			}
		}

		if ws.AnalyticsJobID > 0 {
			resp.AnalyticsStatus.Done = 1
			job, err := schedulerClient.GetAnalyticsJob(hctx, ws.AnalyticsJobID)
			if err != nil {
				return resp, err
			}
			if job != nil {
				switch job.Status {
				case api5.JobCreated:
					resp.AnalyticsStatus.Done = 2
				case api5.JobInProgress:
					resp.AnalyticsStatus.Done = 3
				case api5.JobCompleted, api5.JobCompletedWithFailure:
					resp.AnalyticsStatus.Done = 4
				}
			}
		}
	} else {
		resp.WorkspaceCreationStatus.Done = resp.WorkspaceCreationStatus.Total
		resp.ComplianceStatus.Done = resp.ComplianceStatus.Total
		resp.DiscoveryStatus.Done = resp.DiscoveryStatus.Total
		resp.AnalyticsStatus.Done = resp.AnalyticsStatus.Total
	}

	return resp, nil
}

// GetBootstrapStatus godoc
//
//	@Summary	Get bootstrap status
//	@Security	BearerToken
//	@Tags		workspace
//	@Accept		json
//	@Produce	json
//	@Param		workspace_name	path		string	true	"Workspace Name"
//	@Success	200				{object}	api.BootstrapStatusResponse
//	@Router		/workspace/api/v1/bootstrap/{workspace_name} [get]
func (s *Server) GetBootstrapStatus(c echo.Context) error {
	workspaceName := c.Param("workspace_name")
	ws, err := s.db.GetWorkspaceByName(workspaceName)
	if err != nil {
		return err
	}

	if err := s.CheckRoleInWorkspace(c, &ws.ID, ws.OwnerId, workspaceName); err != nil {
		return err
	}

	if ws == nil {
		return echo.NewHTTPError(http.StatusBadRequest, errors.New("workspace not found"))
	}

	resp, err := s.getBootstrapStatus(ws)
	if err != nil {
		return err
	}

	limits := api.GetLimitsByTier(ws.Tier)

	resp.MinRequiredConnections = 3
	resp.MaxConnections = limits.MaxConnections
	resp.ConnectionCount = make(map[source.Type]int64)
	return c.JSON(http.StatusOK, resp)
}

func (s *Server) GetWorkspace(c echo.Context) error {
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

	if err := s.CheckRoleInWorkspace(c, &workspace.ID, workspace.OwnerId, workspace.Name); err != nil {
		return err
	}

	version := "unspecified"
	var kaytuVersionConfig corev1.ConfigMap
	err = s.kubeClient.Get(c.Request().Context(), k8sclient.ObjectKey{
		Namespace: s.cfg.KaytuOctopusNamespace,
		Name:      "kaytu-version",
	}, &kaytuVersionConfig)
	if err == nil {
		version = kaytuVersionConfig.Data["version"]
	} else {
		fmt.Printf("failed to load version due to %v\n", err)
	}

	return c.JSON(http.StatusOK, api.WorkspaceResponse{
		Workspace: workspace.ToAPI(),
		Version:   version,
	})
}

func (s *Server) GetWorkspaceByName(c echo.Context) error {
	name := c.Param("workspace_name")
	if name == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "workspace name is empty")
	}

	workspace, err := s.db.GetWorkspaceByName(name)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "workspace not found")
		}
		c.Logger().Errorf("find workspace: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, ErrInternalServer)
	}

	if err := s.CheckRoleInWorkspace(c, &workspace.ID, workspace.OwnerId, name); err != nil {
		return err
	}

	version := "unspecified"
	var kaytuVersionConfig corev1.ConfigMap
	err = s.kubeClient.Get(c.Request().Context(), k8sclient.ObjectKey{
		Namespace: workspace.ID,
		Name:      "kaytu-version",
	}, &kaytuVersionConfig)
	if err == nil {
		version = kaytuVersionConfig.Data["version"]
	} else {
		fmt.Printf("failed to load version due to %v\n", err)
	}

	return c.JSON(http.StatusOK, api.WorkspaceResponse{
		Workspace: workspace.ToAPI(),
		Version:   version,
	})
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
	var resp authapi.GetRoleBindingsResponse
	var err error

	userId := httpserver2.GetUserID(c)

	if userId != api2.GodUserID {
		resp, err = s.authClient.GetUserRoleBindings(httpclient.FromEchoContext(c))
		if err != nil {
			return fmt.Errorf("GetUserRoleBindings: %v", err)
		}
	}

	dbWorkspaces, err := s.db.ListWorkspaces()
	if err != nil {
		return fmt.Errorf("ListWorkspaces: %v", err)
	}

	workspaces := make([]*api.WorkspaceResponse, 0)
	for _, workspace := range dbWorkspaces {
		hasRoleInWorkspace := false
		if userId != api2.GodUserID {
			for _, rb := range resp.RoleBindings {
				if rb.WorkspaceID == workspace.ID {
					hasRoleInWorkspace = true
				}
			}
			if resp.GlobalRoles != nil {
				hasRoleInWorkspace = true
			}
		} else {
			// god has role in everything
			hasRoleInWorkspace = true
		}

		if workspace.OwnerId != nil && *workspace.OwnerId == "kaytu|owner|all" {
			hasRoleInWorkspace = true
		}

		if workspace.OwnerId == nil || (*workspace.OwnerId != userId && !hasRoleInWorkspace) {
			continue
		}

		version := "unspecified"

		if workspace.IsCreated {
			var kaytuVersionConfig corev1.ConfigMap
			err = s.kubeClient.Get(c.Request().Context(), k8sclient.ObjectKey{
				Namespace: s.cfg.KaytuOctopusNamespace,
				Name:      "kaytu-version",
			}, &kaytuVersionConfig)
			if err == nil {
				version = kaytuVersionConfig.Data["version"]
			} else {
				fmt.Printf("failed to load version due to %v\n", err)
			}
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
	var kaytuVersionConfig corev1.ConfigMap
	err = s.kubeClient.Get(c.Request().Context(), k8sclient.ObjectKey{
		Namespace: workspace.ID,
		Name:      "kaytu-version",
	}, &kaytuVersionConfig)
	if err == nil {
		version = kaytuVersionConfig.Data["version"]
	} else {
		fmt.Printf("failed to load version due to %v\n", err)
	}

	return c.JSON(http.StatusOK, api.WorkspaceResponse{
		Workspace: workspace.ToAPI(),
		Version:   version,
	})
}

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

	if *w.OwnerId != userID {
		return echo.NewHTTPError(http.StatusForbidden, "operation is forbidden")
	}

	err = s.db.UpdateWorkspaceOwner(workspaceID, request.NewOwnerUserID)
	if err != nil {
		return err
	}

	return c.NoContent(http.StatusOK)
}

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

	if err := s.CheckRoleInWorkspace(c, &dbWorkspace.ID, dbWorkspace.OwnerId, workspaceName); err != nil {
		return err
	}

	if ignoreUsage != "true" {
		ectx := httpclient.FromEchoContext(c)
		ectx.UserRole = api2.AdminRole
		resp, err := s.authClient.GetWorkspaceRoleBindings(ectx, dbWorkspace.ID)
		if err != nil {
			return fmt.Errorf("GetWorkspaceRoleBindings: %v", err)
		}
		response.CurrentUsers = int64(len(resp))

		inventoryURL := strings.ReplaceAll(s.cfg.Inventory.BaseURL, "%NAMESPACE%", s.cfg.KaytuOctopusNamespace)
		inventoryClient := client2.NewInventoryServiceClient(inventoryURL)
		resourceCount, err := inventoryClient.CountResources(httpclient.FromEchoContext(c))
		response.CurrentResources = resourceCount

		onboardURL := strings.ReplaceAll(s.cfg.Onboard.BaseURL, "%NAMESPACE%", s.cfg.KaytuOctopusNamespace)
		onboardClient := client.NewOnboardServiceClient(onboardURL)
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

func (s *Server) GetWorkspaceByID(c echo.Context) error {
	workspaceID := c.Param("workspace_id")

	dbWorkspace, err := s.db.GetWorkspace(workspaceID)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, dbWorkspace.ToAPI())
}

func (s *Server) CreateOrganization(c echo.Context) error {
	var request api.Organization
	if err := c.Bind(&request); err != nil {
		return err
	}

	dbOrg := db.Organization{
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

func (s *Server) ListOrganization(c echo.Context) error {
	orgs, err := s.db.ListOrganizations()
	if err != nil {
		return err
	}

	var apiOrg []api.Organization
	for _, org := range orgs {
		apiOrg = append(apiOrg, org.ToAPI())
	}
	return c.JSON(http.StatusCreated, apiOrg)
}

func (s *Server) DeleteOrganization(c echo.Context) error {
	organizationIDStr := c.Param("organizationId")
	organizationID, err := strconv.ParseInt(organizationIDStr, 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid organization ID")
	}
	_, err = s.db.GetOrganization(uint(organizationID))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
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

// PurgeSampleData godoc
//
//	@Summary		List all workspaces with owner id
//	@Description	Returns all workspaces with owner id
//	@Security		BearerToken
//	@Tags			workspace
//	@Accept			json
//	@Produce		json
//	@Success		200
//	@Router			/workspace/api/v3/sample/purge [put]
func (s *Server) PurgeSampleData(c echo.Context) error {
	ctx := &httpclient.Context{UserRole: api2.InternalRole}

	ws, err := s.db.GetWorkspaceByName("main")
	if err != nil {
		s.logger.Error("failed to get workspace", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to get workspace")
	}
	if ws.ContainSampleData == false {
		return echo.NewHTTPError(http.StatusNotFound, "Workspace does not contain sample data")
	}

	schedulerURL := strings.ReplaceAll(s.cfg.Scheduler.BaseURL, "%NAMESPACE%", s.cfg.KaytuOctopusNamespace)
	schedulerClient := client3.NewSchedulerServiceClient(schedulerURL)

	complianceURL := strings.ReplaceAll(s.cfg.Compliance.BaseURL, "%NAMESPACE%", s.cfg.KaytuOctopusNamespace)
	complianceClient := client4.NewComplianceClient(complianceURL)

	onboardURL := strings.ReplaceAll(s.cfg.Onboard.BaseURL, "%NAMESPACE%", s.cfg.KaytuOctopusNamespace)
	onboardClient := client.NewOnboardServiceClient(onboardURL)

	err = schedulerClient.PurgeSampleData(ctx)
	if err != nil {
		s.logger.Error("failed to purge scheduler data", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to purge scheduler data")
	}
	err = complianceClient.PurgeSampleData(ctx)
	if err != nil {
		s.logger.Error("failed to purge compliance data", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to purge compliance data")
	}
	err = onboardClient.PurgeSampleData(ctx)
	if err != nil {
		s.logger.Error("failed to purge onboard data", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to purge onboard data")
	}

	err = s.db.WorkspaceSampleDataDeleted("main")
	if err != nil {
		s.logger.Error("failed to update workspace sample data check", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to update workspace sample data check")
	}

	return c.NoContent(http.StatusOK)
}

// SyncDemo godoc
//
//	@Summary		Sync demo
//
//	@Description	Syncs demo with the git backend.
//
//	@Security		BearerToken
//	@Tags			compliance
//	@Param			demo_data_s3_url	query	string	false	"Demo Data S3 URL"
//	@Accept			json
//	@Produce		json
//	@Success		200
//	@Router			/workspace/api/v3/sample/sync [put]
func (s *Server) SyncDemo(echoCtx echo.Context) error {
	ctx := echoCtx.Request().Context()

	metadataURL := strings.ReplaceAll(s.cfg.Metadata.BaseURL, "%NAMESPACE%", s.cfg.KaytuOctopusNamespace)
	metadataClient := client5.NewMetadataServiceClient(metadataURL)

	enabled, err := metadataClient.GetConfigMetadata(httpclient.FromEchoContext(echoCtx), models.MetadataKeyCustomizationEnabled)
	if err != nil {
		s.logger.Error("get config metadata", zap.Error(err))
		return err
	}

	if !enabled.GetValue().(bool) {
		return echo.NewHTTPError(http.StatusForbidden, "customization is not allowed")
	}

	demoDataS3URL := echoCtx.QueryParam("demo_data_s3_url")
	if demoDataS3URL != "" {
		// validate url
		_, err := url.ParseRequestURI(demoDataS3URL)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid url")
		}

		err = metadataClient.SetConfigMetadata(httpclient.FromEchoContext(echoCtx), models.DemoDataS3URL, demoDataS3URL)
		if err != nil {
			s.logger.Error("set config metadata", zap.Error(err))
			return err
		}
	}

	var importDemoJob batchv1.Job
	err = s.kubeClient.Get(ctx, k8sclient.ObjectKey{
		Namespace: s.cfg.KaytuOctopusNamespace,
		Name:      "import-es-demo-data",
	}, &importDemoJob)
	if err != nil {
		return err
	}

	err = s.kubeClient.Delete(ctx, &importDemoJob)
	if err != nil {
		return err
	}

	for {
		err = s.kubeClient.Get(ctx, k8sclient.ObjectKey{
			Namespace: s.cfg.KaytuOctopusNamespace,
			Name:      "import-es-demo-data",
		}, &importDemoJob)
		if err != nil {
			if k8sclient.IgnoreNotFound(err) == nil {
				break
			}
			return err
		}

		time.Sleep(1 * time.Second)
	}

	importDemoJob.ObjectMeta = metav1.ObjectMeta{
		Name:      "import-es-demo-data",
		Namespace: s.cfg.KaytuOctopusNamespace,
		Annotations: map[string]string{
			"helm.sh/hook":        "post-install,post-upgrade",
			"helm.sh/hook-weight": "0",
		},
	}
	importDemoJob.Spec.Selector = nil
	importDemoJob.Spec.Suspend = aws.Bool(false)
	importDemoJob.Spec.Template.ObjectMeta = metav1.ObjectMeta{}
	importDemoJob.Status = batchv1.JobStatus{}

	err = s.kubeClient.Create(ctx, &importDemoJob)
	if err != nil {
		return err
	}

	var importDemoDbJob batchv1.Job
	err = s.kubeClient.Get(ctx, k8sclient.ObjectKey{
		Namespace: s.cfg.KaytuOctopusNamespace,
		Name:      "import-psql-demo-data",
	}, &importDemoDbJob)
	if err != nil {
		return err
	}

	err = s.kubeClient.Delete(ctx, &importDemoDbJob)
	if err != nil {
		return err
	}

	for {
		err = s.kubeClient.Get(ctx, k8sclient.ObjectKey{
			Namespace: s.cfg.KaytuOctopusNamespace,
			Name:      "import-psql-demo-data",
		}, &importDemoDbJob)
		if err != nil {
			if k8sclient.IgnoreNotFound(err) == nil {
				break
			}
			return err
		}

		time.Sleep(1 * time.Second)
	}

	importDemoDbJob.ObjectMeta = metav1.ObjectMeta{
		Name:      "import-psql-demo-data",
		Namespace: s.cfg.KaytuOctopusNamespace,
		Annotations: map[string]string{
			"helm.sh/hook":        "post-install,post-upgrade",
			"helm.sh/hook-weight": "0",
		},
	}
	importDemoDbJob.Spec.Selector = nil
	importDemoDbJob.Spec.Suspend = aws.Bool(false)
	importDemoDbJob.Spec.Template.ObjectMeta = metav1.ObjectMeta{}
	importDemoDbJob.Status = batchv1.JobStatus{}

	err = s.kubeClient.Create(ctx, &importDemoDbJob)
	if err != nil {
		return err
	}
	err = s.db.WorkspaceSampleDataSynced("main")
	if err != nil {
		s.logger.Error("failed to update workspace sample data check", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to update workspace sample data check")
	}

	return echoCtx.JSON(http.StatusOK, struct{}{})
}

// WorkspaceLoadedSampleData godoc
//
//	@Summary		Sync demo
//
//	@Description	Syncs demo with the git backend.
//
//	@Security		BearerToken
//	@Tags			compliance
//	@Param			demo_data_s3_url	query	string	false	"Demo Data S3 URL"
//	@Accept			json
//	@Produce		json
//	@Success		200
//	@Router			/workspace/api/v3/sample/loaded [put]
func (s *Server) WorkspaceLoadedSampleData(echoCtx echo.Context) error {
	ws, err := s.db.GetWorkspaceByName("main")
	if err != nil {
		s.logger.Error("failed to get workspace", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to get workspace")
	}

	if ws.ContainSampleData {
		return echoCtx.String(http.StatusOK, "True")
	} else {
		return echoCtx.String(http.StatusOK, "False")
	}
}

// GetMigrationStatus godoc
//
//	@Summary		Sync demo
//
//	@Description	Syncs demo with the git backend.
//
//	@Security		BearerToken
//	@Tags			compliance
//	@Param			demo_data_s3_url	query	string	false	"Demo Data S3 URL"
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	api.GetMigrationStatusResponse
//	@Router			/workspace/api/v3/migration/status [get]
func (s *Server) GetMigrationStatus(echoCtx echo.Context) error {
	var mig model.Migration
	tx := s.migratorDb.Orm.Model(&model.Migration{}).Where("id = ?", "main").First(&mig)
	if tx.Error != nil {
		s.logger.Error("failed to get migration", zap.Error(tx.Error))
		return echoCtx.JSON(http.StatusInternalServerError, "failed to get migration")
	}
	jobsStatus := make(map[string]model.JobsStatus)

	if len(mig.JobsStatus.Bytes) > 0 {
		err := json.Unmarshal(mig.JobsStatus.Bytes, &jobsStatus)
		if err != nil {
			return err
		}
	}

	return echoCtx.JSON(http.StatusOK, api.GetMigrationStatusResponse{
		Status:     mig.Status,
		JobsStatus: jobsStatus,
	})
}

// GetSampleSyncStatus godoc
//
//	@Summary		Sync demo
//
//	@Description	Syncs demo with the git backend.
//
//	@Security		BearerToken
//	@Tags			compliance
//	@Param			demo_data_s3_url	query	string	false	"Demo Data S3 URL"
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	api.GetSampleSyncStatusResponse
//	@Router			/workspace/api/v3/sample/sync/status [get]
func (s *Server) GetSampleSyncStatus(echoCtx echo.Context) error {
	var mig model.Migration
	tx := s.migratorDb.Orm.Model(&model2.Migration{}).Where("id = ?", model2.MigrationJobName).First(&mig)
	if tx.Error != nil {
		s.logger.Error("failed to get migration", zap.Error(tx.Error))
		return echoCtx.JSON(http.StatusInternalServerError, "failed to get migration")
	}
	var jobsStatus model2.ESImportProgress

	if len(mig.JobsStatus.Bytes) > 0 {
		err := json.Unmarshal(mig.JobsStatus.Bytes, &jobsStatus)
		if err != nil {
			return err
		}
	}
	return echoCtx.JSON(http.StatusOK, api.GetSampleSyncStatusResponse{
		Status:       mig.Status,
		JobsProgress: jobsStatus,
	})
}

// GetConfiguredStatus godoc
//
//	@Summary		Sync demo
//
//	@Description	Syncs demo with the git backend.
//
//	@Security		BearerToken
//	@Tags			compliance
//	@Param			demo_data_s3_url	query	string	false	"Demo Data S3 URL"
//	@Accept			json
//	@Produce		json
//	@Success		200
//	@Router			/workspace/api/v3/configured/status [get]
func (s *Server) GetConfiguredStatus(echoCtx echo.Context) error {
	ws, err := s.db.GetWorkspaceByName("main")
	if err != nil {
		s.logger.Error("failed to get workspace", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to get workspace")
	}

	if ws.Configured {
		return echoCtx.String(http.StatusOK, "True")
	} else {
		return echoCtx.String(http.StatusOK, "False")
	}
}

// SetConfiguredStatus godoc
//
//	@Summary		Sync demo
//
//	@Description	Syncs demo with the git backend.
//
//	@Security		BearerToken
//	@Tags			compliance
//	@Param			demo_data_s3_url	query	string	false	"Demo Data S3 URL"
//	@Accept			json
//	@Produce		json
//	@Success		200
//	@Router			/workspace/api/v3/configured/set [put]
func (s *Server) SetConfiguredStatus(echoCtx echo.Context) error {
	err := s.db.WorkspaceConfigured("main", true)
	if err != nil {
		s.logger.Error("failed to set workspace configured", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to set workspace configured")
	}
	return echoCtx.NoContent(http.StatusOK)
}

// UnsetConfiguredStatus godoc
//
//	@Summary		Sync demo
//
//	@Description	Syncs demo with the git backend.
//
//	@Security		BearerToken
//	@Tags			compliance
//	@Param			demo_data_s3_url	query	string	false	"Demo Data S3 URL"
//	@Accept			json
//	@Produce		json
//	@Success		200
//	@Router			/workspace/api/v3/configured/unset [put]
func (s *Server) UnsetConfiguredStatus(echoCtx echo.Context) error {
	err := s.db.WorkspaceConfigured("main", false)
	if err != nil {
		s.logger.Error("failed to unset workspace configured", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to unset workspace configured")
	}
	return echoCtx.NoContent(http.StatusOK)
}

// GetAbout godoc
//
//	@Summary		Get About info
//
//	@Description	Syncs demo with the git backend.
//
//	@Security		BearerToken
//	@Tags			compliance
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	api.About
//	@Router			/workspace/api/v3/configured/status [put]
func (s *Server) GetAbout(echoCtx echo.Context) error {
	ctx := httpclient.FromEchoContext(echoCtx)
	ctx.UserRole = api2.AdminRole

	ws, err := s.db.GetWorkspaceByName("main")
	if err != nil {
		s.logger.Error("failed to get workspace info", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to get workspace info")
	}

	version := ""
	var kaytuVersionConfig corev1.ConfigMap
	err = s.kubeClient.Get(echoCtx.Request().Context(), k8sclient.ObjectKey{
		Namespace: s.cfg.KaytuOctopusNamespace,
		Name:      "kaytu-version",
	}, &kaytuVersionConfig)
	if err == nil {
		version = kaytuVersionConfig.Data["version"]
	} else {
		fmt.Printf("failed to load version due to %v\n", err)
	}

	users, err := s.authClient.GetWorkspaceRoleBindings(ctx, ws.ID)
	if err != nil {
		s.logger.Error("failed to get users", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to get users")
	}
	apiKeys, err := s.authClient.ListAPIKeys(ctx, ws.ID)
	if err != nil {
		s.logger.Error("failed to get api keys", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to get api keys")
	}

	onboardURL := strings.ReplaceAll(s.cfg.Onboard.BaseURL, "%NAMESPACE%", s.cfg.KaytuOctopusNamespace)
	onboardClient := client.NewOnboardServiceClient(onboardURL)
	connections, err := onboardClient.ListSources(ctx, nil)

	integrations := make(map[string][]onboardApi.Connection)
	for _, c := range connections {
		if _, ok := integrations[c.Connector.String()]; !ok {
			integrations[c.Connector.String()] = make([]onboardApi.Connection, 0)
		}
		integrations[c.Connector.String()] = append(integrations[c.Connector.String()], c)
	}

	inventoryURL := strings.ReplaceAll(s.cfg.Inventory.BaseURL, "%NAMESPACE%", s.cfg.KaytuOctopusNamespace)
	inventoryClient := client2.NewInventoryServiceClient(inventoryURL)

	var engine inventoryApi.QueryEngine
	engine = inventoryApi.QueryEngine_OdysseusSQL
	query := `SELECT
    (SELECT SUM(cost) FROM azure_costmanagement_costbyresourcetype) +
    (SELECT SUM(amortized_cost_amount) FROM aws_cost_by_service_daily) AS total_cost;`
	results, err := inventoryClient.RunQuery(ctx, inventoryApi.RunQueryRequest{
		Page: inventoryApi.Page{
			No:   1,
			Size: 1000,
		},
		Engine: &engine,
		Query:  &query,
		Sorts:  nil,
	})
	if err != nil {
		s.logger.Error("failed to run query", zap.Error(err))
	}

	var floatValue float64
	if results != nil {
		s.logger.Info("query result", zap.Any("result", results.Result))
		if len(results.Result) > 0 && len(results.Result[0]) > 0 {
			totalSpent := results.Result[0][0]
			floatValue, _ = totalSpent.(float64)
		}
	}

	var dexConnectors []api.DexConnectorInfo
	dexClient, err := newDexClient(s.cfg.DexGrpcAddr)
	if err != nil {
		s.logger.Error("failed to create dex client", zap.Error(err))
		return echo.NewHTTPError(http.StatusBadRequest, "failed to create dex client")
	}

	if dexClient != nil {
		dexRes, err := dexClient.ListConnectors(context.Background(), &dexApi.ListConnectorReq{})
		if err != nil {
			s.logger.Error("failed to list dex connectors", zap.Error(err))
			return echo.NewHTTPError(http.StatusBadRequest, "failed to list dex connectors")
		}
		if dexRes != nil {
			for _, c := range dexRes.Connectors {
				dexConnectors = append(dexConnectors, api.DexConnectorInfo{
					ID:   c.Id,
					Name: c.Name,
					Type: c.Type,
				})
			}
		}
	}

	response := api.About{
		DexConnectors:         dexConnectors,
		AppVersion:            version,
		WorkspaceCreationTime: ws.CreatedAt,
		Users:                 users,
		PrimaryDomainURL:      s.cfg.PrimaryDomainURL,
		APIKeys:               apiKeys,
		Integrations:          integrations,
		SampleData:            ws.ContainSampleData,
		TotalSpendGoverned:    floatValue,
	}

	return echoCtx.JSON(http.StatusOK, response)
}

func newDexClient(hostAndPort string) (dexApi.DexClient, error) {
	conn, err := grpc.NewClient(hostAndPort, grpc.WithInsecure())
	if err != nil {
		return nil, fmt.Errorf("dial: %v", err)
	}
	return dexApi.NewDexClient(conn), nil
}
