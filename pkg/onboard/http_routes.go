package onboard

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/subscription/armsubscription"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	api2 "github.com/kaytu-io/kaytu-engine/pkg/inventory/api"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/kaytu-io/kaytu-engine/pkg/internal/httpclient"
	"github.com/kaytu-io/kaytu-engine/pkg/internal/httpserver"
	"go.uber.org/zap"

	api3 "github.com/kaytu-io/kaytu-engine/pkg/auth/api"
	"github.com/kaytu-io/kaytu-engine/pkg/utils"
	"github.com/kaytu-io/kaytu-util/pkg/source"
	"github.com/labstack/echo/v4"

	awsOrgTypes "github.com/aws/aws-sdk-go-v2/service/organizations/types"
	"github.com/google/uuid"
	kaytuAws "github.com/kaytu-io/kaytu-aws-describer/aws"
	kaytuAzure "github.com/kaytu-io/kaytu-azure-describer/azure"
	"github.com/kaytu-io/kaytu-engine/pkg/describe"
	"github.com/kaytu-io/kaytu-engine/pkg/onboard/api"
	"gorm.io/gorm"
)

const (
	paramSourceId     = "sourceId"
	paramCredentialId = "credentialId"
)

func (h HttpHandler) Register(r *echo.Echo) {
	v1 := r.Group("/api/v1")

	v1.GET("/sources", httpserver.AuthorizeHandler(h.ListSources, api3.ViewerRole))
	v1.POST("/sources", httpserver.AuthorizeHandler(h.GetSources, api3.KaytuAdminRole))
	v1.GET("/sources/count", httpserver.AuthorizeHandler(h.CountSources, api3.ViewerRole))
	v1.GET("/catalog/metrics", httpserver.AuthorizeHandler(h.CatalogMetrics, api3.ViewerRole))

	connector := v1.Group("/connector")
	connector.GET("", httpserver.AuthorizeHandler(h.ListConnectors, api3.ViewerRole))

	sourceApiGroup := v1.Group("/source")
	sourceApiGroup.POST("/aws", httpserver.AuthorizeHandler(h.PostSourceAws, api3.EditorRole))
	sourceApiGroup.POST("/azure", httpserver.AuthorizeHandler(h.PostSourceAzure, api3.EditorRole))
	sourceApiGroup.GET("/:sourceId", httpserver.AuthorizeHandler(h.GetSource, api3.KaytuAdminRole))
	sourceApiGroup.GET("/:sourceId/healthcheck", httpserver.AuthorizeHandler(h.GetConnectionHealth, api3.EditorRole))
	sourceApiGroup.GET("/:sourceId/credentials/full", httpserver.AuthorizeHandler(h.GetSourceFullCred, api3.KaytuAdminRole))
	sourceApiGroup.DELETE("/:sourceId", httpserver.AuthorizeHandler(h.DeleteSource, api3.EditorRole))

	credential := v1.Group("/credential")
	credential.POST("", httpserver.AuthorizeHandler(h.PostCredentials, api3.EditorRole))
	credential.PUT("/:credentialId", httpserver.AuthorizeHandler(h.PutCredentials, api3.EditorRole))
	credential.GET("", httpserver.AuthorizeHandler(h.ListCredentials, api3.ViewerRole))
	credential.DELETE("/:credentialId", httpserver.AuthorizeHandler(h.DeleteCredential, api3.EditorRole))
	credential.GET("/:credentialId", httpserver.AuthorizeHandler(h.GetCredential, api3.ViewerRole))
	credential.POST("/:credentialId/autoonboard", httpserver.AuthorizeHandler(h.AutoOnboardCredential, api3.EditorRole))

	connections := v1.Group("/connections")
	connections.GET("/summary", httpserver.AuthorizeHandler(h.ListConnectionsSummaries, api3.ViewerRole))
	connections.POST("/:connectionId/state", httpserver.AuthorizeHandler(h.ChangeConnectionLifecycleState, api3.EditorRole))

	connectionGroups := v1.Group("/connection-groups")
	connectionGroups.GET("", httpserver.AuthorizeHandler(h.ListConnectionGroups, api3.ViewerRole))
	connectionGroups.GET("/:connectionGroupName", httpserver.AuthorizeHandler(h.GetConnectionGroup, api3.ViewerRole))
}

func bindValidate(ctx echo.Context, i interface{}) error {
	if err := ctx.Bind(i); err != nil {
		return err
	}

	if err := ctx.Validate(i); err != nil {
		return err
	}

	return nil
}

// ListConnectors godoc
//
//	@Summary		List connectors
//	@Description	Returns list of all connectors
//	@Security		BearerToken
//	@Tags			onboard
//	@Produce		json
//	@Success		200	{object}	[]api.ConnectorCount
//	@Router			/onboard/api/v1/connector [get]
func (h HttpHandler) ListConnectors(ctx echo.Context) error {
	connectors, err := h.db.ListConnectors()
	if err != nil {
		return err
	}

	var res []api.ConnectorCount
	for _, c := range connectors {
		count, err := h.db.CountSourcesOfType(c.Name)
		if err != nil {
			return err
		}
		tags := make(map[string]any)
		err = json.Unmarshal(c.Tags, &tags)
		if err != nil {
			return err
		}
		res = append(res, api.ConnectorCount{
			Connector: api.Connector{
				Name:                c.Name,
				Label:               c.Label,
				ShortDescription:    c.ShortDescription,
				Description:         c.Description,
				Direction:           c.Direction,
				Status:              c.Status,
				Logo:                c.Logo,
				AutoOnboardSupport:  c.AutoOnboardSupport,
				AllowNewConnections: c.AllowNewConnections,
				MaxConnectionLimit:  c.MaxConnectionLimit,
				Tags:                tags,
			},
			ConnectionCount: count,
		})

	}
	return ctx.JSON(http.StatusOK, res)
}

// PostSourceAws godoc
//
//	@Summary		Create AWS source
//	@Description	Creating AWS source
//	@Security		BearerToken
//	@Tags			onboard
//	@Produce		json
//	@Success		200		{object}	api.CreateSourceResponse
//	@Param			request	body		api.SourceAwsRequest	true	"Request"
//	@Router			/onboard/api/v1/source/aws [post]
func (h HttpHandler) PostSourceAws(ctx echo.Context) error {
	var req api.SourceAwsRequest
	if err := bindValidate(ctx, &req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}

	sdkCnf, err := kaytuAws.GetConfig(context.Background(), req.Config.AccessKey, req.Config.SecretKey, "", "", nil)
	if err != nil {
		return err
	}
	isAttached, err := kaytuAws.CheckAttachedPolicy(h.logger, sdkCnf, "", "")
	if err != nil {
		fmt.Printf("error in checking security audit permission: %v", err)
		return echo.NewHTTPError(http.StatusUnauthorized, PermissionError.Error())
	}
	if !isAttached {
		return echo.NewHTTPError(http.StatusUnauthorized, "Failed to find read access policy")
	}

	// Create source section
	cfg, err := kaytuAws.GetConfig(context.Background(), req.Config.AccessKey, req.Config.SecretKey, "", "", nil)
	if err != nil {
		return err
	}

	if cfg.Region == "" {
		cfg.Region = "us-east-1"
	}

	acc, err := currentAwsAccount(context.Background(), h.logger, cfg)
	if err != nil {
		return err
	}
	if req.Name != "" {
		acc.AccountName = &req.Name
	}

	count, err := h.db.CountSources()
	if err != nil {
		return err
	}
	if count >= httpserver.GetMaxConnections(ctx) {
		return echo.NewHTTPError(http.StatusBadRequest, "maximum number of connections reached")
	}

	src := NewAWSSource(h.logger, describe.AWSAccountConfig{AccessKey: req.Config.AccessKey, SecretKey: req.Config.SecretKey}, *acc, req.Description)
	secretBytes, err := h.kms.Encrypt(req.Config.AsMap(), h.keyARN)
	if err != nil {
		return err
	}
	src.Credential.Secret = string(secretBytes)

	err = h.db.CreateSource(&src)
	if err != nil {
		return err
	}

	return ctx.JSON(http.StatusOK, src.ToSourceResponse())
}

