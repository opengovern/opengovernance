package onboard

import (
	_ "embed"
	"errors"
	api3 "github.com/opengovern/og-util/pkg/api"
	"github.com/opengovern/og-util/pkg/httpserver"
	"github.com/opengovern/opengovernance/pkg/onboard/api/entities"
	apiv2 "github.com/opengovern/opengovernance/pkg/onboard/api/v2"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/opengovern/og-util/pkg/source"
	"github.com/opengovern/opengovernance/pkg/utils"

	"github.com/opengovern/opengovernance/pkg/onboard/api"
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
	sourceApiGroup.GET("/:sourceId/healthcheck", httpserver.AuthorizeHandler(h.GetConnectionHealth, api3.EditorRole))
	sourceApiGroup.GET("/:sourceId/credentials/full", httpserver.AuthorizeHandler(h.GetSourceFullCred, api3.AdminRole))
	sourceApiGroup.DELETE("/:sourceId", httpserver.AuthorizeHandler(h.DeleteSource, api3.EditorRole))

	credential := v1.Group("/credential")
	credential.PUT("/:credentialId", httpserver.AuthorizeHandler(h.PutCredentials, api3.EditorRole))
	credential.GET("", httpserver.AuthorizeHandler(h.ListCredentials, api3.ViewerRole))
	credential.DELETE("/:credentialId", httpserver.AuthorizeHandler(h.DeleteCredential, api3.EditorRole))
	credential.GET("/:credentialId", httpserver.AuthorizeHandler(h.GetCredential, api3.ViewerRole))

	credentialV2 := v2.Group("/credential")
	credentialV2.POST("", httpserver.AuthorizeHandler(h.CreateCredential, api3.EditorRole))

	connections := v1.Group("/connections")
	connections.GET("/summary", httpserver.AuthorizeHandler(h.ListConnectionsSummaries, api3.ViewerRole))
	connections.POST("/:connectionId/state", httpserver.AuthorizeHandler(h.ChangeConnectionLifecycleState, api3.EditorRole))

	v3 := r.Group("/api/v3")
	v3.GET("/connector", httpserver.AuthorizeHandler(h.ListConnectorsV2, api3.ViewerRole))
	v3.PUT("/sample/purge", httpserver.AuthorizeHandler(h.PurgeSampleData, api3.AdminRole))
	v3.GET("/integrations", httpserver.AuthorizeHandler(h.ListIntegrations, api3.ViewerRole))
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
	var res []api.ConnectorCount

	return ctx.JSON(http.StatusOK, res)
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

	apiCredentials := make([]api.Credential, 0, 0)

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
	apiCredential := entities.NewCredential(nil)

	return ctx.JSON(http.StatusOK, apiCredential)
}

func (h HttpHandler) putAzureCredentials(ctx echo.Context, req api.UpdateCredentialRequest) error {

	return ctx.JSON(http.StatusOK, struct{}{})
}

func (h HttpHandler) putAWSCredentials(ctx echo.Context, req api.UpdateCredentialRequest) error {

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

	return ctx.JSON(http.StatusOK, struct{}{})
}

func (h HttpHandler) GetSourceFullCred(ctx echo.Context) error {

	return errors.New("invalid provider")

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

	return ctx.JSON(http.StatusOK, entities.NewConnection())
}

func (h HttpHandler) GetSource(ctx echo.Context) error {

	apiRes := entities.NewConnection()

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

	return ctx.NoContent(http.StatusOK)
}

func (h HttpHandler) ListSources(ctx echo.Context) error {

	resp := api.GetSourcesResponse{}

	return ctx.JSON(http.StatusOK, resp)
}

func (h HttpHandler) GetSources(ctx echo.Context) error {
	var res []api.Connection

	return ctx.JSON(http.StatusOK, res)
}

func (h HttpHandler) CountSources(ctx echo.Context) error {
	var count int64

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
	result := api.ListConnectionSummaryResponse{
		ConnectionCount:       0,
		TotalCost:             0,
		TotalResourceCount:    0,
		TotalOldResourceCount: 0,
		TotalUnhealthyCount:   0,

		TotalDisabledCount:   0,
		TotalDiscoveredCount: 0,
		TotalOnboardedCount:  0,
		TotalArchivedCount:   0,
		Connections:          make([]api.Connection, 0, 0),
	}

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
//	@Router			/onboard/api/v3/sample/purge [put]
func (s HttpHandler) PurgeSampleData(c echo.Context) error {
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
	return ctx.JSON(http.StatusOK, api.ListConnectorsV2Response{})
}

// ListIntegrations godoc
//
//	@Summary		List Integrations
//	@Description	List Integrations with filters
//	@Security		BearerToken
//	@Tags			onboard
//	@Produce		json
//	@Param			health_state	query		string		false	"health state"
//	@Param			connectors		query		[]string	false	"connectors"
//	@Param			integration_id	query		[]string	false	"integration tracker"
//	@Param			name_regex		query		string		false	"name regex"
//	@Param			id_regex		query		string		false	"id regex"
//	@Param			per_page		query		int			false	"PerPage"
//	@Param			cursor			query		int			false	"Cursor"
//	@Success		200				{object}	[]api.ConnectorCount
//	@Router			/onboard/api/v3/integrations [get]
func (h HttpHandler) ListIntegrations(ctx echo.Context) error {
	return ctx.JSON(http.StatusOK, api.ListIntegrationsResponse{})
}
