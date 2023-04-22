package auth

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/ory/dockertest/v3"
	"github.com/stretchr/testify/suite"
	"gitlab.com/keibiengine/keibi-engine/pkg/auth/api"
	"gitlab.com/keibiengine/keibi-engine/pkg/auth/auth0"
	"gitlab.com/keibiengine/keibi-engine/pkg/auth/db"
	idocker "gitlab.com/keibiengine/keibi-engine/pkg/internal/dockertest"
	"gitlab.com/keibiengine/keibi-engine/pkg/internal/httpserver"
	"gitlab.com/keibiengine/keibi-engine/pkg/internal/postgres"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func TestSuite(t *testing.T) {
	suite.Run(t, &testSuite{})
}

type testSuite struct {
	suite.Suite
	testServer *httptest.Server
	service    *auth0.Service
	httpRoutes httpRoutes
}

func (ts *testSuite) FetchData() (error, string) {

	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/idrandom", ts.testServer.URL), nil)
	if err != nil {
		return err, ""
	}

	resp, err := ts.testServer.Client().Do(req)
	if err != nil {
		return err, ""
	}
	defer resp.Body.Close()

	b, err := io.ReadAll(resp.Body)
	// b, err := ioutil.ReadAll(resp.Body)  Go.1.15 and earlier
	if err != nil {
		return err, ""
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%w. %s", err, http.StatusText(resp.StatusCode)), "Not OK"
	}

	return nil, string(b)
}

func (ts *testSuite) SetupSuite() {
	t := ts.T()
	pool, err := dockertest.NewPool("")
	ts.NoError(err, "pool constructed")
	err = pool.Client.Ping()
	ts.NoError(err, "pinged pool")
	user, pass := "postgres", "123456"
	resource, err := pool.Run(user, "14.2-alpine", []string{fmt.Sprintf("POSTGRES_PASSWORD=%s", pass)})
	ts.NoError(err, "status postgres")
	t.Cleanup(func() {
		err := pool.Purge(resource)
		ts.NoError(err, "purge resource %s", resource)
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
		ts.NoError(err, "new zap logger")

		adb, err = postgres.NewClient(cfg, logger)
		ts.NoError(err, "new postgres client")

		d, err := adb.DB()
		if err != nil {
			return err
		}

		return d.Ping()
	})
	ts.NoError(err, "wait for postgres connection")

	// setup external APIs mock
	mux := http.NewServeMux()

	mux.HandleFunc("/oauth/token", mockFillTocken)
	mux.HandleFunc("/api/v2/users/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			mockGetUser(w, r)
		} else if r.Method == http.MethodDelete {
			mockDeleteUser(w, r)
		} else if r.Method == http.MethodPatch {
			mockPatchUser(w, r)
		}
	}))
	mux.HandleFunc("/api/v2/users", mockGetUsers)
	mux.HandleFunc("/api/v2/clients/", mockGetClient)

	ts.testServer = httptest.NewServer(mux)

	ts.service = auth0.New(ts.testServer.URL, "test_auth0ClientIDNative", "test_auth0ClientID", "test_auth0ManageClientID",
		"test_auth0ManageClientSecret", "test_auth0Connection", int(1))
	ts.httpRoutes = httpRoutes{
		auth0Service: ts.service,
		db:           db.Database{Orm: adb},
	}
	e := echo.New()
	ts.httpRoutes.Register(e)

}

func (ts *testSuite) TearDownSuite() {
}

func (ts *testSuite) TearDownTest() {
}

