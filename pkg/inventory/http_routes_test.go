package inventory

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"github.com/labstack/echo/v4"
	"github.com/ory/dockertest/v3"
	"github.com/stretchr/testify/suite"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
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
		Repository:   "docker.elastic.co/elasticsearch/elasticsearch",
		Tag:          "7.15.1",
		ExposedPorts: []string{"9200"},
		Env: []string{
			"xpack.security.enabled=false",
			"discovery.type=single-node",
		},
	})
	require.NoError(err, "status elasticsearch")
	esUrl := "http://localhost:" + resource.GetPort("9200/tcp")
	t.Cleanup(func() {
		err := pool.Purge(resource)
		require.NoError(err, "purge resource %s", resource)
	})

	// exponential backoff-retry, because the application in the container might not be ready to accept connections yet
	err = pool.Retry(func() error {
		res, err := http.Get(esUrl)
		if err != nil {
			return err
		}

		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return err
		}

		var resp map[string]interface{}
		err = json.Unmarshal(body, &resp)
		if err != nil {
			return err
		}

		return nil
	})
	require.NoError(err, "wait for elastic search connection")

	err = PopulateElastic(esUrl)
	require.NoError(err, "populating elastic")

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
	rec, err := doRequestJSONResponse(s.router, echo.POST, "/api/v1/resources", GetResourcesRequest{
		Filters: Filters{},
		Sort:    Sort{},
		Page: Page{
			Size: 10,
		},
	}, &response)
	require.NoError(err, "request")
	require.Equal(http.StatusOK, rec.Code)
	require.Len(response.Resources, 4)
}

func (s *HttpHandlerSuite) TestGetAllResources_Sort() {
	require := s.Require()

	var response GetResourcesResponse
	rec, err := doRequestJSONResponse(s.router, echo.POST, "/api/v1/resources", GetResourcesRequest{
		Filters: Filters{},
		Sort: Sort{
			SortBy: []SortItem{
				{
					Field:     SortFieldResourceID,
					Direction: DirectionDescending,
				},
			},
		},
		Page: Page{
			Size: 10,
		},
	}, &response)
	require.NoError(err, "request")
	require.Equal(http.StatusOK, rec.Code)
	require.Len(response.Resources, 4)
	require.Equal(response.Resources[0].ResourceID, "aaa3")
}

func (s *HttpHandlerSuite) TestGetAllResources_Paging() {
	require := s.Require()

	req := GetResourcesRequest{
		Filters: Filters{},
		Sort: Sort{
			SortBy: []SortItem{
				{
					Field:     SortFieldResourceID,
					Direction: DirectionDescending,
				},
			},
		},
		Page: Page{
			Size: 1,
		},
	}
	var response GetResourcesResponse
	rec, err := doRequestJSONResponse(s.router, echo.POST, "/api/v1/resources", req, &response)
	require.NoError(err, "request")
	require.Equal(http.StatusOK, rec.Code)
	require.Len(response.Resources, 1)
	require.Equal(response.Resources[0].ResourceID, "aaa3")

	req.Page.NextMarker = response.Page.NextMarker
	rec, err = doRequestJSONResponse(s.router, echo.POST, "/api/v1/resources", req, &response)
	require.NoError(err, "request")
	require.Equal(http.StatusOK, rec.Code)
	require.Len(response.Resources, 1)
	require.Equal(response.Resources[0].ResourceID, "aaa2")

	req.Page.NextMarker = response.Page.NextMarker
	rec, err = doRequestJSONResponse(s.router, echo.POST, "/api/v1/resources", req, &response)
	require.NoError(err, "request")
	require.Equal(http.StatusOK, rec.Code)
	require.Len(response.Resources, 1)
	require.Equal(response.Resources[0].ResourceID, "aaa1")

	req.Page.NextMarker = response.Page.NextMarker
	rec, err = doRequestJSONResponse(s.router, echo.POST, "/api/v1/resources", req, &response)
	require.NoError(err, "request")
	require.Equal(http.StatusOK, rec.Code)
	require.Len(response.Resources, 1)
	require.Equal(response.Resources[0].ResourceID, "aaa0")

	req.Page.NextMarker = response.Page.NextMarker
	rec, err = doRequestJSONResponse(s.router, echo.POST, "/api/v1/resources", req, &response)
	require.NoError(err, "request")
	require.Equal(http.StatusOK, rec.Code)
	require.Len(response.Resources, 0)
}

