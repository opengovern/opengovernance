package auth

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"testing"
	"time"

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
		logger: logger,
		verifier: oidc.NewVerifier(testIssuer, &testKeySet{}, &oidc.Config{
			SkipClientIDCheck: true,
			SkipExpiryCheck:   true,
			SkipIssuerCheck:   true,
		}),
		//extAuth: &mocks.Provider{},
		db:   s.db,
		host: "app.keibi.io",
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
		Email:         user1.Email,
		WorkspaceName: "workspace1",
		Role:          api.AdminRole,
		AssignedAt:    time.Now(),
	}

	rb2 := RoleBinding{
		UserID:        user2.ID,
		Email:         user2.Email,
		WorkspaceName: "workspace1",
		Role:          api.ViewerRole,
		AssignedAt:    time.Now(),
	}

	rb3 := RoleBinding{
		UserID:        user1.ID,
		Email:         user1.Email,
		WorkspaceName: "workspace2",
		Role:          api.EditorRole,
		AssignedAt:    time.Now(),
	}

	require.NoError(server.db.CreateUser(&user1))
	require.NoError(server.db.CreateUser(&user2))

	require.NoError(server.db.CreateOrUpdateRoleBinding(&rb1), "create rolebinding 1", rb1.Email)
	require.NoError(server.db.CreateOrUpdateRoleBinding(&rb2), "create rolebinding 2", rb2.Email)
	require.NoError(server.db.CreateOrUpdateRoleBinding(&rb3), "create rolebinding 3", rb3.Email)
	//
	//server.extAuth.(*mocks.Provider).On("FetchUser", mock.Anything, "3").Return(extauth.AzureADUser{
	//	ID:   "3",
	//	Mail: "3@gmail.com",
	//}, error(nil))
	//
	//server.extAuth.(*mocks.Provider).On("FetchUser", mock.Anything, "4").Return(extauth.AzureADUser{}, extauth.ErrUserNotExists)

	type args struct {
		token             string
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
			token:             "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxIiwiZ2l2ZW5fbmFtZSI6InVzZXIxIiwiZW1haWxzIjpbIjFAZ21haWwuY29tIl19.SM3WDS8i_yBHKEigcd9bYE4N8dH4eJwiqo5Q5QtTsh9oYyOE2_kkfSpH5h5if6-z2pr6uPgJlY1a0VEE0MTVueeCsfyag3uDP941AKpchgYgJQCXxCpXL9fBf5I691hl4kZ9eVMraOrb39AK1wCn9anjrIEonKTmfdIxJMa6UpsvwUBjMLPubKSzv66UcqfMRbdSHh__cXWbH1lsj9MURMUpkd_HmJVLrhVGPCdd9jaQWQAUwWmGN9UOr7WJaEbpCN93b_6f508awN8F8ByEmmhXj1JG2zlHQctgjAt8sVeUh_LVBEq29hG1f3AyZvTxOMY_Xz14yrasDV79x1KWaA",
			workspace:         "workspace1",
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
			token:             "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIyIiwiZ2l2ZW5fbmFtZSI6InVzZXIyIiwiZW1haWxzIjpbIjJAZ21haWwuY29tIl19.ei5Agr-hbKhaAAzgkaz_ou5IxpIsiVr0roo0FF46P-t4v2zGRyd63NaIp6mrCZkheA2_gK54N6t0v8SQuJr9nr0d2w0rWypMRXda2i2cltpNgvPJk3undh4-YcJZ8qfNI71zk_5NZK-iygNh6qC7N62SNu9kG_DAaIzvzq3hCyE6p7jzptKLP4ZVR6_lNqyukMdufPZATwJNWwtEGJRrpYOY73XwjP4dezRNe1dRFHrS6kCl8AGfLQcbVdnFZT27fuYAuT2z0LrmRs91msrWVdKz-GL4ro09L-mN9HUU2_lbgvnE7bMyBAdTpgqnrFbL3XSFdnQJtRFbdGl0SawEjg",
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
			token:             "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxIiwiZ2l2ZW5fbmFtZSI6InVzZXIxIiwiZW1haWxzIjpbIjFAZ21haWwuY29tIl19.SM3WDS8i_yBHKEigcd9bYE4N8dH4eJwiqo5Q5QtTsh9oYyOE2_kkfSpH5h5if6-z2pr6uPgJlY1a0VEE0MTVueeCsfyag3uDP941AKpchgYgJQCXxCpXL9fBf5I691hl4kZ9eVMraOrb39AK1wCn9anjrIEonKTmfdIxJMa6UpsvwUBjMLPubKSzv66UcqfMRbdSHh__cXWbH1lsj9MURMUpkd_HmJVLrhVGPCdd9jaQWQAUwWmGN9UOr7WJaEbpCN93b_6f508awN8F8ByEmmhXj1JG2zlHQctgjAt8sVeUh_LVBEq29hG1f3AyZvTxOMY_Xz14yrasDV79x1KWaA",
			workspace:         "workspace2",
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
			token:     "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIyIiwiZ2l2ZW5fbmFtZSI6InVzZXIyIiwiZW1haWxzIjpbIjJAZ21haWwuY29tIl19.ei5Agr-hbKhaAAzgkaz_ou5IxpIsiVr0roo0FF46P-t4v2zGRyd63NaIp6mrCZkheA2_gK54N6t0v8SQuJr9nr0d2w0rWypMRXda2i2cltpNgvPJk3undh4-YcJZ8qfNI71zk_5NZK-iygNh6qC7N62SNu9kG_DAaIzvzq3hCyE6p7jzptKLP4ZVR6_lNqyukMdufPZATwJNWwtEGJRrpYOY73XwjP4dezRNe1dRFHrS6kCl8AGfLQcbVdnFZT27fuYAuT2z0LrmRs91msrWVdKz-GL4ro09L-mN9HUU2_lbgvnE7bMyBAdTpgqnrFbL3XSFdnQJtRFbdGl0SawEjg",
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
			token:     "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIzIiwiZ2l2ZW5fbmFtZSI6InVzZXIzIiwiZW1haWxzIjpbIjNAZ21haWwuY29tIl19.kKrCbMYjsZcvoc_LZ-z0mUgijMpGArFAPtRY_ETjgLJT4cz3a_VM6xU5HiKGR3BtVHRy9DSyjiaP4q_hARHJL2PivaxXp2LZvgIdz_pG1kA0b_6PiOqwtNlBKU8Nzrka1cn2sA5XkCHHagQ-mDPJBlx32MReVKnQePOg5CyREyVs2pgAzNBNC5YFzXIq5rKgaf0fpncvlebxBlermfuyXTCFxcLfnMIaAwTqw6Xo901Qq2GSG755O6G21TxMYDbwZARJC_-4on1BQA0BtKdcSuC15xRx29qAdl0Dkz4SkZZjVhqsYKDBjH01SHJfq6Rukmgt2aA1_B0JZdUrruEY-Q",
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
			token:     "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiI0IiwiZ2l2ZW5fbmFtZSI6InVzZXI0IiwiZW1haWxzIjpbIjRAZ21haWwuY29tIl19.bghMsGaEoRauL6QrV1T5yhL3ylECd7jjMWy-6XripZuKu1x1oToqJFW-id1ZgzqyeScM8qiA8DK2Nl8iPT4hY0BjeTpziq85FTBm8dKyKAWI4JrjVbzgicArD25fuvtL5zEG1zr0PCtvYiVxF6wpDmrBnDjxAFw_U3HcHyVZ6axKiR9LMifTiimdEt0elPUcsVpNj8TO0MTTMvX6l6jcwwhJAO_30LexV8-xfjtQ9dzxdCDN9f4YuY3F87bANgU8w2LxVLZgxclhDZ4oXbHBcRLnmekChY0UiXO7Q4XHU17BscBcLAy5X0lmmhZsmKNQrfbkpinylPATmSnZ41x2Gg",
			workspace: "ThirdWorkspace",
			auth:      false,
		},
		// wrong token
		{
			token:     "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIzIiwiZ2l2ZW5fbmFtZSI6InVzZXIzIiwiZW1haWxzIjpbIjNAZ21haWwuY29tIl19.owxoniWrQ_jefJU3rxzQGtJPiz3-ww4V6AxD_-fbsD0",
			workspace: "ThirdWorkspace",
			auth:      false,
		},
	}

	for i, tc := range tests {
		req := &envoyauth.CheckRequest{
			Attributes: &envoyauth.AttributeContext{
				Request: &envoyauth.AttributeContext_Request{
					Http: &envoyauth.AttributeContext_HttpRequest{
						Headers: map[string]string{
							"authorization": "Bearer " + tc.token,
						},
						Host: fmt.Sprintf("%s.%s", tc.workspace, server.host),
					},
				},
			},
		}
		resp, err := server.Check(context.Background(), req)
		require.NoError(err)

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
	require.ErrorIs(err, gorm.ErrRecordNotFound)
}
