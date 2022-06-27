package inventory

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	compliance_es "gitlab.com/keibiengine/keibi-engine/pkg/compliance-report/es"
	"go.uber.org/zap"

	api3 "gitlab.com/keibiengine/keibi-engine/pkg/compliance-report/api"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/ory/dockertest/v3"
	"github.com/stretchr/testify/suite"
	compliance_report "gitlab.com/keibiengine/keibi-engine/pkg/compliance-report"
	"gitlab.com/keibiengine/keibi-engine/pkg/compliance-report/test"
	"gitlab.com/keibiengine/keibi-engine/pkg/describe"
	api2 "gitlab.com/keibiengine/keibi-engine/pkg/describe/api"
	pagination "gitlab.com/keibiengine/keibi-engine/pkg/internal/api"
	idocker "gitlab.com/keibiengine/keibi-engine/pkg/internal/dockertest"
	"gitlab.com/keibiengine/keibi-engine/pkg/internal/httpclient"
	"gitlab.com/keibiengine/keibi-engine/pkg/internal/httpserver"
	"gitlab.com/keibiengine/keibi-engine/pkg/internal/postgres"
	"gitlab.com/keibiengine/keibi-engine/pkg/inventory/api"
	"gorm.io/gorm"
)

type HttpHandlerSuite struct {
	suite.Suite

	elasticUrl string
	handler    *HttpHandler
	router     *echo.Echo
	describe   *DescribeMock
}

