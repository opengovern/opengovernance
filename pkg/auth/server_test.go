package auth

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt"
	workspaceClient "gitlab.com/keibiengine/keibi-engine/pkg/workspace/client"

	"github.com/coreos/go-oidc/v3/oidc"
	envoyauth "github.com/envoyproxy/go-control-plane/envoy/service/auth/v3"
	"github.com/gogo/googleapis/google/rpc"
	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
	"gitlab.com/keibiengine/keibi-engine/pkg/auth/api"
	"gitlab.com/keibiengine/keibi-engine/pkg/internal/dockertest"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type ServerSuite struct {
	suite.Suite

	orm *gorm.DB
	db  Database
}

func (s *ServerSuite) SetupSuite() {
	s.orm = dockertest.StartupPostgreSQL(s.T())
	s.db = NewDatabase(s.orm)

	http.HandleFunc("/api/v1/workspaces/limits/workspace1", func(writer http.ResponseWriter, request *http.Request) {
		writer.Write([]byte("{}"))
	})
	http.HandleFunc("/api/v1/workspaces/limits/workspace2", func(writer http.ResponseWriter, request *http.Request) {
		writer.Write([]byte("{}"))
	})
	http.HandleFunc("/api/v1/workspaces/limits/ThirdWorkspace", func(writer http.ResponseWriter, request *http.Request) {
		writer.Write([]byte("{}"))
	})
	go http.ListenAndServe("localhost:8080", nil)
}

func (s *ServerSuite) BeforeTest(suiteName, testName string) {
	require := s.Require()

	err := s.db.Initialize()
	require.NoError(err, "initialize db")
}

func (s *ServerSuite) AfterTest(suiteName, testName string) {
	require := s.Require()

	tx := s.db.orm.Exec("DROP TABLE IF EXISTS role_bindings;")
	require.NoError(tx.Error, "drop role_bindings")

	tx = s.db.orm.Exec("DROP TABLE IF EXISTS users;")
	require.NoError(tx.Error, "drop users")
}

func TestServer(t *testing.T) {
	suite.Run(t, &ServerSuite{})
}

const testIssuer = "http://example.com"

type testKeySet struct {
}

func (tks *testKeySet) VerifySignature(ctx context.Context, jwt string) (payload []byte, err error) {
	parts := strings.Split(jwt, ".")
	if len(parts) < 2 {
		return nil, fmt.Errorf("oidc: malformed jwt, expected 3 parts got %d", len(parts))
	}
	payload, err = base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("oidc: malformed jwt payload: %v", err)
	}
	return payload, nil
}

