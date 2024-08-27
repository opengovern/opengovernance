package credential

import (
	"encoding/json"
	"errors"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	kaytuAws "github.com/kaytu-io/kaytu-aws-describer/aws"
	"github.com/kaytu-io/kaytu-util/pkg/api"
	"github.com/kaytu-io/kaytu-util/pkg/httpserver"
	"net/http"
	"sort"
	"strconv"

	"github.com/google/uuid"
	"github.com/kaytu-io/kaytu-engine/pkg/utils"
	"github.com/kaytu-io/kaytu-engine/services/integration/api/entity"
	"github.com/kaytu-io/kaytu-engine/services/integration/model"
	"github.com/kaytu-io/kaytu-engine/services/integration/repository"
	"github.com/kaytu-io/kaytu-engine/services/integration/service"
	"github.com/kaytu-io/kaytu-util/pkg/fp"
	"github.com/kaytu-io/kaytu-util/pkg/source"
	"github.com/labstack/echo/v4"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

type API struct {
	credentialSvc service.Credential
	connectionSvc service.Connection
	tracer        trace.Tracer
	logger        *zap.Logger
}

func New(
	credentialSvc service.Credential,
	connectionSvc service.Connection,
	logger *zap.Logger,
) API {
	return API{
		credentialSvc: credentialSvc,
		connectionSvc: connectionSvc,
		tracer:        otel.GetTracerProvider().Tracer("integration.http.sources"),
		logger:        logger.Named("source"),
	}
}

// Get godoc
//
//	@Summary		Get Credential
//	@Description	Retrieving credential details by credential ID
//	@Security		BearerToken
//	@Tags			credentials
//	@Produce		json
//	@Success		200				{object}	entity.Credential
//	@Param			credentialId	path		string	true	"Credential ID"
//	@Router			/integration/api/v1/credentials/{credentialId} [get]
func (h API) Get(c echo.Context) error {
	ctx := otel.GetTextMapPropagator().Extract(c.Request().Context(), propagation.HeaderCarrier(c.Request().Header))

	id, err := uuid.Parse(c.Param("credentialId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}

	ctx, span := h.tracer.Start(ctx, "delete")
	defer span.End()

	credential, err := h.credentialSvc.Get(ctx, id.String())
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())

		if err == repository.ErrCredentialNotFound {
			return echo.NewHTTPError(http.StatusNotFound, "credential not found")
		}

		return err
	}
	span.AddEvent("information", trace.WithAttributes(
		attribute.String("credential id", id.String()),
	))

	connections, err := h.connectionSvc.ListByCredential(ctx, id.String())
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())

		return err
	}

	metadata := make(map[string]any)
	err = json.Unmarshal(credential.Metadata, &metadata)
	if err != nil {
		return err
	}

	apiCredential := entity.NewCredential(*credential)
	if err != nil {
		return err
	}

	for _, conn := range connections {
		apiCredential.Connections = append(apiCredential.Connections, entity.NewConnection(conn))

		switch conn.LifecycleState {
		case model.ConnectionLifecycleStateDiscovered:
			apiCredential.DiscoveredConnections = utils.PAdd(apiCredential.DiscoveredConnections, fp.Optional[int64](1))
		case model.ConnectionLifecycleStateInProgress:
			fallthrough
		case model.ConnectionLifecycleStateOnboard:
			apiCredential.OnboardConnections = utils.PAdd(apiCredential.OnboardConnections, fp.Optional[int64](1))
		case model.ConnectionLifecycleStateDisabled:
			apiCredential.DisabledConnections = utils.PAdd(apiCredential.DisabledConnections, fp.Optional[int64](1))
		case model.ConnectionLifecycleStateArchived:
			apiCredential.ArchivedConnections = utils.PAdd(apiCredential.ArchivedConnections, fp.Optional[int64](1))
		}
		if conn.HealthState == source.HealthStatusUnhealthy {
			apiCredential.UnhealthyConnections = utils.PAdd(apiCredential.UnhealthyConnections, fp.Optional[int64](1))
		}

		apiCredential.TotalConnections = utils.PAdd(apiCredential.TotalConnections, fp.Optional[int64](1))
	}

	switch credential.ConnectorType {
	case source.CloudAzure:
		cnf, err := h.credentialSvc.AzureCredentialConfig(ctx, *credential)
		if err != nil {
			return err
		}
		apiCredential.Config = entity.AzureCredentialConfig{
			TenantId: cnf.TenantID,
			ObjectId: cnf.ObjectID,
			ClientId: cnf.ClientID,
		}
	case source.CloudAWS:
		cnf, err := h.credentialSvc.AWSCredentialConfig(ctx, *credential)
		if err != nil {
			return err
		}
		apiCredential.Config = entity.AWSCredentialConfig{
			AssumeRoleName: cnf.AssumeRoleName,
			ExternalId:     cnf.ExternalId,
		}
	}

	return c.JSON(http.StatusOK, apiCredential)
}