func (s *HttpHandlerSuite) TestGetAllResources_Filters() {
	require := s.Require()

	req := GetResourcesRequest{
		Filters: Filters{},
		Sort: Sort{
			SortBy: []SortItem{
				{
					Field:     SortFieldResourceID,
					Direction: DirectionAscending,
				},
			},
		},
		Page: Page{
			Size: 10,
		},
	}

	var response GetResourcesResponse
	req.Filters = Filters{}
	req.Filters.Location = []string{"us-east1"}
	rec, err := doRequestJSONResponse(s.router, echo.POST, "/api/v1/resources", req, &response)
	require.NoError(err, "request")
	require.Equal(http.StatusOK, rec.Code)
	require.Len(response.Resources, 2)
	require.Equal(response.Resources[0].ResourceID, "aaa0")
	require.Equal(response.Resources[1].ResourceID, "aaa2")

	req.Filters = Filters{}
	req.Filters.KeibiSource = []string{"ss1"}
	rec, err = doRequestJSONResponse(s.router, echo.POST, "/api/v1/resources", req, &response)
	require.NoError(err, "request")
	require.Equal(http.StatusOK, rec.Code)
	require.Len(response.Resources, 2)
	require.Equal(response.Resources[0].ResourceID, "aaa0")
	require.Equal(response.Resources[1].ResourceID, "aaa1")

	req.Filters = Filters{}
	req.Filters.ResourceType = []string{"Microsoft.Network/virtualNetworks"}
	rec, err = doRequestJSONResponse(s.router, echo.POST, "/api/v1/resources", req, &response)
	require.NoError(err, "request")
	require.Equal(http.StatusOK, rec.Code)
	require.Len(response.Resources, 2)
	require.Equal(response.Resources[0].ResourceID, "aaa2")
	require.Equal(response.Resources[1].ResourceID, "aaa3")
}

func (s *HttpHandlerSuite) TestGetAllResources_CSV() {
	require := s.Require()

	req := GetResourcesRequest{
		Filters: Filters{},
		Sort: Sort{
			SortBy: []SortItem{
				{
					Field:     SortFieldResourceID,
					Direction: DirectionAscending,
				},
			},
		},
		Page: Page{
			Size: 10,
		},
	}

	rec, response, err := doRequestCSVResponse(s.router, echo.POST, "/api/v1/resources/csv", req)
	require.NoError(err, "request")
	require.Equal(http.StatusOK, rec.Code)
	require.Len(response, 5) // first is header
	require.Equal(response[0][4], "ResourceID")
	require.Equal(response[1][4], "aaa0")
	require.Equal(response[2][4], "aaa1")
	require.Equal(response[3][4], "aaa2")
	require.Equal(response[4][4], "aaa3")

	req.Filters = Filters{}
	req.Filters.Location = []string{"us-east1"}
	rec, response, err = doRequestCSVResponse(s.router, echo.POST, "/api/v1/resources/csv", req)
	require.NoError(err, "request")
	require.Equal(http.StatusOK, rec.Code)
	require.Len(response, 3)
	require.Equal(response[1][4], "aaa0")
	require.Equal(response[2][4], "aaa2")
}

func (s *HttpHandlerSuite) TestGetAWSResources() {
	require := s.Require()

	var response GetAWSResourceResponse
	rec, err := doRequestJSONResponse(s.router, echo.POST, "/api/v1/resources/aws", GetResourcesRequest{
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
	require.Len(response.Resources, 2)
	for _, resource := range response.Resources {
		require.Equal(true, strings.HasPrefix(resource.ResourceType, "AWS"))
	}
}

func (s *HttpHandlerSuite) TestGetAzureResources() {
	require := s.Require()

	var response GetAzureResourceResponse
	rec, err := doRequestJSONResponse(s.router, echo.POST, "/api/v1/resources/azure", GetResourcesRequest{
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
	require.Len(response.Resources, 2)
	for _, resource := range response.Resources {
		require.Equal(true, strings.HasPrefix(resource.ResourceType, "Microsoft"), "resource type is %s", resource.ResourceType)
	}
}

func TestHttpHandlerSuite(t *testing.T) {
	suite.Run(t, &HttpHandlerSuite{})
}

func doRequestJSONResponse(router *echo.Echo, method string, path string, request, response interface{}) (*httptest.ResponseRecorder, error) {
	var requestBytes []byte
	var err error

	if request != nil {
		requestBytes, err = json.Marshal(request)
		if err != nil {
			return nil, err
		}
	}

	rec, resp, err := sendRequest(router, method, path, requestBytes)
	if err != nil {
		return nil, err
	}

	if response != nil {
		if err := json.Unmarshal(resp, response); err != nil {
			return nil, err
		}
	}
	return rec, nil
}

func doRequestCSVResponse(router *echo.Echo, method string, path string, request interface{}) (*httptest.ResponseRecorder, [][]string, error) {
	var requestBytes []byte
	var err error

	if request != nil {
		requestBytes, err = json.Marshal(request)
		if err != nil {
			return nil, nil, err
		}
	}

	rec, resp, err := sendRequest(router, method, path, requestBytes)
	if err != nil {
		return nil, nil, err
	}

	reader := csv.NewReader(bytes.NewReader(resp))
	response, err := reader.ReadAll()
	if err != nil {
		return nil, nil, err
	}
	return rec, response, nil
}

func sendRequest(router *echo.Echo, method string, path string, request []byte) (*httptest.ResponseRecorder, []byte, error) {
	var r io.Reader
	if request != nil {
		r = bytes.NewReader(request)
	}

	req := httptest.NewRequest(method, path, r)
	req.Header.Add("content-type", "application/json")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	// Wrap in NopCloser in case the calling method wants to also read the body
	b, err := ioutil.ReadAll(io.NopCloser(rec.Body))
	if err != nil {
		return nil, nil, err
	}
	return rec, b, nil
}
