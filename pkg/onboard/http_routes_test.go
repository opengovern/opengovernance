package onboard

import (
	"bytes"
	"encoding/json"
	"fmt"
	idocker "github.com/kaytu-io/kaytu-util/pkg/dockertest"
	"github.com/kaytu-io/kaytu-util/pkg/httpserver"
	"github.com/kaytu-io/kaytu-util/pkg/postgres"
	"github.com/kaytu-io/kaytu-util/pkg/queue/mocks"
	"github.com/ory/dockertest/v3"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"gitlab.com/keibiengine/keibi-engine/pkg/source"
	vaultmocks "gitlab.com/keibiengine/keibi-engine/pkg/vault/mocks"

	describeapi "gitlab.com/keibiengine/keibi-engine/pkg/describe/api"
	inventoryapi "gitlab.com/keibiengine/keibi-engine/pkg/inventory/api"
	"go.uber.org/zap"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	_ "github.com/ory/dockertest/v3"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"gitlab.com/keibiengine/keibi-engine/pkg/onboard/api"

	"gorm.io/gorm"
)

type HttpHandlerSuite struct {
	suite.Suite

	handler *HttpHandler
	router  *echo.Echo
}

func (s *HttpHandlerSuite) SetupSuite() {
	t := s.T()
	require := s.Require()

	pool, err := dockertest.NewPool("")
	require.NoError(err, "connect to docker")

	resource, err := pool.Run("postgres", "14.2-alpine", []string{"POSTGRES_PASSWORD=mysecretpassword"})
	require.NoError(err, "status postgres")

	t.Cleanup(func() {
		err := pool.Purge(resource)
		require.NoError(err, "purge resource %s", resource)
	})
	time.Sleep(5 * time.Second)

	var orm *gorm.DB
	// exponential backoff-retry, because the application in the container might not be ready to accept connections yet
	err = pool.Retry(func() error {
		cfg := &postgres.Config{
			Host:   idocker.GetDockerHost(),
			Port:   resource.GetPort("5432/tcp"),
			User:   "postgres",
			Passwd: "mysecretpassword",
			DB:     "postgres",
		}

		logger, err := zap.NewProduction()
		require.NoError(err, "new zap logger")

		orm, err = postgres.NewClient(cfg, logger)
		require.NoError(err, "new postgres client")

		d, err := orm.DB()
		if err != nil {
			return err
		}

		return d.Ping()
	})
	require.NoError(err, "wait for postgres connection")

	tx := orm.Exec(`CREATE EXTENSION IF NOT EXISTS "uuid-ossp";`)
	require.NoError(tx.Error, "enable uuid v4")

	s.handler = &HttpHandler{
		db:                Database{orm: orm},
		sourceEventsQueue: &mocks.Interface{},
		vault:             &vaultmocks.SourceConfig{},
	}

	logger, err := zap.NewProduction()
	require.NoError(err, "new logger")

	s.router = httpserver.Register(logger, s.handler)
}

func (s *HttpHandlerSuite) BeforeTest(suiteName, testName string) {
	require := s.Require()

	err := s.handler.db.Initialize()
	require.NoError(err, "initialize db")
}

func (s *HttpHandlerSuite) AfterTest(suiteName, testName string) {
	require := s.Require()

	tx := s.handler.db.orm.Exec("DROP TABLE IF EXISTS sources;")
	require.NoError(tx.Error, "drop sources")
}

func (s *HttpHandlerSuite) TestGetSource() {
	require := s.Require()

	srcId := uuid.New()
	err := s.handler.db.CreateSource(&Source{
		ID:          srcId,
		SourceId:    "12312312312312321",
		Name:        "123123",
		Description: "123123123",
		Type:        source.CloudAWS,
	})
	require.NoError(err)

	req := httptest.NewRequest(echo.GET, fmt.Sprintf("/api/v1/source/%s", srcId), nil)
	rec := httptest.NewRecorder()
	s.router.ServeHTTP(rec, req)

	require.Equal(http.StatusOK, rec.Code)
}

