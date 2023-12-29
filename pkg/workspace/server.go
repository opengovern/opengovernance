package workspace

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/subscription/armsubscription"
	"github.com/aws/aws-sdk-go-v2/aws"
	config2 "github.com/aws/aws-sdk-go-v2/config"
	types2 "github.com/aws/aws-sdk-go-v2/service/iam/types"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	"github.com/aws/aws-sdk-go-v2/service/organizations/types"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	kms2 "github.com/aws/aws-sdk-go/service/kms"
	"github.com/aws/smithy-go"
	kaytuAws "github.com/kaytu-io/kaytu-aws-describer/aws"
	"github.com/kaytu-io/kaytu-aws-describer/aws/describer"
	kaytuAzure "github.com/kaytu-io/kaytu-azure-describer/azure"
	api5 "github.com/kaytu-io/kaytu-engine/pkg/analytics/api"
	"github.com/kaytu-io/kaytu-engine/pkg/describe"
	api3 "github.com/kaytu-io/kaytu-engine/pkg/describe/api"
	client3 "github.com/kaytu-io/kaytu-engine/pkg/describe/client"
	"github.com/kaytu-io/kaytu-engine/pkg/httpclient"
	"github.com/kaytu-io/kaytu-engine/pkg/httpserver"
	api4 "github.com/kaytu-io/kaytu-engine/pkg/insight/api"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/config"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/db"
	db2 "github.com/kaytu-io/kaytu-engine/pkg/workspace/db"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/statemanager"
	"github.com/kaytu-io/kaytu-util/pkg/source"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/kaytu-io/kaytu-engine/pkg/onboard/client"

	client2 "github.com/kaytu-io/kaytu-engine/pkg/inventory/client"

	v1 "k8s.io/api/apps/v1"

	"github.com/labstack/gommon/log"

	corev1 "k8s.io/api/core/v1"

	authapi "github.com/kaytu-io/kaytu-engine/pkg/auth/api"
	authclient "github.com/kaytu-io/kaytu-engine/pkg/auth/client"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/api"
	"github.com/labstack/echo/v4"
	contourv1 "github.com/projectcontour/contour/apis/projectcontour/v1"
	"go.uber.org/zap"
	"gorm.io/gorm"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/sony/sonyflake"
)

var (
	ErrInternalServer = errors.New("internal server error")
)

type Server struct {
	logger       *zap.Logger
	e            *echo.Echo
	cfg          config.Config
	db           *db.Database
	authClient   authclient.AuthServiceClient
	kubeClient   k8sclient.Client // the kubernetes client
	StateManager *statemanager.Service
	awsMasterCnf aws.Config
	kms          *kms.Client
}

func NewServer(cfg config.Config) (*Server, error) {
	s := &Server{
		cfg: cfg,
	}

	logger, err := zap.NewProduction()
	if err != nil {
		return nil, fmt.Errorf("new zap logger: %s", err)
	}
	s.e, _ = httpserver.Register(logger, s)

	dbs, err := db.NewDatabase(cfg, logger)
	if err != nil {
		return nil, fmt.Errorf("new database: %w", err)
	}
	s.db = dbs

	kubeClient, err := statemanager.NewKubeClient()
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

	s.authClient = authclient.NewAuthServiceClient(cfg.Auth.BaseURL)

	s.logger = logger

	s.StateManager, err = statemanager.New(cfg)
	if err != nil {
		return nil, err
	}

	awsCfg, err := config2.LoadDefaultConfig(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to load SDK configuration: %v", err)
	}
	awsCfg.Region = cfg.KMSAccountRegion
	s.kms = kms.NewFromConfig(awsCfg)

	return s, nil
}

