package metadata

import (
	"errors"
	api3 "github.com/kaytu-io/kaytu-util/pkg/api"
	"github.com/kaytu-io/kaytu-util/pkg/httpserver"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"gorm.io/gorm"
	_ "gorm.io/gorm"
	"net/http"

	"github.com/kaytu-io/open-governance/pkg/metadata/api"
	"github.com/kaytu-io/open-governance/pkg/metadata/internal/src"
	"github.com/kaytu-io/open-governance/pkg/metadata/models"
	"github.com/labstack/echo/v4"
)

func (h HttpHandler) Register(r *echo.Echo) {
	v1 := r.Group("/api/v1")

	filter := v1.Group("/filter")
	filter.POST("", httpserver.AuthorizeHandler(h.AddFilter, api3.ViewerRole))
	filter.GET("", httpserver.AuthorizeHandler(h.GetFilters, api3.ViewerRole))

	metadata := v1.Group("/metadata")
	metadata.GET("/:key", httpserver.AuthorizeHandler(h.GetConfigMetadata, api3.ViewerRole))
	metadata.POST("", httpserver.AuthorizeHandler(h.SetConfigMetadata, api3.AdminRole))

	queryParameter := v1.Group("/query_parameter")
	queryParameter.POST("", httpserver.AuthorizeHandler(h.SetQueryParameter, api3.AdminRole))
	queryParameter.GET("", httpserver.AuthorizeHandler(h.ListQueryParameters, api3.ViewerRole))
}

var tracer = otel.Tracer("metadata")

func bindValidate(ctx echo.Context, i interface{}) error {
	if err := ctx.Bind(i); err != nil {
		return err
	}

	if err := ctx.Validate(i); err != nil {
		return err
	}

	return nil
}

// GetConfigMetadata godoc
//
//	@Summary		Get key metadata
//	@Description	Returns the config metadata for the given key
//	@Security		BearerToken
//	@Tags			metadata
//	@Produce		json
//	@Param			key	path		string	true	"Key"
//	@Success		200	{object}	models.ConfigMetadata
//	@Router			/metadata/api/v1/metadata/{key} [get]
func (h HttpHandler) GetConfigMetadata(ctx echo.Context) error {
	key := ctx.Param("key")
	_, span := tracer.Start(ctx.Request().Context(), "new_GetConfigMetadata", trace.WithSpanKind(trace.SpanKindServer))
	span.SetName("new_GetConfigMetadata")

	metadata, err := src.GetConfigMetadata(h.db, key)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())

		if errors.Is(err, gorm.ErrRecordNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "config not found")
		}
		return err
	}
	span.AddEvent("information", trace.WithAttributes(
		attribute.String("key", key),
	))
	span.End()
	return ctx.JSON(http.StatusOK, metadata.GetCore())
}

// SetConfigMetadata godoc
//
//	@Summary		Set key metadata
//	@Description	Sets the config metadata for the given key
//	@Security		BearerToken
//	@Tags			metadata
//	@Produce		json
//	@Param			req	body	api.SetConfigMetadataRequest	true	"Request Body"
//	@Success		200
//	@Router			/metadata/api/v1/metadata [post]
func (h HttpHandler) SetConfigMetadata(ctx echo.Context) error {
	var req api.SetConfigMetadataRequest
	if err := bindValidate(ctx, &req); err != nil {
		return err
	}

	key, err := models.ParseMetadataKey(req.Key)
	if err != nil {
		return err
	}

	err = httpserver.RequireMinRole(ctx, key.GetMinAuthRole())
	if err != nil {
		return err
	}
	_, span := tracer.Start(ctx.Request().Context(), "new_SetConfigMetadata", trace.WithSpanKind(trace.SpanKindServer))
	span.SetName("new_SetConfigMetadata")

	err = src.SetConfigMetadata(h.db, key, req.Value)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	span.AddEvent("information", trace.WithAttributes(
		attribute.String("key", key.String()),
	))
	span.End()

	return ctx.JSON(http.StatusOK, nil)
}

