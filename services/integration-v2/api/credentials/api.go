package credentials

import (
	"github.com/labstack/echo/v4"
	"github.com/opengovern/og-util/pkg/api"
	"github.com/opengovern/og-util/pkg/httpserver"
	"github.com/opengovern/opengovernance/services/integration-v2/api/models"
	"github.com/opengovern/opengovernance/services/integration-v2/db"
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
		logger:   logger.Named("credentials"),
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
//	@Param			credentialId	path	string	true	"credentialId"
//	@Router			/integration/api/v1/credentials/{credentialId} [delete]
func (h API) Delete(c echo.Context) error {
	credentialId := c.Param("credentialId")

	err := h.database.DeleteCredential(credentialId)
	if err != nil {
		h.logger.Error("failed to delete credential", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to delete credential")
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
//	@Router			/integration/api/v1/credentials [get]
func (h API) List(c echo.Context) error {
	credentials, err := h.database.ListCredentials()
	if err != nil {
		h.logger.Error("failed to list credentials", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to list credential")
	}

	var items []models.Credential
	for _, credential := range credentials {
		item, err := credential.ToApi()
		if err != nil {
			h.logger.Error("failed to convert credentials to API model", zap.Error(err))
			return echo.NewHTTPError(http.StatusInternalServerError, "failed to convert credentials to API model")
		}
		items = append(items, *item)
	}

	return c.JSON(http.StatusOK, models.ListCredentialsResponse{
		Credentials: items,
		TotalCount:  len(items),
	})
}

// Get godoc
//
//	@Summary		Get credential
//	@Description	Get credential
//	@Security		BearerToken
//	@Tags			credentials
//	@Produce		json
//	@Success		200
//	@Param			credentialId	path	string	true	"credentialId"
//	@Router			/integration/api/v1/credentials/{credentialId} [get]
func (h API) Get(c echo.Context) error {
	credentialId := c.Param("credentialId")

	credential, err := h.database.GetCredential(credentialId)
	if err != nil {
		h.logger.Error("failed to get credential", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to get credential")
	}

	item, err := credential.ToApi()
	if err != nil {
		h.logger.Error("failed to convert credentials to API model", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to convert integration to API model")
	}
	return c.JSON(http.StatusOK, item)
}

func (h API) Register(g *echo.Group) {
	g.GET("", httpserver.AuthorizeHandler(h.List, api.ViewerRole))
	g.DELETE("/:credentialId", httpserver.AuthorizeHandler(h.Delete, api.EditorRole))
	g.GET("/:credentialId", httpserver.AuthorizeHandler(h.Get, api.ViewerRole))
}
