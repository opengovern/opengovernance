package auth

import (
	"context"
	"crypto/rsa"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/kaytu-io/kaytu-engine/pkg/internal/httpclient"
	"github.com/kaytu-io/kaytu-engine/pkg/internal/httpserver"

	"github.com/golang-jwt/jwt"

	"github.com/labstack/echo/v4"

	api2 "github.com/kaytu-io/kaytu-engine/pkg/workspace/api"

	"github.com/kaytu-io/kaytu-engine/pkg/workspace/client"

	"github.com/kaytu-io/kaytu-engine/pkg/auth/api"

	"github.com/coreos/go-oidc/v3/oidc"
	envoycore "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	envoyauth "github.com/envoyproxy/go-control-plane/envoy/service/auth/v3"
	envoytype "github.com/envoyproxy/go-control-plane/envoy/type/v3"
	"github.com/go-redis/cache/v8"
	"github.com/gogo/googleapis/google/rpc"
	"go.uber.org/zap"
	"google.golang.org/genproto/googleapis/rpc/status"
)

type Server struct {
	host string

	kaytuPublicKey  *rsa.PublicKey
	verifier        *oidc.IDTokenVerifier
	verifierNative  *oidc.IDTokenVerifier
	logger          *zap.Logger
	workspaceClient client.WorkspaceServiceClient
	cache           *cache.Cache
}

func (s Server) Check(ctx context.Context, req *envoyauth.CheckRequest) (*envoyauth.CheckResponse, error) {
	unAuth := &envoyauth.CheckResponse{
		Status: &status.Status{
			Code: int32(rpc.UNAUTHENTICATED),
		},
		HttpResponse: &envoyauth.CheckResponse_DeniedResponse{
			DeniedResponse: &envoyauth.DeniedHttpResponse{
				Status: &envoytype.HttpStatus{Code: 401},
				Body:   http.StatusText(http.StatusUnauthorized),
			},
		},
	}

	httpRequest := req.GetAttributes().GetRequest().GetHttp()
	headers := httpRequest.GetHeaders()

	user, err := s.Verify(ctx, headers[strings.ToLower(echo.HeaderAuthorization)])
	if err != nil {
		s.logger.Warn("denied access due to unsuccessful token verification",
			zap.String("reqId", httpRequest.Id),
			zap.String("path", httpRequest.Path),
			zap.String("method", httpRequest.Method),
			zap.Error(err))
		return unAuth, nil
	}

	user.Email = strings.ToLower(strings.TrimSpace(user.Email))
	if user.Email == "" {
		s.logger.Warn("denied access due to failure to get email from token",
			zap.String("reqId", httpRequest.Id),
			zap.String("path", httpRequest.Path),
			zap.String("method", httpRequest.Method),
			zap.Error(err))
		return unAuth, nil
	}

	workspaceName := strings.TrimPrefix(httpRequest.Path, "/")
	if idx := strings.Index(workspaceName, "/"); idx > 0 {
		workspaceName = workspaceName[:idx]
	}

	if headerWorkspace, ok := headers["workspace-name"]; ok {
		workspaceName = headerWorkspace
	}

	rb, limits, err := s.GetWorkspaceByName(workspaceName, user)
	if err != nil {
		s.logger.Warn("denied access due to failure in getting workspace",
			zap.String("reqId", httpRequest.Id),
			zap.String("path", httpRequest.Path),
			zap.String("method", httpRequest.Method),
			zap.String("workspace", workspaceName),
			zap.Error(err))
		return unAuth, nil
	}

	return &envoyauth.CheckResponse{
		Status: &status.Status{
			Code: int32(rpc.OK),
		},

		HttpResponse: &envoyauth.CheckResponse_OkResponse{
			OkResponse: &envoyauth.OkHttpResponse{
				Headers: []*envoycore.HeaderValueOption{
					{
						Header: &envoycore.HeaderValue{
							Key:   httpserver.XKaytuWorkspaceIDHeader,
							Value: rb.WorkspaceID,
						},
					},
					{
						Header: &envoycore.HeaderValue{
							Key:   httpserver.XKaytuWorkspaceNameHeader,
							Value: rb.WorkspaceName,
						},
					},
					{
						Header: &envoycore.HeaderValue{
							Key:   httpserver.XKaytuUserIDHeader,
							Value: rb.UserID,
						},
					},
					{
						Header: &envoycore.HeaderValue{
							Key:   httpserver.XKaytuUserRoleHeader,
							Value: string(rb.RoleName),
						},
					},
					{
						Header: &envoycore.HeaderValue{
							Key:   httpserver.XKaytuMaxUsersHeader,
							Value: fmt.Sprintf("%d", limits.MaxUsers),
						},
					},
					{
						Header: &envoycore.HeaderValue{
							Key:   httpserver.XKaytuMaxConnectionsHeader,
							Value: fmt.Sprintf("%d", limits.MaxConnections),
						},
					},
					{
						Header: &envoycore.HeaderValue{
							Key:   httpserver.XKaytuMaxResourcesHeader,
							Value: fmt.Sprintf("%d", limits.MaxResources),
						},
					},
				},
			},
		},
	}, nil
}

