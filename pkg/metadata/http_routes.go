package metadata

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	_ "gorm.io/gorm"
	"net/http"

	api3 "github.com/kaytu-io/kaytu-engine/pkg/auth/api"
	"github.com/kaytu-io/kaytu-engine/pkg/internal/httpserver"
	"github.com/kaytu-io/kaytu-engine/pkg/metadata/api"
	"github.com/kaytu-io/kaytu-engine/pkg/metadata/internal/src"
	"github.com/kaytu-io/kaytu-engine/pkg/metadata/models"
	"github.com/labstack/echo/v4"
)

func (h HttpHandler) Register(r *echo.Echo) {
	v1 := r.Group("/api/v1")

	v1.POST("/filter", httpserver.AuthorizeHandler(h.AddFilter, api3.ViewerRole))
	v1.GET("/filter", httpserver.AuthorizeHandler(h.GetFilters, api3.ViewerRole))
	v1.GET("/metadata/:key", httpserver.AuthorizeHandler(h.GetConfigMetadata, api3.ViewerRole))
	v1.POST("/metadata", httpserver.AuthorizeHandler(h.SetConfigMetadata, api3.AdminRole))
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

	metadata, err := src.GetConfigMetadata(h.db, h.redis, key)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
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

	err = src.SetConfigMetadata(h.db, h.redis, key, req.Value)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
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
