package auth

import (
	"fmt"
	"net/http"
	"testing"

	envoyauth "github.com/envoyproxy/go-control-plane/envoy/service/auth/v2"
	"github.com/stretchr/testify/require"
	"gitlab.com/keibiengine/keibi-engine/pkg/auth/api"
	"go.uber.org/zap"
)

func TestServer_Authorize(t *testing.T) {
	logger, err := zap.NewDevelopment()
	require.NoError(t, err)

	authEcho := buildEchoRoutes()

	server := Server{
		logger:   logger,
		authEcho: authEcho,
	}

	type args struct {
		method string
		path   string
		role   api.Role
	}

	tests := []struct {
		args args
		auth bool
	}{
		{
			args: args{
				method: http.MethodGet,
				path:   "/inventory/api/v1/locations/aws",
				role:   api.ViewerRole,
			},
			auth: true,
		},
		{
			args: args{
				method: http.MethodPost,
				path:   "/inventory/api/v1/locations/aws",
				role:   api.ViewerRole,
			},
			auth: false,
		},
		{
			args: args{
				method: http.MethodPost,
				path:   "/onboard/api/v1/source/aws",
				role:   api.ViewerRole,
			},
			auth: false,
		},
		{
			args: args{
				method: http.MethodPost,
				path:   "/onboard/api/v1/source/aws",
				role:   api.EditorRole,
			},
			auth: true,
		},
		{
			args: args{
				method: http.MethodPost,
				path:   "/onboard/api/v1/source/aws",
				role:   api.AdminRole,
			},
			auth: true,
		},
	}
	for _, tt := range tests {
		var name string
		if tt.auth {
			name = fmt.Sprintf("User with role (%s) HAS access to endpoint (%s %s)", tt.args.role, tt.args.method, tt.args.path)
		} else {
			name = fmt.Sprintf("User with role (%s) DOESN'T HAVE access to endpoint (%s %s)", tt.args.role, tt.args.method, tt.args.path)
		}
		t.Run(name, func(t *testing.T) {
			req := &envoyauth.CheckRequest{
				Attributes: &envoyauth.AttributeContext{
					Request: &envoyauth.AttributeContext_Request{
						Http: &envoyauth.AttributeContext_HttpRequest{
							Method: tt.args.method,
							Path:   tt.args.path,
						},
					},
				},
			}

			rb := RoleBinding{
				UserID: "some-random-id",
				Role:   tt.args.role,
			}

			if err := server.Authorize(req, rb); (err == nil) != tt.auth {
				t.Errorf("Authorize() error = %v, auth %v", err, tt.auth)
			}
		})
	}
}