func (ts *testSuite) TestDeleteInvitation() {
	getUserTestCases := []struct {
		UserId    string
		UserRole  api.Role
		DelUserId string
		Response  int
		Error     int
	}{
		{
			UserId:    "test1",
			UserRole:  api.AdminRole,
			DelUserId: "test3",
			Response:  http.StatusOK,
		},
		{
			UserId:    "test4",
			UserRole:  api.ViewerRole,
			DelUserId: "test2",
			Response:  http.StatusOK,
		},
		{
			UserId:    "test1",
			UserRole:  api.AdminRole,
			DelUserId: "test14",
			Error:     http.StatusNoContent,
		},
		{
			UserId:    "test5",
			UserRole:  api.AdminRole,
			DelUserId: "test5",
			Error:     http.StatusBadRequest,
		},
	}
	for i, tc := range getUserTestCases {
		ts.T().Run(fmt.Sprintf("deleteInvitationTestCases-%d", i), func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			r.Header.Set("Content-Type", "application/json; charset=utf8")
			r.Header.Set(httpserver.XKeibiUserIDHeader, tc.UserId)
			r.Header.Set(httpserver.XKeibiWorkspaceIDHeader, "ws1")
			w := httptest.NewRecorder()

			c := echo.New().NewContext(r, w)
			c.SetPath("/auth/api/v1/invite")
			c.QueryParams().Add("userId", tc.DelUserId)

			err := ts.httpRoutes.DeleteInvitation(c)
			if tc.Error == http.StatusNoContent {
				ts.Equal("[GetUser] invalid status code: 204, body=", err.Error())
				return
			} else if tc.Error == http.StatusBadRequest {
				ts.Equal("code=400, message=admin user permission can't be modified by self", err.Error())
			} else {
				ts.Nil(err)
			}
		})
	}
}

func (ts *testSuite) TestGetUsers() {
	getUserTestCases := []struct {
		UserId      string
		Request     []byte
		WorkspaceID string
		Error       int
	}{
		{
			UserId: "test1",
			Request: []byte(`
			{
				"email": "testmail@gmail.com",
				"emailVerified": true,
				"role":	"ADMIN"
			}`),
			WorkspaceID: "ws1",
		},
		{
			UserId: "test3",
			Request: []byte(`
			{
				"email": "testmail@gmail.com",
				"emailVerified": false
			}`),
			WorkspaceID: "ws2",
		},
		{
			UserId: "test2",
			Error:  http.StatusNoContent,
			Request: []byte(`
			{
				"email": "testmail@gmail.com"
			}`),
			WorkspaceID: "ws1",
		},
		{
			UserId: "test4",
			Error:  http.StatusNoContent,
			Request: []byte(`
			{
			}`),
			WorkspaceID: "ws4",
		},
		{
			UserId: "test1",
			Error:  http.StatusNoContent,
			Request: []byte(`
			{
			}`),
			WorkspaceID: "ws4",
		},
	}
	for i, tc := range getUserTestCases {
		ts.T().Run(fmt.Sprintf("getUserTestCases-%d", i), func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/", bytes.NewBuffer(tc.Request))
			r.Header.Set("Content-Type", "application/json; charset=utf8")
			r.Header.Set(httpserver.XKeibiUserIDHeader, tc.UserId)
			w := httptest.NewRecorder()

			c := echo.New().NewContext(r, w)
			c.SetPath("/auth/api/v1/users")
			c.SetParamNames("workspace_id")
			c.SetParamValues(tc.WorkspaceID)

			err := ts.httpRoutes.GetUsers(c)
			if tc.UserId == "test1" && tc.WorkspaceID == "ws4" {
				ts.Equal("code=403, message=This request is only available for ADMIN and EDITOR of the workspace.", err.Error())
			} else {
				ts.NoError(err)
				var response api.GetUsersResponse
				if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
					ts.T().Fatalf("json decode: %v", err)
				}
				fmt.Println(response)
			}
		})
	}

}

func (ts *testSuite) TestGetUser() {
	getUserTestCases := []struct {
		UserId   string
		Response string
		Error    int
	}{
		{
			UserId:   "test1",
			Response: "user1@test.com",
		},
		{
			UserId: "dontHave",
			Error:  http.StatusNoContent,
		},
	}
	for i, tc := range getUserTestCases {
		ts.T().Run(fmt.Sprintf("getUserTestCases-%d", i), func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			r.Header.Set("Content-Type", "application/json; charset=utf8")
			w := httptest.NewRecorder()

			c := echo.New().NewContext(r, w)
			c.SetPath("/auth/api/v1/user/:user_id")
			c.SetParamNames("user_id")
			c.SetParamValues(tc.UserId)

			err := ts.httpRoutes.GetUserDetails(c)
			if err != nil {
				ts.Equal("[GetUser] invalid status code: 204, body=", err.Error())
				return
			}

			var response api.GetUserResponse
			if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
				ts.T().Fatalf("json decode: %v", err)
			}
			ts.Equal(tc.Response, response.Email)
		})
	}

}

