package metadata

import (
	"gitlab.com/keibiengine/keibi-engine/pkg/internal/httpserver"
	"net/http"

	"github.com/labstack/echo/v4"
	api3 "gitlab.com/keibiengine/keibi-engine/pkg/auth/api"
	"gitlab.com/keibiengine/keibi-engine/pkg/metadata/api"
	"gitlab.com/keibiengine/keibi-engine/pkg/metadata/internal/src"
	"gitlab.com/keibiengine/keibi-engine/pkg/metadata/models"
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
//	@Summary		Returns the config metadata for the given key
//	@Description	Returns the config metadata for the given key
//	@Tags			metadata
//	@Produce		json
//	@Success		200	{object}	models.ConfigMetadata
//	@Router			/metadata/api/v1/metadata/{key} [get]
func (h HttpHandler) GetConfigMetadata(ctx echo.Context) error {
	key := ctx.Param("key")

	metadata, err := src.GetConfigMetadata(h.db, h.redis, key)
	if err != nil {
		return err
	}
	return ctx.JSON(http.StatusOK, metadata.GetCore())
}

// SetConfigMetadata godoc
//
//	@Summary		Sets the config metadata for the given key
//	@Description	Sets the config metadata for the given key
//	@Tags			metadata
//	@Produce		json
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