func (s *HttpHandlerSuite) TestCreateAWSSource_Success() {
	require := s.Require()

	qmock := s.handler.sourceEventsQueue.(*mocks.Interface)
	vmock := s.handler.vault.(*vaultmocks.SourceConfig)

	qmock.On("Publish", mock.Anything).Return(error(nil))
	vmock.On("Write", mock.Anything, mock.Anything).Return(error(nil))

	var response api.CreateSourceResponse
	rec, err := doSimpleJSONRequest(s.router, echo.POST, "/api/v1/source/aws", api.SourceAwsRequest{
		Name:        "Account1",
		Description: "Account1 Description",
		Config: api.SourceConfigAWS{
			AccountId: "123456789012",
			AccessKey: "ACCESS_KEY",
			SecretKey: "SECRET_KEY",
		},
	}, &response)
	require.NoError(err, "request")
	require.Equal(http.StatusOK, rec.Code)
	require.NotEmpty(response.ID)

	pathRef := fmt.Sprintf("sources/aws/%s", response.ID)
	vmock.AssertCalled(s.T(), "Write", pathRef, map[string]interface{}{
		"accountId": "123456789012",
		"accessKey": "ACCESS_KEY",
		"secretKey": "SECRET_KEY",
	})

	qmock.AssertCalled(s.T(), "Publish", api.SourceEvent{
		Action:     api.SourceCreated,
		SourceID:   response.ID,
		AccountID:  "123456789012",
		SourceType: source.CloudAWS,
		Secret:     pathRef,
	})
}

func (s *HttpHandlerSuite) TestCreateAzureSourceWithSPN_SPNNotExists() {
	require := s.Require()

	var response api.CreateSourceResponse
	rec, err := doSimpleJSONRequest(s.router, echo.POST, "/api/v1/source/azure/spn", api.SourceAzureSPNRequest{
		Name:           "Account1",
		Description:    "Account1 Description",
		SubscriptionId: "6948DF80-14BD-4E04-8842-7668D9C001F5", // RANDOM UUID
		SPNId:          uuid.New(),
	}, &response)
	require.NoError(err, "request")
	require.Equal(http.StatusNotFound, rec.Code)
}

func (s *HttpHandlerSuite) TestCreateAzureSourceWithSPN_Success() {
	require := s.Require()

	qmock := s.handler.sourceEventsQueue.(*mocks.Interface)
	vmock := s.handler.vault.(*vaultmocks.SourceConfig)

	qmock.On("Publish", mock.Anything).Return(error(nil))
	vmock.On("Write", mock.Anything, mock.Anything).Return(error(nil))

	var spnResponse api.CreateSPNResponse
	rec, err := doSimpleJSONRequest(s.router, echo.POST, "/api/v1/spn/azure", api.CreateSPNRequest{
		Config: api.SPNConfigAzure{
			TenantId:     "297bd476-4f40-499e-81e8-420621500a1b", // RANDOM UUID
			ClientId:     "2578aa74-e8ec-445c-8142-7bdebf33fed4", // RANDOM UUID
			ClientSecret: "SECRET",
		},
	}, &spnResponse)
	require.NoError(err, "request")
	require.Equal(http.StatusOK, rec.Code)
	require.NotEmpty(spnResponse.ID)

	pathRef := fmt.Sprintf("sources/azure/spn/%s", spnResponse.ID)
	vmock.AssertCalled(s.T(), "Write", pathRef, map[string]interface{}{
		"tenantId":     "297bd476-4f40-499e-81e8-420621500a1b",
		"clientId":     "2578aa74-e8ec-445c-8142-7bdebf33fed4",
		"clientSecret": "SECRET",
	})

	var response api.CreateSourceResponse
	rec, err = doSimpleJSONRequest(s.router, echo.POST, "/api/v1/source/azure/spn", api.SourceAzureSPNRequest{
		Name:           "Account1",
		Description:    "Account1 Description",
		SubscriptionId: "6948DF80-14BD-4E04-8842-7668D9C001F5", // RANDOM UUID
		SPNId:          spnResponse.ID,
	}, &response)
	require.NoError(err, "request")
	require.Equal(http.StatusOK, rec.Code)
	require.NotEmpty(response.ID)

	qmock.AssertCalled(s.T(), "Publish", api.SourceEvent{
		Action:     api.SourceCreated,
		SourceID:   response.ID,
		AccountID:  "6948DF80-14BD-4E04-8842-7668D9C001F5",
		SourceType: source.CloudAzure,
		Secret:     pathRef,
	})
}

