package inventory

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/labstack/echo/v4"
	"github.com/ory/dockertest/v3"
	"github.com/stretchr/testify/suite"
	"gitlab.com/keibiengine/keibi-engine/pkg/keibi-es-sdk"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
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

	resource, err := pool.RunWithOptions(&dockertest.RunOptions{
		Repository: "docker.elastic.co/elasticsearch/elasticsearch",
		Tag: "7.15.1",
		ExposedPorts: []string{"9200"},
	})
	require.NoError(err, "status elasticsearch")
	esUrl := "https://localhost:" + resource.GetPort("9200/tcp")
	t.Cleanup(func() {
		err := pool.Purge(resource)
		require.NoError(err, "purge resource %s", resource)
	})

	// exponential backoff-retry, because the application in the container might not be ready to accept connections yet
	err = pool.Retry(func() error {
		accountID := "all"
		_, err := keibi.NewClient(keibi.ClientConfig{
			Addresses: []string{esUrl},
			Username:  nil,
			Password:  nil,
			AccountID: &accountID,
		})

		if err != nil {
			fmt.Println(err)
		}
		return nil
	})
	require.NoError(err, "wait for postgres connection")
	//
	//tx := orm.Exec(`CREATE EXTENSION IF NOT EXISTS "uuid-ossp";`)
	//require.NoError(tx.Error, "enable uuid v4")

	s.router = InitializeRouter()
	s.handler, _ = InitializeHttpHandler(esUrl, "", "")

	s.handler.Register(s.router.Group("/api/v1"))
}

func (s *HttpHandlerSuite) BeforeTest(suiteName, testName string) {
	//require := s.Require()
}

func (s *HttpHandlerSuite) AfterTest(suiteName, testName string) {
	//require := s.Require()
}

func (s *HttpHandlerSuite) TestGetAllResources() {
	require := s.Require()

	var response GetResourcesResponse
	rec, err := doSimpleJSONRequest(s.router, echo.POST, "/api/v1/resources", GetResourcesRequest{
		Filters: Filters{
			ResourceType: nil,
			Location:     nil,
			KeibiSource:  nil,
		},
		Sort: Sort{
			SortBy: nil,
		},
		Page: Page{
			NextMarker: nil,
			Size:       10,
		},
	}, &response)
	require.NoError(err, "request")
	require.Equal(http.StatusOK, rec.Code)
	require.Len(response.Resources, 4)
}

func (s *HttpHandlerSuite) TestGetAllResources_Sort() {
	require := s.Require()

	var response GetResourcesResponse
	rec, err := doSimpleJSONRequest(s.router, echo.POST, "/api/v1/resources", GetResourcesRequest{
		Filters: Filters{
			ResourceType: nil,
			Location:     nil,
			KeibiSource:  nil,
		},
		Sort: Sort{
			SortBy: []SortItem{
				{
					Field:     SortFieldResourceID,
					Direction: DirectionDescending,
				},
			},
		},
		Page: Page{
			NextMarker: nil,
			Size:       10,
		},
	}, &response)
	require.NoError(err, "request")
	require.Equal(http.StatusOK, rec.Code)
	require.Len(response.Resources, 4)
	require.Equal(response.Resources[0].ResourceID, "aaaaab")
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