func (s *HttpHandlerSuite) SetupSuite() {
	t := s.T()
	require := s.Require()

	s.describe = &DescribeMock{}
	s.describe.Run()

	pool, err := dockertest.NewPool("")
	require.NoError(err, "connect to docker")

	net, err := pool.CreateNetwork("keibi")
	require.NoError(err)
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
	s.elasticUrl = fmt.Sprintf("http://%s:", idocker.GetDockerHost()) + elasticResource.GetPort("9200/tcp")

	t.Cleanup(func() {
		err := pool.Purge(elasticResource)
		require.NoError(err, "purge resource %s", elasticResource)
	})

	postgresResource, err := pool.RunWithOptions(&dockertest.RunOptions{
		Name:         "keibi_test_psql",
		Repository:   "postgres",
		Tag:          "12.2-alpine",
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
	time.Sleep(5 * time.Second)

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
	time.Sleep(5 * time.Second)

	// exponential backoff-retry, because the application in the container might not be ready to accept connections yet
	err = pool.Retry(func() error {
		res, err := http.Get(s.elasticUrl)
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

	err = PopulateElastic(s.elasticUrl, s.describe)
	require.NoError(err, "populating elastic")

	logger, err := zap.NewProduction()
	require.NoError(err, "new zap logger")

	var orm *gorm.DB
	// exponential backoff-retry, because the application in the container might not be ready to accept connections yet
	err = pool.Retry(func() error {
		cfg := &postgres.Config{
			Host:   idocker.GetDockerHost(),
			Port:   postgresResource.GetPort("5432/tcp"),
			User:   "postgres",
			Passwd: "mysecretpassword",
			DB:     "postgres",
		}
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

	db := Database{orm: orm}
	err = db.Initialize()
	require.NoError(err, "initializing postgres")

	err = PopulatePostgres(db)
	require.NoError(err, "populating postgres")

	// exponential backoff-retry, because the application in the container might not be ready to accept connections yet
	err = pool.Retry(func() error {
		cfg := &postgres.Config{
			Host:   idocker.GetDockerHost(),
			Port:   steampipeResource.GetPort("9193/tcp"),
			User:   "steampipe",
			Passwd: "abcd",
			DB:     "steampipe",
		}

		orm, err = postgres.NewClient(cfg, logger)
		require.NoError(err, "new postgres client")

		d, err := orm.DB()
		if err != nil {
			return err
		}

		return d.Ping()
	})
	require.NoError(err, "wait for postgres connection")

	s.handler, err = InitializeHttpHandler(s.elasticUrl, "", "",
		idocker.GetDockerHost(), postgresResource.GetPort("5432/tcp"), "postgres", "postgres", "mysecretpassword",
		idocker.GetDockerHost(), steampipeResource.GetPort("9193/tcp"), "steampipe", "steampipe", "abcd",
		s.describe.MockServer.URL, logger,
	)
	require.NoError(err, "init http handler")

	s.router = httpserver.Register(logger, s.handler)
}

func (s *HttpHandlerSuite) BeforeTest(suiteName, testName string) {
	//require := s.Require()
}

func (s *HttpHandlerSuite) AfterTest(suiteName, testName string) {
	//require := s.Require()
}

func (s *HttpHandlerSuite) TestGetAllResources() {
	require := s.Require()

	var response api.GetResourcesResponse
	rec, err := doRequestJSONResponse(s.router, echo.POST, "/api/v1/resources", api.GetResourcesRequest{
		Filters: api.Filters{},
		Sorts:   []api.ResourceSortItem{},
		Page: pagination.PageRequest{
			Size: 10,
		},
	}, &response)
	require.NoError(err, "request")
	require.Equal(http.StatusOK, rec.Code)
	require.Len(response.Resources, 4)
	require.Equal("Name", response.Resources[0].SourceName)
	for _, r := range response.Resources {
		if r.ResourceType == "AWS::EC2::Region" {
			require.Fail("AWS::EC2::Region should be excluded from get resource api")
		}
	}
}

func (s *HttpHandlerSuite) TestGetAllResources_Sort() {
	require := s.Require()

	var response api.GetResourcesResponse
	rec, err := doRequestJSONResponse(s.router, echo.POST, "/api/v1/resources", api.GetResourcesRequest{
		Filters: api.Filters{},
		Sorts: []api.ResourceSortItem{
			{
				Field:     api.SortFieldResourceID,
				Direction: api.DirectionDescending,
			},
		},
		Page: pagination.PageRequest{
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

	req := api.GetResourcesRequest{
		Filters: api.Filters{},
		Sorts: []api.ResourceSortItem{
			{
				Field:     api.SortFieldResourceID,
				Direction: api.DirectionDescending,
			},
		},
		Page: pagination.PageRequest{
			Size: 1,
		},
	}
	var response api.GetResourcesResponse
	rec, err := doRequestJSONResponse(s.router, echo.POST, "/api/v1/resources", req, &response)
	require.NoError(err, "request")
	require.Equal(http.StatusOK, rec.Code)
	require.Len(response.Resources, 1)
	require.Equal("aaa3", response.Resources[0].ResourceID)
	require.Equal(int64(4), response.Page.TotalCount)

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

	req := api.GetResourcesRequest{
		Filters: api.Filters{},
		Sorts: []api.ResourceSortItem{
			{
				Field:     api.SortFieldResourceID,
				Direction: api.DirectionAscending,
			},
		},
		Page: pagination.PageRequest{
			Size: 10,
		},
	}

	var response api.GetResourcesResponse
	req.Filters = api.Filters{}
	req.Filters.Location = []string{"us-east1"}
	rec, err := doRequestJSONResponse(s.router, echo.POST, "/api/v1/resources", req, &response)
	require.NoError(err, "request")
	require.Equal(http.StatusOK, rec.Code)
	require.Len(response.Resources, 2)
	require.Equal(response.Resources[0].ResourceID, "aaa0")
	require.Equal(response.Resources[1].ResourceID, "aaa2")

	req.Filters = api.Filters{}
	req.Filters.SourceID = []string{"ss1"}
	rec, err = doRequestJSONResponse(s.router, echo.POST, "/api/v1/resources", req, &response)
	require.NoError(err, "request")
	require.Equal(http.StatusOK, rec.Code)
	require.Len(response.Resources, 2)
	require.Equal(response.Resources[0].ResourceID, "aaa0")
	require.Equal(response.Resources[1].ResourceID, "aaa1")

	req.Filters = api.Filters{}
	req.Filters.ResourceType = []string{"Microsoft.Network/virtualNetworks"}
	rec, err = doRequestJSONResponse(s.router, echo.POST, "/api/v1/resources", req, &response)
	require.NoError(err, "request")
	require.Equal(http.StatusOK, rec.Code)
	require.Len(response.Resources, 2)
	require.Equal(response.Resources[0].ResourceID, "aaa1")
	require.Equal(response.Resources[1].ResourceID, "aaa2")

	req.Filters = api.Filters{}
	req.Filters.ResourceType = []string{"AWS::EC2::Instance"}
	req.Filters.Location = []string{"us-east1"}
	rec, err = doRequestJSONResponse(s.router, echo.POST, "/api/v1/resources", req, &response)
	require.NoError(err, "request")
	require.Equal(http.StatusOK, rec.Code)
	require.Len(response.Resources, 1)
	require.Equal(response.Resources[0].ResourceID, "abcd")
}

func (s *HttpHandlerSuite) TestGetAllResources_Query() {
	require := s.Require()

	req := api.GetResourcesRequest{
		Query: "EC2",
		Filters: api.Filters{
			Location: []string{"us-east1"},
		},
		Sorts: []api.ResourceSortItem{
			{
				Field:     api.SortFieldResourceID,
				Direction: api.DirectionAscending,
			},
		},
		Page: pagination.PageRequest{
			Size: 10,
		},
	}

	var response api.GetResourcesResponse
	rec, err := doRequestJSONResponse(s.router, echo.POST, "/api/v1/resources", req, &response)
	require.NoError(err, "request")
	require.Equal(http.StatusOK, rec.Code)
	require.Len(response.Resources, 1)
	require.Equal(response.Resources[0].ResourceID, "aaa0")
}

func (s *HttpHandlerSuite) TestGetAllResources_QueryMicrosoft() {
	s.T().Skip("This test fails due to a known bug and we're gonna fix it later")

	require := s.Require()

	req := api.GetResourcesRequest{
		Query: "Microsoft",
		Filters: api.Filters{
			Location: []string{"us-east1"},
		},
		Sorts: []api.ResourceSortItem{
			{
				Field:     api.SortFieldResourceID,
				Direction: api.DirectionAscending,
			},
		},
		Page: pagination.PageRequest{
			Size: 10,
		},
	}

	var response api.GetResourcesResponse
	req.Query = "Microsoft"
	rec, err := doRequestJSONResponse(s.router, echo.POST, "/api/v1/resources", req, &response)
	require.NoError(err, "request")
	require.Equal(http.StatusOK, rec.Code)
	require.Len(response.Resources, 1)
	require.Equal(response.Resources[0].ResourceID, "aaa2")
}

func (s *HttpHandlerSuite) TestGetAllResources_CSV() {
	require := s.Require()

	req := api.GetResourcesRequest{
		Filters: api.Filters{},
		Sorts: []api.ResourceSortItem{
			{
				Field:     api.SortFieldResourceID,
				Direction: api.DirectionAscending,
			},
		},
		Page: pagination.PageRequest{
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

	req.Filters = api.Filters{}
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

	var response api.GetAWSResourceResponse
	rec, err := doRequestJSONResponse(s.router, echo.POST, "/api/v1/resources/aws", api.GetResourcesRequest{
		Filters: api.Filters{
			ResourceType: nil,
			Location:     nil,
			SourceID:     nil,
		},
		Sorts: []api.ResourceSortItem{},
		Page: pagination.PageRequest{
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

	var response api.GetAzureResourceResponse
	rec, err := doRequestJSONResponse(s.router, echo.POST, "/api/v1/resources/azure", api.GetResourcesRequest{
		Filters: api.Filters{
			ResourceType: nil,
			Location:     nil,
			SourceID:     nil,
		},
		Sorts: []api.ResourceSortItem{},
		Page: pagination.PageRequest{
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

func (s *HttpHandlerSuite) TestGetResource() {
	require := s.Require()

	var response map[string]string
	rec, err := doRequestJSONResponse(s.router, echo.POST, "/api/v1/resource", api.GetResourceRequest{
		ResourceType: "AWS::EC2::Instance",
		ID:           "abcd",
	}, &response)
	require.NoError(err, "request")
	require.Equal(http.StatusOK, rec.Code)
	_, ok := response["tags"]
	require.True(ok)
	fmt.Println(response)
}

func (s *HttpHandlerSuite) TestGetQueries() {
	require := s.Require()

	req := api.ListQueryRequest{}
	var response []api.SmartQueryItem

	rec, err := doRequestJSONResponse(s.router, echo.GET, "/api/v1/query", &req, &response)
	require.NoError(err, "request")
	require.Equal(http.StatusOK, rec.Code)
	require.Len(response, 4)
	containsTag := false
	for _, sm := range response {
		if len(sm.Tags) > 0 {
			containsTag = true
		}
	}
	require.True(containsTag)

	var c int
	rec, err = doRequestJSONResponse(s.router, echo.GET, "/api/v1/query/count", &req, &c)
	require.NoError(err, "request")
	require.Equal(http.StatusOK, rec.Code)
	require.Equal(c, 4)

	req.TitleFilter = "4"
	rec, err = doRequestJSONResponse(s.router, echo.GET, "/api/v1/query", &req, &response)
	require.NoError(err, "request")
	require.Equal(http.StatusOK, rec.Code)
	require.Len(response, 1)

	req.TitleFilter = ""
	req.Labels = []string{"tag1"}
	rec, err = doRequestJSONResponse(s.router, echo.GET, "/api/v1/query", &req, &response)
	require.NoError(err, "request")
	require.Equal(http.StatusOK, rec.Code)
	require.Len(response, 1)
}

func (s *HttpHandlerSuite) TestRunQuery() {
	require := s.Require()

	var queryList []api.SmartQueryItem
	rec, err := doRequestJSONResponse(s.router, echo.GET, "/api/v1/query", nil, &queryList)
	require.NoError(err, "request")
	require.Equal(http.StatusOK, rec.Code)
	require.Len(queryList, 4)

	var id uint
	for _, q := range queryList {
		if q.Title == "Query 1" {
			id = q.ID
		}
	}
	req := api.RunQueryRequest{
		Page: pagination.PageRequest{
			NextMarker: "",
			Size:       10,
		},
	}
	var response api.RunQueryResponse
	rec, err = doRequestJSONResponse(s.router, echo.POST, fmt.Sprintf("/api/v1/query/%d", id), &req, &response)
	require.NoError(err, "request")
	require.Equal(http.StatusOK, rec.Code)
	require.Len(response.Result, 1)
	require.Len(response.Result[0], 1)
	require.Equal(float64(1), response.Result[0][0])
}

func (s *HttpHandlerSuite) TestRunQuery_Sort() {
	require := s.Require()

	var queryList []api.SmartQueryItem
	rec, err := doRequestJSONResponse(s.router, echo.GET, "/api/v1/query", nil, &queryList)
	require.NoError(err, "request")
	require.Equal(http.StatusOK, rec.Code)
	require.Len(queryList, 4)

	req := api.RunQueryRequest{
		Page: pagination.PageRequest{
			NextMarker: "",
			Size:       10,
		},
		Sorts: []api.SmartQuerySortItem{
			{
				Field:     "subscription_id",
				Direction: api.DirectionAscending,
			},
		},
	}
	var id uint
	for _, q := range queryList {
		if q.Title == "Query 3" {
			id = q.ID
		}
	}
	var response api.RunQueryResponse
	rec, err = doRequestJSONResponse(s.router, echo.POST, fmt.Sprintf("/api/v1/query/%d", id), &req, &response)
	require.NoError(err, "request")
	require.Equal(http.StatusOK, rec.Code)
	require.Len(response.Result, 2)
	require.Greater(len(response.Result[0]), 0)
	require.Equal("ss1", response.Result[0][17])
	require.Equal("ss2", response.Result[1][17])

	req = api.RunQueryRequest{
		Page: pagination.PageRequest{
			NextMarker: "",
			Size:       10,
		},
		Sorts: []api.SmartQuerySortItem{
			{
				Field:     "subscription_id",
				Direction: api.DirectionDescending,
			},
		},
	}
	rec, err = doRequestJSONResponse(s.router, echo.POST, fmt.Sprintf("/api/v1/query/%d", id), &req, &response)
	require.NoError(err, "request")
	require.Equal(http.StatusOK, rec.Code)
	require.Len(response.Result, 2)
	require.Greater(len(response.Result[0]), 0)
	require.Equal("ss2", response.Result[0][17])
	require.Equal("ss1", response.Result[1][17])
}

func (s *HttpHandlerSuite) TestRunQuery_Page() {
	require := s.Require()

	var queryList []api.SmartQueryItem
	rec, err := doRequestJSONResponse(s.router, echo.GET, "/api/v1/query", nil, &queryList)
	require.NoError(err, "request")
	require.Equal(http.StatusOK, rec.Code)
	require.Len(queryList, 4)

	var id uint
	for _, q := range queryList {
		if q.Title == "Query 3" {
			id = q.ID
		}
	}
	req := api.RunQueryRequest{
		Page: pagination.PageRequest{
			NextMarker: "",
			Size:       1,
		},
		Sorts: []api.SmartQuerySortItem{
			{
				Field:     "subscription_id",
				Direction: api.DirectionAscending,
			},
		},
	}
	var response api.RunQueryResponse
	rec, err = doRequestJSONResponse(s.router, echo.POST, fmt.Sprintf("/api/v1/query/%d", id), &req, &response)
	require.NoError(err, "request")
	require.Equal(http.StatusOK, rec.Code)
	require.Len(response.Result, 1)
	require.Greater(len(response.Result[0]), 0)
	require.Equal("ss1", response.Result[0][17])

	req.Page = response.Page.ToRequest()
	rec, err = doRequestJSONResponse(s.router, echo.POST, fmt.Sprintf("/api/v1/query/%d", id), &req, &response)
	require.NoError(err, "request")
	require.Equal(http.StatusOK, rec.Code)
	require.Len(response.Result, 1)
	require.Greater(len(response.Result[0]), 0)
	require.Equal("ss2", response.Result[0][17])
}

func (s *HttpHandlerSuite) TestRunQuery_CSV() {
	require := s.Require()

	var queryList []api.SmartQueryItem
	rec, err := doRequestJSONResponse(s.router, echo.GET, "/api/v1/query", nil, &queryList)
	require.NoError(err, "request")
	require.Equal(http.StatusOK, rec.Code)
	require.Len(queryList, 4)

	var id uint
	for _, q := range queryList {
		if q.Title == "Query 3" {
			id = q.ID
		}
	}
	req := api.RunQueryRequest{
		Page: pagination.PageRequest{
			NextMarker: "",
			Size:       1,
		},
		Sorts: []api.SmartQuerySortItem{
			{
				Field:     "subscription_id",
				Direction: api.DirectionAscending,
			},
		},
	}
	rec, response, err := doRequestCSVResponse(s.router, echo.POST, fmt.Sprintf("/api/v1/query/%d", id), &req)
	require.NoError(err, "request")
	require.Equal(http.StatusOK, rec.Code)
	require.Len(response, 3) // first is header
	require.Equal(response[0][17], "subscription_id")
	require.Equal(response[1][17], "ss1")
	require.Equal(response[2][17], "ss2")
}

func (s *HttpHandlerSuite) TestGetComplianceReport_Benchmark() {
	s.T().Skip("deprecated")
	require := s.Require()

	err := test.PopulateElastic(s.elasticUrl)
	require.NoError(err)

	uuid1, _ := uuid.Parse("c29c0dae-823f-4726-ade0-5fa94a941e88")

	source := api2.Source{
		ID:   uuid1,
		Type: api2.SourceCloudAWS,
	}
	date1 := time.Now().UnixMilli()
	j1 := describe.ComplianceReportJob{
		Model: gorm.Model{
			ID:        1,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			DeletedAt: gorm.DeletedAt{},
		},
		SourceID:       source.ID,
		Status:         api3.ComplianceReportJobCompleted,
		FailureMessage: "",
	}
	time.Sleep(10 * time.Millisecond)
	date2 := time.Now().UnixMilli()
	time.Sleep(10 * time.Millisecond)
	j2 := describe.ComplianceReportJob{
		Model: gorm.Model{
			ID:        2,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			DeletedAt: gorm.DeletedAt{},
		},
		SourceID:       source.ID,
		Status:         api3.ComplianceReportJobCompleted,
		FailureMessage: "",
	}
	time.Sleep(10 * time.Millisecond)
	require.NoError(err)
	date3 := time.Now().UnixMilli()

	s.describe.SetResponse(j1, j2)

	var res api.GetComplianceReportResponse
	_, err = doRequestJSONResponse(s.router, "GET",
		"/api/v1/reports/compliance/"+source.ID.String(), &api.GetComplianceReportRequest{
			ReportType: compliance_report.ReportTypeBenchmark,
			Page:       pagination.PageRequest{Size: 10},
		}, &res)
	require.NoError(err)
	require.True(len(res.Reports) > 0)
	for _, report := range res.Reports {
		require.NotNil(report.Group)
		require.Equal(2, int(report.ReportJobId))
	}

	res = api.GetComplianceReportResponse{}
	_, err = doRequestJSONResponse(s.router, "GET",
		"/api/v1/reports/compliance/"+source.ID.String(), &api.GetComplianceReportRequest{
			Filters: api.ComplianceReportFilters{
				TimeRange: &api.TimeRangeFilter{
					From: date1,
					To:   date2,
				},
			},
			ReportType: compliance_report.ReportTypeBenchmark,
			Page:       pagination.PageRequest{Size: 10},
		}, &res)
	require.NoError(err)
	require.True(len(res.Reports) > 0)
	for _, report := range res.Reports {
		require.NotNil(report.Group)
		require.Equal(1, int(report.ReportJobId))
	}

	res = api.GetComplianceReportResponse{}
	_, err = doRequestJSONResponse(s.router, "GET",
		"/api/v1/reports/compliance/"+source.ID.String(), &api.GetComplianceReportRequest{
			Filters: api.ComplianceReportFilters{
				TimeRange: &api.TimeRangeFilter{
					From: date2,
					To:   date3,
				},
			},
			ReportType: compliance_report.ReportTypeBenchmark,
			Page:       pagination.PageRequest{Size: 10},
		}, &res)
	require.NoError(err)
	require.True(len(res.Reports) > 0)
	for _, report := range res.Reports {
		require.NotNil(report.Group)
		require.Equal(2, int(report.ReportJobId))
	}

	res = api.GetComplianceReportResponse{}
	groupID := "cis.001"
	_, err = doRequestJSONResponse(s.router, "GET",
		"/api/v1/reports/compliance/"+source.ID.String(), &api.GetComplianceReportRequest{
			Filters: api.ComplianceReportFilters{
				GroupID: &groupID,
			},
			ReportType: compliance_report.ReportTypeBenchmark,
			Page:       pagination.PageRequest{Size: 10},
		}, &res)
	require.NoError(err)
	require.True(len(res.Reports) > 0)
	for _, report := range res.Reports {
		require.NotNil(report.Group)
		require.Equal(groupID, report.Group.ID)
	}

	res = api.GetComplianceReportResponse{}
	_, err = doRequestJSONResponse(s.router, "GET",
		"/api/v1/reports/compliance/"+source.ID.String()+"/1", &api.GetComplianceReportRequest{
			ReportType: compliance_report.ReportTypeBenchmark,
			Page:       pagination.PageRequest{Size: 10},
		}, &res)
	require.NoError(err)
	require.True(len(res.Reports) > 0)
	for _, report := range res.Reports {
		require.NotNil(report.Group)
		require.Equal(1, int(report.ReportJobId))
	}
}

func (s *HttpHandlerSuite) TestGetComplianceReport_Result() {
	s.T().Skip("deprecated")
	require := s.Require()

	err := test.PopulateElastic(s.elasticUrl)
	require.NoError(err)

	uuid1, _ := uuid.Parse("c29c0dae-823f-4726-ade0-5fa94a941e88")
	source := api2.Source{
		ID:   uuid1,
		Type: api2.SourceCloudAWS,
	}

	j1 := describe.ComplianceReportJob{
		Model: gorm.Model{
			ID:        1,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			DeletedAt: gorm.DeletedAt{},
		},
		SourceID: source.ID,
		Status:   api3.ComplianceReportJobCompleted,
	}
	require.NoError(err)
	s.describe.SetResponse(j1)

	var res api.GetComplianceReportResponse
	_, err = doRequestJSONResponse(s.router, "GET",
		"/api/v1/reports/compliance/"+source.ID.String(), &api.GetComplianceReportRequest{
			ReportType: compliance_report.ReportTypeResult,
			Page:       pagination.PageRequest{Size: 10},
		}, &res)
	require.NoError(err)
	require.True(len(res.Reports) > 0)
	for _, report := range res.Reports {
		require.NotNil(report.Result)
		require.Equal(1, int(report.ReportJobId))
	}
}

func (s *HttpHandlerSuite) TestGetBenchmarks() {
	require := s.Require()

	var res []api.Benchmark
	_, err := doRequestJSONResponse(s.router, "GET",
		"/api/v1/benchmarks", nil, &res)
	require.NoError(err)
	require.Len(res, 2)

	_, err = doRequestJSONResponse(s.router, "GET",
		"/api/v1/benchmarks?provider=AWS", nil, &res)
	require.NoError(err)
	require.Len(res, 1)

	_, err = doRequestJSONResponse(s.router, "GET",
		"/api/v1/benchmarks?provider=Azure", nil, &res)
	require.NoError(err)
	require.Len(res, 1)

	_, err = doRequestJSONResponse(s.router, "GET",
		"/api/v1/benchmarks?tagKey=tagValue", nil, &res)
	require.NoError(err)
	require.Len(res, 2)

	_, err = doRequestJSONResponse(s.router, "GET",
		"/api/v1/benchmarks?tag1=val1", nil, &res)
	require.NoError(err)
	require.Len(res, 1)

	_, err = doRequestJSONResponse(s.router, "GET",
		"/api/v1/benchmarks?tag1=val2", nil, &res)
	require.NoError(err)
	require.Len(res, 0)
}

func (s *HttpHandlerSuite) TestGetBenchmarkTags() {
	require := s.Require()

	var res []api.GetBenchmarkTag
	_, err := doRequestJSONResponse(s.router, "GET",
		"/api/v1/benchmarks/tags", nil, &res)
	require.NoError(err)
	require.True(len(res) > 0)
	for _, b := range res {
		require.Equal(b.Count, 1)
	}
}

func (s *HttpHandlerSuite) TestGetBenchmarkDetails() {
	s.T().Skip("deprecated")
	require := s.Require()
	var res api.GetBenchmarkDetailsResponse
	_, err := doRequestJSONResponse(s.router, "GET",
		"/api/v1/benchmarks/test_compliance.benchmark1", nil, &res)
	require.NoError(err)
	require.Len(res.Sections, 1)
	require.Len(res.Categories, 1)
	require.Len(res.Subcategories, 1)
}

func (s *HttpHandlerSuite) TestGetPolicies() {
	require := s.Require()

	var res []api.Policy
	_, err := doRequestJSONResponse(s.router, "GET",
		"/api/v1/benchmarks/test_compliance.benchmark1/policies", nil, &res)
	require.NoError(err)
	require.Len(res, 1)
	for _, policy := range res {
		require.Equal("test_compliance.benchmark1.policy1", policy.ID)
		require.Equal("category1", policy.Category)
	}

	_, err = doRequestJSONResponse(s.router, "GET",
		"/api/v1/benchmarks/mod.azure_compliance/policies?category=category2", nil, &res)
	require.NoError(err)
	require.Len(res, 1)
	for _, policy := range res {
		require.Equal("category2", policy.Category)
	}

	_, err = doRequestJSONResponse(s.router, "GET",
		"/api/v1/benchmarks/mod.azure_compliance/policies?category=category10", nil, &res)
	require.NoError(err)
	require.Len(res, 0)
}

func (s *HttpHandlerSuite) TestGetBenchmarkResult() {
	s.T().Skip("deprecated")
	require := s.Require()

	sourceId, err := uuid.Parse("2a87b978-b8bf-4d7e-bc19-cf0a99a430cf")
	require.NoError(err)

	j1 := describe.ComplianceReportJob{
		Model: gorm.Model{
			ID:        1020,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			DeletedAt: gorm.DeletedAt{},
		},
		SourceID:       sourceId,
		Status:         api3.ComplianceReportJobCompleted,
		FailureMessage: "",
	}
	s.describe.SetResponse(j1)

	var res compliance_report.ReportGroupObj
	_, err = doRequestJSONResponse(s.router, "GET",
		"/api/v1/benchmarks/mod.azure_compliance/2a87b978-b8bf-4d7e-bc19-cf0a99a430cf/result", nil, &res)
	require.NoError(err)
	require.Equal(2783, res.Summary.Status.OK)
}

func (s *HttpHandlerSuite) TestGetBenchmarkResultPolicies() {
	s.T().Skip("deprecated")
	require := s.Require()

	sourceId, err := uuid.Parse("2a87b978-b8bf-4d7e-bc19-cf0a99a430cf")
	require.NoError(err)

	j1 := describe.ComplianceReportJob{
		Model: gorm.Model{
			ID:        1020,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			DeletedAt: gorm.DeletedAt{},
		},
		SourceID:       sourceId,
		Status:         api3.ComplianceReportJobCompleted,
		FailureMessage: "",
	}
	s.describe.SetResponse(j1)

	var res []*api.PolicyResult
	_, err = doRequestJSONResponse(s.router, "GET",
		"/api/v1/benchmarks/mod.azure_compliance/2a87b978-b8bf-4d7e-bc19-cf0a99a430cf/result/policies", nil, &res)
	require.NoError(err)
	require.Len(res, 2)
	require.Equal(api.PolicyResultStatusFailed, res[0].Status)
	require.Equal(api.PolicyResultStatusPassed, res[1].Status)

	require.Equal(3, res[0].TotalResources)
	require.Equal(0, res[0].CompliantResources)

	require.Equal(0, res[1].TotalResources)
	require.Equal(0, res[1].CompliantResources)

	_, err = doRequestJSONResponse(s.router, "GET",
		"/api/v1/benchmarks/mod.azure_compliance/2a87b978-b8bf-4d7e-bc19-cf0a99a430cf/result/policies?status=passed", nil, &res)
	require.NoError(err)
	require.Len(res, 1)
	require.Equal(api.PolicyResultStatusPassed, res[0].Status)

	_, err = doRequestJSONResponse(s.router, "GET",
		"/api/v1/benchmarks/mod.azure_compliance/2a87b978-b8bf-4d7e-bc19-cf0a99a430cf/result/policies?severity=warn", nil, &res)
	require.NoError(err)
	require.Len(res, 1)
	require.Equal(api.PolicyResultStatusPassed, res[0].Status)

	_, err = doRequestJSONResponse(s.router, "GET",
		"/api/v1/benchmarks/mod.azure_compliance/2a87b978-b8bf-4d7e-bc19-cf0a99a430cf/result/policies?section=section2", nil, &res)
	require.NoError(err)
	require.Len(res, 1)
	require.Equal(api.PolicyResultStatusFailed, res[0].Status)
}

func (s *HttpHandlerSuite) TestGetBenchmarksInTime() {
	require := s.Require()

	var res []api.Benchmark
	_, err := doRequestJSONResponse(s.router, "GET",
		"/api/v1/benchmarks/history/list/Azure/"+strconv.FormatInt(time.Now().UnixMilli(), 10), nil, &res)
	require.NoError(err)
	require.Len(res, 1)
	require.Equal("azure_compliance.benchmark.cis_v130", res[0].ID)
}

func (s *HttpHandlerSuite) TestGetBenchmarkComplianceTrend() {
	require := s.Require()

	sourceId, err := uuid.Parse("2a87b978-b8bf-4d7e-bc19-cf0a99a430cf")
	require.NoError(err)

	var res []api.TrendDataPoint
	_, err = doRequestJSONResponse(s.router, "GET",
		"/api/v1/benchmarks/mod.azure_compliance/"+sourceId.String()+"/compliance/trend", nil, &res)
	require.NoError(err)
	require.Len(res, 2)
}

func (s *HttpHandlerSuite) TestGetBenchmarkAccountCompliance() {
	require := s.Require()
	sourceID, err := uuid.Parse("2a87b978-b8bf-4d7e-bc19-cf0a99a430cf")
	require.NoError(err)

	t := time.Now()
	start := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())

	cr, err := s.handler.schedulerClient.ListComplianceReportJobs(&httpclient.Context{}, sourceID.String(), &api.TimeRangeFilter{
		From: start.UnixMilli(),
		To:   t.UnixMilli(),
	})
	require.NoError(err)

	var reportTimes []int64
	for _, c := range cr {
		reportTimes = append(reportTimes, c.ReportCreatedAt)
	}

	url := fmt.Sprintf("/api/v1/benchmarks/%s/%d/accounts/compliance",
		"azure_compliance.benchmark.cis_v130",
		reportTimes[0],
	)
	var res api.BenchmarkAccountComplianceResponse
	_, err = doRequestJSONResponse(s.router, "GET", url, nil, &res)

	require.NoError(err)
	require.Equal(1, res.TotalNonCompliantAccounts)
	require.Equal(0, res.TotalCompliantAccounts)
}

func (s *HttpHandlerSuite) TestGetBenchmarkAccounts() {
	require := s.Require()
	sourceID, err := uuid.Parse("2a87b978-b8bf-4d7e-bc19-cf0a99a430cf")
	require.NoError(err)

	t := time.Now()
	start := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())

	cr, err := s.handler.schedulerClient.ListComplianceReportJobs(&httpclient.Context{}, sourceID.String(), &api.TimeRangeFilter{
		From: start.UnixMilli(),
		To:   t.UnixMilli(),
	})
	require.NoError(err)

	var reportTimes []int64
	for _, c := range cr {
		reportTimes = append(reportTimes, c.ReportCreatedAt)
	}

	url := fmt.Sprintf("/api/v1/benchmarks/%s/%d/accounts?order=asc&size=10",
		"azure_compliance.benchmark.cis_v130",
		reportTimes[0],
	)
	var res []compliance_report.AccountReport
	_, err = doRequestJSONResponse(s.router, "GET", url, nil, &res)

	require.NoError(err)
	require.Len(res, 1)
	require.Equal(res[0].CompliancePercentage, 0.99)
}

func (s *HttpHandlerSuite) TestGetResourceGrowthTrendOfProvider() {
	require := s.Require()

	url := "/api/v1/resources/trend?provider=AWS&timeWindow=24h"
	var res []api.TrendDataPoint
	_, err := doRequestJSONResponse(s.router, "GET", url, nil, &res)

	require.NoError(err)
	require.Len(res, 3)
	fmt.Println(res)
	require.Equal(int64(20), res[0].Value)
	require.Equal(int64(20), res[1].Value)
	require.Equal(int64(30), res[2].Value)
}

func (s *HttpHandlerSuite) TestGetResourceGrowthTrendOfAccount() {
	require := s.Require()
	sourceID, err := uuid.Parse("2a87b978-b8bf-4d7e-bc19-cf0a99a430cf")
	require.NoError(err)

	url := fmt.Sprintf("/api/v1/resources/trend?sourceId=%s&timeWindow=24h", sourceID.String())
	var res []api.TrendDataPoint
	_, err = doRequestJSONResponse(s.router, "GET", url, nil, &res)

	require.NoError(err)
	require.Len(res, 3)
	require.Equal(int64(10), res[0].Value)
	require.Equal(int64(20), res[1].Value)
	require.Equal(int64(30), res[2].Value)
}

func (s *HttpHandlerSuite) TestGetCompliancyTrendOfProvider() {
	require := s.Require()

	url := "/api/v1/compliancy/trend?provider=AWS&timeWindow=24h"
	var res []api.TrendDataPoint
	_, err := doRequestJSONResponse(s.router, "GET", url, nil, &res)

	require.NoError(err)
	require.Len(res, 3)
	require.Equal(int64(10), res[0].Value)
	require.Equal(int64(15), res[1].Value)
	require.Equal(int64(25), res[2].Value)
}

func (s *HttpHandlerSuite) TestGetCompliancyTrendOfAccount() {
	require := s.Require()
	sourceID, err := uuid.Parse("2a87b978-b8bf-4d7e-bc19-cf0a99a430cf")
	require.NoError(err)

	url := fmt.Sprintf("/api/v1/compliancy/trend?sourceId=%s&timeWindow=24h", sourceID.String())
	var res []api.TrendDataPoint
	_, err = doRequestJSONResponse(s.router, "GET", url, nil, &res)

	require.NoError(err)
	require.Len(res, 3)
	require.Equal(int64(5), res[0].Value)
	require.Equal(int64(15), res[1].Value)
	require.Equal(int64(25), res[2].Value)
}

func (s *HttpHandlerSuite) TestGetTopNAccount() {
	require := s.Require()

	url := "/api/v1/resources/top/accounts?count=1&provider=AWS"
	var res []api.TopAccountResponse
	_, err := doRequestJSONResponse(s.router, "GET", url, nil, &res)

	require.NoError(err)
	require.Len(res, 1)
	require.Equal("2a87b978-b8bf-4d7e-bc19-cf0a99a430cf", res[0].SourceID)
	require.Equal(20, res[0].ResourceCount)
}

func (s *HttpHandlerSuite) TestGetTopNServices() {
	require := s.Require()

	url := "/api/v1/resources/top/services?count=1&provider=AWS"
	var res []api.TopServicesResponse
	_, err := doRequestJSONResponse(s.router, "GET", url, nil, &res)

	require.NoError(err)
	require.Len(res, 1)
	require.Equal("EC2 Instance", res[0].ServiceName)
	require.Equal(20, res[0].ResourceCount)
}

func (s *HttpHandlerSuite) TestGetCategories() {
	require := s.Require()

	url := "/api/v1/resources/categories?provider=AWS"
	var res []api.CategoriesResponse
	_, err := doRequestJSONResponse(s.router, "GET", url, nil, &res)

	require.NoError(err)
	require.Len(res, 2)
	require.Contains(res, api.CategoriesResponse{
		CategoryName:  "Infrastructure",
		ResourceCount: 20,
	})
	require.Contains(res, api.CategoriesResponse{
		CategoryName:  "Security",
		ResourceCount: 10,
	})
}

func (s *HttpHandlerSuite) TestGetListOfBenchmarks() {
	require := s.Require()

	url := "/api/v1/benchmarks/Azure/list?count=5"
	var res []api.BenchmarkScoreResponse
	_, err := doRequestJSONResponse(s.router, "GET", url, nil, &res)

	require.NoError(err)
	require.Len(res, 1)
	require.Contains(res, api.BenchmarkScoreResponse{
		BenchmarkID:       "azure_compliance.benchmark.cis_v130",
		NonCompliantCount: 2,
	})

	url = "/api/v1/benchmarks/Azure/list?count=5&sourceId=2a87b978-b8bf-4d7e-bc19-cf0a99a430cf"
	_, err = doRequestJSONResponse(s.router, "GET", url, nil, &res)

	require.NoError(err)
	require.Len(res, 1)
	require.Contains(res, api.BenchmarkScoreResponse{
		BenchmarkID:       "azure_compliance.benchmark.cis_v130",
		NonCompliantCount: 1,
	})
}

func (s *HttpHandlerSuite) TestGetTopAccountsByCompliancy() {
	require := s.Require()

	url := "/api/v1/benchmarks/compliancy/Azure/top/accounts?count=5&order=desc"
	var res []api.AccountCompliancyResponse
	_, err := doRequestJSONResponse(s.router, "GET", url, nil, &res)

	require.NoError(err)
	require.Len(res, 2)
	sourceID, _ := uuid.Parse("2a87b978-b8bf-4d7e-bc19-cf0a99a430cf")
	require.Contains(res, api.AccountCompliancyResponse{
		SourceID:       sourceID,
		TotalResources: 21,
		TotalCompliant: 20,
	})
}

func (s *HttpHandlerSuite) TestGetTopServicesByCompliancy() {
	require := s.Require()

	url := "/api/v1/benchmarks/compliancy/Azure/top/services?count=5&order=desc"
	var res []api.ServiceCompliancyResponse
	_, err := doRequestJSONResponse(s.router, "GET", url, nil, &res)

	require.NoError(err)
	require.Len(res, 2)
	require.Contains(res, api.ServiceCompliancyResponse{
		ServiceName:    "EC2 Instance",
		TotalResources: 21,
		TotalCompliant: 20,
	})
}

func (s *HttpHandlerSuite) TestCountBenchmarksAndPolicies() {
	require := s.Require()

	url := "/api/v1/benchmarks/count?provider=AWS"
	var res int64
	_, err := doRequestJSONResponse(s.router, "GET", url, nil, &res)

	require.NoError(err)
	require.Equal(int64(1), res)

	url = "/api/v1/policies/count?provider=AWS"
	_, err = doRequestJSONResponse(s.router, "GET", url, nil, &res)

	require.NoError(err)
	require.Equal(int64(1), res)
}

func (s *HttpHandlerSuite) TestGetLocationDistributionOfAccount() {
	require := s.Require()
	sourceID, err := uuid.Parse("2a87b978-b8bf-4d7e-bc19-cf0a99a430cf")
	require.NoError(err)

	url := fmt.Sprintf("/api/v1/resources/distribution?sourceId=%s", sourceID.String())
	var res map[string]int
	_, err = doRequestJSONResponse(s.router, "GET", url, nil, &res)

	require.NoError(err)
	require.Len(res, 2)
}

func (s *HttpHandlerSuite) TestGetServiceDistributionOfAccount() {
	require := s.Require()
	sourceID, err := uuid.Parse("2a87b978-b8bf-4d7e-bc19-cf0a99a430cf")
	require.NoError(err)

	url := fmt.Sprintf("/api/v1/services/distribution?sourceId=%s", sourceID.String())
	var res []api.ServiceDistributionItem
	_, err = doRequestJSONResponse(s.router, "GET", url, nil, &res)

	require.NoError(err)
	require.Len(res, 1)
	require.Equal(res[0].ServiceName, "EC2 Instance")
	require.Equal(res[0].Distribution["us-east-1"], 5)
}

func (s *HttpHandlerSuite) TestGetLocationDistributionOfProvider() {
	require := s.Require()

	url := "/api/v1/resources/distribution?provider=AWS"
	var res map[string]int
	_, err := doRequestJSONResponse(s.router, "GET", url, nil, &res)

	require.NoError(err)
	require.Len(res, 4)
}

func (s *HttpHandlerSuite) TestGetBenchmarkResultSummary() {
	require := s.Require()

	var res compliance_report.SummaryStatus
	_, err := doRequestJSONResponse(s.router, "GET",
		"/api/v1/benchmarks/mod.azure_compliance/result/summary", nil, &res)
	require.NoError(err)

	require.Equal(1, res.OK)
	require.Equal(0, res.Alarm)
}

func (s *HttpHandlerSuite) TestGetBenchmarkResultPoliciesNew() {
	require := s.Require()

	var res []api.ResultPolicy
	_, err := doRequestJSONResponse(s.router, "GET",
		"/api/v1/benchmarks/mod.azure_compliance/result/policies", nil, &res)
	require.NoError(err)

	require.Len(res, 2)
	require.Equal(api.PolicyResultStatusPassed, res[0].Status)
	require.Equal(api.PolicyResultStatusPassed, res[1].Status)
}

func (s *HttpHandlerSuite) TestGetBenchmarkResultCompliancy() {
	require := s.Require()

	var res []api.ResultCompliancy
	_, err := doRequestJSONResponse(s.router, "GET",
		"/api/v1/benchmarks/mod.azure_compliance/result/compliancy", nil, &res)
	require.NoError(err)

	require.Len(res, 2)
	for _, r := range res {
		if r.ID == "control.cis_v130_1_21" {
			require.Equal(api.PolicyResultStatusPassed, r.Status)
			require.Equal(1, r.TotalResources)
			require.Equal(0, r.ResourcesWithIssue)
		}
	}
}

func (s *HttpHandlerSuite) TestGetBenchmarkResultPolicyFindings() {
	require := s.Require()

	var res []compliance_es.Finding
	_, err := doRequestJSONResponse(s.router, "GET",
		"/api/v1/benchmarks/mod.azure_compliance/result/policies/control.cis_v130_1_21/findings", nil, &res)
	require.NoError(err)

	require.Len(res, 1)
	require.Equal("resource1", res[0].ResourceID)
	require.Equal(compliance_report.ResultStatusOK, res[0].Status)
}

func (s *HttpHandlerSuite) TestGetBenchmarkResultPolicyResourcesSummary() {
	require := s.Require()

	var res api.ResultPolicyResourceSummary
	_, err := doRequestJSONResponse(s.router, "GET",
		"/api/v1/benchmarks/mod.azure_compliance/result/policies/control.cis_v130_1_21/resources/summary", nil, &res)
	require.NoError(err)

	require.Equal(1, res.CompliantResourceCount)
	require.Equal(0, res.NonCompliantResourceCount)
	require.Equal(1, res.ResourcesByLocation["ResourceLocation"])
}

func (s *HttpHandlerSuite) TestGetInsightResult() {
	require := s.Require()

	var res api.ListInsightResultsResponse
	_, err := doRequestJSONResponse(s.router, "GET",
		"/api/v1/insight/results", nil, &res)
	require.NoError(err)

	require.Len(res.Results, 3)
	require.Equal(int64(20), res.Results[0].Result)
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

	if rec.Code != 200 {
		return nil, errors.New("not ok response code")
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