func (s *HttpHandlerSuite) TestChangeAzureSPNSecret() {
	require := s.Require()

	qmock := s.handler.sourceEventsQueue.(*mocks.Interface)
	vmock := s.handler.vault.(*vaultmocks.SourceConfig)

	qmock.On("Publish", mock.Anything).Return(error(nil))
	vmock.On("Write", mock.Anything, mock.Anything).Return(error(nil))

	var spnResponse api.CreateSPNResponse
	rec, err := doSimpleJSONRequest(s.router, echo.POST, "/api/v1/spn/azure", api.CreateSPNRequest{
		Config: api.SPNConfigAzure{
			TenantId:     "4B8302DA-21AD-401F-AF45-1DFD956B80B5", // RANDOM UUID
			ClientId:     "8628FE7C-A4E9-4056-91BD-FD6AA7817E39", // RANDOM UUID
			ClientSecret: "SECRET",
		},
	}, &spnResponse)
	require.NoError(err, "request")
	require.Equal(http.StatusOK, rec.Code)
	require.NotEmpty(spnResponse.ID)

	pathRef := fmt.Sprintf("sources/azure/spn/%s", spnResponse.ID)
	vmock.AssertCalled(s.T(), "Write", pathRef, map[string]interface{}{
		"tenantId":     "4B8302DA-21AD-401F-AF45-1DFD956B80B5",
		"clientId":     "8628FE7C-A4E9-4056-91BD-FD6AA7817E39",
		"clientSecret": "SECRET",
	})

	vmock.On("Read", fmt.Sprintf("sources/azure/spn/%s", spnResponse.ID.String())).Return(map[string]interface{}{
		"tenantId":     "4B8302DA-21AD-401F-AF45-1DFD956B80B5",
		"clientId":     "8628FE7C-A4E9-4056-91BD-FD6AA7817E39",
		"clientSecret": "SECRET",
	}, error(nil))

	path := fmt.Sprintf("/api/v1/spn/%s", spnResponse.ID.String())
	var credRes api.AzureCredential
	rec, err = doSimpleJSONRequest(s.router, echo.GET, path, nil, &credRes)
	require.NoError(err, "request")
	require.Equal(http.StatusOK, rec.Code)

	require.Equal("8628FE7C-A4E9-4056-91BD-FD6AA7817E39", credRes.ClientID)
	require.Equal("4B8302DA-21AD-401F-AF45-1DFD956B80B5", credRes.TenantID)
	require.Equal("", credRes.ClientSecret)

	rec, err = doSimpleJSONRequest(s.router, echo.PUT, path, &api.AzureCredential{
		ClientID:     uuid.New().String(),
		TenantID:     uuid.New().String(),
		ClientSecret: "SECRET2",
	}, nil)
	require.NoError(err, "request")
	require.Equal(http.StatusOK, rec.Code)
	vmock.AssertCalled(s.T(), "Write", pathRef, map[string]interface{}{
		"tenantId":     "4B8302DA-21AD-401F-AF45-1DFD956B80B5",
		"clientId":     "8628FE7C-A4E9-4056-91BD-FD6AA7817E39",
		"clientSecret": "SECRET2",
	})
}