func (s *Server) Register(e *echo.Echo) {
	v1Group := e.Group("/api/v1")

	workspaceGroup := v1Group.Group("/workspace")
	workspaceGroup.POST("", httpserver.AuthorizeHandler(s.CreateWorkspace, authapi.EditorRole))
	workspaceGroup.DELETE("/:workspace_id", httpserver.AuthorizeHandler(s.DeleteWorkspace, authapi.EditorRole))
	workspaceGroup.GET("/current", httpserver.AuthorizeHandler(s.GetCurrentWorkspace, authapi.ViewerRole))
	workspaceGroup.POST("/:workspace_id/owner", httpserver.AuthorizeHandler(s.ChangeOwnership, authapi.EditorRole))
	workspaceGroup.POST("/:workspace_id/organization", httpserver.AuthorizeHandler(s.ChangeOrganization, authapi.KaytuAdminRole))

	bootstrapGroup := v1Group.Group("/bootstrap")
	bootstrapGroup.GET("/:workspace_name", httpserver.AuthorizeHandler(s.GetBootstrapStatus, authapi.EditorRole))
	bootstrapGroup.POST("/:workspace_name/credential", httpserver.AuthorizeHandler(s.AddCredential, authapi.EditorRole))
	bootstrapGroup.POST("/:workspace_name/finish", httpserver.AuthorizeHandler(s.FinishBootstrap, authapi.EditorRole))

	workspacesGroup := v1Group.Group("/workspaces")
	workspacesGroup.GET("/limits/:workspace_name", httpserver.AuthorizeHandler(s.GetWorkspaceLimits, authapi.ViewerRole))
	workspacesGroup.GET("/byid/:workspace_id", httpserver.AuthorizeHandler(s.GetWorkspaceByID, authapi.InternalRole))
	workspacesGroup.GET("", httpserver.AuthorizeHandler(s.ListWorkspaces, authapi.ViewerRole))
	workspacesGroup.GET("/:workspace_id", httpserver.AuthorizeHandler(s.GetWorkspace, authapi.ViewerRole))
	workspacesGroup.GET("/byname/:workspace_name", httpserver.AuthorizeHandler(s.GetWorkspaceByName, authapi.ViewerRole))

	organizationGroup := v1Group.Group("/organization")
	organizationGroup.GET("", httpserver.AuthorizeHandler(s.ListOrganization, authapi.KaytuAdminRole))
	organizationGroup.POST("", httpserver.AuthorizeHandler(s.CreateOrganization, authapi.KaytuAdminRole))
	organizationGroup.DELETE("/:organizationId", httpserver.AuthorizeHandler(s.DeleteOrganization, authapi.KaytuAdminRole))

	costEstimatorGroup := v1Group.Group("/costestimator")
	costEstimatorGroup.GET("/aws", httpserver.AuthorizeHandler(s.GetAwsCost, authapi.ViewerRole))
	costEstimatorGroup.GET("/azure", httpserver.AuthorizeHandler(s.GetAzureCost, authapi.ViewerRole))
}

func (s *Server) Start() error {
	go s.StateManager.StartReconciler()

	s.e.Logger.SetLevel(log.DEBUG)
	s.e.Logger.Infof("workspace service is started on %s", s.cfg.Http.Address)
	return s.e.Start(s.cfg.Http.Address)
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
	userID := httpserver.GetUserID(c)

	var request api.CreateWorkspaceRequest
	if err := c.Bind(&request); err != nil {
		c.Logger().Errorf("bind the request: %v", err)
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}
	if request.Name == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "name is empty")
	}
	if request.Name == "kaytu" || request.Name == "workspaces" {
		return echo.NewHTTPError(http.StatusBadRequest, "name cannot be kaytu or workspaces")
	}
	if !regexp.MustCompile(`^[a-zA-Z0-9\-]*$`).MatchString(request.Name) {
		return echo.NewHTTPError(http.StatusBadRequest, "name is invalid. only characters, numbers and - is allowed")
	}
	if len(request.Name) >= 150 {
		return echo.NewHTTPError(http.StatusBadRequest, "name must be under 150 characters")
	}

	switch request.Tier {
	case string(api.Tier_Free), string(api.Tier_Teams):
	default:
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid Tier")
	}
	sf := sonyflake.NewSonyflake(sonyflake.Settings{})
	id, err := sf.NextID()
	if err != nil {
		return err
	}

	awsUID, err := sf.NextID()
	if err != nil {
		return err
	}

	var organizationID *int
	if request.OrganizationID != -1 {
		organizationID = &request.OrganizationID
	}

	workspace := &db.Workspace{
		ID:                       fmt.Sprintf("ws-%d", id),
		Name:                     strings.ToLower(request.Name),
		AWSUniqueId:              aws.String(fmt.Sprintf("aws-uid-%d", awsUID)),
		OwnerId:                  &userID,
		Status:                   api.StateID_WaitingForCredential,
		Size:                     api.SizeXS,
		Tier:                     api.Tier(request.Tier),
		OrganizationID:           organizationID,
		IsCreated:                false,
		IsBootstrapInputFinished: false,
		AnalyticsJobID:           0,
		InsightJobsID:            "",
		ComplianceTriggered:      false,
	}

	if err := s.db.CreateWorkspace(workspace); err != nil {
		if strings.Contains(err.Error(), "duplicate key value") {
			return echo.NewHTTPError(http.StatusFound, "workspace already exists")
		}
		c.Logger().Errorf("create workspace: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, ErrInternalServer)
	}

	err = s.authClient.UpdateWorkspaceMap(&httpclient.Context{UserRole: authapi.InternalRole})
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, api.CreateWorkspaceResponse{
		ID: workspace.ID,
	})
}