// PostSourceAzure godoc
//
//	@Summary		Create Azure source
//	@Description	Creating Azure source
//	@Security		BearerToken
//	@Tags			onboard
//	@Produce		json
//	@Success		200		{object}	api.CreateSourceResponse
//	@Param			request	body		api.SourceAzureRequest	true	"Request"
//	@Router			/onboard/api/v1/source/azure [post]
func (h HttpHandler) PostSourceAzure(ctx echo.Context) error {
	var req api.SourceAzureRequest

	if err := bindValidate(ctx, &req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}

	count, err := h.db.CountSources()
	if err != nil {
		return err
	}
	if count >= httpserver.GetMaxConnections(ctx) {
		return echo.NewHTTPError(http.StatusBadRequest, "maximum number of connections reached")
	}

	isAttached, err := kaytuAzure.CheckRole(kaytuAzure.AuthConfig{
		TenantID:     req.Config.TenantId,
		ObjectID:     req.Config.ObjectId,
		SecretID:     req.Config.SecretId,
		ClientID:     req.Config.ClientId,
		ClientSecret: req.Config.ClientSecret,
	}, req.Config.SubscriptionId, kaytuAzure.DefaultReaderRoleDefinitionIDTemplate)
	if err != nil {
		fmt.Printf("error in checking reader role roleAssignment: %v", err)
		return echo.NewHTTPError(http.StatusUnauthorized, PermissionError.Error())
	}
	if !isAttached {
		return echo.NewHTTPError(http.StatusUnauthorized, "Failed to find reader role roleAssignment")
	}

	cred, err := createAzureCredential(
		ctx.Request().Context(),
		fmt.Sprintf("%s - %s - default credentials", source.CloudAzure, req.Config.SubscriptionId),
		CredentialTypeAutoAzure,
		req.Config,
	)
	if err != nil {
		return err
	}

	azSub, err := currentAzureSubscription(ctx.Request().Context(), h.logger, req.Config.SubscriptionId, kaytuAzure.AuthConfig{
		TenantID:     req.Config.TenantId,
		ObjectID:     req.Config.ObjectId,
		SecretID:     req.Config.SecretId,
		ClientID:     req.Config.ClientId,
		ClientSecret: req.Config.ClientSecret,
	})
	if err != nil {
		return err
	}

	src := NewAzureConnectionWithCredentials(*azSub, source.SourceCreationMethodManual, req.Description, *cred)
	secretBytes, err := h.kms.Encrypt(req.Config.AsMap(), h.keyARN)
	if err != nil {
		return err
	}
	src.Credential.Secret = string(secretBytes)

	err = h.db.CreateSource(&src)
	if err != nil {
		return err
	}

	return ctx.JSON(http.StatusOK, src.ToSourceResponse())
}

func (h HttpHandler) checkCredentialHealth(cred Credential) (bool, error) {
	config, err := h.kms.Decrypt(cred.Secret, h.keyARN)
	if err != nil {
		return false, echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	switch cred.ConnectorType {
	case source.CloudAWS:
		var awsConfig describe.AWSAccountConfig
		awsConfig, err = describe.AWSAccountConfigFromMap(config)
		if err != nil {
			return false, echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		var sdkCnf aws.Config
		sdkCnf, err = kaytuAws.GetConfig(context.Background(), awsConfig.AccessKey, awsConfig.SecretKey, "", "", nil)
		if err != nil {
			return false, echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
		err = kaytuAws.CheckGetUserPermission(h.logger, sdkCnf)
		if err == nil {
			metadata, err := getAWSCredentialsMetadata(context.Background(), h.logger, awsConfig)
			if err != nil {
				return false, echo.NewHTTPError(http.StatusBadRequest, err.Error())
			}
			jsonMetadata, err := json.Marshal(metadata)
			if err != nil {
				return false, echo.NewHTTPError(http.StatusInternalServerError, err.Error())
			}
			cred.Metadata = jsonMetadata
		}
	case source.CloudAzure:
		var azureConfig describe.AzureSubscriptionConfig
		azureConfig, err = describe.AzureSubscriptionConfigFromMap(config)
		if err != nil {
			return false, echo.NewHTTPError(http.StatusInternalServerError, err.Error())
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
		if err == nil {
			metadata, err := getAzureCredentialsMetadata(context.Background(), azureConfig)
			if err != nil {
				return false, echo.NewHTTPError(http.StatusBadRequest, err.Error())
			}
			jsonMetadata, err := json.Marshal(metadata)
			if err != nil {
				return false, echo.NewHTTPError(http.StatusInternalServerError, err.Error())
			}
			cred.Metadata = jsonMetadata
		}
	}

	if err != nil {
		errStr := err.Error()
		cred.HealthReason = &errStr
		cred.HealthStatus = source.HealthStatusUnhealthy
	} else {
		cred.HealthStatus = source.HealthStatusHealthy
		cred.HealthReason = utils.GetPointer("")
	}
	cred.LastHealthCheckTime = time.Now()

	_, dbErr := h.db.UpdateCredential(&cred)
	if dbErr != nil {
		return false, echo.NewHTTPError(http.StatusInternalServerError, dbErr.Error())
	}

	if err != nil {
		return false, echo.NewHTTPError(http.StatusBadRequest, "credential is not healthy")
	}

	return true, nil
}

func createAzureCredential(ctx context.Context, name string, credType CredentialType, config api.AzureCredentialConfig) (*Credential, error) {
	azureCnf, err := describe.AzureSubscriptionConfigFromMap(config.AsMap())
	if err != nil {
		return nil, err
	}

	metadata, err := getAzureCredentialsMetadata(ctx, azureCnf)
	if err != nil {
		return nil, err
	}
	if credType == CredentialTypeManualAzureSpn {
		name = metadata.SpnName
	}
	cred, err := NewAzureCredential(name, credType, metadata)
	if err != nil {
		return nil, err
	}
	return cred, nil
}

func (h HttpHandler) postAzureCredentials(ctx echo.Context, req api.CreateCredentialRequest) error {
	configStr, err := json.Marshal(req.Config)
	if err != nil {
		return err
	}
	config := api.AzureCredentialConfig{}
	err = json.Unmarshal(configStr, &config)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, "invalid config")
	}

	cred, err := createAzureCredential(ctx.Request().Context(), "", CredentialTypeManualAzureSpn, config)
	if err != nil {
		return err
	}
	secretBytes, err := h.kms.Encrypt(config.AsMap(), h.keyARN)
	if err != nil {
		return err
	}
	cred.Secret = string(secretBytes)

	err = h.db.orm.Transaction(func(tx *gorm.DB) error {
		if err := h.db.CreateCredential(cred); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}

	_, err = h.checkCredentialHealth(*cred)
	if err != nil {
		return err
	}

	return ctx.JSON(http.StatusOK, api.CreateCredentialResponse{ID: cred.ID.String()})
}

func (h HttpHandler) postAWSCredentials(ctx echo.Context, req api.CreateCredentialRequest) error {
	configStr, err := json.Marshal(req.Config)
	if err != nil {
		return err
	}
	config := api.AWSCredentialConfig{}
	err = json.Unmarshal(configStr, &config)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, "invalid config")
	}

	awsCnf, err := describe.AWSAccountConfigFromMap(config.AsMap())
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, "invalid config")
	}

	metadata, err := getAWSCredentialsMetadata(ctx.Request().Context(), h.logger, awsCnf)
	if err != nil {
		return err
	}

	name := metadata.AccountID
	config.AccountId = metadata.AccountID
	if metadata.OrganizationID != nil {
		name = *metadata.OrganizationID
	}

	cred, err := NewAWSCredential(name, metadata, CredentialTypeManualAwsOrganization)
	if err != nil {
		return err
	}
	secretBytes, err := h.kms.Encrypt(config.AsMap(), h.keyARN)
	if err != nil {
		return err
	}
	cred.Secret = string(secretBytes)

	err = h.db.orm.Transaction(func(tx *gorm.DB) error {
		if err := h.db.CreateCredential(cred); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}

	_, err = h.checkCredentialHealth(*cred)
	if err != nil {
		return err
	}

	return ctx.JSON(http.StatusOK, api.CreateCredentialResponse{ID: cred.ID.String()})
}

// PostCredentials godoc
//
//	@Summary		Create connection credentials
//	@Description	Creating connection credentials
//	@Security		BearerToken
//	@Tags			onboard
//	@Produce		json
//	@Success		200		{object}	api.CreateCredentialResponse
//	@Param			config	body		api.CreateCredentialRequest	true	"config"
//	@Router			/onboard/api/v1/credential [post]
func (h HttpHandler) PostCredentials(ctx echo.Context) error {
	var req api.CreateCredentialRequest

	if err := bindValidate(ctx, &req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}

	switch req.SourceType {
	case source.CloudAzure:
		return h.postAzureCredentials(ctx, req)
	case source.CloudAWS:
		return h.postAWSCredentials(ctx, req)
	}

	return echo.NewHTTPError(http.StatusBadRequest, "invalid source type")
}