func (ts *testSuite) TestGetRoleUsers() {
	getUserTestCases := []struct {
		Role     api.Role
		Response string
		Error    int
	}{
		{
			Role:     api.AdminRole,
			Response: "user1@test.com",
		},
		{
			Role:     api.EditorRole,
			Response: "user1@test.com",
		},
		{
			Role:     api.ViewerRole,
			Response: "user1@test.com",
		},
		{
			Role:     api.KeibiAdminRole,
			Response: "",
		},
	}
	for i, tc := range getUserTestCases {
		ts.T().Run(fmt.Sprintf("getRoleUsersTestCases-%d", i), func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			r.Header.Set("Content-Type", "application/json; charset=utf8")
			r.Header.Set(httpserver.XKeibiWorkspaceIDHeader, "ws1")
			w := httptest.NewRecorder()

			c := echo.New().NewContext(r, w)

			c.SetPath("/auth/api/v1/role/:role/users")
			c.SetParamNames("role")
			c.SetParamValues(string(tc.Role))

			err := ts.httpRoutes.GetRoleUsers(c)
			if err != nil {
				ts.Equal("[SearchUsersByRole] invalid status code: 204, body=", err.Error())
				return
			}

			var response api.GetRoleUsersResponse
			if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
				ts.T().Fatalf("json decode: %v", err)
			}
			if tc.Role == api.KeibiAdminRole {
				ts.Equal(len(response), 0)
			} else {
				ts.Equal(tc.Role, response[0].RoleName)
				ts.Equal("user1@test.com", response[0].Email)
				ts.True(response[0].EmailVerified)
			}
			fmt.Println(response)
		})
	}
}

func (ts *testSuite) TestCreateAPIKey() { // not finished yet. has problem with some header parameters
	createAPIKeyTestCase := []struct {
		Request     api.CreateAPIKeyRequest
		UserID      string
		WorkspaceID string
		Role        string
		Error       int
	}{
		{
			Request:     api.CreateAPIKeyRequest{Name: "Key1", RoleName: api.AdminRole},
			UserID:      "test4",
			WorkspaceID: "ws4",
			Role:        "ADMIN",
		},
		{
			Request:     api.CreateAPIKeyRequest{Name: "Key2", RoleName: api.EditorRole},
			UserID:      "test2",
			WorkspaceID: "ws1",
			Role:        "ADMIN",
		},
		{
			Request:     api.CreateAPIKeyRequest{Name: "Key3", RoleName: api.ViewerRole},
			UserID:      "test1",
			WorkspaceID: "ws1",
			Role:        "EDITOR",
		},
		{
			Request:     api.CreateAPIKeyRequest{Name: "Key4", RoleName: api.ViewerRole},
			UserID:      "test3",
			WorkspaceID: "ws2",
			Role:        "VIEWER",
		},
	}
	for i, tc := range createAPIKeyTestCase {
		ts.T().Run(fmt.Sprintf("getRoleUsersTestCases-%d", i), func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			r.Header.Set("Content-Type", "application/json; charset=utf8")
			r.Header.Set(httpserver.XKeibiUserIDHeader, tc.UserID)
			r.Header.Set(httpserver.XKeibiWorkspaceIDHeader, tc.WorkspaceID)
			r.Header.Set(httpserver.XKeibiWorkspaceNameHeader, tc.WorkspaceID)
			r.Header.Set(httpserver.XKeibiUserRoleHeader, tc.Role)
			r.Header.Set(httpserver.XKeibiMaxUsersHeader, "10")
			r.Header.Set(httpserver.XKeibiMaxConnectionsHeader, "10")
			r.Header.Set(httpserver.XKeibiMaxResourcesHeader, "10")
			w := httptest.NewRecorder()

			c := echo.New().NewContext(r, w)

			c.SetPath("/auth/api/v1/key/create")
			err := (&echo.DefaultBinder{}).BindBody(c, tc.Request)
			ts.NoError(err, "request faild to be added to the body")
			err = ts.httpRoutes.CreateAPIKey(c)
			if err != nil {
				ts.Equal("[SearchUsersByRole] invalid status code: 204, body=", err.Error())
				return
			}

			var response api.GetRoleUsersResponse
			if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
				ts.T().Fatalf("json decode: %v", err)
			}
			fmt.Println("the response is : ", response)

		})
	}
}