// UpdateAzure godoc
//
//	@Summary		Edit azure credential
//	@Description	Edit an azure credential by ID
//	@Security		BearerToken
//	@Tags			credentials
//	@Produce		json
//	@Success		200
//	@Param			credentialId	path	string								true	"Credential ID"
//	@Param			config			body	entity.UpdateAzureCredentialRequest	true	"config"
//	@Router			/integration/api/v1/credentials/azure/{credentialId} [put]
func (h API) UpdateAzure(c echo.Context) error {
	ctx := otel.GetTextMapPropagator().Extract(c.Request().Context(), propagation.HeaderCarrier(c.Request().Header))

	id, err := uuid.Parse(c.Param("credentialId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}

	var req entity.UpdateAzureCredentialRequest

	ctx, span := h.tracer.Start(ctx, "update-azure")
	defer span.End()

	if err := c.Bind(&req); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())

		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	if err := c.Validate(&req); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())

		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	if err := h.credentialSvc.AzureUpdate(ctx, id, req); err != nil {
		return err
	}

	return c.NoContent(http.StatusOK)
}

// UpdateAWS godoc
//
//	@Summary		Edit aws credential
//	@Description	Edit an aws credential by ID
//	@Security		BearerToken
//	@Tags			credentials
//	@Produce		json
//	@Success		200
//	@Param			credentialId	path	string								true	"Credential ID"
//	@Param			config			body	entity.UpdateAWSCredentialRequest	true	"config"
//	@Router			/integration/api/v1/credentials/aws/{credentialId} [put]
func (h API) UpdateAWS(c echo.Context) error {
	ctx := otel.GetTextMapPropagator().Extract(c.Request().Context(), propagation.HeaderCarrier(c.Request().Header))

	id, err := uuid.Parse(c.Param("credentialId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}

	var req entity.UpdateAWSCredentialRequest

	ctx, span := h.tracer.Start(ctx, "update-aws")
	defer span.End()

	if err := c.Bind(&req); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())

		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	if err := c.Validate(&req); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())

		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	if err := h.credentialSvc.AWSUpdate(ctx, id, req); err != nil {
		return err
	}

	return c.NoContent(http.StatusOK)
}

// List godoc
//
//	@Summary		List credentials
//	@Description	Retrieving list of credentials with their details
//	@Security		BearerToken
//	@Tags			credentials
//	@Produce		json
//	@Success		200				{object}	entity.ListCredentialResponse
//	@Param			connector		query		source.Type				false	"filter by connector type"
//	@Param			health			query		string					false	"filter by health status"	Enums(healthy, unhealthy)
//	@Param			credentialType	query		[]entity.CredentialType	false	"filter by credential type"
//	@Param			pageSize		query		int						false	"page size"		default(50)
//	@Param			pageNumber		query		int						false	"page number"	default(1)
//	@Router			/integration/api/v1/credentials [get]
func (h API) List(c echo.Context) error {
	ctx := otel.GetTextMapPropagator().Extract(c.Request().Context(), propagation.HeaderCarrier(c.Request().Header))

	connector, _ := source.ParseType(c.QueryParam("connector"))

	health, _ := source.ParseHealthStatus(c.QueryParam("health"))

	credentialTypes := model.ParseCredentialTypes(httpserver.QueryArrayParam(c, "credentialType"))
	if len(credentialTypes) == 0 {
		// take note if you want the change this,
		// the default is used in the frontend AND the checkup worker.
		credentialTypes = model.GetManualCredentialTypes()
	}

	pageSizeStr := c.QueryParam("pageSize")
	pageNumberStr := c.QueryParam("pageNumber")

	pageSize := int64(50)
	pageNumber := int64(1)
	if pageSizeStr != "" {
		pageSize, _ = strconv.ParseInt(pageSizeStr, 10, 64)
	}
	if pageNumberStr != "" {
		pageNumber, _ = strconv.ParseInt(pageNumberStr, 10, 64)
	}

	ctx, span := h.tracer.Start(ctx, "list", trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	credentials, err := h.credentialSvc.ListWithFilters(ctx, connector, health, credentialTypes)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())

		return err
	}

	apiCredentials := make([]entity.Credential, 0, len(credentials))

	for _, cred := range credentials {
		totalConnectionCount, err := h.connectionSvc.CountByCredential(ctx, cred.ID.String(), nil, nil)
		if err != nil {
			return err
		}

		unhealthyConnectionCount, err := h.connectionSvc.CountByCredential(ctx, cred.ID.String(), nil, []source.HealthStatus{source.HealthStatusUnhealthy})
		if err != nil {
			return err
		}

		onboardConnectionCount, err := h.connectionSvc.CountByCredential(ctx, cred.ID.String(),
			[]model.ConnectionLifecycleState{model.ConnectionLifecycleStateInProgress, model.ConnectionLifecycleStateOnboard}, nil)
		if err != nil {
			return err
		}

		discoveredConnectionCount, err := h.connectionSvc.CountByCredential(ctx, cred.ID.String(), []model.ConnectionLifecycleState{model.ConnectionLifecycleStateDiscovered}, nil)
		if err != nil {
			return err
		}

		disabledConnectionCount, err := h.connectionSvc.CountByCredential(ctx, cred.ID.String(), []model.ConnectionLifecycleState{model.ConnectionLifecycleStateDisabled}, nil)
		if err != nil {
			return err
		}

		archivedConnectionCount, err := h.connectionSvc.CountByCredential(ctx, cred.ID.String(), []model.ConnectionLifecycleState{model.ConnectionLifecycleStateArchived}, nil)
		if err != nil {
			return err
		}

		apiCredential := entity.NewCredential(cred)
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

	result := entity.ListCredentialResponse{
		TotalCredentialCount: len(apiCredentials),
		Credentials:          utils.Paginate(pageNumber, pageSize, apiCredentials),
	}

	return c.JSON(http.StatusOK, result)
}

