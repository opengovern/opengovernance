package describe

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	idocker "github.com/kaytu-io/kaytu-util/pkg/dockertest"
	"github.com/kaytu-io/kaytu-util/pkg/httpserver"
	"io"
	"io/ioutil"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
)

type HttpServerSuite struct {
	suite.Suite

	handler *HttpServer
	router  *echo.Echo
}

func (s *HttpServerSuite) SetupSuite() {
	require := s.Require()
	orm := dockertest.StartupPostgreSQL(s.T())
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