func (s *Server) getBootstrapStatus(ws *db2.Workspace, azureCount, awsCount int64) (api.BootstrapStatusResponse, error) {
	complianceTotal := 0
	if azureCount > 0 {
		complianceTotal += 4
	}
	if awsCount > 0 {
		complianceTotal += 4
	}
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
			Total: int64(complianceTotal),
		},
		InsightsStatus: api.BootstrapProgress{
			Total: 2,
		},
	}

	hctx := &httpclient.Context{UserRole: authapi.InternalRole}
	schedulerURL := strings.ReplaceAll(s.cfg.Scheduler.BaseURL, "%NAMESPACE%", ws.ID)
	schedulerClient := client3.NewSchedulerServiceClient(schedulerURL)

	if ws.Status == api.StateID_WaitingForCredential || ws.Status == api.StateID_Provisioning {
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

		if ws.ComplianceTriggered {
			if awsCount > 0 {
				awsComplianceJob, err := schedulerClient.GetLatestComplianceJobForBenchmark(hctx, "aws_cis_v200")
				if err != nil {
					return resp, err
				}

				if awsComplianceJob != nil {
					switch awsComplianceJob.Status {
					case api3.ComplianceJobCreated:
						resp.ComplianceStatus.Done += 1
					case api3.ComplianceJobRunnersInProgress:
						resp.ComplianceStatus.Done += 2
					case api3.ComplianceJobSummarizerInProgress:
						resp.ComplianceStatus.Done += 3
					case api3.ComplianceJobSucceeded, api3.ComplianceJobFailed:
						resp.ComplianceStatus.Done += 4
					}
				}
			}

			if azureCount > 0 {
				azureComplianceJob, err := schedulerClient.GetLatestComplianceJobForBenchmark(hctx, "azure_cis_v200")
				if err != nil {
					return resp, err
				}

				if azureComplianceJob != nil {
					switch azureComplianceJob.Status {
					case api3.ComplianceJobCreated:
						resp.ComplianceStatus.Done += 1
					case api3.ComplianceJobRunnersInProgress:
						resp.ComplianceStatus.Done += 2
					case api3.ComplianceJobSummarizerInProgress:
						resp.ComplianceStatus.Done += 3
					case api3.ComplianceJobSucceeded, api3.ComplianceJobFailed:
						resp.ComplianceStatus.Done += 4
					}
				}
			}
		}

		if len(ws.InsightJobsID) > 0 {
			resp.InsightsStatus.Done = 1
			inProgress := false
			for _, insJobIDStr := range strings.Split(ws.InsightJobsID, ",") {
				insJobID, err := strconv.ParseUint(insJobIDStr, 10, 64)
				if err != nil {
					return resp, err
				}

				job, err := schedulerClient.GetInsightJob(hctx, uint(insJobID))
				if err != nil {
					return resp, err
				}

				if job == nil {
					continue
				}

				if job.Status == api4.InsightJobSucceeded {
					inProgress = false
					break
				}
				if job.Status == api4.InsightJobCreated || job.Status == api4.InsightJobInProgress {
					inProgress = true
				}
			}

			if !inProgress {
				resp.InsightsStatus.Done = 2
			}
		}
	} else {
		resp.WorkspaceCreationStatus.Done = resp.WorkspaceCreationStatus.Total
		resp.InsightsStatus.Done = resp.InsightsStatus.Total
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

	if err := s.CheckRoleInWorkspace(c, ws.OwnerId); err != nil {
		return err
	}

	if ws == nil {
		return echo.NewHTTPError(http.StatusBadRequest, errors.New("workspace not found"))
	}

	currentConnectionCount := map[source.Type]int64{}
	awsCount, err := s.db.CountConnectionsByConnector(ws.ID, source.CloudAWS)
	if err != nil {
		return err
	}
	currentConnectionCount[source.CloudAWS] = awsCount

	azureCount, err := s.db.CountConnectionsByConnector(ws.ID, source.CloudAzure)
	if err != nil {
		return err
	}
	currentConnectionCount[source.CloudAzure] = azureCount

	resp, err := s.getBootstrapStatus(ws, azureCount, awsCount)
	if err != nil {
		return err
	}

	limits := api.GetLimitsByTier(ws.Tier)

	resp.MinRequiredConnections = 3
	resp.MaxConnections = limits.MaxConnections
	resp.ConnectionCount = currentConnectionCount
	return c.JSON(http.StatusOK, resp)
}