// AddFilter godoc
//
//	@Summary	add filter
//	@Security	BearerToken
//	@Tags		metadata
//	@Produce	json
//	@Param		req	body	models.Filter	true	"Request Body"
//	@Success	200
//	@Router		/metadata/api/v1/filter [post]
func (h HttpHandler) AddFilter(ctx echo.Context) error {
	var req models.Filter
	if err := bindValidate(ctx, &req); err != nil {
		return err
	}
	// trace :
	_, span := tracer.Start(ctx.Request().Context(), "new_AddFilter", trace.WithSpanKind(trace.SpanKindServer))
	span.SetName("new_AddFilter")

	err := h.db.AddFilter(models.Filter{Name: req.Name, KeyValue: req.KeyValue})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	span.AddEvent("information", trace.WithAttributes(
		attribute.String("name", req.Name),
	))
	span.End()
	return ctx.JSON(http.StatusOK, nil)
}

// GetFilters godoc
//
//	@Summary	list filters
//	@Security	BearerToken
//	@Tags		metadata
//	@Produce	json
//	@Success	200	{object}	[]models.Filter
//	@Router		/metadata/api/v1/filter [get]
func (h HttpHandler) GetFilters(ctx echo.Context) error {
	// trace :
	_, span := tracer.Start(ctx.Request().Context(), "new_ListFilters", trace.WithSpanKind(trace.SpanKindServer))
	span.SetName("new_ListFilters")

	filters, err := h.db.ListFilters()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil
	}
	span.End()
	return ctx.JSON(http.StatusOK, filters)
}

// SetQueryParameter godoc
//
//	@Summary		Set query parameter
//	@Description	Sets the query parameters from the request body
//	@Security		BearerToken
//	@Tags			metadata
//	@Produce		json
//	@Param			req	body	api.SetQueryParameterRequest	true	"Request Body"
//	@Success		200
//	@Router			/metadata/api/v1/query_parameter [post]
func (h HttpHandler) SetQueryParameter(ctx echo.Context) error {
	var req api.SetQueryParameterRequest
	if err := bindValidate(ctx, &req); err != nil {
		return err
	}

	if len(req.QueryParameters) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "no query parameters provided")
	}

	dbQueryParams := make([]*models.QueryParameter, 0, len(req.QueryParameters))
	for _, apiParam := range req.QueryParameters {
		//key, err := models.ParseQueryParameterKey(apiParam.Key)
		//if err != nil {
		//	return err
		//}
		dbParam := models.QueryParameterFromAPI(apiParam)
		dbParam.Key = apiParam.Key
		dbQueryParams = append(dbQueryParams, &dbParam)
	}

	_, span := tracer.Start(ctx.Request().Context(), "new_SetQueryParameter", trace.WithSpanKind(trace.SpanKindServer))
	span.SetName("new_SetQueryParameter")
	err := h.db.SetQueryParameters(dbQueryParams)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		h.logger.Error("error setting query parameters", zap.Error(err))
		return err
	}
	span.End()

	return ctx.JSON(http.StatusOK, nil)
}

// ListQueryParameters godoc
//
//	@Summary		List query parameters
//	@Description	Returns the list of query parameters
//	@Security		BearerToken
//	@Tags			metadata
//	@Produce		json
//	@Success		200	{object}	api.ListQueryParametersResponse
//	@Router			/metadata/api/v1/query_parameter [get]
func (h HttpHandler) ListQueryParameters(ctx echo.Context) error {
	_, span := tracer.Start(ctx.Request().Context(), "new_ListQueryParameters", trace.WithSpanKind(trace.SpanKindServer))
	span.SetName("new_ListQueryParameters")

	queryParams, err := h.db.GetQueryParameters()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		h.logger.Error("error getting query parameters", zap.Error(err))
		return err
	}
	span.End()

	result := api.ListQueryParametersResponse{
		QueryParameters: make([]api.QueryParameter, 0, len(queryParams)),
	}
	for _, dbParam := range queryParams {
		apiParam := dbParam.ToAPI()
		result.QueryParameters = append(result.QueryParameters, apiParam)
	}

	return ctx.JSON(http.StatusOK, result)
}
