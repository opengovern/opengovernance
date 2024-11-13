package integrations

import (
	"encoding/json"
	"github.com/google/uuid"
	"github.com/jackc/pgtype"
	"github.com/labstack/echo/v4"
	"github.com/opengovern/og-util/pkg/api"
	"github.com/opengovern/og-util/pkg/httpserver"
	"github.com/opengovern/og-util/pkg/integration"
	"github.com/opengovern/og-util/pkg/steampipe"
	"github.com/opengovern/og-util/pkg/vault"
	"github.com/opengovern/opengovernance/pkg/utils"
	"github.com/opengovern/opengovernance/services/integration/api/models"
	"github.com/opengovern/opengovernance/services/integration/db"
	"github.com/opengovern/opengovernance/services/integration/entities"
	integration_type "github.com/opengovern/opengovernance/services/integration/integration-type"
	models2 "github.com/opengovern/opengovernance/services/integration/models"
	"go.uber.org/zap"
	ioutil "io/ioutil"
	"net/http"
	"os"
	"sort"
	strconv "strconv"
	strings "strings"
	"time"
)

type API struct {
	vault         vault.VaultSourceConfig
	logger        *zap.Logger
	database      db.Database
	steampipeConn *steampipe.Database
}

func New(
	vault vault.VaultSourceConfig,
	database db.Database,
	logger *zap.Logger,
	steampipeConn *steampipe.Database,
) API {
	return API{
		vault:         vault,
		database:      database,
		logger:        logger.Named("integrations"),
		steampipeConn: steampipeConn,
	}
}

func (h API) Register(g *echo.Group) {
	g.GET("", httpserver.AuthorizeHandler(h.List, api.ViewerRole))
	g.POST("/list", httpserver.AuthorizeHandler(h.ListByFilters, api.ViewerRole))
	g.POST("/discover", httpserver.AuthorizeHandler(h.DiscoverIntegrations, api.EditorRole))
	g.POST("/add", httpserver.AuthorizeHandler(h.AddIntegrations, api.EditorRole))
	g.PUT("/:IntegrationID/healthcheck", httpserver.AuthorizeHandler(h.IntegrationHealthcheck, api.EditorRole))
	g.DELETE("/:IntegrationID", httpserver.AuthorizeHandler(h.Delete, api.EditorRole))
	g.GET("/:IntegrationID", httpserver.AuthorizeHandler(h.Get, api.ViewerRole))
	g.POST("/:IntegrationID", httpserver.AuthorizeHandler(h.Update, api.EditorRole))
	g.GET("/integration-groups", httpserver.AuthorizeHandler(h.ListIntegrationGroups, api.ViewerRole))
	g.GET("/integration-groups/:integrationGroupName", httpserver.AuthorizeHandler(h.GetIntegrationGroup, api.ViewerRole))

	types := g.Group("/types")
	types.GET("", httpserver.AuthorizeHandler(h.ListIntegrationTypes, api.ViewerRole))
	types.GET("/:integrationTypeId", httpserver.AuthorizeHandler(h.GetIntegrationType, api.ViewerRole))
	types.GET("/:integrationTypeId/ui/spec", httpserver.AuthorizeHandler(h.GetIntegrationTypeUiSpec, api.ViewerRole))
	types.DELETE("/:integrationTypeId", httpserver.AuthorizeHandler(h.DeleteIntegrationType, api.EditorRole))
}