// FinishBootstrap godoc
//
//	@Summary	finish bootstrap
//	@Security	BearerToken
//	@Tags		workspace
//	@Accept		json
//	@Produce	json
//	@Param		workspace_name	path		string	true	"Workspace Name"
//	@Success	200				{object}	string
//	@Router		/workspace/api/v1/bootstrap/{workspace_name}/finish [post]
func (s *Server) FinishBootstrap(c echo.Context) error {
	workspaceName := c.Param("workspace_name")

	ws, err := s.db.GetWorkspaceByName(workspaceName)
	if err != nil {
		return err
	}

	userID := httpserver.GetUserID(c)
	if *ws.OwnerId != userID {
		return echo.NewHTTPError(http.StatusForbidden, "operation is forbidden")
	}

	err = s.db.SetWorkspaceBootstrapInputFinished(ws.ID)
	if err != nil {
		return err
	}

	err = s.StateManager.UseReservationIfPossible(*ws)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, "")
}

func ignoreAwsOrgError(err error) bool {
	var ae smithy.APIError
	return errors.As(err, &ae) &&
		(ae.ErrorCode() == (&types.AWSOrganizationsNotInUseException{}).ErrorCode() ||
			ae.ErrorCode() == (&types.AccessDeniedException{}).ErrorCode())
}

func generateRoleARN(accountID, roleName string) string {
	return fmt.Sprintf("arn:aws:iam::%s:role/%s", accountID, roleName)
}

