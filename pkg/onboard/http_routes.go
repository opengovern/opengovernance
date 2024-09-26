package onboard

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/subscription/armsubscription"
	"github.com/aws/aws-sdk-go-v2/aws"
	api3 "github.com/kaytu-io/kaytu-util/pkg/api"
	"github.com/kaytu-io/kaytu-util/pkg/httpclient"
	"github.com/kaytu-io/kaytu-util/pkg/httpserver"
	"github.com/kaytu-io/open-governance/pkg/demo"
	"github.com/kaytu-io/open-governance/pkg/describe/connectors"
	"github.com/kaytu-io/open-governance/pkg/metadata/models"
	"github.com/kaytu-io/open-governance/pkg/onboard/api/entities"
	apiv2 "github.com/kaytu-io/open-governance/pkg/onboard/api/v2"
	"github.com/kaytu-io/open-governance/services/integration/api/entity"
	"github.com/kaytu-io/open-governance/services/integration/model"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	api2 "github.com/kaytu-io/open-governance/pkg/inventory/api"

	"go.uber.org/zap"

	"github.com/kaytu-io/kaytu-util/pkg/source"
	"github.com/kaytu-io/open-governance/pkg/utils"
	"github.com/labstack/echo/v4"

	awsOrgTypes "github.com/aws/aws-sdk-go-v2/service/organizations/types"
	"github.com/google/uuid"
	kaytuAws "github.com/kaytu-io/kaytu-aws-describer/aws"
	kaytuAzure "github.com/kaytu-io/kaytu-azure-describer/azure"
	"github.com/kaytu-io/open-governance/pkg/onboard/api"
	"gorm.io/gorm"
)

const (
	paramSourceId     = "sourceId"
	paramCredentialId = "credentialId"
)

var tracer = otel.Tracer("onboard")