func (ts *testSuite) TestListAPIKeys() {
	mockKeysInDb(&ts.httpRoutes.db)
	listAPIKeysTestCases := []struct {
		WorkspaceID string
		Error       error
	}{
		{
			WorkspaceID: "ws1",
		},
		{
			WorkspaceID: "ws2",
		},
		{
			WorkspaceID: "ws3",
		},
		{
			WorkspaceID: "ws4",
		},
	}
	for i, tc := range listAPIKeysTestCases {
		ts.T().Run(fmt.Sprintf("listAPIKeysTestCases-%d", i), func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			r.Header.Set("Content-Type", "application/json; charset=utf8")
			r.Header.Set(httpserver.XKeibiWorkspaceIDHeader, tc.WorkspaceID)
			w := httptest.NewRecorder()

			c := echo.New().NewContext(r, w)

			c.SetPath("/auth/api/v1/key")

			err := ts.httpRoutes.ListAPIKeys(c)
			ts.NoError(err, "error while running the API")

			var response []api.WorkspaceApiKey
			if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
				ts.T().Fatalf("json decode: %v", err)
			}
			fmt.Println(response)
		})
	}
}

func (ts *testSuite) TestSuspendAPIKey() {
	mockKeysInDb(&ts.httpRoutes.db)
	suspendAPIKeyTestCases := []struct {
		WorkspaceID string
		KeyID       uint
		Error       error
	}{
		{
			WorkspaceID: "ws1",
			KeyID:       1,
		},
		{
			WorkspaceID: "ws1",
			KeyID:       3,
		},
		{
			WorkspaceID: "ws1",
			KeyID:       5,
		},
	}
	for i, tc := range suspendAPIKeyTestCases {
		ts.T().Run(fmt.Sprintf("suspendAPIKeyTestCases-%d", i), func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			r.Header.Set("Content-Type", "application/json; charset=utf8")
			r.Header.Set(httpserver.XKeibiWorkspaceIDHeader, tc.WorkspaceID)
			w := httptest.NewRecorder()

			c := echo.New().NewContext(r, w)

			c.SetPath("/auth/api/v1/key/:id/suspend")
			c.SetParamNames("id")
			c.SetParamValues(strconv.FormatUint(uint64(tc.KeyID), 10))

			err := ts.httpRoutes.SuspendAPIKey(c)
			ts.NoError(err, "error while running the API")

			after, err := ts.httpRoutes.db.GetApiKey(tc.WorkspaceID, tc.KeyID)
			if tc.KeyID == 2 {
				ts.Equal(uint(0), after.ID)
			} else {
				ts.False(after.Active)
			}
		})
	}
}

func (ts *testSuite) TestActiveAPIKey() {
	mockKeysInDb(&ts.httpRoutes.db)
	activeAPIKeyTestCases := []struct {
		WorkspaceID string
		KeyID       uint
		Error       error
	}{
		{
			WorkspaceID: "ws1",
			KeyID:       1,
		},
		{
			WorkspaceID: "ws1",
			KeyID:       2,
		},
		{
			WorkspaceID: "ws1",
			KeyID:       5,
		},
	}
	for i, tc := range activeAPIKeyTestCases {
		ts.T().Run(fmt.Sprintf("activeAPIKeyTestCases-%d", i), func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			r.Header.Set("Content-Type", "application/json; charset=utf8")
			r.Header.Set(httpserver.XKeibiWorkspaceIDHeader, tc.WorkspaceID)
			w := httptest.NewRecorder()

			c := echo.New().NewContext(r, w)

			c.SetPath("/auth/api/v1/key/:id/activate")
			c.SetParamNames("id")
			c.SetParamValues(strconv.FormatUint(uint64(tc.KeyID), 10))

			err := ts.httpRoutes.ActivateAPIKey(c)
			ts.NoError(err, "error while running the API")

			after, err := ts.httpRoutes.db.GetApiKey(tc.WorkspaceID, tc.KeyID)
			if tc.KeyID == 2 {
				ts.Equal(uint(0), after.ID)
			} else {
				ts.True(after.Active)
			}
		})
	}
}