// ListCredentials godoc
//
//	@Summary		List credentials
//	@Description	Retrieving list of credentials with their details
//	@Security		BearerToken
//	@Tags			onboard
//	@Produce		json
//	@Success		200				{object}	api.ListCredentialResponse
//	@Param			connector		query		source.Type				false	"filter by connector type"
//	@Param			health			query		string					false	"filter by health status"	Enums(healthy, unhealthy)
//	@Param			credentialType	query		[]api.CredentialType	false	"filter by credential type"
//	@Param			pageSize		query		int						false	"page size"		default(50)
//	@Param			pageNumber		query		int						false	"page number"	default(1)
//	@Router			/onboard/api/v1/credential [get]
func (h HttpHandler) ListCredentials(ctx echo.Context) error {
	connector, _ := source.ParseType(ctx.QueryParam("connector"))
	health, _ := source.ParseHealthStatus(ctx.QueryParam("health"))
	credentialTypes := ParseCredentialTypes(ctx.QueryParams()["credentialType"])
	if len(credentialTypes) == 0 {
		// Take note if you want the change this, the default is used in the frontend AND the checkup worker
		credentialTypes = GetManualCredentialTypes()
	}
	pageSizeStr := ctx.QueryParam("pageSize")
	pageNumberStr := ctx.QueryParam("pageNumber")
	pageSize := int64(50)
	pageNumber := int64(1)
	if pageSizeStr != "" {
		pageSize, _ = strconv.ParseInt(pageSizeStr, 10, 64)
	}
	if pageNumberStr != "" {
		pageNumber, _ = strconv.ParseInt(pageNumberStr, 10, 64)
	}

	credentials, err := h.db.GetCredentialsByFilters(connector, health, credentialTypes)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	apiCredentials := make([]api.Credential, 0, len(credentials))
	for _, cred := range credentials {
		totalConnectionCount, err := h.db.CountConnectionsByCredential(cred.ID.String(), nil, nil)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		unhealthyConnectionCount, err := h.db.CountConnectionsByCredential(cred.ID.String(), nil, []source.HealthStatus{source.HealthStatusUnhealthy})
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}

		onboardConnectionCount, err := h.db.CountConnectionsByCredential(cred.ID.String(),
			[]ConnectionLifecycleState{ConnectionLifecycleStateInProgress, ConnectionLifecycleStateOnboard}, nil)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		discoveredConnectionCount, err := h.db.CountConnectionsByCredential(cred.ID.String(), []ConnectionLifecycleState{ConnectionLifecycleStateDiscovered}, nil)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		disabledConnectionCount, err := h.db.CountConnectionsByCredential(cred.ID.String(), []ConnectionLifecycleState{ConnectionLifecycleStateDisabled}, nil)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		archivedConnectionCount, err := h.db.CountConnectionsByCredential(cred.ID.String(), []ConnectionLifecycleState{ConnectionLifecycleStateArchived}, nil)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}

		apiCredential := cred.ToAPI()
		apiCredential.TotalConnections = &totalConnectionCount
		apiCredential.UnhealthyConnections = &unhealthyConnectionCount

		apiCredential.DiscoveredConnections = &discoveredConnectionCount
		apiCredential.OnboardConnections = &onboardConnectionCount
		apiCredential.DisabledConnections = &disabledConnectionCount
		apiCredential.ArchivedConnections = &archivedConnectionCount

		apiCredentials = append(apiCredentials, apiCredential)
	}

	sort.Slice(apiCredentials, func(i, j int) bool {
		return apiCredentials[i].OnboardDate.After(apiCredentials[j].OnboardDate)
	})

	result := api.ListCredentialResponse{
		TotalCredentialCount: len(apiCredentials),
		Credentials:          utils.Paginate(pageNumber, pageSize, apiCredentials),
	}

	return ctx.JSON(http.StatusOK, result)
}

// GetCredential godoc
//
//	@Summary		Get Credential
//	@Description	Retrieving credential details by credential ID
//	@Security		BearerToken
//	@Tags			onboard
//	@Produce		json
//	@Success		200				{object}	api.Credential
//	@Param			credentialId	path		string	true	"Credential ID"
//	@Router			/onboard/api/v1/credential/{credentialId} [get]
func (h HttpHandler) GetCredential(ctx echo.Context) error {
	credId, err := uuid.Parse(ctx.Param(paramCredentialId))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}

	credential, err := h.db.GetCredentialByID(credId)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return echo.NewHTTPError(http.StatusNotFound, "credential not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	connections, err := h.db.GetSourcesByCredentialID(credId.String())
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	metadata := make(map[string]any)
	err = json.Unmarshal(credential.Metadata, &metadata)
	if err != nil {
		return err
	}

	apiCredential := credential.ToAPI()
	if err != nil {
		return err
	}
	for _, conn := range connections {
		apiCredential.Connections = append(apiCredential.Connections, conn.toAPI())
		switch conn.LifecycleState {
		case ConnectionLifecycleStateDiscovered:
			apiCredential.DiscoveredConnections = utils.PAdd(apiCredential.DiscoveredConnections, utils.GetPointer(1))
		case ConnectionLifecycleStateInProgress:
			fallthrough
		case ConnectionLifecycleStateOnboard:
			apiCredential.OnboardConnections = utils.PAdd(apiCredential.OnboardConnections, utils.GetPointer(1))
		case ConnectionLifecycleStateDisabled:
			apiCredential.DisabledConnections = utils.PAdd(apiCredential.DisabledConnections, utils.GetPointer(1))
		case ConnectionLifecycleStateArchived:
			apiCredential.ArchivedConnections = utils.PAdd(apiCredential.ArchivedConnections, utils.GetPointer(1))
		}
		if conn.HealthState == source.HealthStatusUnhealthy {
			apiCredential.UnhealthyConnections = utils.PAdd(apiCredential.UnhealthyConnections, utils.GetPointer(1))
		}

		apiCredential.TotalConnections = utils.PAdd(apiCredential.TotalConnections, utils.GetPointer(1))
	}

	switch credential.ConnectorType {
	case source.CloudAzure:
		cnf, err := h.kms.Decrypt(credential.Secret, h.keyARN)
		if err != nil {
			return err
		}
		azureCnf, err := describe.AzureSubscriptionConfigFromMap(cnf)
		if err != nil {
			return err
		}
		apiCredential.Config = api.AzureCredentialConfig{
			SubscriptionId: azureCnf.SubscriptionID,
			TenantId:       azureCnf.TenantID,
			ObjectId:       azureCnf.ObjectID,
			SecretId:       azureCnf.SecretID,
			ClientId:       azureCnf.ClientID,
		}
	case source.CloudAWS:
		cnf, err := h.kms.Decrypt(credential.Secret, h.keyARN)
		if err != nil {
			return err
		}
		awsCnf, err := describe.AWSAccountConfigFromMap(cnf)
		if err != nil {
			return err
		}
		apiCredential.Config = api.AWSCredentialConfig{
			AccountId:            awsCnf.AccountID,
			Regions:              awsCnf.Regions,
			AccessKey:            awsCnf.AccessKey,
			AssumeRoleName:       awsCnf.AssumeRoleName,
			AssumeRolePolicyName: awsCnf.AssumeRolePolicyName,
			ExternalId:           awsCnf.ExternalID,
		}
	}

	return ctx.JSON(http.StatusOK, apiCredential)
}

func (h HttpHandler) autoOnboardAzureSubscriptions(ctx context.Context, credential Credential, maxConnections int64) ([]api.Connection, error) {
	onboardedSources := make([]api.Connection, 0)
	cnf, err := h.kms.Decrypt(credential.Secret, h.keyARN)
	if err != nil {
		return nil, err
	}
	azureCnf, err := describe.AzureSubscriptionConfigFromMap(cnf)
	if err != nil {
		return nil, err
	}
	h.logger.Info("discovering subscriptions", zap.String("credentialId", credential.ID.String()))
	subs, err := discoverAzureSubscriptions(ctx, h.logger, kaytuAzure.AuthConfig{
		TenantID:     azureCnf.TenantID,
		ObjectID:     azureCnf.ObjectID,
		SecretID:     azureCnf.SecretID,
		ClientID:     azureCnf.ClientID,
		ClientSecret: azureCnf.ClientSecret,
	})
	if err != nil {
		h.logger.Error("failed to discover subscriptions", zap.Error(err))
		return nil, err
	}
	h.logger.Info("discovered subscriptions", zap.Int("count", len(subs)))

	existingConnections, err := h.db.GetSourcesOfType(credential.ConnectorType)
	if err != nil {
		return nil, err
	}

	existingConnectionSubIDs := make([]string, 0, len(existingConnections))
	subsToOnboard := make([]azureSubscription, 0)
	for _, conn := range existingConnections {
		existingConnectionSubIDs = append(existingConnectionSubIDs, conn.SourceId)
	}
	for _, sub := range subs {
		if sub.SubModel.State != nil && *sub.SubModel.State == armsubscription.SubscriptionStateEnabled && !utils.Includes(existingConnectionSubIDs, sub.SubscriptionID) {
			subsToOnboard = append(subsToOnboard, sub)
		} else {
			for _, conn := range existingConnections {
				if conn.SourceId == sub.SubscriptionID {
					name := sub.SubscriptionID
					if sub.SubModel.DisplayName != nil {
						name = *sub.SubModel.DisplayName
					}
					localConn := conn
					if conn.Name != name {
						localConn.Name = name
					}
					if sub.SubModel.State != nil && *sub.SubModel.State != armsubscription.SubscriptionStateEnabled {
						localConn.LifecycleState = ConnectionLifecycleStateDisabled
					}
					if conn.Name != name || localConn.LifecycleState != conn.LifecycleState {
						_, err := h.db.UpdateSource(&localConn)
						if err != nil {
							h.logger.Error("failed to update source", zap.Error(err))
							return nil, err
						}
					}
				}
			}
		}
	}

	h.logger.Info("onboarding subscriptions", zap.Int("count", len(subsToOnboard)))

	for _, sub := range subsToOnboard {
		h.logger.Info("onboarding subscription", zap.String("subscriptionId", sub.SubscriptionID))
		count, err := h.db.CountSources()
		if err != nil {
			return nil, err
		}
		if count >= maxConnections {
			return nil, echo.NewHTTPError(http.StatusBadRequest, "maximum number of connections reached")
		}

		isAttached, err := kaytuAzure.CheckRole(kaytuAzure.AuthConfig{
			TenantID:     azureCnf.TenantID,
			ObjectID:     azureCnf.ObjectID,
			SecretID:     azureCnf.SecretID,
			ClientID:     azureCnf.ClientID,
			ClientSecret: azureCnf.ClientSecret,
		}, sub.SubscriptionID, kaytuAzure.DefaultReaderRoleDefinitionIDTemplate)
		if err != nil {
			h.logger.Warn("failed to check role", zap.Error(err))
			continue
		}
		if !isAttached {
			h.logger.Warn("role not attached", zap.String("subscriptionId", sub.SubscriptionID))
			continue
		}

		src := NewAzureConnectionWithCredentials(
			sub,
			source.SourceCreationMethodAutoOnboard,
			fmt.Sprintf("Auto onboarded subscription %s", sub.SubscriptionID),
			credential,
		)

		err = h.db.CreateSource(&src)
		if err != nil {
			return nil, err
		}

		metadata := make(map[string]any)
		if src.Metadata.String() != "" {
			err := json.Unmarshal(src.Metadata, &metadata)
			if err != nil {
				return nil, err
			}
		}

		onboardedSources = append(onboardedSources, api.Connection{
			ID:                   src.ID,
			ConnectionID:         src.SourceId,
			ConnectionName:       src.Name,
			Email:                src.Email,
			Connector:            src.Type,
			Description:          src.Description,
			CredentialID:         src.CredentialID.String(),
			CredentialName:       src.Credential.Name,
			OnboardDate:          src.CreatedAt,
			LifecycleState:       api.ConnectionLifecycleState(src.LifecycleState),
			AssetDiscoveryMethod: src.AssetDiscoveryMethod,
			LastHealthCheckTime:  src.LastHealthCheckTime,
			HealthReason:         src.HealthReason,
			Metadata:             metadata,
		})
	}

	return onboardedSources, nil
}

