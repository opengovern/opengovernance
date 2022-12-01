package auth

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"net/http"
	"strings"
	"time"

	"github.com/go-redis/cache/v8"
	"gitlab.com/keibiengine/keibi-engine/pkg/internal/httpclient"

	"github.com/go-redis/redis/v8"

	api2 "gitlab.com/keibiengine/keibi-engine/pkg/workspace/api"

	"gitlab.com/keibiengine/keibi-engine/pkg/workspace/client"

	"gitlab.com/keibiengine/keibi-engine/pkg/auth/api"

	"github.com/coreos/go-oidc/v3/oidc"
	envoycore "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	envoyauth "github.com/envoyproxy/go-control-plane/envoy/service/auth/v3"
	envoytype "github.com/envoyproxy/go-control-plane/envoy/type/v3"
	"github.com/gogo/googleapis/google/rpc"
	"github.com/labstack/echo/v4"
	"gitlab.com/keibiengine/keibi-engine/pkg/internal/httpserver"
	"go.uber.org/zap"
	"google.golang.org/genproto/googleapis/rpc/status"
	"gorm.io/gorm"
)

type Server struct {
	host string

	db              Database
	verifier        *oidc.IDTokenVerifier
	logger          *zap.Logger
	workspaceClient client.WorkspaceServiceClient
	rdb             *redis.Client
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
	internalUser, err := s.GetUserByExternalID(user.ExternalUserID)
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			s.logger.Warn("denied access due to failure in retrieving internal user",
				zap.String("reqId", httpRequest.Id),
				zap.String("path", httpRequest.Path),
				zap.String("method", httpRequest.Method),
				zap.Error(err))
			return unAuth, nil
		}

		// user does not exists
		internalUser = User{
			ExternalID: user.ExternalUserID,
		}

		if len(user.Email) == 0 {
			s.logger.Warn("denied access due to failure in retrieving auth user email",
				zap.String("reqId", httpRequest.Id),
				zap.String("path", httpRequest.Path),
				zap.String("method", httpRequest.Method),
				zap.Error(err))
			return unAuth, nil
		}
		internalUser.Email = user.Email

		if err := s.db.CreateUser(&internalUser); err != nil {
			s.logger.Warn("denied access due to failure in creating the user",
				zap.String("reqId", httpRequest.Id),
				zap.String("path", httpRequest.Path),
				zap.String("method", httpRequest.Method),
				zap.Error(err))
			return unAuth, nil
		}
	}

	workspaceName := strings.TrimPrefix(httpRequest.Path, "/")
	if idx := strings.Index(workspaceName, "/"); idx > 0 {
		workspaceName = workspaceName[:idx]
	}

	var rb RoleBinding
	if workspaceName == "keibi" {
		rb = RoleBinding{
			UserID: internalUser.ID,
			Role:   api.EditorRole,
		}
	} else {
		rb, err = s.GetRoleBindingForWorkspace(internalUser.ID, workspaceName)
		if err != nil {
			s.logger.Warn("denied access due to failure in retrieving auth user host",
				zap.String("reqId", httpRequest.Id),
				zap.String("path", httpRequest.Path),
				zap.String("method", httpRequest.Method),
				zap.String("workspace", workspaceName),
				zap.Error(err))
			return unAuth, nil
		}

		if s.rdb != nil {
			err = s.rdb.SetEX(context.Background(), "last_access_"+workspaceName, time.Now().UnixMilli(),
				30*24*time.Hour).Err()
			if err != nil {
				s.logger.Warn("denied access due to failure in setting last access",
					zap.String("reqId", httpRequest.Id),
					zap.String("path", httpRequest.Path),
					zap.String("method", httpRequest.Method),
					zap.String("workspace", workspaceName),
					zap.Error(err))
				return unAuth, nil
			}
		}
	}

	s.logger.Debug("granted access",
		zap.String("userId", rb.UserID.String()),
		zap.String("role", string(rb.Role)),
		zap.String("reqId", httpRequest.Id),
		zap.String("path", httpRequest.Path),
		zap.String("method", httpRequest.Method),
	)

	var limits api2.WorkspaceLimits
	if workspaceName != "keibi" {
		limits, err = s.GetWorkspaceLimits(rb, workspaceName)
		if err != nil {
			s.logger.Warn("denied access due to failure in retrieving limits",
				zap.String("reqId", httpRequest.Id),
				zap.String("path", httpRequest.Path),
				zap.String("method", httpRequest.Method),
				zap.String("workspace", workspaceName),
				zap.Error(err))
			return nil, err
		}
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
							Key:   httpserver.XKeibiWorkspaceNameHeader,
							Value: rb.WorkspaceName,
						},
					},
					{
						Header: &envoycore.HeaderValue{
							Key:   httpserver.XKeibiUserIDHeader,
							Value: rb.UserID.String(),
						},
					},
					{
						Header: &envoycore.HeaderValue{
							Key:   httpserver.XKeibiUserRoleHeader,
							Value: string(rb.Role),
						},
					},
					{
						Header: &envoycore.HeaderValue{
							Key:   httpserver.XKeibiMaxUsersHeader,
							Value: fmt.Sprintf("%d", limits.MaxUsers),
						},
					},
					{
						Header: &envoycore.HeaderValue{
							Key:   httpserver.XKeibiMaxConnectionsHeader,
							Value: fmt.Sprintf("%d", limits.MaxConnections),
						},
					},
					{
						Header: &envoycore.HeaderValue{
							Key:   httpserver.XKeibiMaxResourcesHeader,
							Value: fmt.Sprintf("%d", limits.MaxResources),
						},
					},
				},
			},
		},
	}, nil
}

