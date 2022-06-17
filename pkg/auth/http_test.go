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

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"gitlab.com/keibiengine/keibi-engine/pkg/auth/api"
	"gitlab.com/keibiengine/keibi-engine/pkg/auth/extauth"
	extauthmocks "gitlab.com/keibiengine/keibi-engine/pkg/auth/extauth/mocks"
	"gitlab.com/keibiengine/keibi-engine/pkg/internal/dockertest"
	emailmocks "gitlab.com/keibiengine/keibi-engine/pkg/internal/email/mocks"
	"gitlab.com/keibiengine/keibi-engine/pkg/internal/httpserver"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type HTTPRouteSuite struct {
	suite.Suite

	orm        *gorm.DB
	db         Database
	router     *echo.Echo
	httpRoutes *httpRoutes
}

func TestHTTPRoutes(t *testing.T) {
	suite.Run(t, &HTTPRouteSuite{})
}

func (s *HTTPRouteSuite) SetupSuite() {
	require := s.Require()

	s.orm = dockertest.StartupPostgreSQL(s.T())
	s.db = NewDatabase(s.orm)

	logger, err := zap.NewDevelopment()
	require.NoError(err, "failed to create logger")

	s.httpRoutes = &httpRoutes{
		logger: logger,
		db:     s.db,
	}

	s.router = httpserver.Register(logger, s.httpRoutes)
}

func (s *HTTPRouteSuite) BeforeTest(suiteName, testName string) {
	require := s.Require()

	err := s.httpRoutes.db.Initialize()
	require.NoError(err, "initialize db")

	s.httpRoutes.authProvider = &extauthmocks.Provider{}
	s.httpRoutes.emailService = &emailmocks.Service{}
}

func (s *HTTPRouteSuite) AfterTest(suiteName, testName string) {
	require := s.Require()

	db := s.httpRoutes.db

	tx := db.orm.Exec("DROP TABLE IF EXISTS role_bindings;")
	require.NoError(tx.Error, "drop role_bindings")

	tx = db.orm.Exec("DROP TABLE IF EXISTS users;")
	require.NoError(tx.Error, "drop users")
}

func (s *HTTPRouteSuite) TestGetRoleBindings_Empty() {
	require := s.Require()

	var resp api.GetRoleBindingsResponse
	recorder, err := doSimpleJSONRequest(s.router, http.MethodGet, "/api/v1/user/role/bindings",
		uuid.New(),
		api.ViewerRole,
		"workspace1",
		nil, &resp)
	require.NoError(err, "get role bindings")
	require.Equal(http.StatusOK, recorder.Result().StatusCode, mustRead(recorder.Result().Body))

	require.Equal(0, len(resp))
}

func (s *HTTPRouteSuite) TestCreateAndGetRoleBindings() {
	require := s.Require()

	cases := []RoleBinding{
		{
			UserID:        uuid.New(),
			ExternalID:    "user-1",
			WorkspaceName: "workspace1",
			Role:          api.AdminRole,
			AssignedAt:    time.Now(),
		},
		{
			UserID:        uuid.New(),
			ExternalID:    "user-2",
			WorkspaceName: "workspace2",
			Role:          api.EditorRole,
			AssignedAt:    time.Now(),
		},
		{
			UserID:        uuid.New(),
			ExternalID:    "user-3",
			WorkspaceName: "workspace3",
			Role:          api.ViewerRole,
			AssignedAt:    time.Now(),
		},
	}

	for _, rb := range cases {
		require.NoError(s.httpRoutes.db.CreateOrUpdateRoleBinding(&rb))

		var resp api.GetRoleBindingsResponse
		recorder, err := doSimpleJSONRequest(s.router, http.MethodGet, "/api/v1/user/role/bindings", rb.UserID, rb.Role, rb.WorkspaceName, nil, &resp)
		require.NoError(err, "get role binding")
		require.Equal(http.StatusOK, recorder.Result().StatusCode, mustRead(recorder.Result().Body))

		require.Len(resp, 1)
		require.Equal(rb.Role, resp[0].Role)
		require.Equal(rb.WorkspaceName, resp[0].WorkspaceName)
		require.Equal(rb.AssignedAt.UnixMilli(), resp[0].AssignedAt.UnixMilli())
	}
}