func (s *ServerSuite) TestServer_Check() {
	require := s.Require()

	logger, err := zap.NewDevelopment()
	require.NoError(err)

	server := Server{
		host: "app.keibi.io",
		//extAuth: &mocks.Provider{},
		db: s.db,
		verifier: oidc.NewVerifier(testIssuer, &testKeySet{}, &oidc.Config{
			SupportedSigningAlgs: []string{"HS256"},
			SkipClientIDCheck:    true,
			SkipExpiryCheck:      true,
			SkipIssuerCheck:      true,
		}),
		logger:          logger,
		workspaceClient: workspaceClient.NewWorkspaceClient("http://localhost:8080"),
	}

	user1 := User{
		ID:         uuid.New(),
		ExternalID: "1",
		Email:      "1@gmail.com",
	}

	user2 := User{
		ID:         uuid.New(),
		ExternalID: "2",
		Email:      "2@gmail.com",
	}

	rb1 := RoleBinding{
		UserID:        user1.ID,
		WorkspaceName: "workspace1",
		Role:          api.AdminRole,
		AssignedAt:    time.Now(),
	}

	rb2 := RoleBinding{
		UserID:        user2.ID,
		WorkspaceName: "workspace1",
		Role:          api.ViewerRole,
		AssignedAt:    time.Now(),
	}

	rb3 := RoleBinding{
		UserID:        user1.ID,
		WorkspaceName: "workspace2",
		Role:          api.EditorRole,
		AssignedAt:    time.Now(),
	}

	require.NoError(server.db.CreateUser(&user1))
	require.NoError(server.db.CreateUser(&user2))

	require.NoError(server.db.CreateOrUpdateRoleBinding(&rb1), "create rolebinding 1")
	require.NoError(server.db.CreateOrUpdateRoleBinding(&rb2), "create rolebinding 2")
	require.NoError(server.db.CreateOrUpdateRoleBinding(&rb3), "create rolebinding 3")
	//
	//server.extAuth.(*mocks.Provider).On("FetchUser", mock.Anything, "3").Return(extauth.AzureADUser{
	//	ID:   "3",
	//	Mail: "3@gmail.com",
	//}, error(nil))
	//
	//server.extAuth.(*mocks.Provider).On("FetchUser", mock.Anything, "4").Return(extauth.AzureADUser{}, extauth.ErrUserNotExists)

	type args struct {
		email             string
		workspace         string
		auth              bool
		expectedId        uuid.UUID
		expectedRole      api.Role
		expectedWorkspace string
	}

	tests := []args{
		{
			// {
			//  "sub": "1",
			//  "given_name": "user1",
			//  "emails": [
			//    "1@gmail.com"
			//  ]
			//}
			workspace:         "workspace1",
			email:             "1@gmail.com",
			auth:              true,
			expectedId:        rb1.UserID,
			expectedRole:      rb1.Role,
			expectedWorkspace: rb1.WorkspaceName,
		},
		{
			// {
			//  "sub": "2",
			//  "given_name": "user2",
			//  "emails": [
			//    "2@gmail.com"
			//  ]
			//}
			email:             "2@gmail.com",
			workspace:         "workspace1",
			auth:              true,
			expectedId:        rb2.UserID,
			expectedRole:      rb2.Role,
			expectedWorkspace: rb2.WorkspaceName,
		},
		{
			// {
			//  "sub": "1",
			//  "given_name": "user1",
			//  "emails": [
			//    "1@gmail.com"
			//  ]
			//}
			workspace:         "workspace2",
			email:             "1@gmail.com",
			auth:              true,
			expectedId:        rb3.UserID,
			expectedRole:      rb3.Role,
			expectedWorkspace: rb3.WorkspaceName,
		},
		{
			// {
			//  "sub": "2",
			//  "given_name": "user2",
			//  "emails": [
			//    "2@gmail.com"
			//  ]
			//}
			email:     "2@gmail.com",
			workspace: "workspace2",
			auth:      false,
		},
		{
			// {
			//  "sub": "3",
			//  "given_name": "user3",
			//  "emails": [
			//    "3@gmail.com"
			//  ]
			//}
			email:     "3@gmail.com",
			workspace: "workspace1",
			auth:      false,
		},
		{
			// {
			//  "sub": "4",
			//  "given_name": "user4",
			//  "emails": [
			//    "4@gmail.com"
			//  ]
			//}
			email:     "4@gmail.com",
			workspace: "ThirdWorkspace",
			auth:      false,
		},
		// wrong token
		{
			email:     "5@gmail.com",
			workspace: "ThirdWorkspace",
			auth:      false,
		},
	}
	key := []byte("SecretYouShouldHide")
	for i, tc := range tests {
		claims := userClaim{
			Access:         map[string]api.Role{tc.workspace: api.AdminRole},
			Email:          tc.email,
			ExternalUserID: strings.Split(tc.email, "@")[0],
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenString, err := token.SignedString(key)
		require.NoError(err)

		req := &envoyauth.CheckRequest{
			Attributes: &envoyauth.AttributeContext{
				Request: &envoyauth.AttributeContext_Request{
					Http: &envoyauth.AttributeContext_HttpRequest{
						Path: tc.workspace + "/",
						Headers: map[string]string{
							"authorization": "Bearer " + tokenString,
						},
					},
				},
			},
		}
		resp, err := server.Check(context.Background(), req)
		if tc.auth {
			require.EqualValues(rpc.OK, resp.Status.Code, i)
			headers := resp.GetOkResponse().GetHeaders()

			ws := headers[0].GetHeader().GetValue()
			id := headers[1].GetHeader().GetValue()
			role := headers[2].GetHeader().GetValue()
			require.Equal(tc.expectedWorkspace, ws)
			require.Equal(tc.expectedId.String(), id)
			require.Equal(string(tc.expectedRole), role)
		} else {
			require.EqualValues(rpc.UNAUTHENTICATED, resp.Status.Code)
			require.Empty(resp.GetDeniedResponse().GetHeaders())
		}
	}

	u, err := s.db.GetUserByExternalID("3")
	require.NoError(err)
	require.Equal(u.Email, "3@gmail.com")
	require.Equal(uuid.RFC4122, u.ID.Variant())

	u, err = s.db.GetUserByExternalID("4")
	require.NoError(err)
}