func (h HttpHandler) autoOnboardAWSAccounts(ctx context.Context, credential Credential, maxConnections int64) ([]api.Connection, error) {
	onboardedSources := make([]api.Connection, 0)
	cnf, err := h.kms.Decrypt(credential.Secret, h.keyARN)
	if err != nil {
		return nil, err
	}
	awsCnf, err := describe.AWSAccountConfigFromMap(cnf)
	if err != nil {
		return nil, err
	}
	cfg, err := kaytuAws.GetConfig(
		ctx,
		awsCnf.AccessKey,
		awsCnf.SecretKey,
		"",
		"",
		nil)
	h.logger.Info("discovering accounts", zap.String("credentialId", credential.ID.String()))
	if cfg.Region == "" {
		cfg.Region = "us-east-1"
	}
	accounts, err := discoverAWSAccounts(ctx, cfg)
	if err != nil {
		h.logger.Error("failed to discover accounts", zap.Error(err))
		return nil, err
	}
	h.logger.Info("discovered accounts", zap.Int("count", len(accounts)))
	existingConnections, err := h.db.GetSourcesOfType(credential.ConnectorType)
	if err != nil {
		return nil, err
	}
	existingConnectionAccountIDs := make([]string, 0, len(existingConnections))
	for _, conn := range existingConnections {
		existingConnectionAccountIDs = append(existingConnectionAccountIDs, conn.SourceId)
	}
	accountsToOnboard := make([]awsAccount, 0)
	for _, account := range accounts {
		if account.Account.Status == awsOrgTypes.AccountStatusActive && !utils.Includes(existingConnectionAccountIDs, account.AccountID) {
			accountsToOnboard = append(accountsToOnboard, account)
		} else {
			for _, conn := range existingConnections {
				if conn.SourceId == account.AccountID {
					name := account.AccountID
					if account.AccountName != nil {
						name = *account.AccountName
					}

					localConn := conn
					if conn.Name != name {
						localConn.Name = name
					}
					if account.Account.Status != awsOrgTypes.AccountStatusActive {
						localConn.LifecycleState = ConnectionLifecycleStateArchived
					}
					if conn.Name != name || account.Account.Status != awsOrgTypes.AccountStatusActive {
						_, err := h.db.UpdateSource(&localConn)
						if err != nil {
							h.logger.Error("failed to update source", zap.Error(err))
							return nil, err
						}
					}
				}
			}
		}
	}

	// TODO add tag filter

	h.logger.Info("onboarding accounts", zap.Int("count", len(accountsToOnboard)))
	for _, account := range accountsToOnboard {
		//assumeRoleArn := kaytuAws.GetRoleArnFromName(account.AccountID, awsCnf.AssumeRoleName)
		//sdkCnf, err := kaytuAws.GetConfig(ctx.Request().Context(), awsCnf.AccessKey, awsCnf.SecretKey, assumeRoleArn, assumeRoleArn, awsCnf.ExternalID)
		//if err != nil {
		//	h.logger.Warn("failed to get config", zap.Error(err))
		//	return err
		//}
		//isAttached, err := kaytuAws.CheckAttachedPolicy(h.logger, sdkCnf, awsCnf.AssumeRoleName, kaytuAws.SecurityAuditPolicyARN)
		//if err != nil {
		//	h.logger.Warn("failed to check get user permission", zap.Error(err))
		//	continue
		//}
		//if !isAttached {
		//	h.logger.Warn("security audit policy not attached", zap.String("accountID", account.AccountID))
		//	continue
		//}
		h.logger.Info("onboarding account", zap.String("accountID", account.AccountID))
		count, err := h.db.CountSources()
		if err != nil {
			return nil, err
		}
		if count >= maxConnections {
			return nil, echo.NewHTTPError(http.StatusBadRequest, "maximum number of connections reached")
		}

		src := NewAWSAutoOnboardedConnection(
			h.logger,
			awsCnf,
			account,
			source.SourceCreationMethodAutoOnboard,
			fmt.Sprintf("Auto onboarded account %s", account.AccountID),
			credential,
		)

		err = h.db.orm.Transaction(func(tx *gorm.DB) error {
			err := h.db.CreateSource(&src)
			if err != nil {
				return err
			}

			//TODO: add enable account

			return nil
		})
		if err != nil {
			return nil, err
		}

		metadata := make(map[string]any)
		if src.Metadata.String() != "" {
			err := json.Unmarshal(src.Metadata, &metadata)
			if err != nil {
				return nil, err
			}
		}

		onboardedSources = append(onboardedSources, api.Connection{
			ID:                   src.ID,
			ConnectionID:         src.SourceId,
			ConnectionName:       src.Name,
			Email:                src.Email,
			Connector:            src.Type,
			Description:          src.Description,
			CredentialID:         src.CredentialID.String(),
			CredentialName:       src.Credential.Name,
			OnboardDate:          src.CreatedAt,
			LifecycleState:       api.ConnectionLifecycleState(src.LifecycleState),
			AssetDiscoveryMethod: src.AssetDiscoveryMethod,
			LastHealthCheckTime:  src.LastHealthCheckTime,
			HealthReason:         src.HealthReason,
			Metadata:             metadata,
		})
	}

	return onboardedSources, nil
}

// AutoOnboardCredential godoc
//
//	@Summary		Onboard credential connections
//	@Description	Onboard all available connections for a credential
//	@Security		BearerToken
//	@Tags			onboard
//	@Produce		json
//	@Param			credentialId	path		string	true	"CredentialID"
//	@Success		200				{object}	[]api.Connection
//	@Router			/onboard/api/v1/credential/{credentialId}/autoonboard [post]
func (h HttpHandler) AutoOnboardCredential(ctx echo.Context) error {
	credId, err := uuid.Parse(ctx.Param(paramCredentialId))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}

	credential, err := h.db.GetCredentialByID(credId)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return echo.NewHTTPError(http.StatusNotFound, "credential not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	maxConns := httpserver.GetMaxConnections(ctx)

	onboardedSources := make([]api.Connection, 0)
	switch credential.ConnectorType {
	case source.CloudAzure:
		onboardedSources, err = h.autoOnboardAzureSubscriptions(ctx.Request().Context(), *credential, maxConns)
		if err != nil {
			return err
		}
	case source.CloudAWS:
		onboardedSources, err = h.autoOnboardAWSAccounts(ctx.Request().Context(), *credential, maxConns)
		if err != nil {
			return err
		}
	default:
		return echo.NewHTTPError(http.StatusBadRequest, "connector doesn't support auto onboard")
	}

	return ctx.JSON(http.StatusOK, onboardedSources)
}

