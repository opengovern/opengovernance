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
	payload []byte
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
		db:         s.db,
		hostSuffix: ".app.keibi.io",
	}

	rb1 := RoleBinding{
		UserID:        uuid.New(),
		ExternalID:    "1",
		WorkspaceName: "workspace1",
		Role:          api.AdminRole,
		AssignedAt:    time.Now(),
	}

	rb2 := RoleBinding{
		UserID:        uuid.New(),
		ExternalID:    "2",
		WorkspaceName: "workspace1",
		Role:          api.ViewerRole,
		AssignedAt:    time.Now(),
	}

	rb3 := RoleBinding{
		UserID:        rb1.UserID,
		ExternalID:    "1",
		WorkspaceName: "workspace2",
		Role:          api.EditorRole,
		AssignedAt:    time.Now(),
	}

	require.NoError(server.db.CreateOrUpdateRoleBinding(&rb1), "create rolebinding 1", rb1.ExternalID)
	require.NoError(server.db.CreateOrUpdateRoleBinding(&rb2), "create rolebinding 2", rb2.ExternalID)
	require.NoError(server.db.CreateOrUpdateRoleBinding(&rb3), "create rolebinding 3", rb3.ExternalID)

	type args struct {
		token             string
		host              string
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
			host:              "workspace1.app.keibi.io",
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
			host:              "workspace1.app.keibi.io",
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
			host:              "workspace2.app.keibi.io",
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
			token: "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIyIiwiZ2l2ZW5fbmFtZSI6InVzZXIyIiwiZW1haWxzIjpbIjJAZ21haWwuY29tIl19.ei5Agr-hbKhaAAzgkaz_ou5IxpIsiVr0roo0FF46P-t4v2zGRyd63NaIp6mrCZkheA2_gK54N6t0v8SQuJr9nr0d2w0rWypMRXda2i2cltpNgvPJk3undh4-YcJZ8qfNI71zk_5NZK-iygNh6qC7N62SNu9kG_DAaIzvzq3hCyE6p7jzptKLP4ZVR6_lNqyukMdufPZATwJNWwtEGJRrpYOY73XwjP4dezRNe1dRFHrS6kCl8AGfLQcbVdnFZT27fuYAuT2z0LrmRs91msrWVdKz-GL4ro09L-mN9HUU2_lbgvnE7bMyBAdTpgqnrFbL3XSFdnQJtRFbdGl0SawEjg",
			host:  "workspace2.app.keibi.io",
			auth:  false,
		},
		{
			// {
			//  "sub": "3",
			//  "given_name": "user3",
			//  "emails": [
			//    "3@gmail.com"
			//  ]
			//}
			token: "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIzIiwiZ2l2ZW5fbmFtZSI6InVzZXIzIiwiZW1haWxzIjpbIjNAZ21haWwuY29tIl19.kKrCbMYjsZcvoc_LZ-z0mUgijMpGArFAPtRY_ETjgLJT4cz3a_VM6xU5HiKGR3BtVHRy9DSyjiaP4q_hARHJL2PivaxXp2LZvgIdz_pG1kA0b_6PiOqwtNlBKU8Nzrka1cn2sA5XkCHHagQ-mDPJBlx32MReVKnQePOg5CyREyVs2pgAzNBNC5YFzXIq5rKgaf0fpncvlebxBlermfuyXTCFxcLfnMIaAwTqw6Xo901Qq2GSG755O6G21TxMYDbwZARJC_-4on1BQA0BtKdcSuC15xRx29qAdl0Dkz4SkZZjVhqsYKDBjH01SHJfq6Rukmgt2aA1_B0JZdUrruEY-Q",
			host:  "workspace1.app.keibi.io",
			auth:  false,
		},
		// wrong token
		{
			token: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIzIiwiZ2l2ZW5fbmFtZSI6InVzZXIzIiwiZW1haWxzIjpbIjNAZ21haWwuY29tIl19.owxoniWrQ_jefJU3rxzQGtJPiz3-ww4V6AxD_-fbsD0",
			host:  "ThirdWorkspace",
			auth:  false,
		},
		// user not found
		{
			token: "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJub3QgZm91bmQiLCJnaXZlbl9uYW1lIjoidXNlcm5vdGZvdW5kIiwiZW1haWxzIjpbIm5vdGZvdW5kQGdtYWlsLmNvbSJdfQ.m1LtiG_A5Qy-9JJ4eLM_G3n7rl698f7msC5mMH0bU0JIAbFzeLumbaVvR3__hnxHqCBKvddKmRHM9RflYRoC4hCXk3hDRw9hAfp4E2UYYWhj3SZy7-LLR7uBLQxtIOi_ARtxyRAXAYAZ0fUhjJRFowlKDnCD7BG97HE_SocDFCFGzheOR9UU0iRg0eb6BiJWglovflOke1Ncm8J_0nZj2zUzluITB10ujUjw6AvjUT1uCPMadbZVvnDfDAgM3H046-OpJXwutXvylbeT8WjuBGrOx9hpmQSPbTug-j5egzK4ZySHs6d4GtxtkpbRKegP9FCq2FcjsWbn3lVde4zagA",
			host:  "NotFoundWorkspace",
			auth:  false,
		},
		// host not found
		{
			token: "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiI0IiwiZ2l2ZW5fbmFtZSI6InVzZXI0IiwiZW1haWxzIjpbIjRAZ21haWwuY29tIl19.bghMsGaEoRauL6QrV1T5yhL3ylECd7jjMWy-6XripZuKu1x1oToqJFW-id1ZgzqyeScM8qiA8DK2Nl8iPT4hY0BjeTpziq85FTBm8dKyKAWI4JrjVbzgicArD25fuvtL5zEG1zr0PCtvYiVxF6wpDmrBnDjxAFw_U3HcHyVZ6axKiR9LMifTiimdEt0elPUcsVpNj8TO0MTTMvX6l6jcwwhJAO_30LexV8-xfjtQ9dzxdCDN9f4YuY3F87bANgU8w2LxVLZgxclhDZ4oXbHBcRLnmekChY0UiXO7Q4XHU17BscBcLAy5X0lmmhZsmKNQrfbkpinylPATmSnZ41x2Gg",
			host:  "ThirdWorkspace",
			auth:  false,
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
						Host: tc.host,
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
}
