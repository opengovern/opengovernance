package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	envoycore "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	envoyauth "github.com/envoyproxy/go-control-plane/envoy/service/auth/v3"
	envoytype "github.com/envoyproxy/go-control-plane/envoy/type/v3"
	"github.com/gogo/googleapis/google/rpc"
	"github.com/labstack/echo/v4"
	"gitlab.com/keibiengine/keibi-engine/pkg/auth/api"
	"go.uber.org/zap"
	"google.golang.org/genproto/googleapis/rpc/status"
)

const roleCtxKey = "role"

type Server struct {
	db       Database
	verifier *oidc.IDTokenVerifier

	authEcho *echo.Echo

	logger *zap.Logger
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

	rb, err := s.FindUserRoleBinding(ctx, user)
	if err != nil {
		s.logger.Warn("denied access due to failure in retrieving user role",
			zap.String("reqId", httpRequest.Id),
			zap.String("path", httpRequest.Path),
			zap.String("method", httpRequest.Method),
			zap.Error(err))
		return unAuth, nil
	}

	if err := s.Authorize(req, rb); err != nil {
		s.logger.Warn("denied access due to unauthorized access",
			zap.String("userId", rb.UserID),
			zap.String("role", string(rb.Role)),
			zap.String("reqId", httpRequest.Id),
			zap.String("path", httpRequest.Path),
			zap.String("method", httpRequest.Method),
			zap.Error(err))
		return unAuth, nil
	}

	s.logger.Debug("granted access",
		zap.String("userId", rb.UserID),
		zap.String("role", string(rb.Role)),
		zap.String("reqId", httpRequest.Id),
		zap.String("path", httpRequest.Path),
		zap.String("method", httpRequest.Method),
	)

	return &envoyauth.CheckResponse{
		Status: &status.Status{
			Code: int32(rpc.OK),
		},
		HttpResponse: &envoyauth.CheckResponse_OkResponse{
			OkResponse: &envoyauth.OkHttpResponse{
				Headers: []*envoycore.HeaderValueOption{
					{
						Header: &envoycore.HeaderValue{
							Key:   "X-Keibi-UserId",
							Value: rb.UserID,
						},
					},
				},
			},
		},
	}, nil
}

type User struct {
	UserID    string   `json:"sub"`
	GivenName string   `json:"given_name"`
	Emails    []string `json:"emails"`
}

func (s Server) Verify(ctx context.Context, authToken string) (*User, error) {
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

	var u User
	if err := t.Claims(&u); err != nil {
		return nil, err
	}

	return &u, nil
}

func (s Server) FindUserRoleBinding(ctx context.Context, user *User) (RoleBinding, error) {
	defaultRb := RoleBinding{
		UserID:     user.UserID,
		Name:       user.GivenName,
		Emails:     user.Emails,
		Role:       api.ViewerRole,
		AssignedAt: time.Now(),
	}

	err := s.db.GetRoleBindingOrCreate(&defaultRb)
	if err != nil {
		return RoleBinding{}, err
	}

	return defaultRb, nil
}

func (s Server) Authorize(req *envoyauth.CheckRequest, rb RoleBinding) error {
	eCtx := s.authEcho.NewContext(&http.Request{}, nil)
	httpReq := req.GetAttributes().GetRequest().GetHttp()
	s.authEcho.Router().Find(httpReq.Method, httpReq.Path, eCtx)

	eCtx.Set(roleCtxKey, rb)
	if err := eCtx.Handler()(eCtx); err != nil {
		return err
	}

	return nil
}

func newOidcVerifier(ctx context.Context, tenantName, tenantId, clientId, policy string) (*oidc.IDTokenVerifier, error) {
	// Azure AD B2C OpenID Connect endpoint is not fully compliant. The issuer and discovery endpoint
	// don't exactly match. This is the recommended way to override the expected issuer.
	// See: https://github.com/MicrosoftDocs/azure-docs/issues/38427
	discovery := fmt.Sprintf("https://%s.b2clogin.com/%s/%s/v2.0", tenantName, tenantId, policy)
	issuer := fmt.Sprintf("https://%s.b2clogin.com/%s/v2.0/", tenantName, tenantId)

	provider, err := oidc.NewProvider(oidc.InsecureIssuerURLContext(ctx, issuer), discovery)
	if err != nil {
		return nil, err
	}

	return provider.Verifier(&oidc.Config{
		ClientID: clientId,
	}), nil
}

func buildEchoRoutes() *echo.Echo {
	e := echo.New()
	for _, endpoint := range endpoints {
		e.Add(endpoint.Method, endpoint.Path, authHandlerFunc(endpoint.MinimumRole))
	}

	return e
}

func authHandlerFunc(minRole api.Role) func(ctx echo.Context) error {
	return func(ctx echo.Context) error {
		rb := ctx.Get(roleCtxKey).(RoleBinding)
		if !hasAccess(rb.Role, minRole) {
			return echo.ErrUnauthorized
		}

		return nil
	}
}
