package credential

import (
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strconv"

	"github.com/google/uuid"
	"github.com/kaytu-io/kaytu-engine/pkg/auth/api"
	"github.com/kaytu-io/kaytu-engine/pkg/httpserver"
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

// UpdateAzure godoc
//
//	@Summary		Edit azure credential
//	@Description	Edit an azure credential by ID
//	@Security		BearerToken
//	@Tags			onboard
//	@Produce		json
//	@Success		200
//	@Param			credentialId	path	string						true	"Credential ID"
//	@Param			config			body	api.UpdateCredentialRequest	true	"config"
//	@Router			/integration/api/v1/credentials/azure/{credentialId} [put]
func (h API) UpdateAzure(c echo.Context) error {
	ctx := otel.GetTextMapPropagator().Extract(c.Request().Context(), propagation.HeaderCarrier(c.Request().Header))

	id, err := uuid.Parse(c.Param("credentialId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}

	var req entity.UpdateAzureCredentialRequest

	ctx, span := h.tracer.Start(ctx, "update-credential")
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
//	@Tags			onboard
//	@Produce		json
//	@Success		200
//	@Param			credentialId	path	string						true	"Credential ID"
//	@Param			config			body	api.UpdateCredentialRequest	true	"config"
//	@Router			/integration/api/v1/credentials/aws/{credentialId} [put]
func (h API) UpdateAWS(c echo.Context) error {
	ctx := otel.GetTextMapPropagator().Extract(c.Request().Context(), propagation.HeaderCarrier(c.Request().Header))

	id, err := uuid.Parse(c.Param("credentialId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}

	var req entity.UpdateAWSCredentialRequest

	ctx, span := h.tracer.Start(ctx, "update-credential")
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

	credentialTypes := model.ParseCredentialTypes(c.QueryParams()["credentialType"])
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
//	@Tags			onboard
//	@Produce		json
//	@Success		200
//	@Param			credentialId	path	string	true	"CredentialID"
//	@Router			/onboard/api/v1/credential/{credentialId} [delete]
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
		attribute.String("credential name", *credential.Name),
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

	cred, err := h.credentialSvc.NewAzure(
		ctx,
		model.CredentialTypeManualAzureSpn,
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
//	@Tags			integration
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

	awsConfig, err := h.credentialSvc.AWSSDKConfig(
		ctx,
		fmt.Sprintf("arn:aws:iam::%s:role/%s", req.Config.AccountID, req.Config.AssumeRoleName),
		req.Config.ExternalId,
	)
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
//	@Tags			onboard
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
		attribute.String("credential name", *credential.Name),
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
//	@Tags			onboard
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
		attribute.String("credential name", *credential.Name),
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
	g.POST("/azure", httpserver.AuthorizeHandler(s.CreateAzure, api.EditorRole))
	g.POST("/aws", httpserver.AuthorizeHandler(s.CreateAWS, api.EditorRole))
	g.DELETE("/:credentialId", httpserver.AuthorizeHandler(s.Delete, api.EditorRole))
	g.PUT("/aws/:credentialId", httpserver.AuthorizeHandler(s.UpdateAWS, api.EditorRole))
	g.PUT("/azure/:credentialId", httpserver.AuthorizeHandler(s.UpdateAzure, api.EditorRole))
	g.POST("aws/:credentialId/autoonboard", httpserver.AuthorizeHandler(s.AutoOnboardAWS, api.EditorRole))
	g.POST("azure/:credentialId/autoonboard", httpserver.AuthorizeHandler(s.AutoOnboardAzure, api.EditorRole))
}