func (h HttpHandler) putAzureCredentials(ctx echo.Context, req api.UpdateCredentialRequest) error {
	id, err := uuid.Parse(ctx.Param("credentialId"))
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, "invalid id")
	}

	cred, err := h.db.GetCredentialByID(id)
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			return err
		}
		return ctx.JSON(http.StatusNotFound, "credential not found")
	}

	if req.Name != nil {
		cred.Name = req.Name
	}

	cnf, err := h.kms.Decrypt(cred.Secret, h.keyARN)
	if err != nil {
		return err
	}
	config, err := describe.AzureSubscriptionConfigFromMap(cnf)
	if err != nil {
		return err
	}

	if req.Config != nil {
		configStr, err := json.Marshal(req.Config)
		if err != nil {
			return err
		}
		newConfig := api.AzureCredentialConfig{}
		err = json.Unmarshal(configStr, &newConfig)
		if err != nil {
			return ctx.JSON(http.StatusBadRequest, "invalid config")
		}
		if newConfig.SubscriptionId != "" {
			config.SubscriptionID = newConfig.SubscriptionId
		}
		if newConfig.TenantId != "" {
			config.TenantID = newConfig.TenantId
		}
		if newConfig.ObjectId != "" {
			config.ObjectID = newConfig.ObjectId
		}
		if newConfig.SecretId != "" {
			config.SecretID = newConfig.SecretId
		}
		if newConfig.ClientId != "" {
			config.ClientID = newConfig.ClientId
		}
		if newConfig.ClientSecret != "" {
			config.ClientSecret = newConfig.ClientSecret
		}
	}
	metadata, err := getAzureCredentialsMetadata(ctx.Request().Context(), config)
	if err != nil {
		return err
	}
	jsonMetadata, err := json.Marshal(metadata)
	if err != nil {
		return err
	}
	cred.Metadata = jsonMetadata
	secretBytes, err := h.kms.Encrypt(config.ToMap(), h.keyARN)
	if err != nil {
		return err
	}
	cred.Secret = string(secretBytes)
	if metadata.SpnName != "" {
		cred.Name = &metadata.SpnName
	}

	err = h.db.orm.Transaction(func(tx *gorm.DB) error {
		if _, err := h.db.UpdateCredential(cred); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}

	_, err = h.checkCredentialHealth(*cred)
	if err != nil {
		return err
	}

	return ctx.JSON(http.StatusOK, struct{}{})
}

func (h HttpHandler) putAWSCredentials(ctx echo.Context, req api.UpdateCredentialRequest) error {
	id, err := uuid.Parse(ctx.Param("credentialId"))
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, "invalid id")
	}

	cred, err := h.db.GetCredentialByID(id)
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			return err
		}
		return ctx.JSON(http.StatusNotFound, "credential not found")
	}

	if req.Name != nil {
		cred.Name = req.Name
	}

	cnf, err := h.kms.Decrypt(cred.Secret, h.keyARN)
	if err != nil {
		return err
	}
	config, err := describe.AWSAccountConfigFromMap(cnf)
	if err != nil {
		return err
	}

	if req.Config != nil {
		configStr, err := json.Marshal(req.Config)
		if err != nil {
			return err
		}
		newConfig := api.AWSCredentialConfig{}
		err = json.Unmarshal(configStr, &newConfig)
		if err != nil {
			return ctx.JSON(http.StatusBadRequest, "invalid config")
		}

		if newConfig.AccountId != "" {
			config.AccountID = newConfig.AccountId
		}
		if newConfig.Regions != nil {
			config.Regions = newConfig.Regions
		}
		if newConfig.AccessKey != "" {
			config.AccessKey = newConfig.AccessKey
		}
		if newConfig.SecretKey != "" {
			config.SecretKey = newConfig.SecretKey
		}
		if newConfig.AssumeRoleName != "" {
			config.AssumeRoleName = newConfig.AssumeRoleName
		}
		if newConfig.AssumeRolePolicyName != "" {
			config.AssumeRolePolicyName = newConfig.AssumeRolePolicyName
		}
		if newConfig.ExternalId != nil {
			config.ExternalID = newConfig.ExternalId
		}
	}

	metadata, err := getAWSCredentialsMetadata(ctx.Request().Context(), h.logger, config)
	if err != nil {
		return err
	}
	jsonMetadata, err := json.Marshal(metadata)
	if err != nil {
		return err
	}
	cred.Metadata = jsonMetadata
	secretBytes, err := h.kms.Encrypt(config.ToMap(), h.keyARN)
	if err != nil {
		return err
	}
	cred.Secret = string(secretBytes)
	if metadata.OrganizationID != nil && metadata.OrganizationMasterAccountId != nil &&
		metadata.AccountID == *metadata.OrganizationMasterAccountId &&
		config.AssumeRoleName != "" && config.ExternalID != nil {
		cred.Name = metadata.OrganizationID
		cred.CredentialType = CredentialTypeManualAwsOrganization
	}

	err = h.db.orm.Transaction(func(tx *gorm.DB) error {
		if _, err := h.db.UpdateCredential(cred); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return err
	}

	_, err = h.checkCredentialHealth(*cred)
	if err != nil {
		return err
	}

	return ctx.JSON(http.StatusOK, struct{}{})
}

// PutCredentials godoc
//
//	@Summary		Edit credential
//	@Description	Edit a credential by ID
//	@Security		BearerToken
//	@Tags			onboard
//	@Produce		json
//	@Success		200
//	@Param			credentialId	path	string						true	"Credential ID"
//	@Param			config			body	api.UpdateCredentialRequest	true	"config"
//	@Router			/onboard/api/v1/credential/{credentialId} [put]
func (h HttpHandler) PutCredentials(ctx echo.Context) error {
	var req api.UpdateCredentialRequest
	if err := bindValidate(ctx, &req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}

	switch req.Connector {
	case source.CloudAzure:
		return h.putAzureCredentials(ctx, req)
	case source.CloudAWS:
		return h.putAWSCredentials(ctx, req)
	}

	return ctx.JSON(http.StatusBadRequest, "invalid source type")
}

