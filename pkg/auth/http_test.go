package auth

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/suite"
	"gitlab.com/keibiengine/keibi-engine/pkg/auth/api"
	"gitlab.com/keibiengine/keibi-engine/pkg/internal/dockertest"
	"gitlab.com/keibiengine/keibi-engine/pkg/internal/httpserver"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type HTTPRouteSuite struct {
	suite.Suite

	orm        *gorm.DB
	router     *echo.Echo
	httpRoutes httpRoutes
}

func (s *HTTPRouteSuite) SetupSuite() {
	require := s.Require()

	s.orm = dockertest.StartupPostgreSQL(s.T())

	s.httpRoutes = httpRoutes{
		db: Database{
			orm: s.orm,
		},
	}

	logger, err := zap.NewDevelopment()
	require.NoError(err)

	s.router = httpserver.Register(logger, s.httpRoutes)
}

func (s *HTTPRouteSuite) BeforeTest(suiteName, testName string) {
	require := s.Require()

	err := s.httpRoutes.db.Initialize()
	require.NoError(err, "initialize db")
}

func (s *HTTPRouteSuite) AfterTest(suiteName, testName string) {
	require := s.Require()

	db := s.httpRoutes.db

	tx := db.orm.Exec("DROP TABLE IF EXISTS role_bindings;")
	require.NoError(tx.Error, "drop role_bindings")
}

func (s *HTTPRouteSuite) TestGetRoleBindings_Empty() {
	require := s.Require()

	var resp api.GetRoleBindingsResponse
	recorder, err := doSimpleJSONRequest(s.router, http.MethodGet, "/api/v1/role/bindings", nil, &resp)
	require.NoError(err, "get role bindings")
	require.Equal(http.StatusOK, recorder.Result().StatusCode, mustRead(recorder.Result().Body))

	require.Equal(0, len(resp))
}

func (s *HTTPRouteSuite) TestCreateAndGetRoleBinding() {
	require := s.Require()

	// Need to create users before being able to update their role bindings
	rb := RoleBinding{
		UserID: "RandomUser",
		Role:   api.AdminRole,
		Name:   "Nima",
		Emails: []string{
			"nima@keibi.io",
			"nima2@keibi.io",
		},
		AssignedAt: time.Now(),
	}
	require.NoError(s.httpRoutes.db.GetRoleBindingOrCreate(&rb))

	req := api.GetRoleBindingRequest{
		UserID: "RandomUser",
	}
	var resp api.GetRoleBindingResponse
	recorder, err := doSimpleJSONRequest(s.router, http.MethodGet, "/api/v1/role/binding", req, &resp)
	require.NoError(err, "get role binding")
	require.Equal(http.StatusOK, recorder.Result().StatusCode, mustRead(recorder.Result().Body))

	require.Equal(rb.UserID, resp.UserID)
	require.Equal(rb.Role, resp.Role)
	require.Equal(rb.Name, resp.Name)
	require.Equal([]string(rb.Emails), resp.Emails)
	require.Equal(rb.AssignedAt.UnixMilli(), resp.AssignedAt.UnixMilli())
}

func (s *HTTPRouteSuite) TestCreateRoleBinding_UserDoesNotExist() {
	require := s.Require()

	request := api.PutRoleBindingRequest{
		UserID: "admin-user",
		Role:   api.AdminRole,
	}

	var response api.ErrorResponse
	recorder, err := doSimpleJSONRequest(s.router, http.MethodPut, "/api/v1/role/binding", request, &response)
	require.NoError(err, "put role binding")
	require.Equal(http.StatusBadRequest, recorder.Result().StatusCode, mustRead(recorder.Result().Body))
	require.Equal("update role binding: user with id admin-user doesn't exist", response.Message)
}

func (s *HTTPRouteSuite) TestPutRoleBinding() {
	require := s.Require()

	const (
		admin  = "admin-user"
		viewer = "viewer-user"
		editor = "editor-user"
	)

	// Need to create users before being able to update their role bindings
	for _, user := range []string{admin, viewer, editor} {
		require.NoError(s.httpRoutes.db.GetRoleBindingOrCreate(&RoleBinding{
			UserID: user,
		}))
	}

	requests := []api.PutRoleBindingRequest{
		{
			UserID: admin,
			Role:   api.AdminRole,
		},
		{
			UserID: editor,
			Role:   api.EditorRole,
		},
		{
			UserID: viewer,
			Role:   api.ViewerRole,
		},
	}

	for _, request := range requests {
		recorder, err := doSimpleJSONRequest(s.router, http.MethodPut, "/api/v1/role/binding", request, nil)
		require.NoError(err, "put role binding")
		require.Equal(http.StatusOK, recorder.Result().StatusCode, mustRead(recorder.Result().Body))
	}

	var resp api.GetRoleBindingsResponse
	recorder, err := doSimpleJSONRequest(s.router, http.MethodGet, "/api/v1/role/bindings", nil, &resp)
	require.NoError(err, "get role bindings")
	require.Equal(http.StatusOK, recorder.Result().StatusCode, mustRead(recorder.Result().Body))

	require.Equal(3, len(resp))

	each := []int{0, 0, 0}
	for _, rb := range resp {
		require.Empty(rb.Name)
		require.Empty(rb.Emails)
		require.False(rb.AssignedAt.IsZero())

		switch rb.UserID {
		case admin:
			require.Equal(api.AdminRole, rb.Role)
			each[0]++
		case editor:
			require.Equal(api.EditorRole, rb.Role)
			each[1]++
		case viewer:
			require.Equal(api.ViewerRole, rb.Role)
			each[2]++
		}
	}

	require.Equal([]int{1, 1, 1}, each)
}

func TestHTTPRoutes(t *testing.T) {
	suite.Run(t, &HTTPRouteSuite{})
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
			return nil, fmt.Errorf("%w: %s", err, string(b))
		}
	}

	return rec, nil
}

func mustRead(reader io.ReadCloser) string {
	all, err := ioutil.ReadAll(reader)
	if err != nil {
		panic(err)
	}

	return string(all)
}