func (s Server) GetWorkspaceLimits(rb api.RoleBinding, workspaceName string, ignoreUsage bool) (api2.WorkspaceLimitsUsage, error) {
	key := "cache-limits-" + workspaceName

	var res api2.WorkspaceLimitsUsage
	if s.cache != nil {
		if err := s.cache.Get(context.Background(), key, &res); err == nil {
			return res, nil
		}
	}

	limits, err := s.workspaceClient.GetLimits(&httpclient.Context{UserRole: rb.RoleName, UserID: rb.UserID,
		WorkspaceName: workspaceName}, workspaceName, ignoreUsage)
	if err != nil {
		return api2.WorkspaceLimitsUsage{}, err
	}

	if s.cache != nil {
		s.cache.Set(&cache.Item{
			Ctx:   context.Background(),
			Key:   key,
			Value: limits,
			TTL:   1 * time.Minute,
		})
	}

	return limits, nil
}

type userClaim struct {
	WorkspaceAccess map[string]api.Role `json:"https://app.kaytu.io/workspaceAccess"`
	GlobalAccess    *api.Role           `json:"https://app.kaytu.io/globalAccess"`
	Email           string              `json:"https://app.kaytu.io/email"`
	ExternalUserID  string              `json:"sub"`
}

func (u userClaim) Valid() error {
	return nil
}

func (s Server) Verify(ctx context.Context, authToken string) (*userClaim, error) {
	if !strings.HasPrefix(authToken, "Bearer ") {
		return nil, errors.New("invalid authorization token")
	}
	token := strings.TrimSpace(strings.TrimPrefix(authToken, "Bearer "))
	if token == "" {
		return nil, errors.New("missing authorization token")
	}

	var u userClaim
	t, err := s.verifierNative.Verify(context.Background(), token)
	if err == nil {
		if err := t.Claims(&u); err != nil {
			return nil, err
		}

		return &u, nil
	}

	t, err = s.verifier.Verify(context.Background(), token)
	if err == nil {
		if err := t.Claims(&u); err != nil {
			return nil, err
		}

		return &u, nil
	}

	_, errk := jwt.ParseWithClaims(token, &u, func(token *jwt.Token) (interface{}, error) {
		return s.kaytuPublicKey, nil
	})
	if errk == nil {
		return &u, nil
	} else {
		fmt.Println("failed to auth with kaytu cred due to", errk)
	}
	return nil, err
}

func (s Server) GetWorkspaceByName(workspaceName string, user *userClaim) (api.RoleBinding, api2.WorkspaceLimitsUsage, error) {
	var rb api.RoleBinding
	var limits api2.WorkspaceLimitsUsage
	var err error

	rb = api.RoleBinding{
		UserID:        user.ExternalUserID,
		WorkspaceID:   "",
		WorkspaceName: "",
		RoleName:      api.EditorRole,
	}

	if workspaceName != "kaytu" {
		limits, err = s.GetWorkspaceLimits(rb, workspaceName, true)
		if err != nil {
			return rb, limits, err
		}

		rb.UserID = user.ExternalUserID
		rb.WorkspaceName = workspaceName
		rb.WorkspaceID = limits.ID

		if rl, ok := user.WorkspaceAccess[limits.ID]; ok {
			rb.RoleName = rl
		} else if user.GlobalAccess != nil {
			rb.RoleName = *user.GlobalAccess
		} else {
			return rb, limits, errors.New("access denied")
		}
	}

	return rb, limits, nil
}

func newAuth0OidcVerifier(ctx context.Context, auth0Domain, clientId string) (*oidc.IDTokenVerifier, error) {
	provider, err := oidc.NewProvider(ctx, auth0Domain)
	if err != nil {
		return nil, err
	}

	return provider.Verifier(&oidc.Config{
		ClientID:          clientId,
		SkipClientIDCheck: true,
	}), nil
}