// DeleteCredential godoc
//
//	@Summary		Delete credential
//	@Description	Remove a credential by ID
//	@Security		BearerToken
//	@Tags			onboard
//	@Produce		json
//	@Success		200
//	@Param			credentialId	path	string	true	"CredentialID"
//	@Router			/onboard/api/v1/credential/{credentialId} [delete]
func (h HttpHandler) DeleteCredential(ctx echo.Context) error {
	credId, err := uuid.Parse(ctx.Param(paramCredentialId))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}

	credential, err := h.db.GetCredentialByID(credId)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return echo.NewHTTPError(http.StatusNotFound, "credential not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	sources, err := h.db.GetSourcesByCredentialID(credential.ID.String())
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	err = h.db.orm.Transaction(func(tx *gorm.DB) error {
		if err := h.db.DeleteCredential(credential.ID); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}

		for _, src := range sources {
			if err := h.db.UpdateSourceLifecycleState(src.ID, ConnectionLifecycleStateDisabled); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	return ctx.JSON(http.StatusOK, struct{}{})
}

func (h HttpHandler) GetSourceFullCred(ctx echo.Context) error {
	sourceUUID, err := uuid.Parse(ctx.Param("sourceId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid source uuid")
	}

	src, err := h.db.GetSource(sourceUUID)
	if err != nil {
		return err
	}

	cnf, err := h.kms.Decrypt(src.Credential.Secret, h.keyARN)
	if err != nil {
		return err
	}

	switch src.Type {
	case source.CloudAWS:
		awsCnf, err := describe.AWSAccountConfigFromMap(cnf)
		if err != nil {
			return err
		}
		return ctx.JSON(http.StatusOK, api.AWSCredential{
			AccessKey: awsCnf.AccessKey,
			SecretKey: awsCnf.SecretKey,
		})
	case source.CloudAzure:
		azureCnf, err := describe.AzureSubscriptionConfigFromMap(cnf)
		if err != nil {
			return err
		}
		return ctx.JSON(http.StatusOK, api.AzureCredential{
			ClientID:     azureCnf.ClientID,
			TenantID:     azureCnf.TenantID,
			ClientSecret: azureCnf.ClientSecret,
		})
	default:
		return errors.New("invalid provider")
	}
}

func (h HttpHandler) updateConnectionHealth(connection Source, healthStatus source.HealthStatus, reason *string) (Source, error) {
	connection.HealthState = healthStatus
	connection.HealthReason = reason
	connection.LastHealthCheckTime = time.Now()
	_, err := h.db.UpdateSource(&connection)
	if err != nil {
		return Source{}, err
	}
	//TODO Mahan: record state change in elastic search
	return connection, nil
}

func (h HttpHandler) checkConnectionHealth(ctx context.Context, connection Source, updateMetadata bool) (Source, error) {
	var cnf map[string]any
	cnf, err := h.kms.Decrypt(connection.Credential.Secret, h.keyARN)
	if err != nil {
		h.logger.Error("failed to decrypt credential", zap.Error(err), zap.String("sourceId", connection.SourceId))
		return connection, err
	}

	var isAttached bool
	switch connection.Type {
	case source.CloudAWS:
		var awsCnf describe.AWSAccountConfig
		awsCnf, err = describe.AWSAccountConfigFromMap(cnf)
		if err != nil {
			h.logger.Error("failed to get aws config", zap.Error(err), zap.String("sourceId", connection.SourceId))
			return connection, err
		}
		assumeRoleArn := kaytuAws.GetRoleArnFromName(connection.SourceId, awsCnf.AssumeRoleName)
		var sdkCnf aws.Config
		if awsCnf.AccountID != connection.SourceId {
			sdkCnf, err = kaytuAws.GetConfig(ctx, awsCnf.AccessKey, awsCnf.SecretKey, "", assumeRoleArn, awsCnf.ExternalID)
		} else {
			sdkCnf, err = kaytuAws.GetConfig(ctx, awsCnf.AccessKey, awsCnf.SecretKey, "", "", nil)
		}
		if err != nil {
			h.logger.Error("failed to get aws config", zap.Error(err), zap.String("sourceId", connection.SourceId))
			return connection, err
		}
		if awsCnf.AccountID != connection.SourceId {
			isAttached, err = kaytuAws.CheckAttachedPolicy(h.logger, sdkCnf, awsCnf.AssumeRoleName, kaytuAws.GetPolicyArnFromName(connection.SourceId, awsCnf.AssumeRolePolicyName))
		} else {
			isAttached, err = kaytuAws.CheckAttachedPolicy(h.logger, sdkCnf, "", kaytuAws.GetPolicyArnFromName(connection.SourceId, awsCnf.AssumeRolePolicyName))
		}
		if err == nil && isAttached && updateMetadata {
			if sdkCnf.Region == "" {
				sdkCnf.Region = "us-east-1"
			}
			var awsAccount *awsAccount
			awsAccount, err = currentAwsAccount(ctx, h.logger, sdkCnf)
			if err != nil {
				h.logger.Error("failed to get current aws account", zap.Error(err), zap.String("sourceId", connection.SourceId))
				return connection, err
			}
			metadata, err2 := NewAWSConnectionMetadata(h.logger, awsCnf, connection, *awsAccount)
			if err2 != nil {
				h.logger.Error("failed to get aws connection metadata", zap.Error(err2), zap.String("sourceId", connection.SourceId))
			}
			jsonMetadata, err2 := json.Marshal(metadata)
			if err2 != nil {
				return connection, err
			}
			connection.Metadata = jsonMetadata
		}
	case source.CloudAzure:
		var azureCnf describe.AzureSubscriptionConfig
		azureCnf, err = describe.AzureSubscriptionConfigFromMap(cnf)
		if err != nil {
			h.logger.Error("failed to get azure config", zap.Error(err), zap.String("sourceId", connection.SourceId))
			return connection, err
		}
		authCnf := kaytuAzure.AuthConfig{
			TenantID:            azureCnf.TenantID,
			ClientID:            azureCnf.ClientID,
			ObjectID:            azureCnf.ObjectID,
			SecretID:            azureCnf.SecretID,
			ClientSecret:        azureCnf.ClientSecret,
			CertificatePath:     azureCnf.CertificatePath,
			CertificatePassword: azureCnf.CertificatePass,
			Username:            azureCnf.Username,
			Password:            azureCnf.Password,
		}
		isAttached, err = kaytuAzure.CheckRole(authCnf, connection.SourceId, kaytuAzure.DefaultReaderRoleDefinitionIDTemplate)

		if err == nil && isAttached && updateMetadata {
			var azSub *azureSubscription
			azSub, err = currentAzureSubscription(ctx, h.logger, connection.SourceId, authCnf)
			if err != nil {
				h.logger.Error("failed to get current azure subscription", zap.Error(err), zap.String("sourceId", connection.SourceId))
				return connection, err
			}
			metadata := NewAzureConnectionMetadata(*azSub)
			var jsonMetadata []byte
			jsonMetadata, err = json.Marshal(metadata)
			if err != nil {
				h.logger.Error("failed to marshal azure metadata", zap.Error(err), zap.String("sourceId", connection.SourceId))
				return connection, err
			}
			connection.Metadata = jsonMetadata
		}
	}
	if err != nil {
		h.logger.Warn("failed to check read permission", zap.Error(err), zap.String("sourceId", connection.SourceId))
	}

	if !isAttached {
		var healthMessage string
		if err == nil {
			healthMessage = "Failed to find read permission"
		} else {
			healthMessage = err.Error()
		}
		connection, err = h.updateConnectionHealth(connection, source.HealthStatusUnhealthy, &healthMessage)
		if err != nil {
			h.logger.Warn("failed to update source health", zap.Error(err), zap.String("sourceId", connection.SourceId))
			return connection, err
		}
	} else {
		connection, err = h.updateConnectionHealth(connection, source.HealthStatusHealthy, utils.GetPointer(""))
		if err != nil {
			h.logger.Warn("failed to update source health", zap.Error(err), zap.String("sourceId", connection.SourceId))
			return connection, err
		}
	}

	return connection, nil
}

// GetConnectionHealth godoc
//
//	@Summary		Get source health
//	@Description	Get live source health status with given source ID.
//	@Security		BearerToken
//	@Tags			onboard
//	@Produce		json
//	@Param			sourceId		path		string	true	"Source ID"
//	@Param			updateMetadata	query		bool	false	"Whether to update metadata or not"	default(true)
//	@Success		200				{object}	api.Connection
//	@Router			/onboard/api/v1/source/{sourceId}/healthcheck [get]
func (h HttpHandler) GetConnectionHealth(ctx echo.Context) error {
	sourceUUID, err := uuid.Parse(ctx.Param("sourceId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid source uuid")
	}
	updateMetadata := true
	if strings.ToLower(ctx.QueryParam("updateMetadata")) == "false" {
		updateMetadata = false
	}

	connection, err := h.db.GetSource(sourceUUID)
	if err != nil {
		h.logger.Error("failed to get source", zap.Error(err), zap.String("sourceId", sourceUUID.String()))
		return err
	}

	if !connection.LifecycleState.IsEnabled() {
		connection, err = h.updateConnectionHealth(connection, source.HealthStatusNil, utils.GetPointer("Connection is not enabled"))
		if err != nil {
			h.logger.Error("failed to update source health", zap.Error(err), zap.String("sourceId", connection.SourceId))
			return err
		}
	} else {
		isHealthy, err := h.checkCredentialHealth(connection.Credential)
		if err != nil {
			if herr, ok := err.(*echo.HTTPError); ok {
				if herr.Code == http.StatusInternalServerError {
					h.logger.Error("failed to check credential health", zap.Error(err), zap.String("sourceId", connection.SourceId))
					return herr
				}
			} else {
				h.logger.Error("failed to check credential health", zap.Error(err), zap.String("sourceId", connection.SourceId))
				return err
			}
		}
		if !isHealthy {
			connection, err = h.updateConnectionHealth(connection, source.HealthStatusUnhealthy, utils.GetPointer("Credential is not healthy"))
			if err != nil {
				h.logger.Error("failed to update source health", zap.Error(err), zap.String("sourceId", connection.SourceId))
				return err
			}
		} else {
			connection, err = h.checkConnectionHealth(ctx.Request().Context(), connection, updateMetadata)
		}
	}
	return ctx.JSON(http.StatusOK, connection.toAPI())
}

func (h HttpHandler) GetSource(ctx echo.Context) error {
	srcId, err := uuid.Parse(ctx.Param(paramSourceId))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}

	src, err := h.db.GetSource(srcId)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return echo.NewHTTPError(http.StatusBadRequest, "source not found")
		}
		return err
	}

	metadata := make(map[string]any)
	if src.Metadata.String() != "" {
		err := json.Unmarshal(src.Metadata, &metadata)
		if err != nil {
			return err
		}
	}

	apiRes := src.toAPI()
	if httpserver.GetUserRole(ctx) == api3.KaytuAdminRole {
		apiRes.Credential = src.Credential.ToAPI()
		apiRes.Credential.Config = src.Credential.Secret
	}

	return ctx.JSON(http.StatusOK, apiRes)
}

// DeleteSource godoc
//
//	@Summary		Delete source
//	@Description	Deleting a single source either AWS / Azure for the given source id.
//	@Security		BearerToken
//	@Tags			onboard
//	@Produce		json
//	@Success		200
//	@Param			sourceId	path	string	true	"Source ID"
//	@Router			/onboard/api/v1/source/{sourceId} [delete]
func (h HttpHandler) DeleteSource(ctx echo.Context) error {
	srcId, err := uuid.Parse(ctx.Param(paramSourceId))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}

	src, err := h.db.GetSource(srcId)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return echo.NewHTTPError(http.StatusBadRequest, "source not found")
		}
		return err
	}

	err = h.db.orm.Transaction(func(tx *gorm.DB) error {
		if err := h.db.DeleteSource(srcId); err != nil {
			return err
		}

		if src.Credential.CredentialType.IsManual() {
			err = h.db.DeleteCredential(src.Credential.ID)
			if err != nil {
				return err
			}
		}

		if err := h.sourceEventsQueue.Publish(api.SourceEvent{
			Action:     api.SourceDeleted,
			SourceID:   src.ID,
			SourceType: src.Type,
			Secret:     src.Credential.Secret,
		}); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return err
	}

	return ctx.NoContent(http.StatusOK)
}