func (s *HttpHandlerSuite) TestChangeAWSSecret() {
	require := s.Require()

	qmock := s.handler.sourceEventsQueue.(*mocks.Interface)
	vmock := s.handler.vault.(*vaultmocks.SourceConfig)

	qmock.On("Publish", mock.Anything).Return(error(nil))
	vmock.On("Write", mock.Anything, mock.Anything).Return(error(nil))

	var response api.CreateSourceResponse
	rec, err := doSimpleJSONRequest(s.router, echo.POST, "/api/v1/source/aws", api.SourceAwsRequest{
		Name:        "Account1",
		Description: "Account1 Description",
		Config: api.SourceConfigAWS{
			AccountId: "123456789012",
			AccessKey: "ACCESS_KEY",
			SecretKey: "SECRET_KEY",
		},
	}, &response)
	require.NoError(err, "request")
	require.Equal(http.StatusOK, rec.Code)
	require.NotEmpty(response.ID)

	pathRef := fmt.Sprintf("sources/aws/%s", response.ID)
	vmock.AssertCalled(s.T(), "Write", pathRef, map[string]interface{}{
		"accountId": "123456789012",
		"accessKey": "ACCESS_KEY",
		"secretKey": "SECRET_KEY",
	})

	vmock.On("Read", fmt.Sprintf("sources/aws/%s", response.ID)).Return(map[string]interface{}{
		"accountId": "123456789012",
		"accessKey": "ACCESS_KEY",
		"secretKey": "SECRET_KEY",
	}, error(nil))

	path := fmt.Sprintf("/api/v1/source/%s/credentials", response.ID.String())
	var credRes api.AWSCredential
	rec, err = doSimpleJSONRequest(s.router, echo.GET, path, nil, &credRes)
	require.NoError(err, "request")
	require.Equal(http.StatusOK, rec.Code)
	require.Equal(credRes.AccessKey, "ACCESS_KEY")
	require.Equal(credRes.SecretKey, "")

	rec, err = doSimpleJSONRequest(s.router, echo.PUT, path, &api.AWSCredential{
		AccessKey: "TEMP",
		SecretKey: "SECRET_KEY2",
	}, nil)
	require.NoError(err, "request")
	require.Equal(http.StatusOK, rec.Code)
	vmock.AssertCalled(s.T(), "Write", pathRef, map[string]interface{}{
		"accountId": "123456789012",
		"accessKey": "ACCESS_KEY",
		"secretKey": "SECRET_KEY2",
	})
}

func (s *HttpHandlerSuite) TestCreateAzureSource_Success() {
	require := s.Require()

	qmock := s.handler.sourceEventsQueue.(*mocks.Interface)
	vmock := s.handler.vault.(*vaultmocks.SourceConfig)

	qmock.On("Publish", mock.Anything).Return(error(nil))
	vmock.On("Write", mock.Anything, mock.Anything).Return(error(nil))

	var response api.CreateSourceResponse
	rec, err := doSimpleJSONRequest(s.router, echo.POST, "/api/v1/source/azure", api.SourceAzureRequest{
		Name:        "Account1",
		Description: "Account1 Description",
		Config: api.SourceConfigAzure{
			SubscriptionId: "6948DF80-14BD-4E04-8842-7668D9C001F5", // RANDOM UUID
			TenantId:       "4B8302DA-21AD-401F-AF45-1DFD956B80B5", // RANDOM UUID
			ClientId:       "8628FE7C-A4E9-4056-91BD-FD6AA7817E39", // RANDOM UUID
			ClientSecret:   "SECRET",
		},
	}, &response)
	require.NoError(err, "request")
	require.Equal(http.StatusOK, rec.Code)
	require.NotEmpty(response.ID)

	pathRef := fmt.Sprintf("sources/azure/%s", response.ID)
	vmock.AssertCalled(s.T(), "Write", pathRef, map[string]interface{}{
		"subscriptionId": "6948DF80-14BD-4E04-8842-7668D9C001F5",
		"tenantId":       "4B8302DA-21AD-401F-AF45-1DFD956B80B5",
		"clientId":       "8628FE7C-A4E9-4056-91BD-FD6AA7817E39",
		"clientSecret":   "SECRET",
	})

	qmock.AssertCalled(s.T(), "Publish", api.SourceEvent{
		Action:     api.SourceCreated,
		SourceID:   response.ID,
		AccountID:  "6948DF80-14BD-4E04-8842-7668D9C001F5",
		SourceType: source.CloudAzure,
		Secret:     pathRef,
	})
}

