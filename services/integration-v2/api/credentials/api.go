package credentials

import (
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/opengovern/og-util/pkg/api"
	"github.com/opengovern/og-util/pkg/httpserver"
	"github.com/opengovern/og-util/pkg/vault"
	"github.com/opengovern/opengovernance/services/integration-v2/api/models"
	"github.com/opengovern/opengovernance/services/integration-v2/db"
	models2 "github.com/opengovern/opengovernance/services/integration-v2/models"
	"go.uber.org/zap"
	"net/http"
)

type API struct {
	vault    vault.VaultSourceConfig
	logger   *zap.Logger
	database db.Database
}

func New(
	vault vault.VaultSourceConfig,
	logger *zap.Logger,
) API {
	return API{
		vault:  vault,
		logger: logger.Named("credentials"),
	}
}

// Create godoc
//
//	@Summary		Create credential
//	@Description	Create credential
//	@Security		BearerToken
//	@Tags			credentials
//	@Produce		json
//	@Success		200
//	@Param			request	body		entity.CreateRequest	true	"Request"
//	@Router			/integration/api/v1/credentials [post]
func (h API) Create(c echo.Context) error {
	var req models.CreateRequest

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}

	secret, err := h.vault.Encrypt(c.Request().Context(), req.Config)
	if err != nil {
		h.logger.Error("failed to encrypt secret", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to encrypt config")
	}

	err = h.database.CreateCredential(&models2.Credential{
		ID:             uuid.New(),
		Secret:         secret,
		CredentialType: req.CredentialType,
	})
	if err != nil {
		h.logger.Error("failed to create credential", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create credential")
	}

	return c.NoContent(http.StatusOK)
}

// Delete godoc
//
//	@Summary		Delete credential
//	@Description	Delete credential
//	@Security		BearerToken
//	@Tags			credentials
//	@Produce		json
//	@Success		200
//	@Param			credentialId	path	string	true	"CredentialID"
//	@Router			/integration/api/v1/credentials/{credentialId} [delete]
func (h API) Delete(c echo.Context) error {
	credId, err := uuid.Parse(c.Param("credentialId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}

	err = h.database.DeleteCredential(credId)
	if err != nil {
		h.logger.Error("failed to delete credential", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to delete credential")
	}

	return c.NoContent(http.StatusOK)
}

// List godoc
//
//	@Summary		List credential
//	@Description	List credential
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

	var credentialItems []models.CredentialItem
	for _, cred := range credentials {
		credentialItems = append(credentialItems, models.CredentialItem{
			ID:             cred.ID.String(),
			CredentialType: cred.CredentialType,
		})
	}

	return c.JSON(http.StatusOK, models.ListResponse{
		Credentials: credentialItems,
		TotalCount:  len(credentials),
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
//	@Param			credentialId	path	string	true	"CredentialID"
//	@Router			/integration/api/v1/credentials/{credentialId} [get]
func (h API) Get(c echo.Context) error {
	credId, err := uuid.Parse(c.Param("credentialId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}

	credential, err := h.database.GetCredential(credId)
	if err != nil {
		h.logger.Error("failed to get credential", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to get credential")
	}

	return c.JSON(http.StatusOK, models.CredentialItem{
		ID:             credential.ID.String(),
		CredentialType: credential.CredentialType,
	})
}

// Update godoc
//
//	@Summary		Get credential
//	@Description	Get credential
//	@Security		BearerToken
//	@Tags			credentials
//	@Produce		json
//	@Success		200
//	@Param			credentialId	path	string	true	"CredentialID"
//	@Param			request	body		entity.CreateRequest	true	"Request"
//	@Router			/integration/api/v1/credentials/{credentialId} [post]
func (h API) Update(c echo.Context) error {
	credId, err := uuid.Parse(c.Param("credentialId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}

	var req models.CreateRequest

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}

	credential, err := h.database.GetCredential(credId)
	if err != nil {
		h.logger.Error("failed to get credential", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to get credential")
	}

	credentials, err := h.vault.Decrypt(c.Request().Context(), credential.Secret)
	if err != nil {
		h.logger.Error("failed to decrypt secret", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to decrypt config")
	}

	for k, v := range req.Config {
		credentials[k] = v
	}

	secret, err := h.vault.Encrypt(c.Request().Context(), credentials)
	if err != nil {
		h.logger.Error("failed to encrypt secret", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to encrypt config")
	}

	err = h.database.UpdateCredential(credId, secret)
	if err != nil {
		h.logger.Error("failed to update credential", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to update credential")
	}

	return c.JSON(http.StatusOK, models.CredentialItem{
		ID:             credential.ID.String(),
		CredentialType: credential.CredentialType,
	})
}

func (h API) Register(g *echo.Group) {
	g.GET("", httpserver.AuthorizeHandler(h.List, api.ViewerRole))
	g.POST("", httpserver.AuthorizeHandler(h.Create, api.EditorRole))
	g.DELETE("/:credentialId", httpserver.AuthorizeHandler(h.Delete, api.EditorRole))
	g.GET("/:credentialId", httpserver.AuthorizeHandler(h.Get, api.ViewerRole))
	g.POST("/:credentialId", httpserver.AuthorizeHandler(h.Update, api.EditorRole))
}
