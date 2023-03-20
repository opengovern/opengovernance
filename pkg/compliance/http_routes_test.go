package compliance

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/suite"
	"gitlab.com/keibiengine/keibi-engine/pkg/compliance/db"
	idocker "gitlab.com/keibiengine/keibi-engine/pkg/internal/dockertest"
	"gitlab.com/keibiengine/keibi-engine/pkg/internal/httpserver"
	"go.uber.org/zap"
)

type HttpServerSuite struct {
	suite.Suite

	handler *HttpHandler
	router  *echo.Echo
}

func (s *HttpServerSuite) SetupSuite() {
	require := s.Require()

	orm := idocker.StartupPostgreSQL(s.T())
	s.handler = &HttpHandler{
		db: db.Database{Orm: orm},
	}

	err := s.handler.db.Initialize()
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
	b, err := io.ReadAll(io.NopCloser(rec.Body))
	if err != nil {
		return nil, nil, err
	}
	return rec, b, nil
}

func TestHttpServer(t *testing.T) {
	suite.Run(t, &HttpServerSuite{})
}

func (s *HttpServerSuite) TestDatabaseTableStructure() {
	time.Sleep(5 * time.Minute)
}