func (h HttpHandler) Register(r *echo.Echo) {
	v1 := r.Group("/api/v1")
	v2 := r.Group("/api/v2")

	v1.GET("/sources", httpserver.AuthorizeHandler(h.ListSources, api3.ViewerRole))
	v1.POST("/sources", httpserver.AuthorizeHandler(h.GetSources, api3.ViewerRole))
	v1.GET("/sources/count", httpserver.AuthorizeHandler(h.CountSources, api3.ViewerRole))
	v1.GET("/catalog/metrics", httpserver.AuthorizeHandler(h.CatalogMetrics, api3.ViewerRole))

	connector := v1.Group("/connector")
	connector.GET("", httpserver.AuthorizeHandler(h.ListConnectors, api3.ViewerRole))

	sourceApiGroup := v1.Group("/source")
	sourceApiGroup.POST("/aws", httpserver.AuthorizeHandler(h.PostSourceAws, api3.EditorRole))
	sourceApiGroup.POST("/azure", httpserver.AuthorizeHandler(h.PostSourceAzure, api3.EditorRole))
	sourceApiGroup.GET("/:sourceId", httpserver.AuthorizeHandler(h.GetSource, api3.AdminRole))
	sourceApiGroup.GET("/:sourceId/healthcheck", httpserver.AuthorizeHandler(h.GetConnectionHealth, api3.EditorRole))
	sourceApiGroup.GET("/:sourceId/credentials/full", httpserver.AuthorizeHandler(h.GetSourceFullCred, api3.AdminRole))
	sourceApiGroup.DELETE("/:sourceId", httpserver.AuthorizeHandler(h.DeleteSource, api3.EditorRole))

	credential := v1.Group("/credential")
	credential.POST("", httpserver.AuthorizeHandler(h.PostCredentials, api3.EditorRole))
	credential.PUT("/:credentialId", httpserver.AuthorizeHandler(h.PutCredentials, api3.EditorRole))
	credential.GET("", httpserver.AuthorizeHandler(h.ListCredentials, api3.ViewerRole))
	credential.DELETE("/:credentialId", httpserver.AuthorizeHandler(h.DeleteCredential, api3.EditorRole))
	credential.GET("/:credentialId", httpserver.AuthorizeHandler(h.GetCredential, api3.ViewerRole))
	credential.POST("/:credentialId/autoonboard", httpserver.AuthorizeHandler(h.AutoOnboardCredential, api3.EditorRole))

	credentialV2 := v2.Group("/credential")
	credentialV2.POST("", httpserver.AuthorizeHandler(h.CreateCredential, api3.EditorRole))

	connections := v1.Group("/connections")
	connections.GET("/summary", httpserver.AuthorizeHandler(h.ListConnectionsSummaries, api3.ViewerRole))
	connections.POST("/:connectionId/state", httpserver.AuthorizeHandler(h.ChangeConnectionLifecycleState, api3.EditorRole))
	connections.POST("/aws", httpserver.AuthorizeHandler(h.PostConnectionAws, api3.EditorRole))

	connectionGroups := v1.Group("/connection-groups")
	connectionGroups.GET("", httpserver.AuthorizeHandler(h.ListConnectionGroups, api3.ViewerRole))
	connectionGroups.GET("/:connectionGroupName", httpserver.AuthorizeHandler(h.GetConnectionGroup, api3.ViewerRole))

	v2.GET("/sources/:sourceId", httpserver.AuthorizeHandler(h.GetSourceBySourceId, api3.ViewerRole))
	v2.POST("/sources", httpserver.AuthorizeHandler(h.ListSourcesByFilters, api3.ViewerRole))
	v2.POST("/source", httpserver.AuthorizeHandler(h.GetSourceByFilters, api3.ViewerRole))

	v3 := r.Group("/api/v3")
	v3.GET("/connector", httpserver.AuthorizeHandler(h.ListConnectorsV2, api3.ViewerRole))
	v3.PUT("/sample/purge", httpserver.AuthorizeHandler(h.PurgeSampleData, api3.InternalRole))
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
	//trace :
	outputS, span1 := tracer.Start(ctx.Request().Context(), "new_ListConnectors", trace.WithSpanKind(trace.SpanKindServer))
	span1.SetName("new_ListConnectors")

	connectors, err := h.db.ListConnectors()
	if err != nil {
		span1.RecordError(err)
		span1.SetStatus(codes.Error, err.Error())
		return err
	}
	span1.End()

	var res []api.ConnectorCount

	//trace :
	outputS2, span2 := tracer.Start(outputS, "new_CountSourcesOfType(loop)")
	span2.SetName("new_CountSourcesOfType(loop)")

	for _, c := range connectors {
		_, span3 := tracer.Start(outputS2, "new_CountSourcesOfType", trace.WithSpanKind(trace.SpanKindServer))
		span3.SetName("new_CountSourcesOfType")

		count, err := h.db.CountSourcesOfType(c.Name)
		if err != nil {
			span3.RecordError(err)
			span3.SetStatus(codes.Error, err.Error())
			return err
		}
		span3.AddEvent("information", trace.WithAttributes(
			attribute.String("source name", string(c.Name)),
		))
		span3.End()

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
	span2.End()
	return ctx.JSON(http.StatusOK, res)
}

func (h HttpHandler) CheckMaxConnections(additionCount int64) error {
	count, err := h.db.CountSources()
	if err != nil {
		return err
	}
	cnf, err := h.metadataClient.GetConfigMetadata(&httpclient.Context{UserRole: api3.InternalRole}, models.MetadataKeyConnectionLimit)
	if err != nil {
		return err
	}

	var maxConnections int64
	if v, ok := cnf.GetValue().(int64); ok {
		maxConnections = v
	} else if v, ok := cnf.GetValue().(int); ok {
		maxConnections = int64(v)
	}
	if count+additionCount > maxConnections {
		return echo.NewHTTPError(http.StatusBadRequest, "maximum number of connections reached")
	}
	return nil
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

	sdkCnf, err := kaytuAws.GetConfig(ctx.Request().Context(), req.Config.AccessKey, req.Config.SecretKey, "", "", nil)
	if err != nil {
		return err
	}
	err = kaytuAws.CheckGetUserPermission(h.logger, sdkCnf)
	if err != nil {
		fmt.Printf("error in checking security audit permission: %v", err)
		return echo.NewHTTPError(http.StatusUnauthorized, PermissionError.Error())
	}

	// Create source section
	cfg, err := kaytuAws.GetConfig(ctx.Request().Context(), req.Config.AccessKey, req.Config.SecretKey, "", "", nil)
	if err != nil {
		return err
	}

	if cfg.Region == "" {
		cfg.Region = "us-east-1"
	}

	acc, err := currentAwsAccount(ctx.Request().Context(), h.logger, cfg)
	if err != nil {
		return err
	}
	if req.Name != "" {
		acc.AccountName = &req.Name
	}

	err = h.CheckMaxConnections(1)
	if err != nil {
		return err
	}

	src := NewAWSSource(ctx.Request().Context(), h.logger, connectors.AWSAccountConfig{AccessKey: req.Config.AccessKey, SecretKey: req.Config.SecretKey}, *acc, req.Description)

	secretBytes, err := h.vaultSc.Encrypt(ctx.Request().Context(), req.Config.AsMap())
	if err != nil {
		return err
	}
	src.Credential.Secret = string(secretBytes)

	err = h.db.CreateSource(&src)
	if err != nil {
		return err
	}

	return ctx.JSON(http.StatusOK, api.CreateSourceResponse{
		ID: src.ID,
	})
}

// PostConnectionAws godoc
//
//	@Summary		Create AWS connection
//	@Description	Creating AWS connection
//	@Security		BearerToken
//	@Tags			onboard
//	@Produce		json
//	@Success		200		{object}	api.CreateConnectionResponse
//	@Param			request	body		api.CreateAwsConnectionRequest	true	"Request"
//	@Router			/onboard/api/v1/connections/aws [post]
func (h HttpHandler) PostConnectionAws(ctx echo.Context) error {
	var req api.CreateAwsConnectionRequest
	if err := bindValidate(ctx, &req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}

	err := h.CheckMaxConnections(1)
	if err != nil {
		return err
	}

	sdkCnf, err := h.GetAWSSDKConfig(ctx.Request().Context(), generateRoleARN(req.AWSConfig.AccountID, req.AWSConfig.AssumeRoleName), req.AWSConfig.AccessKey, req.AWSConfig.SecretKey, req.AWSConfig.ExternalId)
	if err != nil {
		return err
	}

	acc, err := currentAwsAccount(ctx.Request().Context(), h.logger, sdkCnf)
	if err != nil {
		return err
	}
	if req.Name != "" {
		acc.AccountName = &req.Name
	}

	aKey := h.masterAccessKey
	sKey := h.masterSecretKey
	if req.AWSConfig.AccessKey != nil {
		aKey = *req.AWSConfig.AccessKey
	}
	if req.AWSConfig.SecretKey != nil {
		sKey = *req.AWSConfig.SecretKey
	}

	src := NewAWSSource(ctx.Request().Context(), h.logger, connectors.AWSAccountConfig{
		AccessKey: aKey,
		SecretKey: sKey,
	}, *acc, "")

	secretBytes, err := h.vaultSc.Encrypt(ctx.Request().Context(), req.AWSConfig.AsMap())
	if err != nil {
		return err
	}
	src.Credential.Version = 2
	src.Credential.Secret = string(secretBytes)

	err = h.db.CreateSource(&src)
	if err != nil {
		return err
	}

	src, err = h.checkConnectionHealth(ctx.Request().Context(), src, true)
	if err != nil {
		return err
	}

	return ctx.JSON(http.StatusOK, api.CreateConnectionResponse{
		ID: src.ID,
	})
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

	err := h.CheckMaxConnections(1)
	if err != nil {
		return err
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
		model.CredentialTypeAutoAzure,
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

	src := NewAzureConnectionWithCredentials(*azSub, source.SourceCreationMethodManual, req.Description, *cred, req.Config.TenantId)

	secretBytes, err := h.vaultSc.Encrypt(ctx.Request().Context(), req.Config.AsMap())
	if err != nil {
		return err
	}
	src.Credential.Secret = string(secretBytes)
	// trace :
	//_, span2 := tracer.Start(outputS, "new_CreateSource", trace.WithSpanKind(trace.SpanKindServer))
	//span2.SetName("new_CreateSource")

	err = h.db.CreateSource(&src)
	if err != nil {
		//span2.RecordError(err)
		//span2.SetStatus(codes.Error, err.Error())
		return err
	}
	//span2.AddEvent("information", trace.WithAttributes(
	//	attribute.String("source name ", src.Name),
	//))
	//span2.End()

	return ctx.JSON(http.StatusOK, api.CreateSourceResponse{
		ID: src.ID,
	})
}

func createAzureCredential(ctx context.Context, name string, credType model.CredentialType, config api.AzureCredentialConfig) (*model.Credential, error) {
	azureCnf, err := connectors.AzureSubscriptionConfigFromMap(config.AsMap())
	if err != nil {
		return nil, err
	}

	metadata, err := getAzureCredentialsMetadata(ctx, azureCnf, credType)
	if err != nil {
		return nil, fmt.Errorf("failed to get credential metadata: %v", err)
	}
	switch credType {
	case model.CredentialTypeManualAzureSpn:
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

	credType := model.CredentialTypeManualAzureSpn

	cred, err := createAzureCredential(ctx.Request().Context(), "", credType, config)
	if err != nil {
		return err
	}

	secretBytes, err := h.vaultSc.Encrypt(ctx.Request().Context(), config.AsMap())
	if err != nil {
		return err
	}
	cred.Secret = string(secretBytes)

	// trace :
	outputS, span := tracer.Start(ctx.Request().Context(), "new_Transaction", trace.WithSpanKind(trace.SpanKindServer))
	span.SetName("new_Transaction And checkCredentialHealth")

	err = h.db.CreateCredential(cred)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	span.End()

	_, err = h.checkCredentialHealth(outputS, *cred)
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

	awsCnf, err := connectors.AWSAccountConfigFromMap(config.AsMap())
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

	cred, err := NewAWSCredential(name, metadata, model.CredentialTypeManualAwsOrganization, 1)
	if err != nil {
		return err
	}

	secretBytes, err := h.vaultSc.Encrypt(ctx.Request().Context(), config.AsMap())
	if err != nil {
		return err
	}
	cred.Secret = string(secretBytes)
	// trace :
	outputS, span := tracer.Start(ctx.Request().Context(), "new_Transaction ", trace.WithSpanKind(trace.SpanKindServer))
	span.SetName("new_Transaction")

	err = h.db.Orm.Transaction(func(tx *gorm.DB) error {
		// trace :
		_, span2 := tracer.Start(outputS, "new_CreateCredential", trace.WithSpanKind(trace.SpanKindServer))
		span2.SetName("new_CreateCredential")

		if err := h.db.CreateCredential(cred); err != nil {
			span2.RecordError(err)
			span2.SetStatus(codes.Error, err.Error())
			return err
		}
		span2.AddEvent("information", trace.WithAttributes(
			attribute.String("credential id", cred.ID.String()),
		))
		span2.End()

		return nil
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	span.End()

	_, err = h.checkCredentialHealth(outputS, *cred)
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

// CreateCredential godoc
//
//	@Summary		Create connection credentials
//	@Description	Creating connection credentials
//	@Security		BearerToken
//	@Tags			onboard
//	@Produce		json
//	@Success		200		{object}	apiv2.CreateCredentialV2Response
//	@Param			config	body		apiv2.CreateCredentialV2Request	true	"config"
//	@Router			/onboard/api/v2/credential [post]
func (h HttpHandler) CreateCredential(ctx echo.Context) error {
	var req apiv2.CreateCredentialV2Request

	if err := bindValidate(ctx, &req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}

	switch req.Connector {
	case source.CloudAzure:
		return echo.NewHTTPError(http.StatusNotImplemented)
	case source.CloudAWS:
		resp, err := h.createAWSCredential(ctx.Request().Context(), req)
		if err != nil {
			return err
		}
		return ctx.JSON(http.StatusOK, *resp)
	}

	return echo.NewHTTPError(http.StatusBadRequest, "invalid connector")
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
	credentialTypes := model.ParseCredentialTypes(ctx.QueryParams()["credentialType"])
	if len(credentialTypes) == 0 {
		// Take note if you want the change this, the default is used in the frontend AND the checkup worker
		credentialTypes = model.GetManualCredentialTypes()
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
	// trace :
	_, span := tracer.Start(ctx.Request().Context(), "new_GetCredentialsByFilters", trace.WithSpanKind(trace.SpanKindServer))
	span.SetName("new_GetCredentialsByFilters")

	credentials, err := h.db.GetCredentialsByFilters(connector, health, credentialTypes)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	span.End()

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
			[]model.ConnectionLifecycleState{model.ConnectionLifecycleStateInProgress, model.ConnectionLifecycleStateOnboard}, nil)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		discoveredConnectionCount, err := h.db.CountConnectionsByCredential(cred.ID.String(), []model.ConnectionLifecycleState{model.ConnectionLifecycleStateDiscovered}, nil)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		disabledConnectionCount, err := h.db.CountConnectionsByCredential(cred.ID.String(), []model.ConnectionLifecycleState{model.ConnectionLifecycleStateDisabled}, nil)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		archivedConnectionCount, err := h.db.CountConnectionsByCredential(cred.ID.String(), []model.ConnectionLifecycleState{model.ConnectionLifecycleStateArchived}, nil)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}

		apiCredential := entities.NewCredential(cred)
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
	// trace :
	outputS, span1 := tracer.Start(ctx.Request().Context(), "new_GetCredentialByID", trace.WithSpanKind(trace.SpanKindServer))
	span1.SetName("new_GetCredentialByID")

	credential, err := h.db.GetCredentialByID(credId)
	if err != nil {
		span1.RecordError(err)
		span1.SetStatus(codes.Error, err.Error())

		if err == gorm.ErrRecordNotFound {
			return echo.NewHTTPError(http.StatusNotFound, "credential not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	span1.AddEvent("information", trace.WithAttributes(
		attribute.String("credential id", credential.ID.String()),
	))
	span1.End()

	// trace :
	_, span2 := tracer.Start(outputS, "new_GetSourcesByCredentialID", trace.WithSpanKind(trace.SpanKindServer))
	span2.SetName("new_GetSourcesByCredentialID")

	connections, err := h.db.GetSourcesByCredentialID(credId.String())
	if err != nil {
		span2.RecordError(err)
		span2.SetStatus(codes.Error, err.Error())
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	span1.AddEvent("information", trace.WithAttributes(
		attribute.String("credential id", credId.String()),
	))
	span2.End()

	metadata := make(map[string]any)
	err = json.Unmarshal(credential.Metadata, &metadata)
	if err != nil {
		return err
	}

	apiCredential := entities.NewCredential(*credential)
	if err != nil {
		return err
	}
	for _, conn := range connections {
		apiCredential.Connections = append(apiCredential.Connections, entities.NewConnection(conn))
		switch conn.LifecycleState {
		case model.ConnectionLifecycleStateDiscovered:
			apiCredential.DiscoveredConnections = utils.PAdd(apiCredential.DiscoveredConnections, utils.GetPointer(1))
		case model.ConnectionLifecycleStateInProgress:
			fallthrough
		case model.ConnectionLifecycleStateOnboard:
			apiCredential.OnboardConnections = utils.PAdd(apiCredential.OnboardConnections, utils.GetPointer(1))
		case model.ConnectionLifecycleStateDisabled:
			apiCredential.DisabledConnections = utils.PAdd(apiCredential.DisabledConnections, utils.GetPointer(1))
		case model.ConnectionLifecycleStateArchived:
			apiCredential.ArchivedConnections = utils.PAdd(apiCredential.ArchivedConnections, utils.GetPointer(1))
		}
		if conn.HealthState == source.HealthStatusUnhealthy {
			apiCredential.UnhealthyConnections = utils.PAdd(apiCredential.UnhealthyConnections, utils.GetPointer(1))
		}

		apiCredential.TotalConnections = utils.PAdd(apiCredential.TotalConnections, utils.GetPointer(1))
	}

	switch credential.ConnectorType {
	case source.CloudAzure:
		cnf, err := h.vaultSc.Decrypt(ctx.Request().Context(), credential.Secret)
		if err != nil {
			return err
		}
		azureCnf, err := connectors.AzureSubscriptionConfigFromMap(cnf)
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
		cnf, err := h.vaultSc.Decrypt(ctx.Request().Context(), credential.Secret)
		if err != nil {
			return err
		}
		if credential.Version == 2 {
			awsCnf, err := apiv2.AWSCredentialV2ConfigFromMap(cnf)
			if err != nil {
				return err
			}
			aKey := h.masterAccessKey
			if awsCnf.AccessKey != nil {
				aKey = *awsCnf.AccessKey
			}
			apiCredential.Config = api.AWSCredentialConfig{
				AccountId:      awsCnf.AccountID,
				AccessKey:      aKey,
				AssumeRoleName: awsCnf.AssumeRoleName,
				ExternalId:     awsCnf.ExternalId,
			}
		} else {
			awsCnf, err := connectors.AWSAccountConfigFromMap(cnf)
			if err != nil {
				return err
			}
			apiCredential.Config = api.AWSCredentialConfig{
				AccountId:      awsCnf.AccountID,
				Regions:        awsCnf.Regions,
				AccessKey:      awsCnf.AccessKey,
				AssumeRoleName: awsCnf.AssumeRoleName,
				ExternalId:     awsCnf.ExternalID,
			}
		}
	}

	for idx, c := range apiCredential.Connections {
		c.ConnectionName = demo.EncodeResponseData(ctx, c.ConnectionName)
		c.ConnectionID = demo.EncodeResponseData(ctx, c.ConnectionID)
		apiCredential.Connections[idx] = c
	}
	return ctx.JSON(http.StatusOK, apiCredential)
}

func (h HttpHandler) autoOnboardAzureSubscriptions(ctx context.Context, credential model.Credential, maxConnections int64) ([]api.Connection, error) {
	onboardedSources := make([]api.Connection, 0)
	cnf, err := h.vaultSc.Decrypt(ctx, credential.Secret)
	if err != nil {
		return nil, err
	}
	azureCnf, err := connectors.AzureSubscriptionConfigFromMap(cnf)
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
	// tracer :
	outputS, span := tracer.Start(ctx, "new_GetSourcesOfType", trace.WithSpanKind(trace.SpanKindServer))
	span.SetName("new_GetSourcesOfType")

	existingConnections, err := h.db.GetSourcesOfType(credential.ConnectorType)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	span.AddEvent("information", trace.WithAttributes(
		attribute.String("connector type", string(credential.ConnectorType)),
	))
	span.End()

	existingConnectionSubIDs := make([]string, 0, len(existingConnections))
	subsToOnboard := make([]azureSubscription, 0)
	for _, conn := range existingConnections {
		existingConnectionSubIDs = append(existingConnectionSubIDs, conn.SourceId)
	}
	outputS2, span2 := tracer.Start(outputS, "new_UpdateSource(loop)", trace.WithSpanKind(trace.SpanKindServer))
	span2.SetName("new_UpdateSource(loop)")

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
						localConn.LifecycleState = model.ConnectionLifecycleStateDisabled
					}
					if conn.Name != name || localConn.LifecycleState != conn.LifecycleState {
						_, span3 := tracer.Start(outputS2, "new_UpdateSource", trace.WithSpanKind(trace.SpanKindServer))
						span3.SetName("new_UpdateSource")

						_, err := h.db.UpdateSource(&localConn)
						if err != nil {
							span3.RecordError(err)
							span3.SetStatus(codes.Error, err.Error())
							h.logger.Error("failed to update source", zap.Error(err))
							return nil, err
						}
						span3.AddEvent("information", trace.WithAttributes(
							attribute.String("source name ", localConn.Name),
						))
						span3.End()
					}
				}
			}
		}
	}
	span2.End()

	h.logger.Info("onboarding subscriptions", zap.Int("count", len(subsToOnboard)))
	// tracer :
	outputS4, span4 := tracer.Start(outputS, "new_CreateSource(loop)", trace.WithSpanKind(trace.SpanKindServer))
	span4.SetName("new_CreateSource(loop)")

	for _, sub := range subsToOnboard {
		h.logger.Info("onboarding subscription", zap.String("subscriptionId", sub.SubscriptionID))
		// tracer :
		_, span6 := tracer.Start(outputS4, "CountSources", trace.WithSpanKind(trace.SpanKindServer))
		span6.SetName("CountSources")

		count, err := h.db.CountSources()
		if err != nil {
			span6.RecordError(err)
			span6.SetStatus(codes.Error, err.Error())
			return nil, err
		}
		span6.End()
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
			azureCnf.TenantID,
		)
		//tracer :
		_, span5 := tracer.Start(outputS4, "new_CreateSource", trace.WithSpanKind(trace.SpanKindServer))
		span5.SetName("new_CreateSource")

		err = h.db.CreateSource(&src)
		if err != nil {
			span5.RecordError(err)
			span5.SetStatus(codes.Error, err.Error())
			return nil, err
		}
		span5.AddEvent("information", trace.WithAttributes(
			attribute.String("source name", src.Name),
		))
		span5.End()

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
			OnboardDate:          src.CreatedAt,
			AssetDiscoveryMethod: src.AssetDiscoveryMethod,
			CredentialID:         src.CredentialID.String(),
			CredentialName:       src.Credential.Name,
			LifecycleState:       api.ConnectionLifecycleState(src.LifecycleState),
			HealthState:          src.HealthState,
			LastHealthCheckTime:  src.LastHealthCheckTime,
			HealthReason:         src.HealthReason,
			Metadata:             metadata,
		})
	}
	span4.End()

	return onboardedSources, nil
}

