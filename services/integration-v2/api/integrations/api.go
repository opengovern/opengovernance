package integrations

import (
	"encoding/json"
	"github.com/google/uuid"
	"github.com/jackc/pgtype"
	"github.com/labstack/echo/v4"
	"github.com/opengovern/og-util/pkg/api"
	"github.com/opengovern/og-util/pkg/httpserver"
	"github.com/opengovern/og-util/pkg/vault"
	"github.com/opengovern/opengovernance/services/integration-v2/api/models"
	"github.com/opengovern/opengovernance/services/integration-v2/db"
	integration_type "github.com/opengovern/opengovernance/services/integration-v2/integration-type"
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
	database db.Database,
	logger *zap.Logger,
) API {
	return API{
		vault:    vault,
		database: database,
		logger:   logger.Named("integrations"),
	}
}

// DiscoverIntegrations godoc
//
//	@Summary		Discover integrations
//	@Description	Discover integrations and return back the list of integrations and credential ID
//	@Security		BearerToken
//	@Tags			integrations
//	@Produce		json
//	@Success		200
//	@Param			request	body		entity.CreateRequest	true	"Request"
//	@Router			/integration/api/v1/integrations/discover [post]
func (h API) DiscoverIntegrations(c echo.Context) error {
	var req models.DiscoverIntegrationRequest

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}

	jsonData, err := json.Marshal(req.Credentials)
	if err != nil {
		h.logger.Error("failed to marshal json data", zap.Error(err))
		return echo.NewHTTPError(http.StatusBadRequest, "failed to marshal json data")
	}

	var mapData map[string]any
	err = json.Unmarshal(jsonData, &mapData)
	if err != nil {
		h.logger.Error("failed to unmarshal json data", zap.Error(err))
	}

	if _, ok := integration_type.IntegrationTypes[req.IntegrationType]; !ok {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid integration type")
	}
	createCredentialFunction := integration_type.IntegrationTypes[req.IntegrationType]
	integration, err := createCredentialFunction()

	if integration == nil {
		return echo.NewHTTPError(http.StatusBadRequest, "failed to marshal json data")
	}

	healthy, err := integration.HealthCheck(req.CredentialType, jsonData)
	if err != nil || !healthy {
		h.logger.Error("healthcheck failed", zap.Error(err))
		return echo.NewHTTPError(http.StatusBadRequest, "healthcheck failed")
	}

	integrations, err := integration.DiscoverIntegrations(req.CredentialType, jsonData)

	secret, err := h.vault.Encrypt(c.Request().Context(), mapData)
	if err != nil {
		h.logger.Error("failed to encrypt secret", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to encrypt config")
	}

	credentialID := uuid.New()

	credentialMetadataJsonb := pgtype.JSONB{}
	err = credentialMetadataJsonb.Set([]byte(""))
	err = h.database.CreateCredential(&models2.Credential{
		ID:             credentialID,
		Secret:         secret,
		CredentialType: req.CredentialType,
		Metadata:       credentialMetadataJsonb,
	})
	if err != nil {
		h.logger.Error("failed to create credential", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create credential")
	}

	var integrationsAPI []models.Integration
	for _, i := range integrations {
		metadata, err := integration.GetMetadata(req.CredentialType, jsonData)
		if err != nil {
			h.logger.Error("failed to get metadata", zap.Error(err))
		}
		metadataJsonData, err := json.Marshal(metadata)
		if err != nil {
			return err
		}
		integrationMetadataJsonb := pgtype.JSONB{}
		err = integrationMetadataJsonb.Set(metadataJsonData)
		i.Metadata = integrationMetadataJsonb

		annotations, err := integration.GetAnnotations(req.CredentialType, jsonData)
		if err != nil {
			h.logger.Error("failed to get annotations", zap.Error(err))
		}
		annotationsJsonData, err := json.Marshal(annotations)
		if err != nil {
			return err
		}
		integrationAnnotationsJsonb := pgtype.JSONB{}
		err = integrationAnnotationsJsonb.Set(annotationsJsonData)
		i.Annotations = integrationAnnotationsJsonb

		labels, err := integration.GetLabels(req.CredentialType, jsonData)
		if err != nil {
			h.logger.Error("failed to get labels", zap.Error(err))
		}
		labelsJsonData, err := json.Marshal(labels)
		if err != nil {
			return err
		}
		integrationLabelsJsonb := pgtype.JSONB{}
		err = integrationLabelsJsonb.Set(labelsJsonData)
		i.Labels = integrationLabelsJsonb

		integrationAPI, err := i.ToApi()
		if err != nil {
			h.logger.Error("failed to create integration api", zap.Error(err))
			return echo.NewHTTPError(http.StatusInternalServerError, "failed to create integration api")
		}
		integrationsAPI = append(integrationsAPI, *integrationAPI)
	}

	return c.JSON(http.StatusOK, models.DiscoverIntegrationResponse{
		CredentialID: credentialID.String(),
		Integrations: integrationsAPI,
	})
}

// AddIntegrations godoc
//
//	@Summary		Add integrations
//	@Description	Add integrations by given credential ID and integration IDs
//	@Security		BearerToken
//	@Tags			integrations
//	@Produce		json
//	@Success		200
//	@Param			request	body		entity.CreateRequest	true	"Request"
//	@Router			/integration/api/v1/integrations/add [post]
func (h API) AddIntegrations(c echo.Context) error {
	var req models.AddIntegrationsRequest

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}

	credentialID, err := uuid.Parse(req.CredentialID)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid credential id")
	}
	credential, err := h.database.GetCredential(credentialID)
	if err != nil {
		h.logger.Error("failed to get credential", zap.Error(err))
		return echo.NewHTTPError(http.StatusNotFound, "credential not found")
	}

	mapData, err := h.vault.Decrypt(c.Request().Context(), credential.Secret)
	if err != nil {
		h.logger.Error("failed to encrypt secret", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to encrypt config")
	}

	if _, ok := integration_type.IntegrationTypes[req.IntegrationType]; !ok {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid integration type")
	}

	jsonData, err := json.Marshal(mapData)
	if err != nil {
		h.logger.Error("failed to marshal json data", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to marshal json data")
	}

	createCredentialFunction := integration_type.IntegrationTypes[req.IntegrationType]
	integration, err := createCredentialFunction()
	if integration == nil {
		return echo.NewHTTPError(http.StatusBadRequest, "failed to marshal json data")
	}

	healthy, err := integration.HealthCheck(req.CredentialType, jsonData)
	if err != nil || !healthy {
		h.logger.Error("healthcheck failed", zap.Error(err))
		return echo.NewHTTPError(http.StatusBadRequest, "healthcheck failed")
	}

	integrations, err := integration.DiscoverIntegrations(req.CredentialType, jsonData)
	if err != nil {
		h.logger.Error("failed to create credential", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create credential")
	}

	integrationIDs := make(map[string]bool)
	for _, i := range req.IntegrationIDs {
		integrationIDs[i] = true
	}

	for _, i := range integrations {
		if _, ok := integrationIDs[i.IntegrationID]; !ok {
			continue
		}

		i.CredentialID = credentialID

		metadata, err := integration.GetMetadata(req.CredentialType, jsonData)
		if err != nil {
			h.logger.Error("failed to get metadata", zap.Error(err))
		}
		metadataJsonData, err := json.Marshal(metadata)
		if err != nil {
			return err
		}
		integrationMetadataJsonb := pgtype.JSONB{}
		err = integrationMetadataJsonb.Set(metadataJsonData)
		i.Metadata = integrationMetadataJsonb

		annotations, err := integration.GetAnnotations(req.CredentialType, jsonData)
		if err != nil {
			h.logger.Error("failed to get annotations", zap.Error(err))
		}
		annotationsJsonData, err := json.Marshal(annotations)
		if err != nil {
			return err
		}
		integrationAnnotationsJsonb := pgtype.JSONB{}
		err = integrationAnnotationsJsonb.Set(annotationsJsonData)
		i.Annotations = integrationAnnotationsJsonb

		labels, err := integration.GetLabels(req.CredentialType, jsonData)
		if err != nil {
			h.logger.Error("failed to get labels", zap.Error(err))
		}
		labelsJsonData, err := json.Marshal(labels)
		if err != nil {
			return err
		}
		integrationLabelsJsonb := pgtype.JSONB{}
		err = integrationLabelsJsonb.Set(labelsJsonData)
		i.Labels = integrationLabelsJsonb

		err = h.database.CreateIntegration(&i)
		if err != nil {
			h.logger.Error("failed to create credential", zap.Error(err))
			return echo.NewHTTPError(http.StatusInternalServerError, "failed to create credential")
		}
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
//	@Param			integrationTracker	path	string	true	"integrationTracker"
//	@Router			/integration/api/v1/integrations/{integrationTracker} [delete]
func (h API) Delete(c echo.Context) error {
	integrationTracker, err := uuid.Parse(c.Param("integrationTracker"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}

	err = h.database.DeleteIntegration(integrationTracker)
	if err != nil {
		h.logger.Error("failed to delete credential", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to delete credential")
	}

	return c.NoContent(http.StatusOK)
}

// List godoc
//
//		@Summary		List integrations
//		@Description	List integrations
//		@Security		BearerToken
//		@Tags			credentials
//		@Produce		json
//	 	@Param			integration_type query []string false "integration type filter"
//		@Success		200				{object}	models.ListResponse
//		@Router			/integration/api/v1/integrations [get]
func (h API) List(c echo.Context) error {
	integrationTypesStr := httpserver.QueryArrayParam(c, "integration_type")

	var integrationTypes []integration_type.IntegrationType
	for _, i := range integrationTypesStr {
		integrationTypes = append(integrationTypes, integration_type.IntegrationType(i))
	}

	integrations, err := h.database.ListIntegration(integrationTypes)
	if err != nil {
		h.logger.Error("failed to list credentials", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to list credential")
	}

	var items []models.Integration
	for _, integration := range integrations {
		item, err := integration.ToApi()
		if err != nil {
			h.logger.Error("failed to convert integration to API model", zap.Error(err))
			return echo.NewHTTPError(http.StatusInternalServerError, "failed to convert integration to API model")
		}
		items = append(items, *item)
	}

	return c.JSON(http.StatusOK, models.ListIntegrationsResponse{
		Integrations: items,
		TotalCount:   len(items),
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
//	@Param			integrationTracker	path	string	true	"integrationTracker"
//	@Router			/integration/api/v1/integrations/{integrationTracker} [get]
func (h API) Get(c echo.Context) error {
	integrationTracker, err := uuid.Parse(c.Param("integrationTracker"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}

	integration, err := h.database.GetIntegration(integrationTracker)
	if err != nil {
		h.logger.Error("failed to get credential", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to get credential")
	}

	item, err := integration.ToApi()
	if err != nil {
		h.logger.Error("failed to convert integration to API model", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to convert integration to API model")
	}
	return c.JSON(http.StatusOK, item)
}

// Update godoc
//
//	@Summary		Get credential
//	@Description	Get credential
//	@Security		BearerToken
//	@Tags			credentials
//	@Produce		json
//	@Success		200
//	@Param			integrationId	path	string	true	"IntegrationID"
//	@Param			request	body		models.UpdateRequest	true	"Request"
//	@Router			/integration/api/v1/integrations/{integrationId} [post]
func (h API) Update(c echo.Context) error {
	integrationTracker, err := uuid.Parse(c.Param("integrationTracker"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}

	var req models.UpdateRequest

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}

	integration, err := h.database.GetIntegration(integrationTracker)
	if err != nil {
		h.logger.Error("failed to get credential", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to get credential")
	}

	credential, err := h.database.GetCredential(integration.CredentialID)
	if err != nil {
		h.logger.Error("failed to get credential", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to get credential")
	}

	credentials, err := h.vault.Decrypt(c.Request().Context(), credential.Secret)
	if err != nil {
		h.logger.Error("failed to decrypt secret", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to decrypt config")
	}

	for k, v := range req.Credentials {
		credentials[k] = v
	}

	secret, err := h.vault.Encrypt(c.Request().Context(), credentials)
	if err != nil {
		h.logger.Error("failed to encrypt secret", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to encrypt config")
	}

	err = h.database.UpdateCredential(integration.CredentialID, secret)
	if err != nil {
		h.logger.Error("failed to update credential", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to update credential")
	}

	return c.NoContent(http.StatusOK)
}

func (h API) Register(g *echo.Group) {
	g.GET("", httpserver.AuthorizeHandler(h.List, api.ViewerRole))
	g.POST("/discover", httpserver.AuthorizeHandler(h.DiscoverIntegrations, api.EditorRole))
	g.POST("/add", httpserver.AuthorizeHandler(h.AddIntegrations, api.EditorRole))
	g.DELETE("/:integrationTracker", httpserver.AuthorizeHandler(h.Delete, api.EditorRole))
	g.GET("/:integrationTracker", httpserver.AuthorizeHandler(h.Get, api.ViewerRole))
	g.POST("/:integrationTracker", httpserver.AuthorizeHandler(h.Update, api.EditorRole))
}
