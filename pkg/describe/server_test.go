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
	"time"

	"gitlab.com/keibiengine/keibi-engine/pkg/describe/api"

	idocker "github.com/kaytu-io/kaytu-util/pkg/dockertest"
	"github.com/kaytu-io/kaytu-util/pkg/postgres"
	"github.com/labstack/echo/v4"
	"github.com/ory/dockertest/v3"
	"github.com/stretchr/testify/suite"
	"gitlab.com/keibiengine/keibi-engine/pkg/internal/httpserver"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type HttpServerSuite struct {
	suite.Suite

	handler *HttpServer
	router  *echo.Echo
}

func (s *HttpServerSuite) SetupSuite() {
	require := s.Require()
	t := s.T()
	pool, err := dockertest.NewPool("")
	s.NoError(err, "pool constructed")
	err = pool.Client.Ping()
	s.NoError(err, "pinged pool")
	user, pass := "postgres", "123456"
	resource, err := pool.Run(user, "14.2-alpine", []string{fmt.Sprintf("POSTGRES_PASSWORD=%s", pass)})
	s.NoError(err, "status postgres")
	t.Cleanup(func() {
		err := pool.Purge(resource)
		s.NoError(err, "purge resource %s", resource)
	})
	time.Sleep(5 * time.Second)

	var adb *gorm.DB
	// exponential backoff-retry, because the application in the container might not be ready to accept connections yet
	err = pool.Retry(func() error {
		cfg := &postgres.Config{
			Host:   idocker.GetDockerHost(),
			Port:   resource.GetPort("5432/tcp"),
			User:   user,
			Passwd: pass,
			DB:     "postgres",
		}

		logger, err := zap.NewProduction()
		s.NoError(err, "new zap logger")

		adb, err = postgres.NewClient(cfg, logger)
		s.NoError(err, "new postgres client")

		d, err := adb.DB()
		if err != nil {
			return err
		}

		return d.Ping()
	})
	s.handler = NewHTTPServer(":8080", Database{orm: adb}, nil)
	err = s.handler.DB.Initialize()
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

func (s *HttpServerSuite) TestCreateStack() []string {
	var stackIds []string
	createStackTestCase := []struct {
		TestId       int
		Request      api.CreateStackRequest
		Result       int
		ErrorMessage string
	}{
		{
			TestId: 1,
			Request: api.CreateStackRequest{
				Statefile: "terraform.tfstate",
				Tags: []api.StackTag{
					{
						Key:   "Key1",
						Value: []string{"value1", "value2"},
					},
				},
			},
			Result: http.StatusOK,
		},
		{
			TestId:       2,
			Request:      api.CreateStackRequest{},
			Result:       http.StatusBadRequest,
			ErrorMessage: "code=400, message=No resource provided",
		},
	}
	for i, tc := range createStackTestCase {
		s.T().Run(fmt.Sprintf("createStack-%d", i), func(t *testing.T) {
			requestBody, err := json.Marshal(tc.Request)
			if err != nil {
				t.Fatalf("Marshal request: %v", err)
			}
			r := httptest.NewRequest(http.MethodPost, "/", bytes.NewBuffer(requestBody))
			r.Header.Set("Content-Type", "application/json; charset=utf8")
			w := httptest.NewRecorder()

			c := echo.New().NewContext(r, w)
			c.SetPath("/stacks/build")

			err = s.handler.CreateStack(c)
			if tc.ErrorMessage != "" {
				s.Equal(tc.ErrorMessage, err.Error())
				s.Equal(tc.Result, err.(*echo.HTTPError).Code)
			} else {
				var response Stack
				if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
					t.Fatalf("json decode: %v", err)
				}
				s.NotEmpty(response)
				stackIds = append(stackIds, response.StackID)
			}

		})
	}
	return stackIds
}

func (s *HttpServerSuite) TestListStacks() {
	s.TestCreateStack()
	s.T().Run(fmt.Sprintf("listStack"), func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.Header.Set("Content-Type", "application/json; charset=utf8")
		w := httptest.NewRecorder()

		c := echo.New().NewContext(r, w)
		c.SetPath("/stacks")

		err := s.handler.ListStack(c)
		if err != nil {
			t.Fatalf("List stacks %v", err)
		}
		var response []Stack
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("json decode: %v", err)
		}

		s.NotEmpty(response)

	})

}

func (s *HttpServerSuite) TestGetStack() {
	ids := s.TestCreateStack()
	var getStackTestCase []struct {
		stackId   string
		resultId  string
		errorCode int
	}
	for _, id := range ids {
		getStackTestCase = append(getStackTestCase, struct {
			stackId   string
			resultId  string
			errorCode int
		}{
			stackId:   id,
			resultId:  id,
			errorCode: http.StatusOK,
		})
	}
	getStackTestCase = append(getStackTestCase, struct {
		stackId   string
		resultId  string
		errorCode int
	}{
		stackId:   "not-a-stack",
		resultId:  "",
		errorCode: http.StatusOK,
	})
	for i, tc := range getStackTestCase {
		s.T().Run(fmt.Sprintf("getstack-%d", i), func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			r.Header.Set("Content-Type", "application/json; charset=utf8")
			w := httptest.NewRecorder()

			c := echo.New().NewContext(r, w)
			c.SetPath("stacks/:stackId")
			c.SetParamNames("stackId")
			c.SetParamValues(tc.stackId)

			err := s.handler.GetStack(c)
			if err != nil {
				t.Fatalf("Get stacks %v", err)
			}
			var response Stack
			if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
				t.Fatalf("json decode: %v", err)
			}
			if tc.stackId == "not-a-stack" {
				s.Equal("", response.StackID)
			} else {
				s.Equal(tc.resultId, response.StackID)
			}
		})
	}
}