func (ts *testSuite) TestDeleteAPIKey() {
	mockKeysInDb(&ts.httpRoutes.db)
	deleteAPIKeyTestCases := []struct {
		WorkspaceID string
		KeyID       uint
		Error       error
	}{
		{
			WorkspaceID: "ws1",
			KeyID:       1,
		},
		{
			WorkspaceID: "ws1",
			KeyID:       2,
		},
		{
			WorkspaceID: "ws1",
			KeyID:       5,
		},
		{
			WorkspaceID: "ws1",
			KeyID:       10,
		},
	}
	for i, tc := range deleteAPIKeyTestCases {
		ts.T().Run(fmt.Sprintf("deleteAPIKeyTestCases-%d", i), func(t *testing.T) {
			r := httptest.NewRequest(http.MethodDelete, "/", nil)
			r.Header.Set("Content-Type", "application/json; charset=utf8")
			r.Header.Set(httpserver.XKeibiWorkspaceIDHeader, tc.WorkspaceID)
			w := httptest.NewRecorder()

			c := echo.New().NewContext(r, w)

			c.SetPath("/auth/api/v1/key/:id/delete")
			c.SetParamNames("id")
			c.SetParamValues(strconv.FormatUint(uint64(tc.KeyID), 10))

			before, err := ts.httpRoutes.db.GetApiKey(tc.WorkspaceID, tc.KeyID)
			if tc.KeyID == 10 || tc.KeyID == 2 {
				ts.Empty(before)
			}

			err = ts.httpRoutes.DeleteAPIKey(c)
			ts.NoError(err, "error while running the API")

			after, err := ts.httpRoutes.db.GetApiKey(tc.WorkspaceID, tc.KeyID)
			ts.Empty(after)
		})
	}
}

func (ts *testSuite) TestGetAPIKey() {
	mockKeysInDb(&ts.httpRoutes.db)
	getAPIKeyTestCases := []struct {
		WorkspaceID string
		KeyID       uint
		Error       error
	}{
		{
			WorkspaceID: "ws1",
			KeyID:       1,
		},
		{
			WorkspaceID: "ws1",
			KeyID:       2,
		},
		{
			WorkspaceID: "ws1",
			KeyID:       5,
		},
		{
			WorkspaceID: "ws1",
			KeyID:       10,
		},
	}
	for i, tc := range getAPIKeyTestCases {
		ts.T().Run(fmt.Sprintf("getAPIKeyTestCases-%d", i), func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			r.Header.Set("Content-Type", "application/json; charset=utf8")
			r.Header.Set(httpserver.XKeibiWorkspaceIDHeader, tc.WorkspaceID)
			w := httptest.NewRecorder()

			c := echo.New().NewContext(r, w)

			c.SetPath("/auth/api/v1/key/:id")
			c.SetParamNames("id")
			c.SetParamValues(strconv.FormatUint(uint64(tc.KeyID), 10))

			err := ts.httpRoutes.GetAPIKey(c)
			if tc.KeyID == 10 || tc.KeyID == 2 {
				ts.Equal("code=404, message=api key not found", err.Error())
			} else {
				ts.NoError(err, "error while running the API")
				var response api.WorkspaceApiKey
				if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
					ts.T().Fatalf("json decode: %v", err)
				}
				ts.NotEmpty(response)
				fmt.Println("RES: ", response)
			}

		})
	}
}

func (ts *testSuite) TestGetRoleKeys() {
	mockKeysInDb(&ts.httpRoutes.db)
	getRoleKeysTestCases := []struct {
		WorkspaceID string
		Role        api.Role
		Error       error
	}{
		{
			WorkspaceID: "ws1",
			Role:        api.AdminRole,
		},
		{
			WorkspaceID: "ws2",
			Role:        api.EditorRole,
		},
		{
			WorkspaceID: "ws3",
			Role:        api.ViewerRole,
		},
		{
			WorkspaceID: "ws1",
			Role:        api.ViewerRole,
		},
	}
	for i, tc := range getRoleKeysTestCases {
		ts.T().Run(fmt.Sprintf("getRoleKeysTestCases-%d", i), func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			r.Header.Set("Content-Type", "application/json; charset=utf8")
			r.Header.Set(httpserver.XKeibiWorkspaceIDHeader, tc.WorkspaceID)
			w := httptest.NewRecorder()

			c := echo.New().NewContext(r, w)

			c.SetPath("/auth/api/v1/role/:role/keys")
			c.SetParamNames("role")
			c.SetParamValues(string(tc.Role))
			err := ts.httpRoutes.GetRoleKeys(c)
			ts.NoError(err, "error while running the API")
			var response []api.WorkspaceApiKey
			if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
				ts.T().Fatalf("json decode: %v", err)
			}
			if (tc.WorkspaceID == "ws3") || (tc.WorkspaceID == "ws1" && tc.Role == api.ViewerRole) {
				ts.Empty(response)
			} else {
				ts.NotEmpty(response)
			}
			fmt.Println("RES: ", response)
		})
	}
}