func (h HttpHandler) ChangeConnectionLifecycleState(ctx echo.Context) error {
	connectionId, err := uuid.Parse(ctx.Param("connectionId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid connection id")
	}

	var req api.ChangeConnectionLifecycleStateRequest
	if err := bindValidate(ctx, &req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	} else if err = req.State.Validate(); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	connection, err := h.db.GetSource(connectionId)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return echo.NewHTTPError(http.StatusBadRequest, "source not found")
		}
		return err
	}

	reqState := ConnectionLifecycleStateFromApi(req.State)
	if reqState == connection.LifecycleState {
		return echo.NewHTTPError(http.StatusBadRequest, "connection already in requested state")
	}

	if reqState.IsEnabled() != connection.LifecycleState.IsEnabled() {
		if err := h.db.UpdateSourceLifecycleState(connectionId, reqState); err != nil {
			return err
		}
	} else {
		err = h.db.UpdateSourceLifecycleState(connectionId, reqState)
	}
	if err != nil {
		return err
	}

	return ctx.NoContent(http.StatusOK)
}

func (h HttpHandler) ListSources(ctx echo.Context) error {
	var err error
	sType := httpserver.QueryArrayParam(ctx, "connector")
	var sources []Source
	if len(sType) > 0 {
		st := source.ParseTypes(sType)
		sources, err = h.db.GetSourcesOfTypes(st)
		if err != nil {
			return err
		}
	} else {
		sources, err = h.db.ListSources()
		if err != nil {
			return err
		}
	}

	resp := api.GetSourcesResponse{}
	for _, s := range sources {
		apiRes := s.toAPI()
		if httpserver.GetUserRole(ctx) == api3.KaytuAdminRole {
			apiRes.Credential = s.Credential.ToAPI()
			apiRes.Credential.Config = s.Credential.Secret
		}
		resp = append(resp, apiRes)
	}

	return ctx.JSON(http.StatusOK, resp)
}

func (h HttpHandler) GetSources(ctx echo.Context) error {
	var req api.GetSourcesRequest
	if err := bindValidate(ctx, &req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}

	srcs, err := h.db.GetSources(req.SourceIDs)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return echo.NewHTTPError(http.StatusBadRequest, "source not found")
		}
		return err
	}

	var res []api.Connection
	for _, src := range srcs {
		apiRes := src.toAPI()
		if httpserver.GetUserRole(ctx) == api3.KaytuAdminRole {
			apiRes.Credential = src.Credential.ToAPI()
			apiRes.Credential.Config = src.Credential.Secret
		}

		res = append(res, apiRes)
	}
	return ctx.JSON(http.StatusOK, res)
}

func (h HttpHandler) CountSources(ctx echo.Context) error {
	sType := ctx.QueryParam("connector")
	var count int64
	if sType != "" {
		st, err := source.ParseType(sType)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("invalid source type: %s", sType))
		}

		count, err = h.db.CountSourcesOfType(st)
		if err != nil {
			return err
		}
	} else {
		var err error
		count, err = h.db.CountSources()
		if err != nil {
			return err
		}
	}

	return ctx.JSON(http.StatusOK, count)
}

// CatalogMetrics godoc
//
//	@Summary		List catalog metrics
//	@Description	Retrieving the list of metrics for catalog page.
//	@Security		BearerToken
//	@Tags			onboard
//	@Produce		json
//	@Success		200	{object}	api.CatalogMetrics
//	@Router			/onboard/api/v1/catalog/metrics [get]
func (h HttpHandler) CatalogMetrics(ctx echo.Context) error {
	var metrics api.CatalogMetrics

	srcs, err := h.db.ListSources()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	for _, src := range srcs {
		metrics.TotalConnections++
		if src.LifecycleState.IsEnabled() {
			metrics.ConnectionsEnabled++
		}
		switch src.HealthState {
		case source.HealthStatusHealthy:
			metrics.HealthyConnections++
		case source.HealthStatusUnhealthy:
			metrics.UnhealthyConnections++
		}
	}

	return ctx.JSON(http.StatusOK, metrics)
}

// ListConnectionsSummaries godoc
//
//	@Summary		List connections summaries
//	@Description	Retrieving a list of connections summaries
//	@Security		BearerToken
//	@Tags			connections
//	@Accept			json
//	@Produce		json
//	@Param			connector			query		[]source.Type	false	"Connector"
//	@Param			connectionId		query		[]string		false	"Connection IDs"
//	@Param			lifecycleState		query		string			false	"lifecycle state filter"	Enums(DISABLED, DISCOVERED, IN_PROGRESS, ONBOARD, ARCHIVED)
//	@Param			healthState			query		string			false	"health state filter"		Enums(healthy,unhealthy)
//	@Param			pageSize			query		int				false	"page size - default is 20"
//	@Param			pageNumber			query		int				false	"page number - default is 1"
//	@Param			startTime			query		int				false	"start time in unix seconds"
//	@Param			endTime				query		int				false	"end time in unix seconds"
//	@Param			needCost			query		boolean			false	"for quicker inquiry send this parameter as false, default: true"
//	@Param			needResourceCount	query		boolean			false	"for quicker inquiry send this parameter as false, default: true"
//	@Param			sortBy				query		string			false	"column to sort by - default is cost"	Enums(onboard_date,resource_count,cost,growth,growth_rate,cost_growth,cost_growth_rate)
//	@Success		200					{object}	api.ListConnectionSummaryResponse
//	@Router			/onboard/api/v1/connections/summary [get]
func (h HttpHandler) ListConnectionsSummaries(ctx echo.Context) error {
	connectors := source.ParseTypes(httpserver.QueryArrayParam(ctx, "connector"))
	connectionIDs := httpserver.QueryArrayParam(ctx, "connectionId")
	endTimeStr := ctx.QueryParam("endTime")
	endTime := time.Now()
	if endTimeStr != "" {
		endTimeUnix, err := strconv.ParseInt(endTimeStr, 10, 64)
		if err != nil {
			return ctx.JSON(http.StatusBadRequest, "endTime is not a valid integer")
		}
		endTime = time.Unix(endTimeUnix, 0)
	}
	startTimeStr := ctx.QueryParam("startTime")
	startTime := endTime.AddDate(0, 0, -7)
	if startTimeStr != "" {
		startTimeUnix, err := strconv.ParseInt(startTimeStr, 10, 64)
		if err != nil {
			return ctx.JSON(http.StatusBadRequest, "startTime is not a valid integer")
		}
		startTime = time.Unix(startTimeUnix, 0)
	}
	pageNumber, pageSize, err := utils.PageConfigFromStrings(ctx.QueryParam("pageNumber"), ctx.QueryParam("pageSize"))
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, err.Error())
	}
	sortBy := strings.ToLower(ctx.QueryParam("sortBy"))
	if sortBy == "" {
		sortBy = "cost"
	}

	if sortBy != "cost" && sortBy != "growth" &&
		sortBy != "growth_rate" && sortBy != "cost_growth" &&
		sortBy != "cost_growth_rate" && sortBy != "onboard_date" &&
		sortBy != "resource_count" {
		return ctx.JSON(http.StatusBadRequest, "sortBy is not a valid value")
	}
	var lifecycleStateSlice []ConnectionLifecycleState
	lifecycleState := ctx.QueryParam("lifecycleState")
	if lifecycleState != "" {
		lifecycleStateSlice = append(lifecycleStateSlice, ConnectionLifecycleState(lifecycleState))
	}

	var healthStateSlice []source.HealthStatus
	healthState := ctx.QueryParam("healthState")
	if healthState != "" {
		healthStateSlice = append(healthStateSlice, source.HealthStatus(healthState))
	}

	connections, err := h.db.ListSourcesWithFilters(connectors, connectionIDs, lifecycleStateSlice, healthStateSlice)
	if err != nil {
		return err
	}

	needCostStr := ctx.QueryParam("needCost")
	needCost := true
	if needCostStr == "false" {
		needCost = false
	}
	needResourceCountStr := ctx.QueryParam("needResourceCount")
	needResourceCount := true
	if needResourceCountStr == "false" {
		needResourceCount = false
	}

	connectionData := map[string]api2.ConnectionData{}
	if needResourceCount || needCost {
		connectionData, err = h.inventoryClient.ListConnectionsData(httpclient.FromEchoContext(ctx), nil, &startTime, &endTime, needCost, needResourceCount)
		if err != nil {
			return err
		}
	}

	result := api.ListConnectionSummaryResponse{
		ConnectionCount:       len(connections),
		TotalCost:             0,
		TotalResourceCount:    0,
		TotalOldResourceCount: 0,
		TotalUnhealthyCount:   0,

		TotalDisabledCount:   0,
		TotalDiscoveredCount: 0,
		TotalOnboardedCount:  0,
		TotalArchivedCount:   0,
		Connections:          make([]api.Connection, 0, len(connections)),
	}

	for _, connection := range connections {
		if data, ok := connectionData[connection.ID.String()]; ok {
			localData := data
			apiConn := connection.toAPI()
			apiConn.Cost = localData.TotalCost
			apiConn.DailyCostAtStartTime = localData.DailyCostAtStartTime
			apiConn.DailyCostAtEndTime = localData.DailyCostAtEndTime
			apiConn.ResourceCount = localData.Count
			apiConn.OldResourceCount = localData.OldCount
			apiConn.LastInventory = localData.LastInventory
			if localData.TotalCost != nil {
				result.TotalCost += *localData.TotalCost
			}
			if localData.Count != nil {
				result.TotalResourceCount += *localData.Count
			}
			result.Connections = append(result.Connections, apiConn)
		} else {
			result.Connections = append(result.Connections, connection.toAPI())
		}
		switch connection.LifecycleState {
		case ConnectionLifecycleStateDiscovered:
			result.TotalDiscoveredCount++
		case ConnectionLifecycleStateDisabled:
			result.TotalDisabledCount++
		case ConnectionLifecycleStateInProgress:
			fallthrough
		case ConnectionLifecycleStateOnboard:
			result.TotalOnboardedCount++
		case ConnectionLifecycleStateArchived:
			result.TotalArchivedCount++
		}
		if connection.HealthState == source.HealthStatusUnhealthy {
			result.TotalUnhealthyCount++
		}
	}

	sort.Slice(result.Connections, func(i, j int) bool {
		switch sortBy {
		case "onboard_date":
			return result.Connections[i].OnboardDate.Before(result.Connections[j].OnboardDate)
		case "resource_count":
			if result.Connections[i].ResourceCount == nil && result.Connections[j].ResourceCount == nil {
				break
			}
			if result.Connections[i].ResourceCount == nil {
				return false
			}
			if result.Connections[j].ResourceCount == nil {
				return true
			}
			if *result.Connections[i].ResourceCount != *result.Connections[j].ResourceCount {
				return *result.Connections[i].ResourceCount > *result.Connections[j].ResourceCount
			}
		case "cost":
			if result.Connections[i].Cost == nil && result.Connections[j].Cost == nil {
				break
			}
			if result.Connections[i].Cost == nil {
				return false
			}
			if result.Connections[j].Cost == nil {
				return true
			}
			if *result.Connections[i].Cost != *result.Connections[j].Cost {
				return *result.Connections[i].Cost > *result.Connections[j].Cost
			}
		case "growth":
			diffi := utils.PSub(result.Connections[i].ResourceCount, result.Connections[i].OldResourceCount)
			diffj := utils.PSub(result.Connections[j].ResourceCount, result.Connections[j].OldResourceCount)
			if diffi == nil && diffj == nil {
				break
			}
			if diffi == nil {
				return false
			}
			if diffj == nil {
				return true
			}
			if *diffi != *diffj {
				return *diffi > *diffj
			}
		case "growth_rate":
			diffi := utils.PSub(result.Connections[i].ResourceCount, result.Connections[i].OldResourceCount)
			diffj := utils.PSub(result.Connections[j].ResourceCount, result.Connections[j].OldResourceCount)
			if diffi == nil && diffj == nil {
				break
			}
			if diffi == nil {
				return false
			}
			if diffj == nil {
				return true
			}
			if result.Connections[i].OldResourceCount == nil && result.Connections[j].OldResourceCount == nil {
				break
			}
			if result.Connections[i].OldResourceCount == nil {
				return true
			}
			if result.Connections[j].OldResourceCount == nil {
				return false
			}
			if *result.Connections[i].OldResourceCount == 0 && *result.Connections[j].OldResourceCount == 0 {
				break
			}
			if *result.Connections[i].OldResourceCount == 0 {
				return false
			}
			if *result.Connections[j].OldResourceCount == 0 {
				return true
			}
			if float64(*diffi)/float64(*result.Connections[i].OldResourceCount) != float64(*diffj)/float64(*result.Connections[j].OldResourceCount) {
				return float64(*diffi)/float64(*result.Connections[i].OldResourceCount) > float64(*diffj)/float64(*result.Connections[j].OldResourceCount)
			}
		case "cost_growth":
			diffi := utils.PSub(result.Connections[i].DailyCostAtEndTime, result.Connections[i].DailyCostAtStartTime)
			diffj := utils.PSub(result.Connections[j].DailyCostAtEndTime, result.Connections[j].DailyCostAtStartTime)
			if diffi == nil && diffj == nil {
				break
			}
			if diffi == nil {
				return false
			}
			if diffj == nil {
				return true
			}
			if *diffi != *diffj {
				return *diffi > *diffj
			}
		case "cost_growth_rate":
			diffi := utils.PSub(result.Connections[i].DailyCostAtEndTime, result.Connections[i].DailyCostAtStartTime)
			diffj := utils.PSub(result.Connections[j].DailyCostAtEndTime, result.Connections[j].DailyCostAtStartTime)
			if diffi == nil && diffj == nil {
				break
			}
			if diffi == nil {
				return false
			}
			if diffj == nil {
				return true
			}
			if result.Connections[i].DailyCostAtStartTime == nil && result.Connections[j].DailyCostAtStartTime == nil {
				break
			}
			if result.Connections[i].DailyCostAtStartTime == nil {
				return true
			}
			if result.Connections[j].DailyCostAtStartTime == nil {
				return false
			}
			if *result.Connections[i].DailyCostAtStartTime == 0 && *result.Connections[j].DailyCostAtStartTime == 0 {
				break
			}
			if *result.Connections[i].DailyCostAtStartTime == 0 {
				return false
			}
			if *result.Connections[j].DailyCostAtStartTime == 0 {
				return true
			}
			if *diffi/(*result.Connections[i].DailyCostAtStartTime) != *diffj/(*result.Connections[j].DailyCostAtStartTime) {
				return *diffi/(*result.Connections[i].DailyCostAtStartTime) > *diffj/(*result.Connections[j].DailyCostAtStartTime)
			}
		}
		return result.Connections[i].ConnectionName < result.Connections[j].ConnectionName
	})

	result.Connections = utils.Paginate(pageNumber, pageSize, result.Connections)
	return ctx.JSON(http.StatusOK, result)
}