func (s *HttpHandlerSuite) TestDeleteAzureSource_Success() {
	require := s.Require()

	qmock := s.handler.sourceEventsQueue.(*mocks.Interface)
	vmock := s.handler.vault.(*vaultmocks.SourceConfig)

	qmock.On("Publish", mock.Anything).Return(error(nil))
	vmock.On("Delete", mock.Anything).Return(error(nil))

	var response api.CreateSourceResponse
	rec, err := doSimpleJSONRequest(s.router, echo.POST, "/api/v1/source/azure", api.SourceAzureRequest{
		Name:        "Account1",
		Description: "Account1 Description",
		Config: api.SourceConfigAzure{
			SubscriptionId: "6948DF80-14BD-4E04-8842-7668D9C001F5", // RANDOM UUID
			TenantId:       "4B8302DA-21AD-401F-AF45-1DFD956B80B5", // RANDOM UUID
			ClientId:       "8628FE7C-A4E9-4056-91BD-FD6AA7817E39", // RANDOM UUID
			ClientSecret:   "SECRET",
		},
	}, &response)
	require.NoError(err, "request")
	require.Equal(http.StatusOK, rec.Code)
	require.NotEmpty(response.ID)

	pathRef := fmt.Sprintf("sources/azure/%s", response.ID)
	rec, err = doSimpleJSONRequest(s.router, echo.DELETE, fmt.Sprintf("/api/v1/source/%s", response.ID.String()), nil, nil)
	require.NoError(err, "request")
	require.Equal(http.StatusOK, rec.Code)

	qmock.AssertCalled(s.T(), "Publish", api.SourceEvent{
		Action:     api.SourceDeleted,
		SourceID:   response.ID,
		SourceType: source.CloudAzure,
		Secret:     pathRef,
	})

	vmock.AssertCalled(s.T(), "Delete", pathRef)
}