func (ts *testSuite) TestUpdateKeyRole() {
	mockKeysInDb(&ts.httpRoutes.db)
	updateKeyRoleTestCases := []struct {
		WorkspaceID string
		Request     api.UpdateKeyRoleRequest
		Error       error
	}{
		{
			WorkspaceID: "ws1",
			Request:     api.UpdateKeyRoleRequest{ID: 1, RoleName: api.EditorRole},
		},
		{
			WorkspaceID: "ws4",
			Request:     api.UpdateKeyRoleRequest{ID: 2, RoleName: api.EditorRole},
		},
		{
			WorkspaceID: "ws1",
			Request:     api.UpdateKeyRoleRequest{ID: 10, RoleName: api.EditorRole},
		},
		{
			WorkspaceID: "ws1",
			Request:     api.UpdateKeyRoleRequest{ID: 3, RoleName: api.ViewerRole},
		},
	}
	for i, tc := range updateKeyRoleTestCases {
		ts.T().Run(fmt.Sprintf("updateKeyRoleTestCases-%d", i), func(t *testing.T) {
			body, err := json.Marshal(tc.Request)
			ts.NoError(err)
			r := httptest.NewRequest(http.MethodPost, "/", bytes.NewBuffer(body))
			r.Header.Set("Content-Type", "application/json; charset=utf8")
			r.Header.Set(httpserver.XKeibiWorkspaceIDHeader, tc.WorkspaceID)

			w := httptest.NewRecorder()

			c := echo.New().NewContext(r, w)

			err = ts.httpRoutes.UpdateKeyRole(c)
			ts.NoError(err, "error while running the API")
			if (tc.Request.ID == 10) || (tc.WorkspaceID == "ws1" && tc.Request.ID == 3) {
			} else {
				after, _ := ts.httpRoutes.db.GetApiKey(tc.WorkspaceID, tc.Request.ID)
				ts.Equal(tc.Request.RoleName, after.Role)
			}
		})
	}
}

func (ts *testSuite) TestGetRoles() {
	GetRolesTestCases := []struct {
		WorkspaceID string
		Error       error
	}{
		{
			WorkspaceID: "ws1",
		},
	}
	for i, tc := range GetRolesTestCases {
		ts.T().Run(fmt.Sprintf("GetRolesTestCases-%d", i), func(t *testing.T) {
			r := httptest.NewRequest(http.MethodPost, "/", nil)
			r.Header.Set("Content-Type", "application/json; charset=utf8")
			r.Header.Set(httpserver.XKeibiWorkspaceIDHeader, tc.WorkspaceID)

			w := httptest.NewRecorder()

			c := echo.New().NewContext(r, w)

			err := ts.httpRoutes.ListRoles(c)
			ts.NoError(err, "error while running the API")
			var response []api.RolesListResponse
			if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
				ts.T().Fatalf("json decode: %v", err)
			}
			fmt.Println(response)
		})
	}
}

func (ts *testSuite) TestGetRoleDetails() {
	GetRoleDetailsTestCases := []struct {
		WorkspaceID string
		RoleName    string
		Error       error
	}{
		{
			WorkspaceID: "ws1",
			RoleName:    "VIEWER",
		},
	}
	for i, tc := range GetRoleDetailsTestCases {
		ts.T().Run(fmt.Sprintf("GetRoleDetailsTestCases-%d", i), func(t *testing.T) {
			r := httptest.NewRequest(http.MethodPost, "/", nil)
			r.Header.Set("Content-Type", "application/json; charset=utf8")
			r.Header.Set(httpserver.XKeibiWorkspaceIDHeader, tc.WorkspaceID)

			w := httptest.NewRecorder()

			c := echo.New().NewContext(r, w)
			c.SetPath("/auth/api/v1/roles/:role")
			c.SetParamNames("role")
			c.SetParamValues(tc.RoleName)

			err := ts.httpRoutes.RoleDetails(c)
			ts.NoError(err, "error while running the API")
			var response api.RoleDetailsResponse
			if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
				ts.T().Fatalf("json decode: %v", err)
			}
			fmt.Println(response)
		})
	}
}