func (h HttpHandler) autoOnboardAWSAccounts(ctx context.Context, credential model.Credential, maxConnections int64) ([]api.Connection, error) {
	onboardedSources := make([]api.Connection, 0)
	cnf, err := h.vaultSc.Decrypt(ctx, credential.Secret)
	if err != nil {
		return nil, err
	}
	awsCnf, err := connectors.AWSAccountConfigFromMap(cnf)
	if err != nil {
		return nil, err
	}
	cfg, err := kaytuAws.GetConfig(
		ctx,
		awsCnf.AccessKey,
		awsCnf.SecretKey,
		"",
		awsCnf.AssumeAdminRoleName,
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
	// tracer :
	outputS, span := tracer.Start(ctx, "new_GetSourcesOfType", trace.WithSpanKind(trace.SpanKindServer))
	span.SetName("new_GetSourcesOfType")

	existingConnections, err := h.db.GetSourcesOfType(credential.ConnectorType)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	span.AddEvent("information", trace.WithAttributes(
		attribute.String("credential connector ", string(credential.ConnectorType)),
	))
	span.End()

	existingConnectionAccountIDs := make([]string, 0, len(existingConnections))
	for _, conn := range existingConnections {
		existingConnectionAccountIDs = append(existingConnectionAccountIDs, conn.SourceId)
	}
	accountsToOnboard := make([]awsAccount, 0)
	// tracer :
	outputS1, span1 := tracer.Start(outputS, "new_UpdateSource(loop)", trace.WithSpanKind(trace.SpanKindServer))
	span1.SetName("new_UpdateSource(loop)")

	for _, account := range accounts {
		if !utils.Includes(existingConnectionAccountIDs, account.AccountID) {
			accountsToOnboard = append(accountsToOnboard, account)
		} else {
			for _, conn := range existingConnections {
				if conn.LifecycleState == model.ConnectionLifecycleStateArchived {
					h.logger.Info("Archived Connection",
						zap.String("accountID", conn.SourceId))
				}
				if conn.SourceId == account.AccountID {
					name := account.AccountID
					if account.AccountName != nil {
						name = *account.AccountName
					}

					if conn.CredentialID.String() != credential.ID.String() {
						h.logger.Warn("organization account is onboarded as an standalone account",
							zap.String("accountID", account.AccountID),
							zap.String("connectionID", conn.ID.String()))
					}

					localConn := conn
					if conn.Name != name {
						localConn.Name = name
					}
					if account.Account.Status != awsOrgTypes.AccountStatusActive {
						localConn.LifecycleState = model.ConnectionLifecycleStateArchived
					} else if localConn.LifecycleState == model.ConnectionLifecycleStateArchived {
						localConn.LifecycleState = model.ConnectionLifecycleStateDiscovered
						if credential.AutoOnboardEnabled {
							localConn.LifecycleState = model.ConnectionLifecycleStateOnboard
						}
					}
					if conn.Name != name || account.Account.Status != awsOrgTypes.AccountStatusActive || conn.LifecycleState != localConn.LifecycleState {
						// tracer :
						_, span2 := tracer.Start(outputS1, "new_UpdateSource", trace.WithSpanKind(trace.SpanKindServer))
						span2.SetName("new_UpdateSource")

						_, err := h.db.UpdateSource(&localConn)
						if err != nil {
							span2.RecordError(err)
							span2.SetStatus(codes.Error, err.Error())
							h.logger.Error("failed to update source", zap.Error(err))
							return nil, err
						}
						span1.AddEvent("information", trace.WithAttributes(
							attribute.String("source name", localConn.Name),
						))
						span2.End()
					}
				}
			}
		}
	}
	span1.End()
	// TODO add tag filter
	// tracer :
	outputS3, span3 := tracer.Start(outputS1, "new_CountSources(loop)", trace.WithSpanKind(trace.SpanKindServer))
	span3.SetName("new_CountSources(loop)")

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
		_, span4 := tracer.Start(outputS3, "new_CountSources", trace.WithSpanKind(trace.SpanKindServer))
		span4.SetName("new_CountSources")

		count, err := h.db.CountSources()
		if err != nil {
			span4.RecordError(err)
			span4.SetStatus(codes.Error, err.Error())
			return nil, err
		}
		span4.End()

		if count >= maxConnections {
			return nil, echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("maximum number of connections reached: [%d/%d]", count, maxConnections))
		}

		src := NewAWSAutoOnboardedConnection(
			ctx,
			h.logger,
			awsCnf,
			account,
			source.SourceCreationMethodAutoOnboard,
			fmt.Sprintf("Auto onboarded account %s", account.AccountID),
			credential,
		)
		// tracer :
		outputS5, span5 := tracer.Start(outputS3, "new_Transaction", trace.WithSpanKind(trace.SpanKindServer))
		span5.SetName("new_Transaction")

		err = h.db.Orm.Transaction(func(tx *gorm.DB) error {
			_, span6 := tracer.Start(outputS5, "new_CreateSource", trace.WithSpanKind(trace.SpanKindServer))
			span6.SetName("new_CreateSource")

			err := h.db.CreateSource(&src)
			if err != nil {
				span6.RecordError(err)
				span6.SetStatus(codes.Error, err.Error())
				return err
			}
			span1.AddEvent("information", trace.WithAttributes(
				attribute.String("source name", src.Name),
			))
			span6.End()

			//TODO: add enable account

			return nil
		})
		if err != nil {
			span5.RecordError(err)
			span5.SetStatus(codes.Error, err.Error())
			return nil, err
		}
		span5.End()

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
	span3.End()

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
	// trace :
	_, span := tracer.Start(ctx.Request().Context(), "new_GetCredentialByID", trace.WithSpanKind(trace.SpanKindServer))
	span.SetName("new_GetCredentialByID")

	credential, err := h.db.GetCredentialByID(credId)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		if err == gorm.ErrRecordNotFound {
			return echo.NewHTTPError(http.StatusNotFound, "credential not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	span.AddEvent("information", trace.WithAttributes(
		attribute.String("credential id", credential.ID.String()),
	))
	span.End()

	cnf, err := h.metadataClient.GetConfigMetadata(&httpclient.Context{UserRole: api3.InternalRole}, models.MetadataKeyConnectionLimit)
	if err != nil {
		return err
	}

	var maxConnections int64

	if v, ok := cnf.GetValue().(int64); ok {
		maxConnections = v
	} else if v, ok := cnf.GetValue().(int); ok {
		maxConnections = int64(v)
	}

	onboardedSources := make([]api.Connection, 0)
	switch credential.ConnectorType {
	case source.CloudAzure:
		onboardedSources, err = h.autoOnboardAzureSubscriptions(ctx.Request().Context(), *credential, maxConnections)
		if err != nil {
			return err
		}

		for _, onboardedSrc := range onboardedSources {
			src, err := h.db.GetSource(onboardedSrc.ID)
			if err != nil {
				return err
			}

			_, err = h.checkConnectionHealth(ctx.Request().Context(), src, true)
			if err != nil {
				return err
			}
		}
	case source.CloudAWS:
		if credential.Version == 2 {
			onboardedSources, err = h.autoOnboardAWSAccountsV2(ctx.Request().Context(), *credential, maxConnections)
			if err != nil {
				return err
			}

			for _, onboardedSrc := range onboardedSources {
				src, err := h.db.GetSource(onboardedSrc.ID)
				if err != nil {
					return err
				}

				_, err = h.checkConnectionHealth(ctx.Request().Context(), src, true)
				if err != nil {
					return err
				}
			}
		} else {
			onboardedSources, err = h.autoOnboardAWSAccounts(ctx.Request().Context(), *credential, maxConnections)
			if err != nil {
				return err
			}

			for _, onboardedSrc := range onboardedSources {
				src, err := h.db.GetSource(onboardedSrc.ID)
				if err != nil {
					return err
				}

				_, err = h.checkConnectionHealth(ctx.Request().Context(), src, true)
				if err != nil {
					return err
				}
			}
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
	// trace :
	outputS, span := tracer.Start(ctx.Request().Context(), "new_GetCredentialByID")
	span.SetName("new_GetCredentialByID")

	cred, err := h.db.GetCredentialByID(id)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		return ctx.JSON(http.StatusNotFound, "credential not found")
	}
	span.AddEvent("information", trace.WithAttributes(
		attribute.String("credential id", cred.ID.String()),
	))
	span.End()

	if req.Name != nil {
		cred.Name = req.Name
	}

	cnf, err := h.vaultSc.Decrypt(ctx.Request().Context(), cred.Secret)
	if err != nil {
		return err
	}
	config, err := connectors.AzureSubscriptionConfigFromMap(cnf)
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
	metadata, err := getAzureCredentialsMetadata(ctx.Request().Context(), config, cred.CredentialType)
	if err != nil {
		return err
	}
	jsonMetadata, err := json.Marshal(metadata)
	if err != nil {
		return err
	}
	cred.Metadata = jsonMetadata

	secretBytes, err := h.vaultSc.Encrypt(ctx.Request().Context(), config.ToMap())
	if err != nil {
		return err
	}
	cred.Secret = string(secretBytes)
	if metadata.SpnName != "" {
		cred.Name = &metadata.SpnName
	}
	// trace :
	outputS1, span1 := tracer.Start(outputS, "new_Transaction")
	span1.SetName("new_Transaction")

	err = h.db.Orm.Transaction(func(tx *gorm.DB) error {
		// trace :
		_, span2 := tracer.Start(outputS1, "new_UpdateCredential")
		span2.SetName("new_UpdateCredential")

		if _, err := h.db.UpdateCredential(cred); err != nil {
			span2.RecordError(err)
			span2.SetStatus(codes.Error, err.Error())
			return err
		}
		span2.AddEvent("information", trace.WithAttributes(
			attribute.String("credential id", cred.ID.String()),
		))
		span2.End()

		return nil
	})
	if err != nil {
		span1.RecordError(err)
		span1.SetStatus(codes.Error, err.Error())
		return err
	}
	span1.End()

	_, err = h.checkCredentialHealth(outputS, *cred)
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
	// trace :
	outputS, span := tracer.Start(ctx.Request().Context(), "new_GetCredentialByID", trace.WithSpanKind(trace.SpanKindServer))
	span.SetName("new_GetCredentialByID")

	cred, err := h.db.GetCredentialByID(id)
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			return err
		}
		return ctx.JSON(http.StatusNotFound, "credential not found")
	}
	span.AddEvent("information", trace.WithAttributes(
		attribute.String("credential id", cred.ID.String()),
	))
	span.End()

	if req.Name != nil {
		cred.Name = req.Name
	}

	cnf, err := h.vaultSc.Decrypt(ctx.Request().Context(), cred.Secret)
	if err != nil {
		return err
	}
	config, err := connectors.AWSAccountConfigFromMap(cnf)
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
		if newConfig.AssumeAdminRoleName != "" {
			config.AssumeAdminRoleName = newConfig.AssumeAdminRoleName
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

	secretBytes, err := h.vaultSc.Encrypt(ctx.Request().Context(), config.ToMap())
	if err != nil {
		return err
	}
	cred.Secret = string(secretBytes)

	if metadata.OrganizationID != nil && metadata.OrganizationMasterAccountId != nil &&
		metadata.AccountID == *metadata.OrganizationMasterAccountId &&
		config.AssumeRoleName != "" && config.ExternalID != nil {
		cred.Name = metadata.OrganizationID
		cred.CredentialType = model.CredentialTypeManualAwsOrganization
		cred.AutoOnboardEnabled = true
	}
	// trace :
	outputS2, span2 := tracer.Start(outputS, "new_Transaction", trace.WithSpanKind(trace.SpanKindServer))
	span2.SetName("new_Transaction")

	err = h.db.Orm.Transaction(func(tx *gorm.DB) error {
		// trace :
		_, span3 := tracer.Start(outputS2, "new_UpdateCredential", trace.WithSpanKind(trace.SpanKindServer))
		span3.SetName("new_UpdateCredential")

		if _, err := h.db.UpdateCredential(cred); err != nil {
			span3.RecordError(err)
			span3.SetStatus(codes.Error, err.Error())
			return err
		}
		span3.AddEvent("information", trace.WithAttributes(
			attribute.String("credential id", cred.ID.String()),
		))
		span3.End()

		return nil
	})
	if err != nil {
		return err
	}
	span2.End()

	_, err = h.checkCredentialHealth(outputS, *cred)
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
	// trace :
	_, span := tracer.Start(ctx.Request().Context(), "new_put(Azure or Aws)Credentials", trace.WithSpanKind(trace.SpanKindServer))
	span.SetName("new_put(Azure or Aws)Credentials")

	switch req.Connector {
	case source.CloudAzure:
		return h.putAzureCredentials(ctx, req)
	case source.CloudAWS:
		return h.putAWSCredentials(ctx, req)
	}
	if req.Name != nil {
		span.AddEvent("information", trace.WithAttributes(
			attribute.String("credential name", *req.Name),
		))
	}
	span.End()
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

	// trace :
	outputS, span1 := tracer.Start(ctx.Request().Context(), "new_GetCredentialByID", trace.WithSpanKind(trace.SpanKindServer))
	span1.SetName("new_GetCredentialByID")

	credential, err := h.db.GetCredentialByID(credId)
	if err != nil {
		span1.RecordError(err)
		span1.SetStatus(codes.Error, err.Error())
		if err == gorm.ErrRecordNotFound {
			return echo.NewHTTPError(http.StatusNotFound, "credential not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	span1.AddEvent("information", trace.WithAttributes(
		attribute.String("credential id", credential.ID.String()),
	))
	span1.End()

	// trace :
	_, span2 := tracer.Start(outputS, "new_GetSourcesByCredentialID", trace.WithSpanKind(trace.SpanKindServer))
	span2.SetName("new_GetSourcesByCredentialID")

	sources, err := h.db.GetSourcesByCredentialID(credential.ID.String())
	if err != nil {
		span2.RecordError(err)
		span2.SetStatus(codes.Error, err.Error())
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	span2.End()

	// trace :
	outputS3, span3 := tracer.Start(outputS, "new_Transaction", trace.WithSpanKind(trace.SpanKindServer))
	span3.SetName("new_Transaction")

	err = h.db.Orm.Transaction(func(tx *gorm.DB) error {
		// trace :
		_, span4 := tracer.Start(outputS3, "new_DeleteCredential", trace.WithSpanKind(trace.SpanKindServer))
		span4.SetName("new_DeleteCredential")

		if err := h.db.DeleteCredential(credential.ID); err != nil {
			span4.RecordError(err)
			span4.SetStatus(codes.Error, err.Error())
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		span4.AddEvent("information", trace.WithAttributes(
			attribute.String("credential id", credential.ID.String()),
		))
		span4.End()

		// trace :
		output5, span5 := tracer.Start(outputS3, "new_UpdateSourceLifecycleState(loop)", trace.WithSpanKind(trace.SpanKindServer))
		span5.SetName("new_UpdateSourceLifecycleState(loop)")
		for _, src := range sources {
			// trace :
			_, span6 := tracer.Start(output5, "new_UpdateSourceLifecycleState", trace.WithSpanKind(trace.SpanKindServer))
			span6.SetName("new_UpdateSourceLifecycleState")
			if err := h.db.UpdateSourceLifecycleState(src.ID, model.ConnectionLifecycleStateDisabled); err != nil {
				span6.RecordError(err)
				span6.SetStatus(codes.Error, err.Error())
				return err
			}
			span6.AddEvent("information", trace.WithAttributes(
				attribute.String("source name", src.Name),
			))
			span6.End()
		}
		span5.End()

		return nil
	})
	if err != nil {
		span3.RecordError(err)
		span3.SetStatus(codes.Error, err.Error())
		return err
	}
	span3.End()

	return ctx.JSON(http.StatusOK, struct{}{})
}

func (h HttpHandler) GetSourceFullCred(ctx echo.Context) error {
	sourceUUID, err := uuid.Parse(ctx.Param("sourceId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid source uuid")
	}
	// trace :
	_, span := tracer.Start(ctx.Request().Context(), "new_GetSource", trace.WithSpanKind(trace.SpanKindServer))
	span.SetName("new_GetSource")

	src, err := h.db.GetSource(sourceUUID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	span.AddEvent("information", trace.WithAttributes(
		attribute.String("source name", src.Name),
	))
	span.End()

	cnf, err := h.vaultSc.Decrypt(ctx.Request().Context(), src.Credential.Secret)
	if err != nil {
		return err
	}

	switch src.Type {
	case source.CloudAWS:
		awsCnf, err := connectors.AWSAccountConfigFromMap(cnf)
		if err != nil {
			return err
		}
		return ctx.JSON(http.StatusOK, api.AWSCredentialConfig{
			AccountId:      awsCnf.AccountID,
			Regions:        awsCnf.Regions,
			AccessKey:      awsCnf.AccessKey,
			SecretKey:      awsCnf.SecretKey,
			AssumeRoleName: awsCnf.AssumeRoleName,
			ExternalId:     awsCnf.ExternalID,
		})
	case source.CloudAzure:
		azureCnf, err := connectors.AzureSubscriptionConfigFromMap(cnf)
		if err != nil {
			return err
		}
		return ctx.JSON(http.StatusOK, api.AzureCredentialConfig{
			SubscriptionId: azureCnf.SubscriptionID,
			TenantId:       azureCnf.TenantID,
			ObjectId:       azureCnf.ObjectID,
			SecretId:       azureCnf.SecretID,
			ClientId:       azureCnf.ClientID,
			ClientSecret:   azureCnf.ClientSecret,
		})
	default:
		return errors.New("invalid provider")
	}
}

func (h HttpHandler) updateConnectionHealth(ctx context.Context, connection model.Connection, healthStatus source.HealthStatus, reason *string, spendDiscovery, assetDiscovery *bool) (model.Connection, error) {
	connection.HealthState = healthStatus
	connection.HealthReason = reason
	connection.LastHealthCheckTime = time.Now()
	connection.SpendDiscovery = spendDiscovery
	connection.AssetDiscovery = assetDiscovery
	// tracer :
	_, span := tracer.Start(ctx, "new_UpdateSource", trace.WithSpanKind(trace.SpanKindServer))
	span.SetName("new_UpdateSource")
	defer span.End()

	_, err := h.db.UpdateSource(&connection)
	if err != nil {
		return model.Connection{}, err
	}
	span.AddEvent("information", trace.WithAttributes(
		attribute.String("source name", connection.Name),
	))
	span.End()
	//TODO Mahan: record state change in elastic search
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

	err = httpserver.CheckAccessToConnectionID(ctx, sourceUUID.String())
	if err != nil {
		h.logger.Error("failed to check access to connection", zap.Error(err))
		return err
	}

	updateMetadata := true
	if strings.ToLower(ctx.QueryParam("updateMetadata")) == "false" {
		updateMetadata = false
	}
	// trace :
	outputS, span := tracer.Start(ctx.Request().Context(), "new_GetSource", trace.WithSpanKind(trace.SpanKindServer))
	span.SetName("new_GetSource")
	defer span.End()

	connection, err := h.db.GetSource(sourceUUID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		span.End()
		h.logger.Error("failed to get source", zap.Error(err), zap.String("sourceId", sourceUUID.String()))
		return err
	}
	span.AddEvent("information", trace.WithAttributes(
		attribute.String("source name", connection.Name),
	))
	span.End()

	if !connection.LifecycleState.IsEnabled() {
		connection, err = h.updateConnectionHealth(outputS, connection, source.HealthStatusNil, utils.GetPointer("Connection is not enabled"), aws.Bool(false), aws.Bool(false))
		if err != nil {
			h.logger.Error("failed to update source health", zap.Error(err), zap.String("sourceId", connection.SourceId))
			return err
		}
	} else {
		isHealthy, err := h.checkCredentialHealth(outputS, connection.Credential)
		if err != nil {
			var herr *echo.HTTPError
			if errors.As(err, &herr) {
				if herr.Code == http.StatusInternalServerError {
					h.logger.Error("failed to check credential health", zap.Error(err), zap.String("sourceId", connection.SourceId))
					return herr
				}
			}
		}
		if !isHealthy {
			connection, err = h.updateConnectionHealth(outputS, connection, source.HealthStatusUnhealthy, utils.GetPointer("Credential is not healthy"), aws.Bool(false), aws.Bool(false))
			if err != nil {
				h.logger.Error("failed to update source health", zap.Error(err), zap.String("sourceId", connection.SourceId))
				return err
			}
		} else {
			connection, err = h.checkConnectionHealth(ctx.Request().Context(), connection, updateMetadata)
			if err != nil {
				h.logger.Error("failed to check connection health", zap.Error(err), zap.String("sourceId", connection.SourceId))
			}
		}
	}
	return ctx.JSON(http.StatusOK, entities.NewConnection(connection))
}

func (h HttpHandler) GetSource(ctx echo.Context) error {
	srcId, err := uuid.Parse(ctx.Param(paramSourceId))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}
	// trace :
	_, span := tracer.Start(ctx.Request().Context(), "new_GetSource", trace.WithSpanKind(trace.SpanKindServer))
	span.SetName("new_GetSource")

	src, err := h.db.GetSource(srcId)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		if err == gorm.ErrRecordNotFound {
			return echo.NewHTTPError(http.StatusBadRequest, "source not found")
		}
		return err
	}
	span.AddEvent("information", trace.WithAttributes(
		attribute.String("source name", src.Name),
	))
	span.End()

	metadata := make(map[string]any)
	if src.Metadata.String() != "" {
		err := json.Unmarshal(src.Metadata, &metadata)
		if err != nil {
			return err
		}
	}

	apiRes := entities.NewConnection(src)
	if httpserver.GetUserRole(ctx) == api3.InternalRole {
		apiRes.Credential = entities.NewCredential(src.Credential)
		apiRes.Credential.Config = src.Credential.Secret
		if apiRes.Credential.Version == 2 {
			apiRes.Credential.Config, err = h.CredentialV2ToV1(ctx.Request().Context(), src.Credential)
			if err != nil {
				return err
			}
		}
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
	// trace :
	outputS, span := tracer.Start(ctx.Request().Context(), "new_GetSource", trace.WithSpanKind(trace.SpanKindServer))
	span.SetName("new_GetSource")

	src, err := h.db.GetSource(srcId)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		if err == gorm.ErrRecordNotFound {
			return echo.NewHTTPError(http.StatusBadRequest, "source not found")
		}
		return err
	}
	span.AddEvent("information", trace.WithAttributes(
		attribute.String("source name", src.Name),
	))
	span.End()

	// trace :
	output1, span1 := tracer.Start(outputS, "new_Transaction", trace.WithSpanKind(trace.SpanKindServer))
	span1.SetName("new_Transaction")

	err = h.db.Orm.Transaction(func(tx *gorm.DB) error {
		// trace :
		outputS2, span2 := tracer.Start(output1, "new_DeleteSource")

		if err := h.db.DeleteSource(srcId); err != nil {
			span2.RecordError(err)
			span2.SetStatus(codes.Error, err.Error())
			return err
		}
		span1.AddEvent("information", trace.WithAttributes(
			attribute.String("source name", src.Name),
		))
		span2.End()

		if src.Credential.CredentialType.IsManual() {
			// trace :
			_, span3 := tracer.Start(outputS2, "new_DeleteCredential", trace.WithSpanKind(trace.SpanKindServer))
			span3.SetName("new_DeleteCredential")
			err = h.db.DeleteCredential(src.Credential.ID)
			if err != nil {
				span3.RecordError(err)
				span3.SetStatus(codes.Error, err.Error())
				return err
			}
			span3.AddEvent("information", trace.WithAttributes(
				attribute.String("credential id", src.Credential.ID.String()),
			))
			span3.End()
		}

		return nil
	})
	if err != nil {
		span1.RecordError(err)
		span1.SetStatus(codes.Error, err.Error())
		return err
	}
	span1.End()

	return ctx.NoContent(http.StatusOK)
}

// ChangeConnectionLifecycleState godoc
//
//	@Summary	Change connection lifecycle state
//	@Security	BearerToken
//	@Tags		onboard
//	@Produce	json
//	@Param		connectionId	path	string										true	"Connection ID"
//	@Param		request			body	api.ChangeConnectionLifecycleStateRequest	true	"Request"
//	@Success	200
//	@Router		/onboard/api/v1/connections/{connectionId}/state [post]
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
	// trace :
	outputS, span := tracer.Start(ctx.Request().Context(), "new_GetSource", trace.WithSpanKind(trace.SpanKindServer))
	span.SetName("new_GetSource")

	connection, err := h.db.GetSource(connectionId)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return echo.NewHTTPError(http.StatusBadRequest, "source not found")
		}
		return err
	}
	span.AddEvent("information", trace.WithAttributes(
		attribute.String("source name", connection.Name),
	))
	span.End()

	reqState := entity.ConnectionLifecycleState(req.State).ToModel()
	if reqState == connection.LifecycleState {
		return echo.NewHTTPError(http.StatusBadRequest, "connection already in requested state")
	}

	if reqState.IsEnabled() != connection.LifecycleState.IsEnabled() {
		// trace :
		_, span2 := tracer.Start(outputS, "new_UpdateSourceLifecycleState")
		span2.SetName("new_UpdateSourceLifecycleState")

		if err := h.db.UpdateSourceLifecycleState(connectionId, reqState); err != nil {
			span2.RecordError(err)
			span2.SetStatus(codes.Error, err.Error())
			return err
		}
		span2.AddEvent("information", trace.WithAttributes(
			attribute.String("source name", connection.Name),
		))
		span2.End()

	} else {
		// trace :
		_, span3 := tracer.Start(outputS, "new_UpdateSourceLifecycleState", trace.WithSpanKind(trace.SpanKindServer))
		span3.SetName("new_UpdateSourceLifecycleState")

		err = h.db.UpdateSourceLifecycleState(connectionId, reqState)
		span3.AddEvent("information", trace.WithAttributes(
			attribute.String("source name", connection.Name),
		))
		span3.End()
	}
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	return ctx.NoContent(http.StatusOK)
}