func (s *HttpHandlerSuite) TestGetSources_Success() {
	require := s.Require()

	qmock := s.handler.sourceEventsQueue.(*mocks.Interface)
	vmock := s.handler.vault.(*vaultmocks.SourceConfig)

	qmock.On("Publish", mock.Anything).Return(error(nil))
	vmock.On("Write", mock.Anything, mock.Anything).Return(error(nil))

	var response1 api.CreateSourceResponse
	rec, err := doSimpleJSONRequest(s.router, echo.POST, "/api/v1/source/azure", api.SourceAzureRequest{
		Name:        "Account1",
		Description: "Account1 Description",
		Config: api.SourceConfigAzure{
			SubscriptionId: "6948DF80-14BD-4E04-8842-7668D9C001F5", // RANDOM UUID
			TenantId:       "4B8302DA-21AD-401F-AF45-1DFD956B80B5", // RANDOM UUID
			ClientId:       "8628FE7C-A4E9-4056-91BD-FD6AA7817E39", // RANDOM UUID
			ClientSecret:   "SECRET",
		},
	}, &response1)
	require.NoError(err, "request")
	require.Equal(http.StatusOK, rec.Code)
	require.NotEmpty(response1.ID)

	var response2 api.CreateSourceResponse
	rec, err = doSimpleJSONRequest(s.router, echo.POST, "/api/v1/source/aws", api.SourceAwsRequest{
		Name:        "Account1",
		Description: "Account1 Description",
		Config: api.SourceConfigAWS{
			AccountId: "123456789012",
			AccessKey: "ACCESS_KEY",
			SecretKey: "SECRET_KEY",
		},
	}, &response2)
	require.NoError(err, "request")
	require.Equal(http.StatusOK, rec.Code)
	require.NotEmpty(response2.ID)

	var response3 api.GetSourcesResponse
	rec, err = doSimpleJSONRequest(s.router, echo.GET, "/api/v1/sources", nil, &response3)
	require.NoError(err, "request")
	require.Equal(http.StatusOK, rec.Code)
	require.Equal(2, len(response3))

	var response4 api.GetSourcesResponse
	rec, err = doSimpleJSONRequest(s.router, echo.GET, "/api/v1/sources?type=aws", nil, &response4)
	require.NoError(err, "request")
	require.Equal(http.StatusOK, rec.Code)
	require.Equal(1, len(response4))

	var response5 api.GetSourcesResponse
	rec, err = doSimpleJSONRequest(s.router, echo.GET, "/api/v1/sources?type=AZURE", nil, &response5)
	require.NoError(err, "request")
	require.Equal(http.StatusOK, rec.Code)
	require.Equal(1, len(response5))

	var response6 int64
	rec, err = doSimpleJSONRequest(s.router, echo.GET, "/api/v1/sources/count", nil, &response6)
	require.NoError(err, "request")
	require.Equal(http.StatusOK, rec.Code)
	require.Equal(int64(2), response6)

	rec, err = doSimpleJSONRequest(s.router, echo.GET, "/api/v1/sources/count?type=aws", nil, &response6)
	require.NoError(err, "request")
	require.Equal(http.StatusOK, rec.Code)
	require.Equal(int64(1), response6)
}

func (s *HttpHandlerSuite) TestCountConnections() {
	require := s.Require()

	var response1 api.CreateSourceResponse
	rec, err := doSimpleJSONRequest(s.router, echo.POST, "/api/v1/source/azure", api.SourceAzureRequest{
		Name:        "Account1",
		Description: "Account1 Description",
		Config: api.SourceConfigAzure{
			SubscriptionId: "6948DF80-14BD-4E04-8842-7668D9C001F5", // RANDOM UUID
			TenantId:       "4B8302DA-21AD-401F-AF45-1DFD956B80B5", // RANDOM UUID
			ClientId:       "8628FE7C-A4E9-4056-91BD-FD6AA7817E39", // RANDOM UUID
			ClientSecret:   "SECRET",
		},
	}, &response1)
	require.NoError(err, "request")
	require.Equal(http.StatusOK, rec.Code)
	require.NotEmpty(response1.ID)

	var response2 api.CreateSourceResponse
	rec, err = doSimpleJSONRequest(s.router, echo.POST, "/api/v1/source/aws", api.SourceAwsRequest{
		Name:        "Account1",
		Description: "Account1 Description",
		Config: api.SourceConfigAWS{
			AccountId: "123456789012",
			AccessKey: "ACCESS_KEY",
			SecretKey: "SECRET_KEY",
		},
	}, &response2)
	require.NoError(err, "request")
	require.Equal(http.StatusOK, rec.Code)
	require.NotEmpty(response2.ID)

	var response6 int64
	rec, err = doSimpleJSONRequest(s.router, echo.POST, "/api/v1/connections/count", api.ConnectionCountRequest{
		ConnectorsNames: nil,
		State:           nil,
		Health:          nil,
	}, &response6)
	require.NoError(err, "request")
	require.Equal(http.StatusOK, rec.Code)
	require.Equal(int64(2), response6)

	rec, err = doSimpleJSONRequest(s.router, echo.POST, "/api/v1/connections/count", api.ConnectionCountRequest{
		ConnectorsNames: []string{"AWS Accounts"},
		State:           nil,
		Health:          nil,
	}, &response6)
	require.NoError(err, "request")
	require.Equal(http.StatusOK, rec.Code)
	require.Equal(int64(1), response6)

	rec, err = doSimpleJSONRequest(s.router, echo.POST, "/api/v1/connections/count", api.ConnectionCountRequest{
		ConnectorsNames: []string{"Azure Subscription"},
		State:           nil,
		Health:          nil,
	}, &response6)
	require.NoError(err, "request")
	require.Equal(http.StatusOK, rec.Code)
	require.Equal(int64(1), response6)

	rec, err = doSimpleJSONRequest(s.router, echo.POST, "/api/v1/connections/count", api.ConnectionCountRequest{
		ConnectorsNames: []string{"ServiceNow"},
		State:           nil,
		Health:          nil,
	}, &response6)
	require.NoError(err, "request")
	require.Equal(http.StatusOK, rec.Code)
	require.Equal(int64(0), response6)

	state := api.ConnectionState_ENABLED
	rec, err = doSimpleJSONRequest(s.router, echo.POST, "/api/v1/connections/count", api.ConnectionCountRequest{
		ConnectorsNames: []string{"Azure Subscription"},
		State:           &state,
		Health:          nil,
	}, &response6)
	require.NoError(err, "request")
	require.Equal(http.StatusOK, rec.Code)
	require.Equal(int64(1), response6)

	health := source.HealthStatusHealthy
	rec, err = doSimpleJSONRequest(s.router, echo.POST, "/api/v1/connections/count", api.ConnectionCountRequest{
		ConnectorsNames: []string{"Azure Subscription"},
		State:           nil,
		Health:          &health,
	}, &response6)
	require.NoError(err, "request")
	require.Equal(http.StatusOK, rec.Code)
	require.Equal(int64(1), response6)
}