// DiscoverIntegrations godoc
//
//	@Summary		Discover integrations
//	@Description	Discover integrations and return back the list of integrations and credential ID
//	@Security		BearerToken
//	@Tags			integrations
//	@Produce		json
//	@Success		200
//	@Param			request	body	models.DiscoverIntegrationRequest	true	"Request"
//	@Router			/integration/api/v1/integrations/discover [post]
func (h API) DiscoverIntegrations(c echo.Context) error {
	var req models.DiscoverIntegrationRequest

	contentType := c.Request().Header.Get("Content-Type")
	if strings.HasPrefix(contentType, "multipart/form-data") {
		h.logger.Info("file imported")
		err := c.Request().ParseMultipartForm(10 << 20) // 10 MB max memory
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Failed to parse multipart form")
		}

		formData := make(map[string]any)

		for key, values := range c.Request().MultipartForm.Value {
			if len(values) > 0 {
				if key == "integrationType" || key == "integration_type" {
					req.IntegrationType = integration_type.ParseType(values[0])
				} else {
					keys := strings.Split(key, ".")
					formData[keys[1]] = values[0]
				}
			}
		}

		for key, fileHeaders := range c.Request().MultipartForm.File {
			if len(fileHeaders) > 0 {
				file, err := fileHeaders[0].Open()
				if err != nil {
					return echo.NewHTTPError(http.StatusInternalServerError, "Failed to open uploaded file")
				}
				defer file.Close()

				content, err := ioutil.ReadAll(file)
				if err != nil {
					return echo.NewHTTPError(http.StatusInternalServerError, "Failed to read uploaded file")
				}
				keys := strings.Split(key, ".")
				formData[keys[1]] = string(content)
			}
		}
		req.Credentials = formData
	} else {
		if err := c.Bind(&req); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
		}
	}

	var jsonData []byte
	var err error
	var integrationType integration.Type
	var credentialIDStr string

	if req.CredentialID != nil {
		credentialIDStr = *req.CredentialID
		credential, err := h.database.GetCredential(*req.CredentialID)
		if err != nil {
			h.logger.Error("failed to get credential", zap.Error(err))
			return echo.NewHTTPError(http.StatusNotFound, "credential not found")
		}
		integrationType = credential.IntegrationType

		mapData, err := h.vault.Decrypt(c.Request().Context(), credential.Secret)
		if err != nil {
			h.logger.Error("failed to decrypt secret", zap.Error(err))
			return echo.NewHTTPError(http.StatusInternalServerError, "failed to decrypt config")
		}

		if _, ok := integration_type.IntegrationTypes[req.IntegrationType]; !ok {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid integration type")
		}

		jsonData, err = json.Marshal(mapData)
		if err != nil {
			h.logger.Error("failed to marshal json data", zap.Error(err))
			return echo.NewHTTPError(http.StatusInternalServerError, "failed to marshal json data")
		}
	} else {
		integrationType = req.IntegrationType
		jsonData, err = json.Marshal(req.Credentials)
		if err != nil {
			h.logger.Error("failed to marshal json data", zap.Error(err))
			return echo.NewHTTPError(http.StatusBadRequest, "failed to marshal json data")
		}
		var mapData map[string]any
		err = json.Unmarshal(jsonData, &mapData)
		if err != nil {
			h.logger.Error("failed to unmarshal json data", zap.Error(err))
		}
		secret, err := h.vault.Encrypt(c.Request().Context(), mapData)
		if err != nil {
			h.logger.Error("failed to encrypt secret", zap.Error(err))
			return echo.NewHTTPError(http.StatusInternalServerError, "failed to encrypt config")
		}

		credentialID := uuid.New()

		metadata := make(map[string]string)
		metadataJsonData, err := json.Marshal(metadata)
		credentialMetadataJsonb := pgtype.JSONB{}
		err = credentialMetadataJsonb.Set(metadataJsonData)
		err = h.database.CreateCredential(&models2.Credential{
			ID:              credentialID,
			IntegrationType: req.IntegrationType,
			Secret:          secret,
			Metadata:        credentialMetadataJsonb,
		})
		if err != nil {
			h.logger.Error("failed to create credential", zap.Error(err))
			return echo.NewHTTPError(http.StatusInternalServerError, "failed to create credential")
		}
		credentialIDStr = credentialID.String()
	}

	integration, ok := integration_type.IntegrationTypes[integrationType]
	if !ok {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid integrationType")
	}

	if integration == nil {
		return echo.NewHTTPError(http.StatusBadRequest, "failed to marshal json data")
	}

	integrations, err := integration.DiscoverIntegrations(jsonData)

	var integrationsAPI []models.Integration
	for _, i := range integrations {
		integrationAPI, err := i.ToApi()
		if err != nil {
			h.logger.Error("failed to create integration api", zap.Error(err))
			return echo.NewHTTPError(http.StatusInternalServerError, "failed to create integration api")
		}

		healthy, err := integration.HealthCheck(jsonData, integrationAPI.ProviderID, integrationAPI.Labels, integrationAPI.Annotations)
		if err != nil || !healthy {
			h.logger.Info("integration is not healthy", zap.String("integration_id", i.IntegrationID.String()), zap.Error(err))
			integrationAPI.State = models.IntegrationStateInactive
		} else {
			integrationAPI.State = models.IntegrationStateActive
		}

		integrationsAPI = append(integrationsAPI, *integrationAPI)
	}

	return c.JSON(http.StatusOK, models.DiscoverIntegrationResponse{
		CredentialID: credentialIDStr,
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
//	@Param			request	body	models.AddIntegrationsRequest	true	"Request"
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
	credential, err := h.database.GetCredential(req.CredentialID)
	if err != nil {
		h.logger.Error("failed to get credential", zap.Error(err))
		return echo.NewHTTPError(http.StatusNotFound, "credential not found")
	}

	mapData, err := h.vault.Decrypt(c.Request().Context(), credential.Secret)
	if err != nil {
		h.logger.Error("failed to decrypt secret", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to decrypt config")
	}

	if _, ok := integration_type.IntegrationTypes[req.IntegrationType]; !ok {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid integration type")
	}

	jsonData, err := json.Marshal(mapData)
	if err != nil {
		h.logger.Error("failed to marshal json data", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to marshal json data")
	}

	integration := integration_type.IntegrationTypes[req.IntegrationType]
	if integration == nil {
		return echo.NewHTTPError(http.StatusBadRequest, "failed to marshal json data")
	}

	integrations, err := integration.DiscoverIntegrations(jsonData)
	if err != nil {
		h.logger.Error("failed to create credential", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create credential")
	}

	providerIDs := make(map[string]bool)
	for _, i := range req.ProviderIDs {
		providerIDs[i] = true
	}

	for _, i := range integrations {
		if _, ok := providerIDs[i.ProviderID]; !ok {
			continue
		}
		i.IntegrationType = req.IntegrationType

		i.CredentialID = credentialID

		healthcheckTime := time.Now()
		i.LastCheck = &healthcheckTime

		if i.Labels.Status != pgtype.Present {
			err = i.Labels.Set("{}")
			if err != nil {
				h.logger.Error("failed to set label", zap.Error(err))
				return echo.NewHTTPError(http.StatusInternalServerError, "failed to set label")
			}
		}

		if i.Annotations.Status != pgtype.Present {
			err = i.Annotations.Set("{}")
			if err != nil {
				h.logger.Error("failed to set annotations", zap.Error(err))
				return echo.NewHTTPError(http.StatusInternalServerError, "failed to set annotations")
			}
		}

		iApi, err := i.ToApi()
		if err != nil {
			h.logger.Error("failed to create integration api", zap.Error(err))
			return echo.NewHTTPError(http.StatusInternalServerError, "failed to create integration api")
		}
		healthy, err := integration.HealthCheck(jsonData, i.ProviderID, iApi.Labels, iApi.Annotations)
		if err != nil || !healthy {
			h.logger.Info("integration is not healthy", zap.String("integration_id", i.IntegrationID.String()), zap.Error(err))
			i.State = models2.IntegrationStateInactive
		} else {
			i.State = models2.IntegrationStateActive
		}

		err = h.database.CreateIntegration(&i)
		if err != nil {
			h.logger.Error("failed to create integration", zap.Error(err))
			return echo.NewHTTPError(http.StatusInternalServerError, "failed to create integration")
		}
	}

	return c.NoContent(http.StatusOK)
}

// IntegrationHealthcheck godoc
//
//	@Summary		Add integrations
//	@Description	Add integrations by given credential ID and integration IDs
//	@Security		BearerToken
//	@Tags			integrations
//	@Produce		json
//	@Success		200
//	@Router			/integration/api/v1/integrations/{IntegrationID}/healthcheck [put]
func (h API) IntegrationHealthcheck(c echo.Context) error {
	IntegrationID, err := uuid.Parse(c.Param("IntegrationID"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}

	integration, err := h.database.GetIntegration(IntegrationID)
	if err != nil {
		h.logger.Error("failed to get integration", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to get integration")
	}

	credential, err := h.database.GetCredential(integration.CredentialID.String())
	if err != nil {
		h.logger.Error("failed to get credential", zap.Error(err))
		return echo.NewHTTPError(http.StatusNotFound, "credential not found")
	}
	if credential == nil {
		h.logger.Error("credential not found", zap.Any("credentialId", integration.CredentialID))
		return echo.NewHTTPError(http.StatusNotFound, "credential not found")
	}

	mapData, err := h.vault.Decrypt(c.Request().Context(), credential.Secret)
	if err != nil {
		h.logger.Error("failed to decrypt secret", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to decrypt config")
	}

	if _, ok := integration_type.IntegrationTypes[integration.IntegrationType]; !ok {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid integration type")
	}

	jsonData, err := json.Marshal(mapData)
	if err != nil {
		h.logger.Error("failed to marshal json data", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to marshal json data")
	}

	integrationType := integration_type.IntegrationTypes[integration.IntegrationType]

	if integrationType == nil {
		return echo.NewHTTPError(http.StatusBadRequest, "failed to marshal json data")
	}
	integrationApi, err := integration.ToApi()
	if err != nil {
		h.logger.Error("failed to create integration api", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create integration api")
	}

	healthy, err := integrationType.HealthCheck(jsonData, integrationApi.ProviderID, integrationApi.Labels, integrationApi.Annotations)
	if err != nil || !healthy {
		h.logger.Error("healthcheck failed", zap.Error(err))
		if integration.State != models2.IntegrationStateArchived {
			integration.State = models2.IntegrationStateInactive
		}
		_, err = integration.AddAnnotations("platform/integration/health-reason", err.Error())
		if err != nil {
			h.logger.Error("failed to add annotations", zap.Error(err))
			return echo.NewHTTPError(http.StatusInternalServerError, "failed to add annotations")
		}
	} else {
		if integration.State != models2.IntegrationStateArchived {
			integration.State = models2.IntegrationStateActive
		}
	}
	healthcheckTime := time.Now()
	integration.LastCheck = &healthcheckTime
	err = h.database.UpdateIntegration(integration)
	if err != nil {
		h.logger.Error("failed to update integration", zap.Error(err), zap.Any("integration", *integration))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to update integration")
	}

	integrationApi, err = integration.ToApi()
	if err != nil {
		h.logger.Error("failed to create integration api", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create integration api")
	}

	return c.JSON(http.StatusOK, *integrationApi)
}

// Delete godoc
//
//	@Summary		Delete credential
//	@Description	Delete credential
//	@Security		BearerToken
//	@Tags			credentials
//	@Produce		json
//	@Success		200
//	@Param			IntegrationID	path	string	true	"IntegrationID"
//	@Router			/integration/api/v1/integrations/{IntegrationID} [delete]
func (h API) Delete(c echo.Context) error {
	IntegrationID, err := uuid.Parse(c.Param("IntegrationID"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}

	err = h.database.DeleteIntegration(IntegrationID)
	if err != nil {
		h.logger.Error("failed to delete credential", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to delete credential")
	}

	return c.NoContent(http.StatusOK)
}

// List godoc
//
//	@Summary		List integrations
//	@Description	List integrations
//	@Security		BearerToken
//	@Tags			credentials
//	@Produce		json
//	@Param			integration_type	query		[]string	false	"integration type filter"
//	@Success		200					{object}	models.ListIntegrationsResponse
//	@Router			/integration/api/v1/integrations [get]
func (h API) List(c echo.Context) error {
	integrationTypesStr := httpserver.QueryArrayParam(c, "integration_type")

	var integrationTypes []integration.Type
	for _, i := range integrationTypesStr {
		integrationTypes = append(integrationTypes, integration.Type(i))
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

// ListByFilters godoc
//
//	@Summary		List credentials with given filters
//	@Description	List credentials with given filters
//	@Security		BearerToken
//	@Tags			credentials
//	@Produce		json
//	@Success		200	{object}	models.ListIntegrationsResponse
//	@Router			/integration/api/v1/integrations/list [post]
func (h API) ListByFilters(c echo.Context) error {
	var req models.ListIntegrationsRequest

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}

	integrations, err := h.database.ListIntegrationsByFilters(req.IntegrationID, req.IntegrationType, req.NameRegex, req.ProviderIDRegex)
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

	totalCount := len(items)
	sort.Slice(items, func(i, j int) bool {
		return items[i].Name < items[j].Name
	})
	if req.PerPage != nil {
		if req.Cursor == nil {
			items = utils.Paginate(1, *req.PerPage, items)
		} else {
			items = utils.Paginate(*req.Cursor, *req.PerPage, items)
		}
	}

	return c.JSON(http.StatusOK, models.ListIntegrationsResponse{
		Integrations: items,
		TotalCount:   totalCount,
	})
}

// ListIntegrationGroups godoc
//
//	@Summary		List integration groups and their integrations
//	@Description	List integration groups and their integrations
//	@Security		BearerToken
//	@Tags			credentials
//	@Produce		json
//	@Param			populateIntegrations	query		bool	false	"Populate connections"	default(false)
//	@Success		200						{object}	[]models.IntegrationGroup
//	@Router			/integration/api/v1/integrations/integration-groups [get]
func (h API) ListIntegrationGroups(c echo.Context) error {
	populateIntegrations := false
	var err error
	if populateIntegrationsStr := c.QueryParam("populateIntegrations"); populateIntegrationsStr != "" {
		populateIntegrations, err = strconv.ParseBool(populateIntegrationsStr)
		if err != nil {
			return c.JSON(http.StatusBadRequest, "populateConnections is not a valid boolean")
		}
	}

	integrationGroups, err := h.database.ListIntegrationGroups()
	if err != nil {
		h.logger.Error("failed to list credentials", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to list credential")
	}

	var items []models.IntegrationGroup
	for _, integrationGroup := range integrationGroups {
		integrationGroupApi, err := entities.NewIntegrationGroup(c.Request().Context(), h.steampipeConn, integrationGroup)
		if err != nil {
			h.logger.Error("failed to convert integration group to API model", zap.Error(err))
			return echo.NewHTTPError(http.StatusInternalServerError, "failed to convert integration group to API model")
		}
		if populateIntegrations {
			integrations, err := h.database.ListIntegrationsByFilters(integrationGroupApi.IntegrationIds, nil, nil, nil)
			if err != nil {
				h.logger.Error("failed to list integrations", zap.Error(err))
				return echo.NewHTTPError(http.StatusInternalServerError, "failed to list integrations")
			}
			var apiIntegrations []models.Integration
			for _, integration := range integrations {
				apiIntegration, err := integration.ToApi()
				if err != nil {
					h.logger.Error("failed to convert integration to API model", zap.Error(err))
					return echo.NewHTTPError(http.StatusInternalServerError, "failed to convert integration to API model")
				}
				apiIntegrations = append(apiIntegrations, *apiIntegration)
			}
			integrationGroupApi.Integrations = apiIntegrations
		}
		items = append(items, *integrationGroupApi)
	}

	return c.JSON(http.StatusOK, items)
}

// GetIntegrationGroup godoc
//
//	@Summary		Get integration group and the integrations
//	@Description	Get integration group and the integrations
//	@Security		BearerToken
//	@Tags			credentials
//	@Produce		json
//	@Param			populateIntegrations	query		bool	false	"Populate connections"	default(false)
//	@Param			integrationGroupName	path		string	true	"integrationGroupName"
//	@Success		200						{object}	models.IntegrationGroup
//	@Router			/integration/api/v1/integrations/integration-groups/{integrationGroupName} [get]
func (h API) GetIntegrationGroup(c echo.Context) error {
	integrationGroupName := c.Param("integrationGroupName")

	populateIntegrations := false
	var err error
	if populateIntegrationsStr := c.QueryParam("populateIntegrations"); populateIntegrationsStr != "" {
		populateIntegrations, err = strconv.ParseBool(populateIntegrationsStr)
		if err != nil {
			return c.JSON(http.StatusBadRequest, "populateConnections is not a valid boolean")
		}
	}

	integrationGroup, err := h.database.GetIntegrationGroup(integrationGroupName)
	if err != nil {
		h.logger.Error("failed to list credentials", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to list credential")
	}

	integrationGroupApi, err := entities.NewIntegrationGroup(c.Request().Context(), h.steampipeConn, *integrationGroup)
	if err != nil {
		h.logger.Error("failed to convert integration group to API model", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to convert integration group to API model")
	}
	if populateIntegrations {
		integrations, err := h.database.ListIntegrationsByFilters(integrationGroupApi.IntegrationIds, nil, nil, nil)
		if err != nil {
			h.logger.Error("failed to list integrations", zap.Error(err))
			return echo.NewHTTPError(http.StatusInternalServerError, "failed to list integrations")
		}
		var apiIntegrations []models.Integration
		for _, integration := range integrations {
			apiIntegration, err := integration.ToApi()
			if err != nil {
				h.logger.Error("failed to convert integration to API model", zap.Error(err))
				return echo.NewHTTPError(http.StatusInternalServerError, "failed to convert integration to API model")
			}
			apiIntegrations = append(apiIntegrations, *apiIntegration)
		}
		integrationGroupApi.Integrations = apiIntegrations
	}

	return c.JSON(http.StatusOK, integrationGroupApi)
}

// Get godoc
//
//	@Summary		Get credential
//	@Description	Get credential
//	@Security		BearerToken
//	@Tags			credentials
//	@Produce		json
//	@Success		200
//	@Param			IntegrationID	path	string	true	"IntegrationID"
//	@Router			/integration/api/v1/integrations/{IntegrationID} [get]
func (h API) Get(c echo.Context) error {
	IntegrationID, err := uuid.Parse(c.Param("IntegrationID"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}

	integration, err := h.database.GetIntegration(IntegrationID)
	if err != nil {
		h.logger.Error("failed to get integration", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to get integration")
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
//	@Param			integrationId	path	string					true	"IntegrationID"
//	@Param			request			body	models.UpdateRequest	true	"Request"
//	@Router			/integration/api/v1/integrations/{integrationId} [post]
func (h API) Update(c echo.Context) error {
	IntegrationID, err := uuid.Parse(c.Param("IntegrationID"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}

	var req models.UpdateRequest

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}

	integration, err := h.database.GetIntegration(IntegrationID)
	if err != nil {
		h.logger.Error("failed to get credential", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to get credential")
	}

	credential, err := h.database.GetCredential(integration.CredentialID.String())
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

	err = h.database.UpdateCredential(integration.CredentialID.String(), secret)
	if err != nil {
		h.logger.Error("failed to update credential", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to update credential")
	}

	return c.NoContent(http.StatusOK)
}

// DeleteIntegrationType godoc
//
//	@Summary		Delete credential
//	@Description	Delete credential
//	@Security		BearerToken
//	@Tags			credentials
//	@Produce		json
//	@Success		200
//	@Param			integrationTypeId	path	string	true	"integrationTypeId"
//	@Router			/integration/api/v1/integrations/types/{integrationTypeId} [delete]
func (h API) DeleteIntegrationType(c echo.Context) error {
	integrationTypeId := c.Param("integrationTypeId")

	err := h.database.DeleteIntegrationType(integrationTypeId)
	if err != nil {
		h.logger.Error("failed to delete integration type", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to delete integration type")
	}

	return c.NoContent(http.StatusOK)
}

// ListIntegrationTypes godoc
//
//	@Summary		List integration types
//	@Description	List integration types
//	@Security		BearerToken
//	@Tags			credentials
//	@Produce		json
//	@Param			per_page	query		int		false	"PerPage"
//	@Param			cursor		query		int		false	"Cursor"
//	@Param			enabled		query		bool	false	"Enabled"
//	@Success		200			{object}	models.ListIntegrationTypesResponse
//	@Router			/integration/api/v1/integrations/types [get]
func (h API) ListIntegrationTypes(c echo.Context) error {
	perPageStr := c.QueryParam("per_page")
	cursorStr := c.QueryParam("cursor")
	enabled := c.QueryParam("enabled")
	var perPage, cursor int64
	if perPageStr != "" {
		perPage, _ = strconv.ParseInt(perPageStr, 10, 64)
	}
	if cursorStr != "" {
		cursor, _ = strconv.ParseInt(cursorStr, 10, 64)
	}

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
		if _, ok := integration_type.IntegrationTypes[integration_type.ParseType(integrationType.IntegrationType)]; ok {
			item.Enabled = true
		} else {
			if enabled == "true" {
				continue
			}
			item.Enabled = false
		}
		items = append(items, *item)
	}

	totalCount := len(items)
	sort.Slice(items, func(i, j int) bool {
		return items[i].ID < items[j].ID
	})
	if perPage != 0 {
		if cursor == 0 {
			items = utils.Paginate(1, perPage, items)
		} else {
			items = utils.Paginate(cursor, perPage, items)
		}
	}

	return c.JSON(http.StatusOK, models.ListIntegrationTypesResponse{
		IntegrationTypes: items,
		TotalCount:       totalCount,
	})
}

// GetIntegrationType godoc
//
//	@Summary		Get integration type
//	@Description	Get integration type
//	@Security		BearerToken
//	@Tags			credentials
//	@Produce		json
//	@Success		200
//	@Param			integrationTypeId	path	string	true	"integrationTypeId"
//	@Router			/integration/api/v1/integrations/types/{integrationTypeId} [get]
func (h API) GetIntegrationType(c echo.Context) error {
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

// GetIntegrationTypeUiSpec godoc
//
//	@Summary		Get integration type UI Spec
//	@Description	Get integration type UI Spec
//	@Security		BearerToken
//	@Tags			credentials
//	@Produce		json
//	@Success		200
//	@Param			integrationTypeId	path	string	true	"integrationTypeId"
//	@Router			/integration/api/v1/integrations/types/{integrationTypeId}/ui/spec [get]
func (h API) GetIntegrationTypeUiSpec(c echo.Context) error {
	integrationTypeId := c.Param("integrationTypeId")

	entries, err := os.ReadDir("/")
	if err != nil {
		h.logger.Error("failed to read dir", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to read dir")
	}

	// Loop through entries
	for _, entry := range entries {
		if entry.IsDir() {
			h.logger.Info("Directory:", zap.String("path", entry.Name()))
		} else {
			h.logger.Info("File:", zap.String("path", entry.Name()))
		}
	}

	integrationType, ok := integration_type.IntegrationTypes[integration.Type(integrationTypeId)]
	if !ok {
		return echo.NewHTTPError(http.StatusNotFound, "invalid integration type")
	}
	cnf := integrationType.GetConfiguration()

	file, err := os.Open("/ui-specs/" + cnf.UISpecFileName)
	if err != nil {
		h.logger.Error("failed to open file", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to open file")
	}
	defer file.Close()

	content, err := ioutil.ReadFile("/ui-specs/" + cnf.UISpecFileName)
	if err != nil {
		h.logger.Error("failed to read the file", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to read the file")
	}

	var result interface{}
	if err := json.Unmarshal(content, &result); err != nil {
		h.logger.Error("failed to unmarshal the file", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to unmarshal the file")
	}

	return c.JSON(http.StatusOK, result)
}