// ListConnectionGroups godoc
//
//	@Summary		List connection groups
//	@Description	Retrieving a list of connection groups
//	@Security		BearerToken
//	@Tags			connection-groups
//	@Accept			json
//	@Produce		json
//	@Param			populateConnections	query		bool	false	"Populate connections"	default(false)
//	@Success		200					{object}	[]api.ConnectionGroup
//	@Router			/onboard/api/v1/connection-groups [get]
func (h HttpHandler) ListConnectionGroups(ctx echo.Context) error {
	var err error
	populateConnections := false
	if populateConnectionsStr := ctx.QueryParam("populateConnections"); populateConnectionsStr != "" {
		populateConnections, err = strconv.ParseBool(populateConnectionsStr)
		if err != nil {
			return ctx.JSON(http.StatusBadRequest, "populateConnections is not a valid boolean")
		}
	}

	connectionGroups, err := h.db.ListConnectionGroups()
	if err != nil {
		h.logger.Error("error listing connection groups", zap.Error(err))
		return err
	}

	result := make([]api.ConnectionGroup, 0, len(connectionGroups))
	for _, connectionGroup := range connectionGroups {
		apiCg, err := connectionGroup.ToAPI(ctx.Request().Context(), h.steampipeConn)
		if err != nil {
			h.logger.Error("error populating connection group", zap.Error(err))
			continue
		}
		if populateConnections {
			connections, err := h.db.GetSources(apiCg.ConnectionIds)
			if err != nil {
				h.logger.Error("error getting connections", zap.Error(err))
				return err
			}
			apiCg.Connections = make([]api.Connection, 0, len(connections))
			for _, connection := range connections {
				apiCg.Connections = append(apiCg.Connections, connection.toAPI())
			}
		}

		result = append(result, *apiCg)
	}

	return ctx.JSON(http.StatusOK, result)
}

// GetConnectionGroup godoc
//
//	@Summary		Get connection group
//	@Description	Retrieving a connection group
//	@Security		BearerToken
//	@Tags			connection-groups
//	@Accept			json
//	@Produce		json
//	@Param			populateConnections	query		bool	false	"Populate connections"	default(false)
//	@Param			connectionGroupName	path		string	true	"ConnectionGroupName"
//	@Success		200					{object}	api.ConnectionGroup
//	@Router			/onboard/api/v1/connection-groups/{connectionGroupName} [get]
func (h HttpHandler) GetConnectionGroup(ctx echo.Context) error {
	connectionGroupName := ctx.Param("connectionGroupName")
	var err error
	populateConnections := false
	if populateConnectionsStr := ctx.QueryParam("populateConnections"); populateConnectionsStr != "" {
		populateConnections, err = strconv.ParseBool(populateConnectionsStr)
		if err != nil {
			return ctx.JSON(http.StatusBadRequest, "populateConnections is not a valid boolean")
		}
	}

	connectionGroup, err := h.db.GetConnectionGroupByName(connectionGroupName)
	if err != nil {
		h.logger.Error("error getting connection group", zap.Error(err))
		return err
	}

	apiCg, err := connectionGroup.ToAPI(ctx.Request().Context(), h.steampipeConn)
	if err != nil {
		h.logger.Error("error populating connection group", zap.Error(err))
		return err
	}

	if populateConnections {
		connections, err := h.db.GetSources(apiCg.ConnectionIds)
		if err != nil {
			h.logger.Error("error getting connections", zap.Error(err))
			return err
		}
		apiCg.Connections = make([]api.Connection, 0, len(connections))
		for _, connection := range connections {
			apiCg.Connections = append(apiCg.Connections, connection.toAPI())
		}
	}

	return ctx.JSON(http.StatusOK, apiCg)
}