// AddCredential godoc
//
//	@Summary	Add credential for workspace to be onboarded
//	@Security	BearerToken
//	@Tags		workspace
//	@Accept		json
//	@Produce	json
//	@Param		workspace_name	path		string						true	"Workspace Name"
//	@Param		request			body		api.AddCredentialRequest	true	"Request"
//	@Success	200				{object}	uint
//	@Router		/workspace/api/v1/bootstrap/{workspace_name}/credential [post]
func (s *Server) AddCredential(ctx echo.Context) error {
	workspaceName := ctx.Param("workspace_name")
	var request api.AddCredentialRequest
	if err := ctx.Bind(&request); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}

	ws, err := s.db.GetWorkspaceByName(workspaceName)
	if err != nil {
		return err
	}

	userID := httpserver.GetUserID(ctx)
	if *ws.OwnerId != userID {
		return echo.NewHTTPError(http.StatusForbidden, "operation is forbidden")
	}

	count := 0
	switch request.ConnectorType {
	case source.CloudAWS:
		masterCred, err := s.db.GetMasterCredentialByWorkspaceUID(*ws.AWSUniqueId)
		if err != nil {
			return err
		}

		var accessKey, secretKey string
		if masterCred != nil {
			decoded, err := base64.StdEncoding.DecodeString(masterCred.Credential)
			if err != nil {
				return err
			}

			result, err := s.kms.Decrypt(context.TODO(), &kms.DecryptInput{
				CiphertextBlob:      decoded,
				EncryptionAlgorithm: kms2.EncryptionAlgorithmSpecSymmetricDefault,
				KeyId:               &s.cfg.KMSKeyARN,
				EncryptionContext:   nil, //TODO-Saleh use workspaceID
			})
			if err != nil {
				return fmt.Errorf("failed to encrypt ciphertext: %v", err)
			}

			var acc types2.AccessKey
			err = json.Unmarshal(result.Plaintext, &acc)
			//err = json.Unmarshal([]byte(masterCred.Credential), &acc)
			if err != nil {
				return err
			}

			accessKey = *acc.AccessKeyId
			secretKey = *acc.SecretAccessKey
		}
		roleARN := generateRoleARN(request.AWSConfig.AccountID, request.AWSConfig.AssumeRoleName)

		var sdkCnf aws.Config
		sdkCnf, err = kaytuAws.GetConfig(context.Background(), accessKey, secretKey, "", roleARN, ws.AWSUniqueId)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
		if sdkCnf.Region == "" {
			sdkCnf.Region = "us-east-1"
		}

		stsClient := sts.NewFromConfig(sdkCnf)
		caller, err := stsClient.GetCallerIdentity(context.Background(), &sts.GetCallerIdentityInput{})
		if err != nil {
			return err
		}

		if sdkCnf.Region == "" {
			sdkCnf.Region = "us-east-1"
		}
		org, err := describer.OrganizationOrganization(context.Background(), sdkCnf)
		if err != nil {
			if !ignoreAwsOrgError(err) {
				return err
			}
		}
		if org != nil && org.MasterAccountId != nil && *org.MasterAccountId == *caller.Account {
			accounts, err := describer.OrganizationAccounts(context.Background(), sdkCnf)
			if err != nil {
				if !ignoreAwsOrgError(err) {
					return err
				}
			}

			for _, account := range accounts {
				if account.Id == nil {
					continue
				}
				count++
			}
		} else {
			count = 1
		}

	case source.CloudAzure:
		var azureConfig describe.AzureSubscriptionConfig
		azureConfig, err = describe.AzureSubscriptionConfigFromMap(request.AzureConfig.AsMap())
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		err = kaytuAzure.CheckSPNAccessPermission(kaytuAzure.AuthConfig{
			TenantID:            azureConfig.TenantID,
			ObjectID:            azureConfig.ObjectID,
			SecretID:            azureConfig.SecretID,
			ClientID:            azureConfig.ClientID,
			ClientSecret:        azureConfig.ClientSecret,
			CertificatePath:     azureConfig.CertificatePath,
			CertificatePassword: azureConfig.CertificatePass,
			Username:            azureConfig.Username,
			Password:            azureConfig.Password,
		})
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}

		identity, err := azidentity.NewClientSecretCredential(
			azureConfig.TenantID,
			azureConfig.ClientID,
			azureConfig.ClientSecret,
			nil)
		if err != nil {
			return err
		}

		subClient, err := armsubscription.NewSubscriptionsClient(identity, nil)
		if err != nil {
			return err
		}

		ctx2 := context.Background()
		it := subClient.NewListPager(nil)
		for it.More() {
			page, err := it.NextPage(ctx2)
			if err != nil {
				return err
			}
			for _, v := range page.Value {
				if v == nil || v.State == nil {
					continue
				}
				count++
			}
		}
	}

	configStr, err := json.Marshal(request)
	if err != nil {
		return err
	}

	result, err := s.kms.Encrypt(context.TODO(), &kms.EncryptInput{
		KeyId:               &s.cfg.KMSKeyARN,
		Plaintext:           configStr,
		EncryptionAlgorithm: kms2.EncryptionAlgorithmSpecSymmetricDefault,
		EncryptionContext:   nil, //TODO-Saleh use workspaceID
		GrantTokens:         nil,
	})
	if err != nil {
		return fmt.Errorf("failed to encrypt ciphertext: %v", err)
	}
	encoded := base64.StdEncoding.EncodeToString(result.CiphertextBlob)

	cred := db2.Credential{
		ConnectorType:    request.ConnectorType,
		WorkspaceID:      ws.ID,
		Metadata:         encoded,
		ConnectionCount:  count,
		SingleConnection: request.SingleConnection,
	}
	err = s.db.CreateCredential(&cred)
	if err != nil {
		return err
	}

	err = s.StateManager.UseReservationIfPossible(*ws)
	if err != nil {
		return err
	}

	return ctx.JSON(http.StatusOK, cred.ID)
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
	if *workspace.OwnerId != userID {
		return echo.NewHTTPError(http.StatusForbidden, "operation is forbidden")
	}

	if err := s.db.UpdateWorkspaceStatus(id, api.StateID_Deleting); err != nil {
		c.Logger().Errorf("delete workspace: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, ErrInternalServer)
	}
	return c.JSON(http.StatusOK, map[string]string{"message": "success"})
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

	if err := s.CheckRoleInWorkspace(c, workspace.OwnerId); err != nil {
		return err
	}

	version := "unspecified"
	var kaytuVersionConfig corev1.ConfigMap
	err = s.kubeClient.Get(context.Background(), k8sclient.ObjectKey{
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

	if err := s.CheckRoleInWorkspace(c, workspace.OwnerId); err != nil {
		return err
	}

	version := "unspecified"
	var kaytuVersionConfig corev1.ConfigMap
	err = s.kubeClient.Get(context.Background(), k8sclient.ObjectKey{
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

	userId := httpserver.GetUserID(c)

	if userId != authapi.GodUserID {
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
		if workspace.Status == api.StateID_Deleted {
			continue
		}

		hasRoleInWorkspace := false
		if userId != authapi.GodUserID {
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

		if workspace.OwnerId == nil || (*workspace.OwnerId != userId && !hasRoleInWorkspace) {
			continue
		}

		version := "unspecified"

		if workspace.IsCreated {
			var kaytuVersionConfig corev1.ConfigMap
			err = s.kubeClient.Get(context.Background(), k8sclient.ObjectKey{
				Namespace: workspace.ID,
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
	wsName := httpserver.GetWorkspaceName(c)

	workspace, err := s.db.GetWorkspaceByName(wsName)
	if err != nil {
		return fmt.Errorf("ListWorkspaces: %v", err)
	}

	version := "unspecified"
	var kaytuVersionConfig corev1.ConfigMap
	err = s.kubeClient.Get(context.Background(), k8sclient.ObjectKey{
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

	if err := s.CheckRoleInWorkspace(c, dbWorkspace.OwnerId); err != nil {
		return err
	}

	if ignoreUsage != "true" {
		ectx := httpclient.FromEchoContext(c)
		ectx.UserRole = authapi.AdminRole
		resp, err := s.authClient.GetWorkspaceRoleBindings(ectx, dbWorkspace.ID)
		if err != nil {
			return fmt.Errorf("GetWorkspaceRoleBindings: %v", err)
		}
		response.CurrentUsers = int64(len(resp))

		inventoryURL := strings.ReplaceAll(s.cfg.Inventory.BaseURL, "%NAMESPACE%", dbWorkspace.ID)
		inventoryClient := client2.NewInventoryServiceClient(inventoryURL)
		resourceCount, err := inventoryClient.CountResources(httpclient.FromEchoContext(c))
		response.CurrentResources = resourceCount

		onboardURL := strings.ReplaceAll(s.cfg.Onboard.BaseURL, "%NAMESPACE%", dbWorkspace.ID)
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