func (h HttpHandler) ListSources(ctx echo.Context) error {
	var err error
	sType := httpserver.QueryArrayParam(ctx, "connector")
	var sources []model.Connection
	if len(sType) > 0 {
		st := source.ParseTypes(sType)
		// trace :
		_, span := tracer.Start(ctx.Request().Context(), "new_GetSourcesOfTypes", trace.WithSpanKind(trace.SpanKindServer))
		span.SetName("new_GetSourcesOfTypes")

		sources, err = h.db.GetSourcesOfTypes(st)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return err
		}
		span.End()
	} else {
		// trace :
		_, span := tracer.Start(ctx.Request().Context(), "new_ListSources", trace.WithSpanKind(trace.SpanKindServer))
		span.SetName("new_ListSources")

		sources, err = h.db.ListSources()
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return err
		}
		span.End()
	}

	resp := api.GetSourcesResponse{}
	for _, s := range sources {
		apiRes := entities.NewConnection(s)
		resp = append(resp, apiRes)
	}

	return ctx.JSON(http.StatusOK, resp)
}

func (h HttpHandler) GetSources(ctx echo.Context) error {
	var req api.GetSourcesRequest
	if err := bindValidate(ctx, &req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	// trace :
	_, span := tracer.Start(ctx.Request().Context(), "new_GetSources", trace.WithSpanKind(trace.SpanKindServer))
	span.SetName("new_GetSources")

	srcs, err := h.db.GetSources(req.SourceIDs)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return echo.NewHTTPError(http.StatusBadRequest, "source not found")
		}
		return err
	}
	span.End()

	var res []api.Connection
	for _, src := range srcs {
		apiRes := entities.NewConnection(src)
		if httpserver.GetUserRole(ctx) == api3.InternalRole {
			apiRes.Credential = entities.NewCredential(src.Credential)
			apiRes.Credential.Config = src.Credential.Secret
			if apiRes.Credential.Version == 2 {
				apiRes.Credential.Config, err = h.CredentialV2ToV1(ctx.Request().Context(), src.Credential)
				if err != nil {
					return err
				}
			}
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
		// trace :
		_, span := tracer.Start(ctx.Request().Context(), "new_CountSourcesOfType", trace.WithSpanKind(trace.SpanKindServer))
		span.SetName("new_CountSourcesOfType")

		count, err = h.db.CountSourcesOfType(st)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return err
		}
		span.AddEvent("information", trace.WithAttributes(
			attribute.String("source type", st.String()),
		))
		span.End()

	} else {
		var err error
		// trace :
		_, span := tracer.Start(ctx.Request().Context(), "new_CountSources", trace.WithSpanKind(trace.SpanKindServer))
		span.SetName("new_CountSources")

		count, err = h.db.CountSources()
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return err
		}
		span.End()
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
//	@Param			connector	query		[]source.Type	false	"Connector"
//	@Success		200			{object}	api.CatalogMetrics
//	@Router			/onboard/api/v1/catalog/metrics [get]
func (h HttpHandler) CatalogMetrics(ctx echo.Context) error {
	var metrics api.CatalogMetrics
	// trace :
	_, span := tracer.Start(ctx.Request().Context(), "new_ListSources", trace.WithSpanKind(trace.SpanKindServer))
	span.SetName("new_ListSources")

	connectors := source.ParseTypes(httpserver.QueryArrayParam(ctx, "connector"))

	srcs, err := h.db.ListSourcesWithFilters(connectors, nil, nil, nil)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return echo.NewHTTPError(http.StatusInternalServerError)
	}
	span.End()

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

		if src.LifecycleState == model.ConnectionLifecycleStateInProgress {
			metrics.InProgressConnections++
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
//	@Param			filter				query		string			false	"Filter costs"
//	@Param			connector			query		[]source.Type	false	"Connector"
//	@Param			connectionId		query		[]string		false	"Connection IDs"
//	@Param			resourceCollection	query		[]string		false	"Resource collection IDs to filter by"
//	@Param			connectionGroups	query		[]string		false	"Connection Groups"
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
	connectionIDs, err := httpserver.ResolveConnectionIDs(ctx, connectionIDs)
	if err != nil {
		return err
	}
	connectionGroups := httpserver.QueryArrayParam(ctx, "connectionGroups")
	resourceCollections := httpserver.QueryArrayParam(ctx, "resourceCollection")
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
	startTime := endTime.AddDate(0, -1, 0)
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
	var lifecycleStateSlice []model.ConnectionLifecycleState
	lifecycleState := ctx.QueryParam("lifecycleState")
	if lifecycleState != "" {
		lifecycleStateSlice = append(lifecycleStateSlice, model.ConnectionLifecycleState(lifecycleState))
	}

	var healthStateSlice []source.HealthStatus
	healthState := ctx.QueryParam("healthState")
	if healthState != "" {
		healthStateSlice = append(healthStateSlice, source.HealthStatus(healthState))
	}
	// trace :
	_, span := tracer.Start(ctx.Request().Context(), "new_ListSourcesWithFilters", trace.WithSpanKind(trace.SpanKindServer))
	span.SetName("new_ListSourcesWithFilters")

	filterStr := ctx.QueryParam("filter")
	if filterStr != "" {
		var filter map[string]interface{}
		err = json.Unmarshal([]byte(filterStr), &filter)
		if err != nil {
			return ctx.JSON(http.StatusBadRequest, "could not parse filter")
		}
		connectionIDs, err = h.connectionsFilter(ctx, filter)
		if err != nil {
			return ctx.JSON(http.StatusBadRequest, fmt.Sprintf("invalid filter: %s", err.Error()))
		}
		h.logger.Warn(fmt.Sprintf("===Filtered Connections: %v", connectionIDs))
	}

	tmpConnections, err := h.db.ListSourcesWithFilters(connectors, connectionIDs, lifecycleStateSlice, healthStateSlice)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	span.End()

	_, span = tracer.Start(ctx.Request().Context(), "new_FilterConnectionGroups", trace.WithSpanKind(trace.SpanKindServer))
	span.SetName("new_FilterConnectionGroups")

	var connections []model.Connection
	if filterStr != "" && len(connectionIDs) == 0 {
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
		return ctx.JSON(http.StatusOK, result)
	} else if len(connectionGroups) > 0 && filterStr == "" {
		var validConnections []string
		for _, group := range connectionGroups {
			connectionGroup, err := h.db.GetConnectionGroupByName(group)
			if err != nil {
				span.RecordError(err)
				span.SetStatus(codes.Error, err.Error())
				h.logger.Error("error getting connection group", zap.Error(err))
				return err
			}

			span.AddEvent("information", trace.WithAttributes(
				attribute.String("connectionGroup name", connectionGroup.Name),
			))
			apiCg, err := entities.NewConnectionGroup(ctx.Request().Context(), h.steampipeConn, *connectionGroup)
			if err != nil {
				h.logger.Error("error populating connection group", zap.Error(err))
				return err
			}
			validConnections = append(validConnections, apiCg.ConnectionIds...)
		}
		for _, c := range tmpConnections {
			for _, vc := range validConnections {
				if c.ID.String() == vc {
					connections = append(connections, c)
					break
				}
			}
		}
	} else {
		connections = tmpConnections
	}

	span.End()

	needCostStr := ctx.QueryParam("needCost")
	needCost := true
	// cost for resource collections is not supported yet
	if nc, err := strconv.ParseBool(needCostStr); (err == nil && !nc) || len(resourceCollections) > 0 {
		needCost = false
	}
	needResourceCountStr := ctx.QueryParam("needResourceCount")
	needResourceCount := true
	if nrc, err := strconv.ParseBool(needResourceCountStr); err == nil && !nrc {
		needResourceCount = false
	}

	connectionData := map[string]api2.ConnectionData{}
	if needResourceCount || needCost {
		connectionData, err = h.inventoryClient.ListConnectionsData(httpclient.FromEchoContext(ctx), nil, resourceCollections, &startTime, &endTime, nil, needCost, needResourceCount)
		if err != nil {
			return err
		}
	}

	pendingDescribeConnections, err := h.describeClient.ListPendingConnections(&httpclient.Context{UserRole: api3.InternalRole})
	if err != nil {
		return err
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
			apiConn := entities.NewConnection(connection)
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
			if (localData.Count == nil || *localData.Count == 0) && len(resourceCollections) > 0 {
				continue
			}
			result.Connections = append(result.Connections, apiConn)
		} else {
			if len(resourceCollections) > 0 {
				continue
			}
			result.Connections = append(result.Connections, entities.NewConnection(connection))
		}
		switch connection.LifecycleState {
		case model.ConnectionLifecycleStateDiscovered:
			result.TotalDiscoveredCount++
		case model.ConnectionLifecycleStateDisabled:
			result.TotalDisabledCount++
		case model.ConnectionLifecycleStateInProgress:
			fallthrough
		case model.ConnectionLifecycleStateOnboard:
			result.TotalOnboardedCount++
		case model.ConnectionLifecycleStateArchived:
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
	for idx, cnn := range result.Connections {
		cnn.ConnectionID = demo.EncodeResponseData(ctx, cnn.ConnectionID)
		cnn.ConnectionName = demo.EncodeResponseData(ctx, cnn.ConnectionName)
		for _, pc := range pendingDescribeConnections {
			if cnn.ID.String() == pc {
				cnn.DescribeJobRunning = true
				break
			}
		}
		result.Connections[idx] = cnn
	}
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
	// trace :
	outputS, span := tracer.Start(ctx.Request().Context(), "new_ListConnectionGroups", trace.WithSpanKind(trace.SpanKindServer))
	span.SetName("new_ListConnectionGroups")

	connectionGroups, err := h.db.ListConnectionGroups()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		h.logger.Error("error listing connection groups", zap.Error(err))
		return err
	}
	span.End()

	result := make([]api.ConnectionGroup, 0, len(connectionGroups))
	// tracer :
	outputS2, span2 := tracer.Start(outputS, "new_GetSources(loop)", trace.WithSpanKind(trace.SpanKindServer))
	span2.SetName("new_GetSources(loop)")

	for _, connectionGroup := range connectionGroups {
		apiCg, err := entities.NewConnectionGroup(ctx.Request().Context(), h.steampipeConn, connectionGroup)
		if err != nil {
			h.logger.Error("error populating connection group", zap.Error(err))
			continue
		}
		if populateConnections {
			// trace :
			_, span3 := tracer.Start(outputS2, "new_GetSources", trace.WithSpanKind(trace.SpanKindServer))
			span3.SetName("new_GetSources")

			connections, err := h.db.GetSources(apiCg.ConnectionIds)
			if err != nil {
				span3.RecordError(err)
				span3.SetStatus(codes.Error, err.Error())
				h.logger.Error("error getting connections", zap.Error(err))
				return err
			}
			span3.End()

			apiCg.Connections = make([]api.Connection, 0, len(connections))
			for _, connection := range connections {
				apiCg.Connections = append(apiCg.Connections, entities.NewConnection(connection))
			}
		}

		result = append(result, *apiCg)
	}
	span2.End()
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
	// trace :
	outputS, span := tracer.Start(ctx.Request().Context(), "new_GetConnectionGroupByName", trace.WithSpanKind(trace.SpanKindServer))
	span.SetName("new_GetConnectionGroupByName")

	connectionGroup, err := h.db.GetConnectionGroupByName(connectionGroupName)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		h.logger.Error("error getting connection group", zap.Error(err))
		return err
	}
	span.AddEvent("information", trace.WithAttributes(
		attribute.String("connectionGroup name", connectionGroup.Name),
	))
	span.End()

	apiCg, err := entities.NewConnectionGroup(ctx.Request().Context(), h.steampipeConn, *connectionGroup)
	if err != nil {
		h.logger.Error("error populating connection group", zap.Error(err))
		return err
	}

	if populateConnections {
		// trace :
		_, span1 := tracer.Start(outputS, "new_GetSources", trace.WithSpanKind(trace.SpanKindServer))
		span1.SetName("new_GetSources")

		connections, err := h.db.GetSources(apiCg.ConnectionIds)
		if err != nil {
			span1.RecordError(err)
			span1.SetStatus(codes.Error, err.Error())
			h.logger.Error("error getting connections", zap.Error(err))
			return err
		}
		span1.End()

		apiCg.Connections = make([]api.Connection, 0, len(connections))
		for _, connection := range connections {
			apiCg.Connections = append(apiCg.Connections, entities.NewConnection(connection))
		}
	}

	return ctx.JSON(http.StatusOK, apiCg)
}

