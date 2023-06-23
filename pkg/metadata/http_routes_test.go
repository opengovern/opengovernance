package metadata

import (
	"bytes"
	"encoding/json"
	"fmt"
	idocker "github.com/kaytu-io/kaytu-util/pkg/dockertest"
	"github.com/kaytu-io/kaytu-util/pkg/postgres"
	"github.com/kaytu-io/kaytu-engine/pkg/internal/httpserver"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
	api2 "github.com/kaytu-io/kaytu-engine/pkg/auth/api"
	"github.com/kaytu-io/kaytu-engine/pkg/metadata/api"
	"github.com/kaytu-io/kaytu-engine/pkg/metadata/internal/cache"
	"github.com/kaytu-io/kaytu-engine/pkg/metadata/internal/database"
	"github.com/kaytu-io/kaytu-engine/pkg/metadata/models"
	"go.uber.org/zap"

	"github.com/labstack/echo/v4"
	"github.com/ory/dockertest/v3"
	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"
)

type HttpHandlerSuite struct {
	suite.Suite

	handler     *HttpHandler
	router      *echo.Echo
	orm         *gorm.DB
	redisClient *redis.Client
}

func (s *HttpHandlerSuite) SetupSuite() {
	t := s.T()
	require := s.Require()

	pool, err := dockertest.NewPool("")
	require.NoError(err, "connect to docker")

	postgresDocker, err := pool.Run("postgres", "14.2-alpine", []string{"POSTGRES_PASSWORD=mysecretpassword"})
	require.NoError(err, "status postgres")
	redisDocker, err := pool.Run("redis", "7.0.7", []string{})
	require.NoError(err, "status redis")

	t.Cleanup(func() {
		err := pool.Purge(postgresDocker)
		require.NoError(err, "purge postgresDocker %s", postgresDocker)
		err = pool.Purge(redisDocker)
		require.NoError(err, "purge redisDocker %s", redisDocker)
	})
	time.Sleep(5 * time.Second)

	// exponential backoff-retry, because the application in the container might not be ready to accept connections yet
	err = pool.Retry(func() error {
		cfg := &postgres.Config{
			Host:   idocker.GetDockerHost(),
			Port:   postgresDocker.GetPort("5432/tcp"),
			User:   "postgres",
			Passwd: "mysecretpassword",
			DB:     "postgres",
		}

		logger, err := zap.NewProduction()
		require.NoError(err, "new zap logger")

		s.orm, err = postgres.NewClient(cfg, logger)
		require.NoError(err, "new postgres client")

		d, err := s.orm.DB()
		if err != nil {
			return err
		}

		err = d.Ping()
		if err != nil {
			return err
		}

		s.redisClient = redis.NewClient(&redis.Options{
			Addr:     fmt.Sprintf("%s:%s", idocker.GetDockerHost(), redisDocker.GetPort("6379/tcp")),
			Password: "",
			DB:       0,
		})

		err = s.redisClient.Ping(s.redisClient.Context()).Err()
		if err != nil {
			return err
		}

		return nil
	})
	require.NoError(err, "wait for postgres connection")

	tx := s.orm.Exec(`CREATE EXTENSION IF NOT EXISTS "uuid-ossp";`)
	require.NoError(tx.Error, "enable uuid v4")

	s.handler = &HttpHandler{
		db:    database.NewDatabase(s.orm),
		redis: cache.NewMetadataRedisCache(s.redisClient, time.Minute),
	}

	logger, err := zap.NewProduction()
	require.NoError(err, "new logger")

	s.router = httpserver.Register(logger, s.handler)
}

func (s *HttpHandlerSuite) BeforeTest(suiteName, testName string) {
	require := s.Require()

	err := s.handler.db.Initialize()
	require.NoError(err, "initialize db")
}

func (s *HttpHandlerSuite) AfterTest(suiteName, testName string) {
	require := s.Require()

	tx := s.orm.Exec("DROP TABLE IF EXISTS config_metadata;")
	require.NoError(tx.Error, "drop ConfigMetadata")
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
	req.Header.Add("X-Keibi-UserRole", string(api2.AdminRole))

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

func (s *HttpHandlerSuite) TestConfigMetadata() {
	require := s.Require()

	key := models.MetadataKeyWorkspaceName
	value := "test-value"
	response := map[string]any{}
	rec, err := doSimpleJSONRequest(s.router, echo.POST, "/api/v1/metadata", api.SetConfigMetadataRequest{
		Key:   key.String(),
		Value: value,
	}, &response)
	require.NoError(err, "request")
	require.Equal(http.StatusOK, rec.Code)

	rec, err = doSimpleJSONRequest(s.router, echo.GET, "/api/v1/metadata/"+key.String(), nil, &response)
	require.NoError(err, "request")
	require.Equal(http.StatusOK, rec.Code)
	require.Equal(value, response["value"])
}

func (s *HttpHandlerSuite) TestMetadataKeys() {
	require := s.Require()

	for _, key := range models.MetadataKeys {
		kType := key.GetConfigMetadataType()
		require.NotEmpty(kType)
		role := key.GetMinAuthRole()
		require.NotEmpty(role)
		fmt.Println(key.String() + "," + string(kType))
	}
}