func (s *HTTPRouteSuite) TestCreateRoleBinding_UserDoesNotExist() {
	require := s.Require()

	request := api.PutRoleBindingRequest{
		UserID: uuid.New(),
		Role:   api.ViewerRole,
	}

	var response = struct {
		Message string
	}{}
	recorder, err := doSimpleJSONRequest(s.router, http.MethodPut, "/api/v1/user/role/binding",
		uuid.New(),
		api.AdminRole,
		"workspace1",
		request,
		&response)
	require.NoError(err, "put role binding")
	require.Equal(http.StatusBadRequest, recorder.Result().StatusCode, mustRead(recorder.Result().Body))
	require.Equal("user not found", response.Message)
}

func (s *HTTPRouteSuite) TestPutRoleBinding() {
	require := s.Require()

	var (
		admin  = uuid.New()
		viewer = uuid.New()
		editor = uuid.New()
	)

	// Need to create users before being able to update their role bindings
	for i, user := range []uuid.UUID{admin, viewer, editor} {
		require.NoError(s.httpRoutes.db.CreateUser(&User{
			ID:         user,
			Email:      fmt.Sprintf("nima%d@keibi.io", i),
			ExternalID: fmt.Sprintf("external-id-%d", i),
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

	adminID := uuid.New()
	for _, request := range requests {
		recorder, err := doSimpleJSONRequest(s.router, http.MethodPut, "/api/v1/user/role/binding",
			adminID,
			api.AdminRole,
			"workspace1",
			request,
			nil)
		require.NoError(err, "put role binding")
		require.Equal(http.StatusOK, recorder.Result().StatusCode, mustRead(recorder.Result().Body))
	}

	var resp api.GetWorkspaceRoleBindingResponse
	recorder, err := doSimpleJSONRequest(s.router, http.MethodGet, "/api/v1/workspace/role/bindings",
		adminID,
		api.AdminRole,
		"workspace1",
		nil, &resp)
	require.NoError(err, "get role bindings")
	require.Equal(http.StatusOK, recorder.Result().StatusCode, mustRead(recorder.Result().Body))

	require.Equal(3, len(resp))

	each := []int{0, 0, 0}
	for _, rb := range resp {
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

	var resp2 api.GetWorkspaceRoleBindingResponse
	recorder, err = doSimpleJSONRequest(s.router, http.MethodGet, "/api/v1/workspace/role/bindings",
		adminID,
		api.AdminRole,
		"workspace2",
		nil,
		&resp2)
	require.NoError(err, "get role bindings")
	require.Equal(http.StatusOK, recorder.Result().StatusCode, mustRead(recorder.Result().Body))

	require.Equal(0, len(resp2))
}

func (s *HTTPRouteSuite) TestPutRoleBinding_UpdateExisting() {
	require := s.Require()

	var (
		user1 = uuid.New()
	)

	require.NoError(s.httpRoutes.db.CreateUser(&User{
		ID:         user1,
		Email:      "nima@keibi.io",
		ExternalID: "external-id-1",
	}))

	requests := []api.PutRoleBindingRequest{
		{
			UserID: user1,
			Role:   api.ViewerRole,
		},
		{
			UserID: user1,
			Role:   api.AdminRole,
		},
		{
			UserID: user1,
			Role:   api.EditorRole,
		},
	}

	adminID := uuid.New()
	for _, request := range requests {
		recorder, err := doSimpleJSONRequest(s.router, http.MethodPut, "/api/v1/user/role/binding",
			adminID,
			api.AdminRole,
			"workspace1",
			request,
			nil)
		require.NoError(err, "put role binding")
		require.Equal(http.StatusOK, recorder.Result().StatusCode, mustRead(recorder.Result().Body))

		var resp api.GetWorkspaceRoleBindingResponse
		recorder, err = doSimpleJSONRequest(s.router, http.MethodGet, "/api/v1/workspace/role/bindings",
			adminID,
			api.AdminRole,
			"workspace1",
			nil, &resp)
		require.NoError(err, "get role bindings")
		require.Equal(http.StatusOK, recorder.Result().StatusCode, mustRead(recorder.Result().Body))

		require.Equal(1, len(resp))
		require.Equal(request.UserID, resp[0].UserID)
		require.Equal(request.Role, resp[0].Role)
	}
}

func (s *HTTPRouteSuite) TestInviteUser_UserDoesntExists() {
	require := s.Require()

	email := "nima@keibi.io"
	extId := "external-user-1"

	s.httpRoutes.authProvider.(*extauthmocks.Provider).On("FetchUser", mock.Anything, email).Return(extauth.AzureADUser{}, extauth.ErrUserNotExists)
	s.httpRoutes.authProvider.(*extauthmocks.Provider).On("CreateUser", mock.Anything, email).Return(extauth.AzureADUser{
		ID:          extId,
		DisplayName: "nima",
		PasswordProfile: extauth.PasswordProfile{
			Password: "123456",
		},
	}, (error)(nil))
	s.httpRoutes.emailService.(*emailmocks.Service).On("SendEmail", mock.Anything, email, "123456", "workspace1").Return((error)(nil))

	req := api.InviteUserRequest{
		Email: email,
	}
	var resp api.InviteUserResponse
	recorder, err := doSimpleJSONRequest(s.router, http.MethodPost, "/api/v1/user/invite", uuid.New(), api.AdminRole, "workspace1", req, &resp)
	require.NoError(err, "invite user")
	require.Equal(http.StatusOK, recorder.Result().StatusCode, mustRead(recorder.Result().Body))

	usr, err := s.db.GetUserByEmail(email)
	require.NoError(err, "get user")
	require.Equal(usr.Email, email)
	require.Equal(usr.ExternalID, extId)
}

func (s *HTTPRouteSuite) TestInviteUser_UserExistsInAzureButNotAuthService() {
	require := s.Require()

	email := "nima@keibi.io"
	extId := "external-user-1"

	s.httpRoutes.authProvider.(*extauthmocks.Provider).On("FetchUser", mock.Anything, email).Return(extauth.AzureADUser{
		ID:          extId,
		DisplayName: "nima",
	}, (error)(nil))

	req := api.InviteUserRequest{
		Email: email,
	}
	var resp api.InviteUserResponse
	recorder, err := doSimpleJSONRequest(s.router, http.MethodPost, "/api/v1/user/invite", uuid.New(), api.AdminRole, "workspace1", req, &resp)
	require.NoError(err, "invite user")
	require.Equal(http.StatusOK, recorder.Result().StatusCode, mustRead(recorder.Result().Body))

	usr, err := s.db.GetUserByEmail(email)
	require.NoError(err, "get user")
	require.Equal(usr.Email, email)
	require.Equal(usr.ExternalID, extId)
	require.Equal(usr.ID, resp.UserID)

	s.httpRoutes.authProvider.(*extauthmocks.Provider).AssertNotCalled(s.T(), "CreateUser", mock.Anything, mock.Anything)
}

func (s *HTTPRouteSuite) TestInviteUser_UserExists() {
	require := s.Require()

	email := "nima@keibi.io"
	extId := "external-user-1"

	user := User{
		Email:      email,
		ExternalID: extId,
	}
	err := s.db.CreateUser(&user)
	require.NoError(err, "create user")

	req := api.InviteUserRequest{
		Email: email,
	}
	var resp api.InviteUserResponse
	recorder, err := doSimpleJSONRequest(s.router, http.MethodPost, "/api/v1/user/invite", uuid.New(), api.AdminRole, "workspace1", req, &resp)
	require.NoError(err, "invite user")
	require.Equal(http.StatusOK, recorder.Result().StatusCode, mustRead(recorder.Result().Body))

	require.NoError(err, "get user")
	require.Equal(user.Email, email)
	require.Equal(user.ExternalID, extId)
	require.Equal(user.ID, resp.UserID)

	s.httpRoutes.authProvider.(*extauthmocks.Provider).AssertNotCalled(s.T(), "FetchUser", mock.Anything, mock.Anything)
	s.httpRoutes.authProvider.(*extauthmocks.Provider).AssertNotCalled(s.T(), "CreateUser", mock.Anything, mock.Anything)
	s.httpRoutes.emailService.(*emailmocks.Service).AssertNotCalled(s.T(), "SendEmail", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func doSimpleJSONRequest(
	router *echo.Echo,
	method string,
	path string,
	userId uuid.UUID,
	userRole api.Role,
	workspaceName string,
	request,
	response interface{}) (*httptest.ResponseRecorder, error) {
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
	req.Header.Add(httpserver.XKeibiUserIDHeader, userId.String())
	req.Header.Add(httpserver.XKeibiUserRoleHeader, string(userRole))
	req.Header.Add(httpserver.XKeibiWorkspaceNameHeader, workspaceName)

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