func (h *HttpHandler) connectionsFilter(ctx echo.Context, filter map[string]interface{}) ([]string, error) {
	var connections []string
	allConnections, err := h.db.ListSources()
	if err != nil {
		return nil, err
	}
	var allConnectionsStr []string
	for _, c := range allConnections {
		allConnectionsStr = append(allConnectionsStr, c.ID.String())
	}
	for key, value := range filter {
		if key == "Match" {
			dimFilter := value.(map[string]interface{})
			if dimKey, ok := dimFilter["Key"]; ok {
				if dimKey == "ConnectionID" {
					connections, err = dimFilterFunction(dimFilter, allConnectionsStr)
					if err != nil {
						return nil, err
					}
					h.logger.Warn(fmt.Sprintf("===Dim Filter Function on filter %v, result: %v", dimFilter, connections))
				} else if dimKey == "Provider" {
					providers, err := dimFilterFunction(dimFilter, []string{"AWS", "Azure"})
					if err != nil {
						return nil, err
					}
					h.logger.Warn(fmt.Sprintf("===Dim Filter Function on filter %v, result: %v", dimFilter, providers))
					for _, c := range allConnections {
						if arrayContains(providers, c.Connector.Name.String()) {
							connections = append(connections, c.ID.String())
						}
					}
				} else if dimKey == "ConnectionGroup" {
					allGroups, err := h.db.ListConnectionGroups()
					if err != nil {
						return nil, err
					}
					allGroupsMap := make(map[string][]string)
					var allGroupsStr []string
					for _, group := range allGroups {
						g, err := entities.NewConnectionGroup(ctx.Request().Context(), h.steampipeConn, group)
						if err != nil {
							return nil, err
						}
						allGroupsMap[g.Name] = make([]string, 0, len(g.ConnectionIds))
						for _, cid := range g.ConnectionIds {
							allGroupsMap[g.Name] = append(allGroupsMap[g.Name], cid)
							allGroupsStr = append(allGroupsStr, cid)
						}
					}
					groups, err := dimFilterFunction(dimFilter, allGroupsStr)
					if err != nil {
						return nil, err
					}
					h.logger.Warn(fmt.Sprintf("===Dim Filter Function on filter %v, result: %v", dimFilter, groups))
					for _, g := range groups {
						for _, conn := range allGroupsMap[g] {
							if !arrayContains(connections, conn) {
								connections = append(connections, conn)
							}
						}
					}
				} else if dimKey == "ConnectionName" {
					var allConnectionsNames []string
					for _, c := range allConnections {
						allConnectionsNames = append(allConnectionsNames, c.Name)
					}
					connectionNames, err := dimFilterFunction(dimFilter, allConnectionsNames)
					if err != nil {
						return nil, err
					}
					h.logger.Warn(fmt.Sprintf("===Dim Filter Function on filter %v, result: %v", dimFilter, connectionNames))
					for _, conn := range allConnections {
						if arrayContains(connectionNames, conn.Name) {
							connections = append(connections, conn.ID.String())
						}
					}
				}
			} else {
				return nil, fmt.Errorf("missing key")
			}
		} else if key == "AND" {
			var andFilters []map[string]interface{}
			for _, v := range value.([]interface{}) {
				andFilter := v.(map[string]interface{})
				andFilters = append(andFilters, andFilter)
			}
			counter := make(map[string]int)
			for _, f := range andFilters {
				values, err := h.connectionsFilter(ctx, f)
				if err != nil {
					return nil, err
				}
				for _, v := range values {
					if c, ok := counter[v]; ok {
						counter[v] = c + 1
					} else {
						counter[v] = 1
					}
					if counter[v] == len(andFilters) {
						connections = append(connections, v)
					}
				}
			}
		} else if key == "OR" {
			var orFilters []map[string]interface{}
			for _, v := range value.([]interface{}) {
				orFilter := v.(map[string]interface{})
				orFilters = append(orFilters, orFilter)
			}
			for _, f := range orFilters {
				values, err := h.connectionsFilter(ctx, f)
				if err != nil {
					return nil, err
				}
				for _, v := range values {
					if !arrayContains(connections, v) {
						connections = append(connections, v)
					}
				}
			}
		} else {
			return nil, fmt.Errorf("invalid key: %s", key)
		}
	}
	return connections, nil
}