func (s *HttpServerSuite) TestTriggerBenchmark() {
	ids := s.TestCreateStack()
	var benchmarkTestCase []struct {
		request api.EvaluateStack
		result  int
	}
	for _, id := range ids {
		benchmarkTestCase = append(benchmarkTestCase, struct {
			request api.EvaluateStack
			result  int
		}{
			request: api.EvaluateStack{
				StackID:    id,
				Benchmarks: []string{"aws_foundational_security"},
			},
			result: http.StatusOK,
		})
	}
	benchmarkTestCase = append(benchmarkTestCase, struct {
		request api.EvaluateStack
		result  int
	}{
		request: api.EvaluateStack{
			StackID:    "not-a-stack",
			Benchmarks: []string{},
		},
		result: http.StatusBadRequest,
	})
	for i, tc := range benchmarkTestCase {
		s.T().Run(fmt.Sprintf("triggerBenchmark-%d", i), func(t *testing.T) {
			requestBody, err := json.Marshal(tc.request)
			if err != nil {
				t.Fatalf("Marshal request: %v", err)
			}
			r := httptest.NewRequest(http.MethodPost, "/", bytes.NewBuffer(requestBody))
			r.Header.Set("Content-Type", "application/json; charset=utf8")
			w := httptest.NewRecorder()

			c := echo.New().NewContext(r, w)
			c.SetPath("stacks/benchmark/trigger")

			err = s.handler.TriggerStackBenchmark(c)
			if tc.request.StackID == "not-a-stack" {
				s.Equal(tc.result, err.(*echo.HTTPError).Code)
			} else {
				if err != nil {
					t.Fatalf("Trigger benchmark: %v", err)
				}
				s.Equal(tc.result, w.Code)
				stackRecord, err := s.handler.DB.GetStack(tc.request.StackID)
				if err != nil {
					t.Fatalf("Database error: %v", err)
				}
				var tags []api.StackTag
				for _, t := range stackRecord.Tags {
					tags = append(tags, api.StackTag{
						Key:   t.Key,
						Value: t.Value,
					})
				}

				var evaluations []api.StackEvaluation
				for _, e := range stackRecord.Evaluations {
					evaluations = append(evaluations, api.StackEvaluation{
						BenchmarkID: e.BenchmarkID,
						JobID:       e.JobID,
						CreatedAt:   e.CreatedAt,
					})
				}

				stack := api.Stack{
					StackID:     stackRecord.StackID,
					CreatedAt:   stackRecord.CreatedAt,
					UpdatedAt:   stackRecord.UpdatedAt,
					Resources:   []string(stackRecord.Resources),
					Tags:        tags,
					Evaluations: evaluations,
				}
				fmt.Println(stack)
				fmt.Println("=================")
				fmt.Println("evaluations:", &stack.Evaluations)
			}

		})
	}

}

func (s *HttpServerSuite) TestDeleteStack() {
	ids := s.TestCreateStack()
	var deleteStackTestCase []struct {
		stackId   string
		resultId  string
		errorCode int
	}
	for _, id := range ids {
		deleteStackTestCase = append(deleteStackTestCase, struct {
			stackId   string
			resultId  string
			errorCode int
		}{
			stackId:   id,
			resultId:  id,
			errorCode: http.StatusOK,
		})
	}
	deleteStackTestCase = append(deleteStackTestCase, struct {
		stackId   string
		resultId  string
		errorCode int
	}{
		stackId:   "not-a-stack",
		resultId:  "",
		errorCode: http.StatusOK,
	})
	for i, tc := range deleteStackTestCase {
		s.T().Run(fmt.Sprintf("getstack-%d", i), func(t *testing.T) {
			if tc.stackId != "not-a-stack" {
				stack, err := s.handler.DB.GetStack(tc.stackId)
				if err != nil {
					t.Fatalf("Database error: %v", err)
				}
				s.NotEqual(Stack{}, stack)
			}

			r := httptest.NewRequest(http.MethodDelete, "/", nil)
			r.Header.Set("Content-Type", "application/json; charset=utf8")
			w := httptest.NewRecorder()

			c := echo.New().NewContext(r, w)
			c.SetPath("stacks/:stackId")
			c.SetParamNames("stackId")
			c.SetParamValues(tc.stackId)

			err := s.handler.DeleteStack(c)

			if err != nil {
				t.Fatalf("Get stacks %v", err)
			}
			stack, err := s.handler.DB.GetStack(tc.stackId)
			if err != nil {
				t.Fatalf("Database error: %v", err)
			}
			s.Equal(Stack{}, stack)
		})
	}
}
