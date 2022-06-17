package inventory

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/labstack/echo/v4"
	"github.com/ory/dockertest/v3"
	"github.com/stretchr/testify/suite"
	describe "gitlab.com/keibiengine/keibi-engine/pkg/describe/api"
	"gitlab.com/keibiengine/keibi-engine/pkg/describe/client"
	idocker "gitlab.com/keibiengine/keibi-engine/pkg/internal/dockertest"
	"gitlab.com/keibiengine/keibi-engine/pkg/internal/postgres"
	"gitlab.com/keibiengine/keibi-engine/pkg/inventory/api"
	"go.uber.org/zap"
)

func TestSuite(t *testing.T) {
	suite.Run(t, &testSuite{})
}

type testSuite struct {
	suite.Suite

	handler     *HttpHandler
	benchmarkId string
	sourceId    string
	sourceType  string
}

func (s *testSuite) mockSchedulerServer(t *testing.T) string {
	r := mux.NewRouter()
	r.HandleFunc("/api/v1/sources/{source_id}", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)

		source, err := uuid.Parse(vars["source_id"])
		if err != nil {
			http.Error(w, fmt.Sprintf("parse source: %v", err), http.StatusBadRequest)
			return
		}

		if err := json.NewEncoder(w).Encode(describe.Source{
			ID:                     source,
			Type:                   describe.SourceType(s.sourceType),
			LastDescribedAt:        time.Now(),
			LastComplianceReportAt: time.Now(),
		}); err != nil {
			http.Error(w, fmt.Sprintf("json encode: %v", err), http.StatusInternalServerError)
			return
		}
	}).Methods(http.MethodGet)

	server := httptest.NewServer(r)
	t.Cleanup(func() {
		server.Close()
	})
	return server.URL
}

func (s *testSuite) SetupSuite() {
	t := s.T()

	s.handler = &HttpHandler{}
	s.handler.schedulerClient = client.NewSchedulerServiceClient(s.mockSchedulerServer(t))

	s.benchmarkId = "benchmark-test"
	s.sourceId = "705e4dcb-3ecd-24f3-3a35-3e926e4bded5"
	s.sourceType = "AWS"

	pool, err := dockertest.NewPool("")
	s.NoError(err, "connect to docker")

	net, err := pool.CreateNetwork("keibi")
	s.NoError(err, "create network")
	t.Cleanup(func() {
		s.NoError(pool.RemoveNetwork(net), "remove network")
	})

	user, pass, name, port := "postgres", "123456", "test", "5432"
	resource, err := pool.RunWithOptions(&dockertest.RunOptions{
		Name:         "keibi_test_psql",
		Repository:   "postgres",
		Tag:          "12.2-alpine",
		ExposedPorts: []string{port},
		Env: []string{
			"POSTGRES_USER=" + user,
			"POSTGRES_PASSWORD=" + pass,
			"POSTGRES_DB=" + name,
		},
		Networks: []*dockertest.Network{net},
	})
	s.NoError(err, "status postgres")
	t.Cleanup(func() {
		s.NoError(pool.Purge(resource), "purge resource")
	})
	time.Sleep(time.Second * 5)

	logger, err := zap.NewProduction()
	s.NoError(err, "new zap logger")
	cfg := postgres.Config{
		Host:   idocker.GetDockerHost(),
		Port:   resource.GetPort("5432/tcp"),
		User:   user,
		Passwd: pass,
		DB:     name,
	}
	orm, err := postgres.NewClient(&cfg, logger)
	s.NoError(err, "new postgres client")

	s.handler.db = Database{orm: orm}
	s.NoError(s.handler.db.Initialize(), "init database schema")
}

func (ts *testSuite) TearDownSuite() {
}

func (ts *testSuite) TearDownTest() {
	tx := ts.handler.db.orm.Exec("delete from benchmarks where id = ?", ts.benchmarkId)
	ts.NoError(tx.Error)
	tx = ts.handler.db.orm.Exec("delete from benchmark_assignments where benchmark_id = ?", ts.benchmarkId)
	ts.NoError(tx.Error)
}