func dimFilterFunction(dimFilter map[string]interface{}, allValues []string) ([]string, error) {
	var values []string
	for _, v := range dimFilter["Values"].([]interface{}) {
		values = append(values, fmt.Sprintf("%v", v))
	}
	var output []string
	if matchOption, ok := dimFilter["MatchOption"]; ok {
		switch {
		case strings.Contains(matchOption.(string), "EQUAL"):
			output = values
		case strings.Contains(matchOption.(string), "STARTS_WITH"):
			for _, v := range values {
				for _, conn := range allValues {
					if strings.HasPrefix(conn, v) {
						if !arrayContains(output, conn) {
							output = append(output, conn)
						}
					}
				}
			}
		case strings.Contains(matchOption.(string), "ENDS_WITH"):
			for _, v := range values {
				for _, conn := range allValues {
					if strings.HasSuffix(conn, v) {
						if !arrayContains(output, conn) {
							output = append(output, conn)
						}
					}
				}
			}
		case strings.Contains(matchOption.(string), "CONTAINS"):
			for _, v := range values {
				for _, conn := range allValues {
					if strings.Contains(conn, v) {
						if !arrayContains(output, conn) {
							output = append(output, conn)
						}
					}
				}
			}
		default:
			return nil, fmt.Errorf("invalid option")
		}
		if strings.HasPrefix(matchOption.(string), "~") {
			var notOutput []string
			for _, v := range allValues {
				if !arrayContains(output, v) {
					notOutput = append(notOutput, v)
				}
			}
			return notOutput, nil
		}
	} else {
		output = values
	}
	return output, nil
}