func (s Server) GetUserByEmail(email string) (User, error) {
	key := "cache-user-email-" + email

	var res User
	if s.cache != nil {
		if err := s.cache.Get(context.Background(), key, &res); err == nil {
			return res, nil
		}
	}

	user, err := s.db.GetUserByEmail(email)
	if err != nil {
		return User{}, err
	}

	if s.cache != nil {
		s.cache.Set(&cache.Item{
			Ctx:   context.Background(),
			Key:   key,
			Value: user,
			TTL:   15 * time.Minute,
		})
	}

	return user, nil
}

func (s Server) GetUserByExternalID(externalID string) (User, error) {
	key := "cache-user-externalID-" + externalID

	var res User
	if s.cache != nil {
		if err := s.cache.Get(context.Background(), key, &res); err == nil {
			return res, nil
		}
	}

	user, err := s.db.GetUserByExternalID(externalID)
	if err != nil {
		return User{}, err
	}

	if s.cache != nil {
		s.cache.Set(&cache.Item{
			Ctx:   context.Background(),
			Key:   key,
			Value: user,
			TTL:   15 * time.Minute,
		})
	}

	return user, nil
}

func (s Server) GetRoleBindingForWorkspace(userId uuid.UUID, workspaceName string) (RoleBinding, error) {
	key := "cache-rb-" + userId.String() + "-" + workspaceName

	var res RoleBinding
	if s.cache != nil {
		if err := s.cache.Get(context.Background(), key, &res); err == nil {
			return res, nil
		}
	}

	rb, err := s.db.GetRoleBindingForWorkspace(userId, workspaceName)
	if err != nil {
		return RoleBinding{}, err
	}

	if s.cache != nil {
		s.cache.Set(&cache.Item{
			Ctx:   context.Background(),
			Key:   key,
			Value: rb,
			TTL:   1 * time.Minute,
		})
	}

	return rb, nil
}

func (s Server) GetWorkspaceLimits(rb RoleBinding, workspaceName string) (api2.WorkspaceLimits, error) {
	key := "cache-limits-" + workspaceName

	var res api2.WorkspaceLimits
	if s.cache != nil {
		if err := s.cache.Get(context.Background(), key, &res); err == nil {
			return res, nil
		}
	}

	limits, err := s.workspaceClient.GetLimits(&httpclient.Context{UserRole: rb.Role, UserID: rb.UserID.String(),
		WorkspaceName: workspaceName}, true)
	if err != nil {
		return api2.WorkspaceLimits{}, err
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
	Workspaces     []string `json:"https://app.keibi.io/workspaces"`
	Email          string   `json:"https://app.keibi.io/email"`
	ExternalUserID string   `json:"sub"`
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

	t, err := s.verifier.Verify(context.Background(), token)
	if err != nil {
		return nil, err
	}

	var u userClaim
	if err := t.Claims(&u); err != nil {
		return nil, err
	}

	return &u, nil
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
