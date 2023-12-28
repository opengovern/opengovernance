package credential

import (
	"fmt"
	"net/http"

	"github.com/kaytu-io/kaytu-engine/pkg/auth/api"
	"github.com/kaytu-io/kaytu-engine/pkg/httpserver"
	"github.com/kaytu-io/kaytu-engine/services/integration/api/entity"
	"github.com/kaytu-io/kaytu-engine/services/integration/model"
	"github.com/kaytu-io/kaytu-engine/services/integration/service"
	"github.com/kaytu-io/kaytu-util/pkg/fp"
	"github.com/kaytu-io/kaytu-util/pkg/source"
	"github.com/labstack/echo/v4"
	"go.opentelemetry.io/otel"
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

func (s API) Register(g *echo.Group) {
	g.POST("/azure", httpserver.AuthorizeHandler(s.CreateAzure, api.EditorRole))
	g.POST("/aws", httpserver.AuthorizeHandler(s.CreateAWS, api.EditorRole))
}