func (ts *testSuite) initBenchmark() error {
	return ts.handler.db.AddBenchmark(&Benchmark{
		ID:          ts.benchmarkId,
		Title:       "benchmark-title",
		Description: "benchmark-description",
		Provider:    ts.sourceType,
		State:       "benchmark-status",
	})
}

func (ts *testSuite) initBenchmarkAssignment() error {
	source, err := uuid.Parse(ts.sourceId)
	if err != nil {
		return err
	}
	return ts.handler.db.AddBenchmarkAssignment(&BenchmarkAssignment{
		BenchmarkId: ts.benchmarkId,
		SourceId:    source,
	})
}

func (ts *testSuite) TestCreateBenchmarkAssignment() {
	ts.NoError(ts.initBenchmark())

	createBenchmarkAssignmentTestCases := []struct {
		SourceId    string
		BenchmarkId string
		Code        int
		Error       string
	}{
		{
			BenchmarkId: ts.benchmarkId,
			Code:        http.StatusBadRequest,
			Error:       "source id is empty",
		},
		{
			SourceId: ts.sourceId,
			Code:     http.StatusBadRequest,
			Error:    "benchmark id is empty",
		},
		{
			SourceId: "invalid source",
			Code:     http.StatusBadRequest,
			Error:    "invalid source uuid",
		},
		{
			BenchmarkId: "not found",
			SourceId:    ts.sourceId,
			Code:        http.StatusNotFound,
		},
		{
			BenchmarkId: ts.benchmarkId,
			SourceId:    ts.sourceId,
			Code:        http.StatusOK,
		},
	}
	for i, tc := range createBenchmarkAssignmentTestCases {
		tc := tc
		ts.T().Run(fmt.Sprintf("CreateBenchmarkAssignmentTestCases-%d", i), func(t *testing.T) {
			r := httptest.NewRequest(http.MethodPost, "/", nil)
			r.Header.Set("Content-Type", "application/json; charset=utf8")
			w := httptest.NewRecorder()

			c := echo.New().NewContext(r, w)
			c.SetPath("/benchmarks/:benchmark_id/source/:source_id")
			c.SetParamNames("benchmark_id", "source_id")
			c.SetParamValues(tc.BenchmarkId, tc.SourceId)

			err := ts.handler.CreateBenchmarkAssignment(c)
			if err != nil {
				var he *echo.HTTPError
				ts.Equal(true, errors.As(err, &he))
				ts.Equal(tc.Code, he.Code)
				ts.Contains(he.Message, tc.Error)
				return
			}

			var assignment api.BenchmarkAssignment
			if err := json.NewDecoder(w.Body).Decode(&assignment); err != nil {
				ts.T().Fatalf("json decode: %v", err)
			}

			ts.Equal(tc.SourceId, assignment.SourceId)
			ts.Equal(tc.BenchmarkId, assignment.BenchmarkId)
		})
	}
}

func (ts *testSuite) TestGetAllBenchmarkAssignmentsBySourceId() {
	ts.NoError(ts.initBenchmarkAssignment())

	getAllBenchmarkAssignmentsBySourceIdTestCases := []struct {
		SourceId string
		Count    int
		Code     int
		Error    string
	}{
		{
			Code:  http.StatusBadRequest,
			Error: "source id is empty",
		},
		{
			SourceId: "invalid source",
			Code:     http.StatusBadRequest,
			Error:    "invalid source uuid",
		},
		{
			SourceId: ts.sourceId,
			Code:     http.StatusOK,
			Count:    1,
		},
		{
			SourceId: "cda6498a-235d-4f7e-ae19-661d41bc154c",
			Code:     http.StatusOK,
			Count:    0,
		},
	}
	for i, tc := range getAllBenchmarkAssignmentsBySourceIdTestCases {
		tc := tc
		ts.T().Run(fmt.Sprintf("GetAllBenchmarkAssignmentsBySourceIdTestCases-%d", i), func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			r.Header.Set("Content-Type", "application/json; charset=utf8")
			w := httptest.NewRecorder()

			c := echo.New().NewContext(r, w)
			c.SetPath("/benchmarks/source/:source_id")
			c.SetParamNames("source_id")
			c.SetParamValues(tc.SourceId)

			err := ts.handler.GetAllBenchmarkAssignmentsBySourceId(c)
			if err != nil {
				var he *echo.HTTPError
				ts.Equal(true, errors.As(err, &he))
				ts.Equal(tc.Code, he.Code)
				ts.Contains(he.Message, tc.Error)
				return
			}

			var assignments []api.BenchmarkAssignment
			if err := json.NewDecoder(w.Body).Decode(&assignments); err != nil {
				ts.T().Fatalf("json decode: %v", err)
			}
			ts.Equal(tc.Count, len(assignments))
		})
	}
}

