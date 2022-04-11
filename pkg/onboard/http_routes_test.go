package onboard

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/ory/dockertest/v3"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	queuemocks "gitlab.com/keibiengine/keibi-engine/pkg/internal/queue/mocks"
	vaultmocks "gitlab.com/keibiengine/keibi-engine/pkg/internal/vault/mocks"
	"gitlab.com/keibiengine/keibi-engine/pkg/onboard/api"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	idocker "gitlab.com/keibiengine/keibi-engine/pkg/internal/dockertest"
)

type HttpHandlerSuite struct {
	suite.Suite

	handler HttpHandler
	router  *echo.Echo
}

func (s *HttpHandlerSuite) SetupSuite() {
	t := s.T()
	require := s.Require()

	pool, err := dockertest.NewPool("")
	require.NoError(err, "connect to docker")

	resource, err := pool.Run("postgres", "latest", []string{"POSTGRES_PASSWORD=mysecretpassword"})
	require.NoError(err, "status postgres")

	t.Cleanup(func() {
		err := pool.Purge(resource)
		require.NoError(err, "purge resource %s", resource)
	})

	var orm *gorm.DB
	// exponential backoff-retry, because the application in the container might not be ready to accept connections yet
	err = pool.Retry(func() error {
		orm, err = gorm.Open(postgres.Open(fmt.Sprintf("postgres://postgres:mysecretpassword@%s:%s/postgres", idocker.GetDockerHost(), resource.GetPort("5432/tcp"))), &gorm.Config{})
		if err != nil {
			return err
		}

		d, err := orm.DB()
		if err != nil {
			return err
		}

		return d.Ping()
	})
	require.NoError(err, "wait for postgres connection")

	tx := orm.Exec(`CREATE EXTENSION IF NOT EXISTS "uuid-ossp";`)
	require.NoError(tx.Error, "enable uuid v4")

	s.router = InitializeRouter()
	s.handler = HttpHandler{
		db:                Database{orm: orm},
		sourceEventsQueue: &queuemocks.Interface{},
		vault:             &vaultmocks.SourceConfig{},
	}

	s.handler.Register(s.router)
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
		Type:        api.SourceCloudAWS,
	})
	require.NoError(err)

	req := httptest.NewRequest(echo.GET, fmt.Sprintf("/api/v1/source/%s", srcId), nil)
	rec := httptest.NewRecorder()
	s.router.ServeHTTP(rec, req)

	require.Equal(http.StatusOK, rec.Code)
}

func (s *HttpHandlerSuite) TestCreateAWSSource_Success() {
	require := s.Require()

	qmock := s.handler.sourceEventsQueue.(*queuemocks.Interface)
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
		SourceType: api.SourceCloudAWS,
		ConfigRef:  pathRef,
	})
}

func (s *HttpHandlerSuite) TestCreateAzureSource_Success() {
	require := s.Require()

	qmock := s.handler.sourceEventsQueue.(*queuemocks.Interface)
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
		SourceType: api.SourceCloudAzure,
		ConfigRef:  pathRef,
	})
}

func (s *HttpHandlerSuite) TestDeleteAzureSource_Success() {
	require := s.Require()

	qmock := s.handler.sourceEventsQueue.(*queuemocks.Interface)
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
		SourceType: api.SourceCloudAzure,
		ConfigRef:  pathRef,
	})

	vmock.AssertCalled(s.T(), "Delete", pathRef)
}

func (s *HttpHandlerSuite) TestGetSources_Success() {
	require := s.Require()

	qmock := s.handler.sourceEventsQueue.(*queuemocks.Interface)
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