// Delete godoc
//
//	@Summary		Delete credential
//	@Description	Remove a credential by ID
//	@Security		BearerToken
//	@Tags			credentials
//	@Produce		json
//	@Success		200
//	@Param			credentialId	path	string	true	"CredentialID"
//	@Router			/integration/api/v1/credential/{credentialId} [delete]
func (h API) Delete(c echo.Context) error {
	// on deleting a credential, we need to delete its accounts / subscription.

	ctx := otel.GetTextMapPropagator().Extract(c.Request().Context(), propagation.HeaderCarrier(c.Request().Header))

	credId, err := uuid.Parse(c.Param("credentialId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}

	ctx, span := h.tracer.Start(ctx, "delete", trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	credential, err := h.credentialSvc.Get(ctx, credId.String())
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())

		if errors.Is(err, repository.ErrCredentialNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "credential not found")
		}

		return err
	}

	span.AddEvent("information", trace.WithAttributes(
		attribute.String("credential id", credId.String()),
	))

	if err := h.credentialSvc.Delete(ctx, *credential); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())

		return err
	}

	return c.NoContent(http.StatusOK)
}

// CreateAzure godoc
//
//	@Summary		Create Azure credential and does onboarding for its subscriptions
//	@Description	Creating Azure credential, testing it and onboard its subscriptions
//	@Security		BearerToken
//	@Tags			integration
//	@Produce		json
//	@Success		200		{object}	entity.CreateCredentialResponse
//	@Param			request	body		entity.CreateAzureCredentialRequest	true	"Request"
//	@Router			/integration/api/v1/credentials/azure [post]
func (h API) CreateAzure(c echo.Context) error {
	ctx := otel.GetTextMapPropagator().Extract(c.Request().Context(), propagation.HeaderCarrier(c.Request().Header))

	ctx, span := h.tracer.Start(ctx, "create-azure-spn")
	defer span.End()

	var req entity.CreateAzureCredentialRequest

	if err := c.Bind(&req); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())

		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	if err := c.Validate(&req); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())

		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	var credType model.CredentialType
	switch req.Type {
	case entity.CredentialTypeAutoAzure:
		credType = model.CredentialTypeAutoAzure
	case entity.CredentialTypeManualAzureSpn:
		credType = model.CredentialTypeManualAzureSpn
	case entity.CredentialTypeManualAzureEntraId:
		credType = model.CredentialTypeManualAzureEntraId
	default:
		credType = model.CredentialTypeManualAzureSpn
	}

	cred, err := h.credentialSvc.NewAzure(
		ctx,
		credType,
		req.Config,
	)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())

		h.logger.Error("creating azure credential failed", zap.Error(err))

		return echo.ErrInternalServerError
	}

	if _, err := h.credentialSvc.AzureHealthCheck(ctx, cred); err != nil {
		return err
	}

	if err := h.credentialSvc.Create(ctx, cred); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())

		h.logger.Error("inserting newly created credential into the database", zap.Error(err))

		return echo.ErrInternalServerError
	}

	// An Azure subscription is a unit of management, billing, and provisioning within Microsoft Azure,
	// which is Microsoft's cloud computing platform.
	// call auto onboard so read current subscriptions of the given azure credentials gathered.
	connections, err := h.credentialSvc.AzureOnboard(ctx, *cred)
	if err != nil {
		h.logger.Error("azure onboarding failed", zap.Error(err))

		return echo.ErrInternalServerError
	}

	response := make([]entity.Connection, len(connections))

	for i, connection := range connections {
		// checking the connection health and update its metadata.
		h.connectionSvc.AzureHealth(ctx, connection, true)

		response[i] = entity.NewConnection(connection)
	}

	// newly created credential id an the list of its subscriptions.
	return c.JSON(http.StatusOK, entity.CreateCredentialResponse{
		Connections: response,
		ID:          cred.ID.String(),
	})
}

