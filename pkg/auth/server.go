package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/coreos/go-oidc/v3/oidc"
	envoycore "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	envoyauth "github.com/envoyproxy/go-control-plane/envoy/service/auth/v3"
	envoytype "github.com/envoyproxy/go-control-plane/envoy/type/v3"
	"github.com/gogo/googleapis/google/rpc"
	"github.com/labstack/echo/v4"
	"gitlab.com/keibiengine/keibi-engine/pkg/auth/api"

	//"gitlab.com/keibiengine/keibi-engine/pkg/auth/extauth"
	"gitlab.com/keibiengine/keibi-engine/pkg/internal/httpserver"
	"go.uber.org/zap"
	"google.golang.org/genproto/googleapis/rpc/status"
	"gorm.io/gorm"
)

type Server struct {
	host string

	db Database
	//extAuth  extauth.Provider
	verifier *oidc.IDTokenVerifier
	logger   *zap.Logger
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

	internalUser, err := s.db.GetUserByExternalID(user.ExternalUserID)
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			s.logger.Warn("denied access due to failure in retrieving internal user",
				zap.String("reqId", httpRequest.Id),
				zap.String("path", httpRequest.Path),
				zap.String("method", httpRequest.Method),
				zap.Error(err))
			return unAuth, nil
		}

		//
		//azureADUser, err := s.extAuth.FetchUser(ctx, user.ExternalUserID)
		//if err != nil {
		//	s.logger.Warn("denied access due to failure in retrieving auth user",
		//		zap.String("reqId", httpRequest.Id),
		//		zap.String("path", httpRequest.Path),
		//		zap.String("method", httpRequest.Method),
		//		zap.Error(err))
		//	return unAuth, nil
		//}
		internalUser = User{
			//Email:      azureADUser.Mail,
			ExternalID: user.ExternalUserID,
		}

		if internalUser.Email == "" {
			if len(user.Emails) == 0 {
				s.logger.Warn("denied access due to failure in retrieving auth user email",
					zap.String("reqId", httpRequest.Id),
					zap.String("path", httpRequest.Path),
					zap.String("method", httpRequest.Method),
					zap.Error(err))
				return unAuth, nil
			}

			sort.Strings(user.Emails)
			internalUser.Email = user.Emails[0]
		}
		if err := s.db.CreateUser(&internalUser); err != nil {
			s.logger.Warn("denied access due to failure in creating the user",
				zap.String("reqId", httpRequest.Id),
				zap.String("path", httpRequest.Path),
				zap.String("method", httpRequest.Method),
				zap.Error(err))
			return unAuth, nil
		}
	}

	var rb RoleBinding
	//TODO-Saleh
	if httpRequest.Host == s.host {
		// e.g: app.keibi.io
		rb = RoleBinding{
			UserID: internalUser.ID,
			Role:   api.EditorRole,
		}
	} else {
		// e.g: workspace.app.keibi.io
		workspaceName := strings.TrimSuffix(httpRequest.Host, s.host)
		workspaceName = strings.TrimSuffix(workspaceName, ".")
		rb, err = s.db.GetRoleBindingForWorkspace(user.ExternalUserID, workspaceName)
		if err != nil {
			s.logger.Warn("denied access due to failure in retrieving auth user host",
				zap.String("reqId", httpRequest.Id),
				zap.String("path", httpRequest.Path),
				zap.String("method", httpRequest.Method),
				zap.String("workspace", workspaceName),
				zap.Error(err))
			return unAuth, nil
		}
	}

	s.logger.Debug("granted access",
		zap.String("userId", rb.UserID.String()),
		zap.String("extUserId", rb.ExternalID),
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
				},
			},
		},
	}, nil
}

type userClaim struct {
	ExternalUserID string   `json:"sub"`
	GivenName      string   `json:"given_name"`
	Emails         []string `json:"emails"`
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