func (s *HttpHandlerSuite) TestGetProviders() {
	require := s.Require()

	var response api.ProvidersResponse
	rec, err := doSimpleJSONRequest(s.router, echo.GET, "/api/v1/providers", nil, &response)
	require.NoError(err, "request")
	require.Equal(http.StatusOK, rec.Code)
	require.Equal(80, len(response))
}

func TestHttpHandlerSuite(t *testing.T) {
	suite.Run(t, &HttpHandlerSuite{})
}

func doSimpleJSONRequest(router *echo.Echo, method string, path string, request, response interface{}) (*httptest.ResponseRecorder, error) {
	var r io.Reader
	if request != nil {
		out, err := json.Marshal(request)
		if err != nil {
			return nil, err
		}

		r = bytes.NewReader(out)
	}

	req := httptest.NewRequest(method, path, r)
	req.Header.Add("content-type", "application/json")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if response != nil {
		// Wrap in NopCloser in case the calling method wants to also read the body
		b, err := ioutil.ReadAll(io.NopCloser(rec.Body))
		if err != nil {
			return nil, err
		}

		if err := json.Unmarshal(b, response); err != nil {
			return nil, err
		}
	}

	return rec, nil
}

type InventoryMockClient struct {
	s *HttpHandlerSuite
}

func (c InventoryMockClient) ListAccountsResourceCount() ([]inventoryapi.TopAccountResponse, error) {
	sources, err := c.s.handler.db.ListSources()
	if err != nil {
		return nil, err
	}

	var resp []inventoryapi.TopAccountResponse
	for _, s := range sources {
		resp = append(resp, inventoryapi.TopAccountResponse{
			SourceID:      s.ID.String(),
			ResourceCount: 10,
		})
	}
	return resp, nil
}

type DescribeMockClient struct {
	s *HttpHandlerSuite
}

func (c DescribeMockClient) ListSources() ([]describeapi.Source, error) {
	sources, err := c.s.handler.db.ListSources()
	if err != nil {
		return nil, err
	}

	var resp []describeapi.Source
	for _, s := range sources {
		resp = append(resp, describeapi.Source{
			ID:                     s.ID,
			Type:                   describeapi.SourceType(s.Type),
			LastDescribedAt:        time.Now(),
			LastComplianceReportAt: time.Now(),
		})
	}
	return resp, nil
}