func arrayContains(array []string, key string) bool {
	for _, v := range array {
		if v == key {
			return true
		}
	}
	return false
}

func (h HttpHandler) GetSourceBySourceId(ctx echo.Context) error {
	srcId := ctx.Param("source_id")

	_, span := tracer.Start(ctx.Request().Context(), "new_GetSource", trace.WithSpanKind(trace.SpanKindServer))
	span.SetName("new_GetSource")

	src, err := h.db.GetSourceBySourceID(srcId)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	span.AddEvent("information", trace.WithAttributes(
		attribute.String("source name", src.Name),
	))
	span.End()

	metadata := make(map[string]any)
	if src.Metadata.String() != "" {
		err := json.Unmarshal(src.Metadata, &metadata)
		if err != nil {
			return err
		}
	}

	apiRes := entities.NewConnection(src)
	if httpserver.GetUserRole(ctx) == api3.InternalRole {
		apiRes.Credential = entities.NewCredential(src.Credential)
		apiRes.Credential.Config = src.Credential.Secret
		if apiRes.Credential.Version == 2 {
			apiRes.Credential.Config, err = h.CredentialV2ToV1(ctx.Request().Context(), src.Credential)
			if err != nil {
				return err
			}
		}
	}

	return ctx.JSON(http.StatusOK, apiRes)
}

