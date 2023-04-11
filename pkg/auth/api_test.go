package auth

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/suite"
	"gitlab.com/keibiengine/keibi-engine/pkg/auth/api"
	"gitlab.com/keibiengine/keibi-engine/pkg/auth/auth0"
	"gitlab.com/keibiengine/keibi-engine/pkg/internal/httpserver"
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
	// ts.testServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	// 	if r.Method == http.MethodDelete {
	// 		mockDeleteUser(w, r)
	// 	}
	// 	if r.Method == http.MethodGet {
	// 		mockGetUsers(w, r)
	// 	}
	// }))

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

			var response api.WorkspaceRoleBinding
			if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
				ts.T().Fatalf("json decode: %v", err)
			}
			fmt.Println(response)
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
				ts.Equal(tc.Role, response[0].Role)
				ts.Equal("testTenant", response[0].TenantId)
				ts.Equal("user1@test.com", response[0].Email)
				ts.True(response[0].EmailVerified)
			}

		})
	}
}
