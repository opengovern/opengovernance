package metadata

import (
	"net/http"

	"github.com/kaytu-io/kaytu-engine/pkg/internal/httpserver"

	api3 "github.com/kaytu-io/kaytu-engine/pkg/auth/api"
	"github.com/kaytu-io/kaytu-engine/pkg/metadata/api"
	"github.com/kaytu-io/kaytu-engine/pkg/metadata/internal/src"
	"github.com/kaytu-io/kaytu-engine/pkg/metadata/models"
	"github.com/labstack/echo/v4"
)

func (h HttpHandler) Register(r *echo.Echo) {
	v1 := r.Group("/api/v1")

	v1.GET("/metadata/:key", httpserver.AuthorizeHandler(h.GetConfigMetadata, api3.ViewerRole))
	v1.POST("/metadata", httpserver.AuthorizeHandler(h.SetConfigMetadata, api3.AdminRole))
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

	// trace :
	//span := jaegertracing.CreateChildSpan(ctx, "GetConfigMetadata")
	//span.SetBaggageItem("metadata", "GetConfigMetadata")

	metadata, err := src.GetConfigMetadata(h.db, h.redis, key)
	//span.Finish()
	if err != nil {
		return err
	}
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

	err = src.SetConfigMetadata(h.db, h.redis, key, req.Value)
	if err != nil {
		return err
	}

	return ctx.JSON(http.StatusOK, nil)
}
