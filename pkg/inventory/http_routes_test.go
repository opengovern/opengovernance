package inventory

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"github.com/labstack/echo/v4"
	"github.com/ory/dockertest/v3"
	"github.com/stretchr/testify/suite"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
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

	net, err := pool.CreateNetwork("keibi")
	require.NoError(err, "create a network")
	t.Cleanup(func() {
		err = pool.RemoveNetwork(net)
		require.NoError(err, "remove network")
	})

	elasticResource, err := pool.RunWithOptions(&dockertest.RunOptions{
		Name:         "keibi_test_es",
		Repository:   "docker.elastic.co/elasticsearch/elasticsearch",
		Tag:          "7.15.1",
		ExposedPorts: []string{"9200"},
		Env: []string{
			"xpack.security.enabled=false",
			"discovery.type=single-node",
		},
		Networks: []*dockertest.Network{net},
	})
	require.NoError(err, "status elasticsearch")
	esUrl := "http://localhost:" + elasticResource.GetPort("9200/tcp")

	t.Cleanup(func() {
		err := pool.Purge(elasticResource)
		require.NoError(err, "purge resource %s", elasticResource)
	})

	postgresResource, err := pool.RunWithOptions(&dockertest.RunOptions{
		Name:         "keibi_test_psql",
		Repository:   "postgres",
		Tag:          "latest",
		ExposedPorts: []string{"5432"},
		Env: []string{
			"POSTGRES_PASSWORD=mysecretpassword",
		},
		Networks: []*dockertest.Network{net},
	})
	require.NoError(err, "status postgres")

	t.Cleanup(func() {
		err := pool.Purge(postgresResource)
		require.NoError(err, "purge resource %s", postgresResource)
	})

	azureSpc, err := BuildTempSpecFile("azure", "http://keibi_test_es:9200")
	t.Cleanup(func() {
		os.Remove(azureSpc)
	})
	require.NoError(err, "azure spc file")

	awsSpc, err := BuildTempSpecFile("aws", "http://keibi_test_es:9200")
	t.Cleanup(func() {
		os.Remove(awsSpc)
	})
	require.NoError(err, "aws spc file")

	azureAdSpc, err := BuildTempSpecFile("azuread", "http://keibi_test_es:9200")
	t.Cleanup(func() {
		os.Remove(azureAdSpc)
	})
	require.NoError(err, "azuread spc file")

	steampipeResource, err := pool.RunWithOptions(
		&dockertest.RunOptions{
			Name:       "keibi_test_steampipe",
			Repository: "registry.digitalocean.com/keibi/steampipe-service",
			Tag:        "0.0.1",
			Mounts: []string{
				azureSpc + ":/home/steampipe/.steampipe/config/azure.spc",
				awsSpc + ":/home/steampipe/.steampipe/config/aws.spc",
				azureAdSpc + ":/home/steampipe/.steampipe/config/azuread.spc",
			},
			Env: []string{
				"STEAMPIPE_LOG=trace",
			},
			Cmd:          []string{"steampipe", "service", "start", "--database-listen", "network", "--database-port", "9193", "--database-password", "abcd", "--foreground"},
			ExposedPorts: []string{"9193"},
			Networks:     []*dockertest.Network{net},
		},
	)
	require.NoError(err, "status steampipe")

	t.Cleanup(func() {
		err := pool.Purge(steampipeResource)
		require.NoError(err, "purge resource %s", steampipeResource)
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

	var orm *gorm.DB
	// exponential backoff-retry, because the application in the container might not be ready to accept connections yet
	err = pool.Retry(func() error {
		orm, err = gorm.Open(postgres.Open(fmt.Sprintf("postgres://postgres:mysecretpassword@localhost:%s/postgres", postgresResource.GetPort("5432/tcp"))), &gorm.Config{})
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

	db := Database{orm: orm}
	err = db.Initialize()
	require.NoError(err, "initializing postgres")

	err = PopulatePostgres(db)
	require.NoError(err, "populating postgres")

	// exponential backoff-retry, because the application in the container might not be ready to accept connections yet
	err = pool.Retry(func() error {
		orm, err = gorm.Open(postgres.Open(fmt.Sprintf("postgres://steampipe:abcd@localhost:%s/steampipe", steampipeResource.GetPort("9193/tcp"))), &gorm.Config{})
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

	s.router = InitializeRouter()
	s.handler, _ = InitializeHttpHandler(esUrl, "", "",
		"localhost", postgresResource.GetPort("5432/tcp"), "postgres", "postgres", "mysecretpassword",
		"localhost", steampipeResource.GetPort("9193/tcp"), "steampipe", "steampipe", "abcd",
	)

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
		Sorts:   []ResourceSortItem{},
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
		Sorts: []ResourceSortItem{
			{
				Field:     SortFieldResourceID,
				Direction: DirectionDescending,
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
		Sorts: []ResourceSortItem{
			{
				Field:     SortFieldResourceID,
				Direction: DirectionDescending,
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
		Sorts: []ResourceSortItem{
			{
				Field:     SortFieldResourceID,
				Direction: DirectionAscending,
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

	req.Filters = Filters{}
	req.Filters.ResourceType = []string{"AWS::EC2::Instance"}
	req.Filters.Location = []string{"us-east1"}
	rec, err = doRequestJSONResponse(s.router, echo.POST, "/api/v1/resources", req, &response)
	require.NoError(err, "request")
	require.Equal(http.StatusOK, rec.Code)
	require.Len(response.Resources, 1)
	require.Equal(response.Resources[0].ResourceID, "aaa0")
}

func (s *HttpHandlerSuite) TestGetAllResources_Query() {
	require := s.Require()

	req := GetResourcesRequest{
		Query: "EC2",
		Filters: Filters{
			Location: []string{"us-east1"},
		},
		Sorts: []ResourceSortItem{
			{
				Field:     SortFieldResourceID,
				Direction: DirectionAscending,
			},
		},
		Page: Page{
			Size: 10,
		},
	}

	var response GetResourcesResponse
	rec, err := doRequestJSONResponse(s.router, echo.POST, "/api/v1/resources", req, &response)
	require.NoError(err, "request")
	require.Equal(http.StatusOK, rec.Code)
	require.Len(response.Resources, 1)
	require.Equal(response.Resources[0].ResourceID, "aaa0")

	req.Query = "Microsoft"
	rec, err = doRequestJSONResponse(s.router, echo.POST, "/api/v1/resources", req, &response)
	require.NoError(err, "request")
	require.Equal(http.StatusOK, rec.Code)
	require.Len(response.Resources, 1)
	require.Equal(response.Resources[0].ResourceID, "aaa2")
}

func (s *HttpHandlerSuite) TestGetAllResources_CSV() {
	require := s.Require()

	req := GetResourcesRequest{
		Filters: Filters{},
		Sorts: []ResourceSortItem{
			{
				Field:     SortFieldResourceID,
				Direction: DirectionAscending,
			},
		},
		Page: Page{
			Size: 10,
		},
	}

	rec, response, err := doRequestCSVResponse(s.router, echo.POST, "/api/v1/resources", req)
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
	rec, response, err = doRequestCSVResponse(s.router, echo.POST, "/api/v1/resources", req)
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
		Sorts: []ResourceSortItem{},
		Page: Page{
			NextMarker: "",
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
		Sorts: []ResourceSortItem{},
		Page: Page{
			NextMarker: "",
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

func (s *HttpHandlerSuite) TestGetQueries() {
	require := s.Require()

	var response []SmartQuery
	rec, err := doRequestJSONResponse(s.router, echo.GET, "/api/v1/query", nil, &response)
	require.NoError(err, "request")
	require.Equal(http.StatusOK, rec.Code)
	require.Len(response, 4)
}

func (s *HttpHandlerSuite) TestRunQuery() {
	require := s.Require()

	var queryList []SmartQuery
	rec, err := doRequestJSONResponse(s.router, echo.GET, "/api/v1/query", nil, &queryList)
	require.NoError(err, "request")
	require.Equal(http.StatusOK, rec.Code)
	require.Len(queryList, 4)

	req := RunQueryRequest{
		Page: Page{
			"", 10,
		},
	}
	var response RunQueryResponse
	rec, err = doRequestJSONResponse(s.router, echo.POST, "/api/v1/query/"+queryList[0].ID.String(), &req, &response)
	require.NoError(err, "request")
	require.Equal(http.StatusOK, rec.Code)
	require.Len(response.Result, 1)
	require.Len(response.Result[0], 1)
	require.Equal(float64(1), response.Result[0][0])
}

func (s *HttpHandlerSuite) TestRunQuery_Sort() {
	require := s.Require()

	var queryList []SmartQuery
	rec, err := doRequestJSONResponse(s.router, echo.GET, "/api/v1/query", nil, &queryList)
	require.NoError(err, "request")
	require.Equal(http.StatusOK, rec.Code)
	require.Len(queryList, 4)

	req := RunQueryRequest{
		Page: Page{
			"", 10,
		},
		Sorts: []SmartQuerySortItem{
			{
				Field:     "subscription_id",
				Direction: DirectionAscending,
			},
		},
	}
	var response RunQueryResponse
	rec, err = doRequestJSONResponse(s.router, echo.POST, "/api/v1/query/"+queryList[2].ID.String(), &req, &response)
	require.NoError(err, "request")
	require.Equal(http.StatusOK, rec.Code)
	require.Len(response.Result, 2)
	require.Greater(len(response.Result[0]), 0)
	require.Equal("ss1", response.Result[0][17])
	require.Equal("ss2", response.Result[1][17])

	req = RunQueryRequest{
		Page: Page{
			"", 10,
		},
		Sorts: []SmartQuerySortItem{
			{
				Field:     "subscription_id",
				Direction: DirectionDescending,
			},
		},
	}
	rec, err = doRequestJSONResponse(s.router, echo.POST, "/api/v1/query/"+queryList[2].ID.String(), &req, &response)
	require.NoError(err, "request")
	require.Equal(http.StatusOK, rec.Code)
	require.Len(response.Result, 2)
	require.Greater(len(response.Result[0]), 0)
	require.Equal("ss2", response.Result[0][17])
	require.Equal("ss1", response.Result[1][17])
}

func (s *HttpHandlerSuite) TestRunQuery_Page() {
	require := s.Require()

	var queryList []SmartQuery
	rec, err := doRequestJSONResponse(s.router, echo.GET, "/api/v1/query", nil, &queryList)
	require.NoError(err, "request")
	require.Equal(http.StatusOK, rec.Code)
	require.Len(queryList, 4)

	req := RunQueryRequest{
		Page: Page{
			"", 1,
		},
		Sorts: []SmartQuerySortItem{
			{
				Field:     "subscription_id",
				Direction: DirectionAscending,
			},
		},
	}
	var response RunQueryResponse
	rec, err = doRequestJSONResponse(s.router, echo.POST, "/api/v1/query/"+queryList[2].ID.String(), &req, &response)
	require.NoError(err, "request")
	require.Equal(http.StatusOK, rec.Code)
	require.Len(response.Result, 1)
	require.Greater(len(response.Result[0]), 0)
	require.Equal("ss1", response.Result[0][17])

	req.Page = response.Page
	rec, err = doRequestJSONResponse(s.router, echo.POST, "/api/v1/query/"+queryList[2].ID.String(), &req, &response)
	require.NoError(err, "request")
	require.Equal(http.StatusOK, rec.Code)
	require.Len(response.Result, 1)
	require.Greater(len(response.Result[0]), 0)
	require.Equal("ss2", response.Result[0][17])
}

func (s *HttpHandlerSuite) TestRunQuery_CSV() {
	require := s.Require()

	var queryList []SmartQuery
	rec, err := doRequestJSONResponse(s.router, echo.GET, "/api/v1/query", nil, &queryList)
	require.NoError(err, "request")
	require.Equal(http.StatusOK, rec.Code)
	require.Len(queryList, 4)

	req := RunQueryRequest{
		Page: Page{
			"", 1,
		},
		Sorts: []SmartQuerySortItem{
			{
				Field:     "subscription_id",
				Direction: DirectionAscending,
			},
		},
	}
	rec, response, err := doRequestCSVResponse(s.router, echo.POST, "/api/v1/query/"+queryList[2].ID.String(), &req)
	require.NoError(err, "request")
	require.Equal(http.StatusOK, rec.Code)
	require.Len(response, 3) // first is header
	require.Equal(response[0][17], "subscription_id")
	require.Equal(response[1][17], "ss1")
	require.Equal(response[2][17], "ss2")
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

	rec, resp, err := sendRequest(router, method, path, requestBytes, "application/json")
	if err != nil {
		return nil, err
	}

	if response != nil {
		if err := json.Unmarshal(resp, response); err != nil {
			fmt.Println("Unmarshal error on response: " + string(resp))
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

	rec, resp, err := sendRequest(router, method, path, requestBytes, "text/csv")
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

func sendRequest(router *echo.Echo, method string, path string, request []byte, accept string) (*httptest.ResponseRecorder, []byte, error) {
	var r io.Reader
	if request != nil {
		r = bytes.NewReader(request)
	}

	req := httptest.NewRequest(method, path, r)
	req.Header.Add("content-type", "application/json")
	req.Header.Add("accept", accept)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	// Wrap in NopCloser in case the calling method wants to also read the body
	b, err := ioutil.ReadAll(io.NopCloser(rec.Body))
	if err != nil {
		return nil, nil, err
	}
	return rec, b, nil
}