func (h HttpHandler) ListSourcesByFilters(ctx echo.Context) error {
	var request api.GetSourceByFiltersRequest
	if err := ctx.Bind(&request); err != nil {
		ctx.Logger().Errorf("bind the request: %v", err)
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}

	_, span := tracer.Start(ctx.Request().Context(), "new_GetSourceByFilters", trace.WithSpanKind(trace.SpanKindServer))
	span.SetName("new_GetSourceByFilters")

	sources, err := h.db.ListSourcesByFilters(request.Connector, request.ProviderNameRegex, request.ProviderIdRegex)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	span.End()
	var results []api.Connection

	for _, src := range sources {
		metadata := make(map[string]any)
		if src.Metadata.String() != "" {
			err := json.Unmarshal(src.Metadata, &metadata)
			if err != nil {
				return err
			}
		}
		apiRes := entities.NewConnection(src)
		if httpserver.GetUserRole(ctx) == api3.InternalRole {
			apiRes.Credential = entities.NewCredential(src.Credential)
			apiRes.Credential.Config = src.Credential.Secret
			if apiRes.Credential.Version == 2 {
				apiRes.Credential.Config, err = h.CredentialV2ToV1(ctx.Request().Context(), src.Credential)
				if err != nil {
					return err
				}
			}
		}
		results = append(results, apiRes)
	}

	return ctx.JSON(http.StatusOK, results)
}

func (h HttpHandler) GetSourceByFilters(ctx echo.Context) error {
	var request api.GetSourceByFiltersRequest
	if err := ctx.Bind(&request); err != nil {
		ctx.Logger().Errorf("bind the request: %v", err)
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}

	_, span := tracer.Start(ctx.Request().Context(), "new_GetSourceByFilters", trace.WithSpanKind(trace.SpanKindServer))
	span.SetName("new_GetSourceByFilters")

	src, err := h.db.GetSourceByFilters(request.Connector, request.ProviderNameRegex, request.ProviderIdRegex)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	span.End()
	var result api.Connection

	metadata := make(map[string]any)
	if src.Metadata.String() != "" {
		err := json.Unmarshal(src.Metadata, &metadata)
		if err != nil {
			return err
		}
	}
	apiRes := entities.NewConnection(src)
	if httpserver.GetUserRole(ctx) == api3.InternalRole {
		apiRes.Credential = entities.NewCredential(src.Credential)
		apiRes.Credential.Config = src.Credential.Secret
		if apiRes.Credential.Version == 2 {
			apiRes.Credential.Config, err = h.CredentialV2ToV1(ctx.Request().Context(), src.Credential)
			if err != nil {
				return err
			}
		}
	}
	result = apiRes

	return ctx.JSON(http.StatusOK, result)
}

// PurgeSampleData godoc
//
//	@Summary		List all workspaces with owner id
//	@Description	Returns all workspaces with owner id
//	@Security		BearerToken
//	@Tags			workspace
//	@Accept			json
//	@Produce		json
//	@Param			ignore_source_ids	query	[]string	false	"ignore_source_ids"
//	@Success		200
//	@Router			/workspace/api/v3/sample/purge [put]
func (s HttpHandler) PurgeSampleData(c echo.Context) error {
	err := s.db.DeleteSources()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to delete sources")
	}
	err = s.db.DeleteCredentials()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to delete credentials")
	}
	err = s.db.DeleteConnectionGroups()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to delete connection groups")
	}

	return c.NoContent(http.StatusOK)
}

// ListConnectorsV2 godoc
//
//	@Summary		List connectors v2
//	@Description	Returns list of all connectors v2
//	@Security		BearerToken
//	@Tags			onboard
//	@Produce		json
//	@Param			tier		query		string	false	"Tier (Community, Enterprise, (default both)"
//	@Param			per_page	query		int		false	"PerPage"
//	@Param			cursor		query		int		false	"Cursor"
//	@Success		200			{object}	[]api.ConnectorCount
//	@Router			/onboard/api/v3/connector [get]
func (h HttpHandler) ListConnectorsV2(ctx echo.Context) error {
	//trace :
	outputS, span1 := tracer.Start(ctx.Request().Context(), "new_ListConnectorsV2", trace.WithSpanKind(trace.SpanKindServer))
	span1.SetName("new_ListConnectorsV2")
	tier := ""
	if model.Tier(ctx.QueryParam("tier")) == model.Tier_Community || model.Tier(ctx.QueryParam("tier")) == model.Tier_Enterprise {
		tier = ctx.QueryParam("tier")
	}

	perPageStr := ctx.QueryParam("per_page")
	cursorStr := ctx.QueryParam("cursor")
	var perPage, cursor int64
	if perPageStr != "" {
		perPage, _ = strconv.ParseInt(perPageStr, 10, 64)
	}
	if cursorStr != "" {
		cursor, _ = strconv.ParseInt(cursorStr, 10, 64)
	}

	connectors, err := h.db.ListConnectorsTierFiltered(tier)
	if err != nil {
		span1.RecordError(err)
		span1.SetStatus(codes.Error, err.Error())
		return err
	}
	span1.End()

	var items []api.ConnectorCount

	//trace :
	outputS2, span2 := tracer.Start(outputS, "new_CountSourcesOfType(loop)")
	span2.SetName("new_CountSourcesOfType(loop)")

	for _, c := range connectors {
		_, span3 := tracer.Start(outputS2, "new_CountSourcesOfType", trace.WithSpanKind(trace.SpanKindServer))
		span3.SetName("new_CountSourcesOfType")

		count, err := h.db.CountSourcesOfType(c.Name)
		if err != nil {
			span3.RecordError(err)
			span3.SetStatus(codes.Error, err.Error())
			return err
		}
		span3.AddEvent("information", trace.WithAttributes(
			attribute.String("source name", string(c.Name)),
		))
		span3.End()

		tags := make(map[string]any)
		err = json.Unmarshal(c.Tags, &tags)
		if err != nil {
			return err
		}
		items = append(items, api.ConnectorCount{
			Connector: api.Connector{
				Name:                c.Name,
				Label:               c.Label,
				ShortDescription:    c.ShortDescription,
				Description:         c.Description,
				Direction:           c.Direction,
				Status:              c.Status,
				Tier:                string(c.Tier),
				Logo:                c.Logo,
				AutoOnboardSupport:  c.AutoOnboardSupport,
				AllowNewConnections: c.AllowNewConnections,
				MaxConnectionLimit:  c.MaxConnectionLimit,
				Tags:                tags,
			},
			ConnectionCount: count,
		})
	}
	totalCount := len(items)

	sort.Slice(items, func(i, j int) bool {
		return items[i].Label < items[j].Label
	})
	if perPage != 0 {
		if cursor == 0 {
			items = utils.Paginate(1, perPage, items)
		} else {
			items = utils.Paginate(cursor, perPage, items)
		}
	}

	span2.End()
	return ctx.JSON(http.StatusOK, api.ListConnectorsV2Response{
		Connectors: items,
		TotalCount: int64(totalCount),
	})
}
