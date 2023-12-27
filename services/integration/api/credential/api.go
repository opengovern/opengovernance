package credential

import (
	"net/http"

	"github.com/kaytu-io/kaytu-engine/pkg/auth/api"
	"github.com/kaytu-io/kaytu-engine/pkg/describe"
	"github.com/kaytu-io/kaytu-engine/pkg/httpserver"
	"github.com/kaytu-io/kaytu-engine/services/integration/api/entity"
	"github.com/kaytu-io/kaytu-engine/services/integration/model"
	"github.com/kaytu-io/kaytu-engine/services/integration/service"
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
//	@Summary			Create Azure credential and does onboarding for its subscriptions
//	@Description	Creating Azure credential, testing it and on-board its subscriptions
//	@Security			BearerToken
//	@Tags					integration
//	@Produce			json
//	@Success			200		{object}	entity.CreateCredentialResponse
//	@Param				request	body		entity.CreateAzureConnectionRequest	true	"Request"
//	@Router				/integration/api/v1/credentials/azure [post]
func (h API) CreateAzure(c echo.Context) error {
	ctx := otel.GetTextMapPropagator().Extract(c.Request().Context(), propagation.HeaderCarrier(c.Request().Header))

	ctx, span := h.tracer.Start(ctx, "create-azure-spn")
	defer span.End()

	var req entity.CreateAzureConnectionRequest

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

	if err := h.credentialSvc.Create(ctx, cred); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())

		h.logger.Error("inserting newly created credential into the database", zap.Error(err))

		return echo.ErrInternalServerError
	}

	// An Azure subscription is a unit of management, billing, and provisioning within Microsoft Azure,
	// which is Microsoft’s cloud computing platform.
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
//	@Summary			Create AWS credential and does onboarding for its subscriptions
//	@Description	Creating AWS credential, testing it and on-board its subscriptions
//	@Security			BearerToken
//	@Tags					integration
//	@Produce			json
//	@Success			200		{object}	entity.CreateCredentialResponse
//	@Param				request	body		entity.CreateAWSConnectionRequest	true	"Request"
//	@Router				/integration/api/v1/credentials/aws [post]
func (h API) CreateAWS(c echo.Context) error {
	ctx := otel.GetTextMapPropagator().Extract(c.Request().Context(), propagation.HeaderCarrier(c.Request().Header))

	ctx, span := h.tracer.Start(ctx, "create-aws")
	defer span.End()

	var req entity.CreateAWSConnectionRequest

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

	awsCnf, err := describe.AWSAccountConfigFromMap(req.Config.AsMap())
	if err != nil {
		h.logger.Error("cannot read aws config from describe", zap.Error(err))

		return echo.NewHTTPError(http.StatusBadRequest, "invalid config")
	}

	metadata, err := h.credentialSvc.AWSMetadata(ctx, awsCnf)
	if err != nil {
		h.logger.Error("cannot read aws credential metadata", zap.Error(err))

		return err
	}

	name := metadata.AccountID
	req.Config.AccountId = metadata.AccountID
	if metadata.OrganizationID != nil {
		name = *metadata.OrganizationID
	}

	cred, err := h.credentialSvc.NewAWS(ctx, name, metadata, model.CredentialTypeManualAwsOrganization, 1, req.Config)
	if err != nil {
		return err
	}

	h.credentialSvc.Create(ctx, cred)

	_, err = h.checkCredentialHealth(outputS, *cred)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, entity.CreateCredentialResponse{ID: cred.ID.String()})
}

func (s API) Register(g *echo.Group) {
	g.POST("/azure", httpserver.AuthorizeHandler(s.CreateAzure, api.EditorRole))
}