// CreateAWS godoc
//
//	@Summary		Create AWS credential and does onboarding for its accounts (organization account)
//	@Description	Creating AWS credential, testing it and onboard its accounts (organization account)
//	@Security		BearerToken
//	@Tags			credentials
//	@Produce		json
//	@Success		200		{object}	entity.CreateCredentialResponse
//	@Param			request	body		entity.CreateAWSCredentialRequest	true	"Request"
//	@Router			/integration/api/v1/credentials/aws [post]
func (h API) CreateAWS(c echo.Context) error {
	ctx := otel.GetTextMapPropagator().Extract(c.Request().Context(), propagation.HeaderCarrier(c.Request().Header))

	ctx, span := h.tracer.Start(ctx, "create-aws")
	defer span.End()

	var req entity.CreateAWSCredentialRequest

	if err := c.Bind(&req); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())

		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	if err := c.Validate(&req); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())

		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	if req.Config.AccountID == "" && req.Config.AccessKey != nil && req.Config.SecretKey != nil {
		awsCred, err := kaytuAws.GetConfig(ctx, *req.Config.AccessKey, *req.Config.SecretKey, "", "", nil)
		if err != nil {
			h.logger.Error("cannot read aws credentials", zap.Error(err))

			return echo.NewHTTPError(http.StatusBadRequest, "cannot read aws credentials")
		}
		stsClient := sts.NewFromConfig(awsCred)
		stsAccount, err := stsClient.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
		if err != nil {
			h.logger.Error("cannot read aws account", zap.Error(err))
			return echo.NewHTTPError(http.StatusBadRequest, "cannot call GetCallerIdentity to read aws account")
		}
		if stsAccount.Account == nil {
			h.logger.Error("cannot read aws account", zap.Error(err))
			return echo.NewHTTPError(http.StatusBadRequest, "GetCallerIdentity returned empty account id")
		}
		req.Config.AccountID = *stsAccount.Account
	}

	// Account id is passed as a pointer, so if it is empty, it'll be filled in the function.
	awsConfig, err := h.credentialSvc.AWSSDKConfig(ctx, req.Config.AssumeRoleName, &req.Config.AccountID, req.Config.AccessKey, req.Config.SecretKey, req.Config.ExternalId)
	if err != nil {
		h.logger.Error("reading aws sdk configuration failed", zap.Error(err))

		return err
	}

	org, accounts, err := h.credentialSvc.AWSOrgAccounts(ctx, awsConfig)
	if err != nil {
		h.logger.Error("getting aws accounts and organizations", zap.Error(err))

		return err
	}

	metadata, err := model.ExtractCredentialMetadata(req.Config.AccountID, org, accounts)
	if err != nil {
		return err
	}

	name := metadata.AccountID
	if metadata.OrganizationID != nil {
		name = *metadata.OrganizationID
	}

	cred, err := h.credentialSvc.NewAWS(ctx, name, metadata, model.CredentialTypeManualAwsOrganization, req.Config)
	if err != nil {
		h.logger.Error("building aws credential failed", zap.Error(err))

		return err
	}

	// we are going to check the credential health but not updating it in the database,
	// because it doesn't exists there yet.
	if _, err := h.credentialSvc.AWSHealthCheck(ctx, cred, false); err != nil {
		return err
	}

	// update credential health before writing it into the database.
	cred.HealthReason = fp.Optional("")
	cred.HealthStatus = source.HealthStatusHealthy

	if err := h.credentialSvc.Create(ctx, cred); err != nil {
		h.logger.Error("creating aws credential failed", zap.Error(err))

		return err
	}

	connections, err := h.credentialSvc.AWSOnboard(ctx, *cred)
	if err != nil {
		h.logger.Error("aws onboarding failed", zap.Error(err))

		return echo.ErrInternalServerError
	}

	response := make([]entity.Connection, len(connections))

	for i, connection := range connections {
		// checking the connection health and update its metadata.
		h.connectionSvc.AWSHealthCheck(ctx, connection, true)

		response[i] = entity.NewConnection(connection)
	}

	return c.JSON(http.StatusOK, entity.CreateCredentialResponse{
		Connections: response,
		ID:          cred.ID.String(),
	})
}