func (ts *testSuite) TestGetAllBenchmarkAssignedSourcesByBenchmarkId() {
	ts.NoError(ts.initBenchmarkAssignment())

	getAllBenchmarkAssignedSourcesByBenchmarkId := []struct {
		BenchmarkId string
		Count       int
		Code        int
		Error       string
	}{
		{
			Code:  http.StatusBadRequest,
			Error: "benchmark id is empty",
		},
		{
			BenchmarkId: ts.benchmarkId,
			Code:        http.StatusOK,
			Count:       1,
		},
		{
			BenchmarkId: "cda6498a-235d-4f7e-ae19-661d41bc154c",
			Code:        http.StatusOK,
			Count:       0,
		},
	}
	for i, tc := range getAllBenchmarkAssignedSourcesByBenchmarkId {
		tc := tc
		ts.T().Run(fmt.Sprintf("GetAllBenchmarkAssignedSourcesByBenchmarkId-%d", i), func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			r.Header.Set("Content-Type", "application/json; charset=utf8")
			w := httptest.NewRecorder()

			c := echo.New().NewContext(r, w)
			c.SetPath("/benchmarks/:benchmark_id/sources")
			c.SetParamNames("benchmark_id")
			c.SetParamValues(tc.BenchmarkId)

			err := ts.handler.GetAllBenchmarkAssignedSourcesByBenchmarkId(c)
			if err != nil {
				var he *echo.HTTPError
				ts.Equal(true, errors.As(err, &he))
				ts.Equal(tc.Code, he.Code)
				ts.Contains(he.Message, tc.Error)
				return
			}

			var assignments []api.BenchmarkAssignment
			if err := json.NewDecoder(w.Body).Decode(&assignments); err != nil {
				ts.T().Fatalf("json decode: %v", err)
			}
			ts.Equal(tc.Count, len(assignments))
		})
	}
}

func (ts *testSuite) TestDeleteBenchmarkAssignment() {
	ts.NoError(ts.initBenchmarkAssignment())

	deleteBenchmarkAssignmentTestCases := []struct {
		SourceId    string
		BenchmarkId string
		Code        int
		Error       string
	}{
		{
			BenchmarkId: ts.benchmarkId,
			Code:        http.StatusBadRequest,
			Error:       "source id is empty",
		},
		{
			SourceId: ts.sourceId,
			Code:     http.StatusBadRequest,
			Error:    "benchmark id is empty",
		},
		{
			SourceId: "invalid source",
			Code:     http.StatusBadRequest,
			Error:    "invalid source uuid",
		},
		{
			BenchmarkId: ts.benchmarkId,
			SourceId:    "cda6498a-235d-4f7e-ae19-661d41bc154c",
			Code:        http.StatusNotFound,
		},
		{
			BenchmarkId: ts.benchmarkId,
			SourceId:    ts.sourceId,
			Code:        http.StatusOK,
		},
	}
	for i, tc := range deleteBenchmarkAssignmentTestCases {
		tc := tc
		ts.T().Run(fmt.Sprintf("DeleteBenchmarkAssignmentTestCases-%d", i), func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			r.Header.Set("Content-Type", "application/json; charset=utf8")
			w := httptest.NewRecorder()

			c := echo.New().NewContext(r, w)
			c.SetPath("/benchmarks/:benchmark_id/source/:source_id")
			c.SetParamNames("benchmark_id", "source_id")
			c.SetParamValues(tc.BenchmarkId, tc.SourceId)

			err := ts.handler.DeleteBenchmarkAssignment(c)
			if err != nil {
				var he *echo.HTTPError
				ts.Equal(true, errors.As(err, &he))
				ts.Equal(tc.Code, he.Code)
				ts.Contains(he.Message, tc.Error)
				return
			}
		})
	}
}
