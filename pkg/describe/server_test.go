package describe

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/suite"
	"gitlab.com/keibiengine/keibi-engine/pkg/describe/api"
	idocker "gitlab.com/keibiengine/keibi-engine/pkg/internal/dockertest"
	"gitlab.com/keibiengine/keibi-engine/pkg/internal/httpserver"
	"go.uber.org/zap"
)

type HttpServerSuite struct {
	suite.Suite

	handler *HttpServer
	router  *echo.Echo
}

func (s *HttpServerSuite) SetupSuite() {
	require := s.Require()
	orm := idocker.StartupPostgreSQL(s.T())
	s.handler = NewHTTPServer(":8080", Database{orm: orm}, nil)
	err := s.handler.DB.Initialize()
	require.NoError(err, "db initialize")

	logger, err := zap.NewProduction()
	require.NoError(err, "new logger")
	s.router = httpserver.Register(logger, s.handler)
}

func (s *HttpServerSuite) BeforeTest(suiteName, testName string) {
	//require := s.Require()
}

func (s *HttpServerSuite) AfterTest(suiteName, testName string) {
	//require := s.Require()
}

func (s *HttpServerSuite) TestInsightAPIs() {
	require := s.Require()

	var response uint
	rec, err := doRequestJSONResponse(s.router, echo.PUT, "/api/v1/insight", api.CreateInsightRequest{
		Description:  "No of users",
		Query:        "select count(*) from aws_users;",
		Provider:     "AWS",
		Category:     "IAM",
		SmartQueryID: 0,
	}, &response)
	require.NoError(err, "request")
	require.Equal(http.StatusOK, rec.Code)

	var insightList []api.Insight
	rec, err = doRequestJSONResponse(s.router, echo.GET, "/api/v1/insight", api.ListInsightsRequest{
		DescriptionFilter: "",
	}, &insightList)
	require.NoError(err, "request")
	require.Equal(http.StatusOK, rec.Code)
	require.Len(insightList, 1)
	require.Equal("No of users", insightList[0].Description)
	require.Equal("select count(*) from aws_users;", insightList[0].Query)
	deleteId := insightList[0].ID

	rec, err = doRequestJSONResponse(s.router, echo.PUT, "/api/v1/insight", api.CreateInsightRequest{
		Description:  "count expired certificates",
		Query:        "select count(*) from aws_expired_certificates;",
		Provider:     "AWS",
		Category:     "IAM",
		SmartQueryID: 0,
	}, &response)
	require.NoError(err, "request")
	require.Equal(http.StatusOK, rec.Code)

	rec, err = doRequestJSONResponse(s.router, echo.GET, "/api/v1/insight", api.ListInsightsRequest{
		DescriptionFilter: "expired",
	}, &insightList)
	require.NoError(err, "request")
	require.Equal(http.StatusOK, rec.Code)
	require.Len(insightList, 1)
	require.Equal("select count(*) from aws_expired_certificates;", insightList[0].Query)

	rec, err = doRequestJSONResponse(s.router, echo.DELETE, fmt.Sprintf("/api/v1/insight/%d", deleteId), nil, nil)
	require.NoError(err, "request")
	require.Equal(http.StatusOK, rec.Code)

	rec, err = doRequestJSONResponse(s.router, echo.GET, "/api/v1/insight", api.ListInsightsRequest{
		DescriptionFilter: "",
	}, &insightList)
	require.NoError(err, "request")
	require.Equal(http.StatusOK, rec.Code)
	require.Len(insightList, 1)
	require.Equal("select count(*) from aws_expired_certificates;", insightList[0].Query)
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

func TestHttpServer(t *testing.T) {
	suite.Run(t, &HttpServerSuite{})
}