// AutoOnboardAWS godoc
//
//	@Summary		Onboard aws credential connections
//	@Description	Onboard all available connections for an aws credential
//	@Security		BearerToken
//	@Tags			credentials
//	@Produce		json
//	@Param			credentialId	path		string	true	"CredentialID"
//	@Success		200				{object}	[]entity.Connection
//	@Router			/integration/api/v1/credentials/aws/{credentialId}/autoonboard [post]
func (h API) AutoOnboardAWS(c echo.Context) error {
	ctx := otel.GetTextMapPropagator().Extract(c.Request().Context(), propagation.HeaderCarrier(c.Request().Header))

	ctx, span := h.tracer.Start(ctx, "auto-onboard-aws")
	defer span.End()

	credID, err := uuid.Parse(c.Param("credentialId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}

	credential, err := h.credentialSvc.Get(ctx, credID.String())
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())

		if errors.Is(err, repository.ErrCredentialNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "credential not found")
		}

		return err
	}

	span.AddEvent("information", trace.WithAttributes(
		attribute.String("credential id", credID.String()),
	))

	connections, err := h.credentialSvc.AWSOnboard(ctx, *credential)
	if err != nil {
		return err
	}

	response := make([]entity.Connection, len(connections))

	for i, connection := range connections {
		// checking the connection health and update its metadata.
		h.connectionSvc.AWSHealthCheck(ctx, connection, true)

		response[i] = entity.NewConnection(connection)
	}

	return c.JSON(http.StatusOK, response)
}

// AutoOnboardAzure godoc
//
//	@Summary		Onboard azure credential connections
//	@Description	Onboard all available connections for an azure credential
//	@Security		BearerToken
//	@Tags			credentials
//	@Produce		json
//	@Param			credentialId	path		string	true	"CredentialID"
//	@Success		200				{object}	[]entity.Connection
//	@Router			/integration/api/v1/credentials/azure/{credentialId}/autoonboard [post]
func (h API) AutoOnboardAzure(c echo.Context) error {
	ctx := otel.GetTextMapPropagator().Extract(c.Request().Context(), propagation.HeaderCarrier(c.Request().Header))

	ctx, span := h.tracer.Start(ctx, "auto-onboard-azure")
	defer span.End()

	credID, err := uuid.Parse(c.Param("credentialId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}

	credential, err := h.credentialSvc.Get(ctx, credID.String())
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())

		if errors.Is(err, repository.ErrCredentialNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "credential not found")
		}

		return err
	}

	span.AddEvent("information", trace.WithAttributes(
		attribute.String("credential id", credID.String()),
	))

	connections, err := h.credentialSvc.AzureOnboard(ctx, *credential)
	if err != nil {
		return err
	}

	response := make([]entity.Connection, len(connections))

	for i, connection := range connections {
		// checking the connection health and update its metadata.
		h.connectionSvc.AzureHealth(ctx, connection, true)

		response[i] = entity.NewConnection(connection)
	}

	return c.JSON(http.StatusOK, response)
}

func (s API) Register(g *echo.Group) {
	g.GET("", httpserver.AuthorizeHandler(s.List, api.ViewerRole))
	g.POST("/azure", httpserver.AuthorizeHandler(s.CreateAzure, api.EditorRole))
	g.POST("/aws", httpserver.AuthorizeHandler(s.CreateAWS, api.EditorRole))
	g.DELETE("/:credentialId", httpserver.AuthorizeHandler(s.Delete, api.EditorRole))
	g.GET("/:credentialId", httpserver.AuthorizeHandler(s.Get, api.ViewerRole))
	g.PUT("/aws/:credentialId", httpserver.AuthorizeHandler(s.UpdateAWS, api.EditorRole))
	g.PUT("/azure/:credentialId", httpserver.AuthorizeHandler(s.UpdateAzure, api.EditorRole))
	g.POST("/aws/:credentialId/autoonboard", httpserver.AuthorizeHandler(s.AutoOnboardAWS, api.EditorRole))
	g.POST("/azure/:credentialId/autoonboard", httpserver.AuthorizeHandler(s.AutoOnboardAzure, api.EditorRole))
}
