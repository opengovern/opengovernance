package integration_types

import (
	"github.com/labstack/echo/v4"
	"github.com/opengovern/og-util/pkg/api"
	"github.com/opengovern/og-util/pkg/httpserver"
	"github.com/opengovern/opengovernance/services/integration/api/models"
	"github.com/opengovern/opengovernance/services/integration/db"
	"go.uber.org/zap"
	"net/http"
)

type API struct {
	logger   *zap.Logger
	database db.Database
}

func New(
	database db.Database,
	logger *zap.Logger,
) API {
	return API{
		database: database,
		logger:   logger.Named("integration_types"),
	}
}

// Delete godoc
//
//	@Summary		Delete credential
//	@Description	Delete credential
//	@Security		BearerToken
//	@Tags			credentials
//	@Produce		json
//	@Success		200
//	@Param			integrationTypeId	path	string	true	"integrationTypeId"
//	@Router			/integration/api/v1/integration-types/{integrationTypeId} [delete]
func (h API) Delete(c echo.Context) error {
	integrationTypeId := c.Param("integrationTypeId")

	err := h.database.DeleteIntegrationType(integrationTypeId)
	if err != nil {
		h.logger.Error("failed to delete integration type", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to delete integration type")
	}

	return c.NoContent(http.StatusOK)
}

// List godoc
//
//	@Summary		List credentials
//	@Description	List credentials
//	@Security		BearerToken
//	@Tags			credentials
//	@Produce		json
//	@Success		200				{object}	models.ListResponse
//	@Router			/integration/api/v1/integration-types [get]
func (h API) List(c echo.Context) error {
	integrationTypes, err := h.database.ListIntegrationTypes()
	if err != nil {
		h.logger.Error("failed to list integration types", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to list integration types")
	}

	var items []models.IntegrationType
	for _, integrationType := range integrationTypes {
		item, err := integrationType.ToApi()
		if err != nil {
			h.logger.Error("failed to convert integration types to API model", zap.Error(err))
			return echo.NewHTTPError(http.StatusInternalServerError, "failed to convert integration types to API model")
		}
		items = append(items, *item)
	}

	return c.JSON(http.StatusOK, models.ListIntegrationTypesResponse{
		IntegrationTypes: items,
		TotalCount:       len(items),
	})
}

// Get godoc
//
//	@Summary		Get integration type
//	@Description	Get integration type
//	@Security		BearerToken
//	@Tags			credentials
//	@Produce		json
//	@Success		200
//	@Param			integrationTypeId	path	string	true	"integrationTypeId"
//	@Router			/integration/api/v1/integration-types/{integrationTypeId} [get]
func (h API) Get(c echo.Context) error {
	integrationTypeId := c.Param("integrationTypeId")

	integrationType, err := h.database.GetIntegrationType(integrationTypeId)
	if err != nil {
		h.logger.Error("failed to get credential", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to get credential")
	}

	item, err := integrationType.ToApi()
	if err != nil {
		h.logger.Error("failed to convert credentials to API model", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to convert integration to API model")
	}
	return c.JSON(http.StatusOK, item)
}

func (h API) Register(g *echo.Group) {
	g.GET("", httpserver.AuthorizeHandler(h.List, api.ViewerRole))
	g.DELETE("/:integrationTypeId", httpserver.AuthorizeHandler(h.Delete, api.EditorRole))
	g.GET("/:integrationTypeId", httpserver.AuthorizeHandler(h.Get, api.ViewerRole))
}
