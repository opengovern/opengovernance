package auth

import (
	"context"
	"crypto/rsa"
	"crypto/sha512"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	dexApi "github.com/dexidp/dex/api/v2"
	envoyauth "github.com/envoyproxy/go-control-plane/envoy/service/auth/v3"
	"github.com/google/uuid"
	"github.com/jackc/pgtype"
	client2 "github.com/opengovern/opengovernance/pkg/compliance/client"
	client4 "github.com/opengovern/opengovernance/pkg/describe/client"
	client3 "github.com/opengovern/opengovernance/services/integration/client"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"time"

	api2 "github.com/opengovern/og-util/pkg/api"
	"github.com/opengovern/og-util/pkg/httpclient"
	"github.com/opengovern/og-util/pkg/httpserver"

	metadataClient "github.com/opengovern/opengovernance/pkg/metadata/client"
	"github.com/opengovern/opengovernance/pkg/metadata/models"

	"github.com/opengovern/opengovernance/pkg/auth/db"

	"github.com/golang-jwt/jwt"
	"github.com/opengovern/opengovernance/pkg/auth/auth0"

	"github.com/labstack/echo/v4"
	"github.com/opengovern/opengovernance/pkg/auth/api"
	"go.uber.org/zap"
)

var (
	//go:embed email/invite.html
	inviteEmailTemplate string
)

type httpRoutes struct {
	logger *zap.Logger
	//emailService    email.Service
	complianceClient  client2.ComplianceServiceClient
	integrationClient client3.IntegrationServiceClient
	schedulerClient   client4.SchedulerServiceClient
	auth0Service      *auth0.Service
	metadataBaseUrl   string
	kaytuPrivateKey   *rsa.PrivateKey
	db                db.Database
	authServer        *Server
}

func (r *httpRoutes) Register(e *echo.Echo) {
	v1 := e.Group("/api/v1")

	v1.GET("/check", r.Check)

	v1.PUT("/user/role/binding", httpserver.AuthorizeHandler(r.PutRoleBinding, api2.AdminRole))
	v1.DELETE("/user/role/binding", httpserver.AuthorizeHandler(r.DeleteRoleBinding, api2.AdminRole))
	v1.GET("/user/role/bindings", httpserver.AuthorizeHandler(r.GetRoleBindings, api2.EditorRole))
	v1.GET("/workspace/role/bindings", httpserver.AuthorizeHandler(r.GetWorkspaceRoleBindings, api2.AdminRole))
	v1.GET("/users", httpserver.AuthorizeHandler(r.GetUsers, api2.EditorRole))
	v1.GET("/user/:user_id", httpserver.AuthorizeHandler(r.GetUserDetails, api2.EditorRole))
	v1.GET("/me", httpserver.AuthorizeHandler(r.GetMe, api2.EditorRole))
	v1.POST("/user/invite", httpserver.AuthorizeHandler(r.Invite, api2.AdminRole))
	v1.PUT("/user/preferences", httpserver.AuthorizeHandler(r.ChangeUserPreferences, api2.ViewerRole))

	v1.POST("/key/create", httpserver.AuthorizeHandler(r.CreateAPIKey, api2.EditorRole))
	v1.GET("/keys", httpserver.AuthorizeHandler(r.ListAPIKeys, api2.EditorRole))
	v1.DELETE("/key/:name/delete", httpserver.AuthorizeHandler(r.DeleteAPIKey, api2.EditorRole))

	v3 := e.Group("/api/v3")
	v3.POST("/user/create", httpserver.AuthorizeHandler(r.CreateUser, api2.ViewerRole))
	v3.POST("/user/update", httpserver.AuthorizeHandler(r.UpdateUser, api2.ViewerRole))
	v3.GET("/user/password/check", httpserver.AuthorizeHandler(r.CheckUserPasswordChangeRequired, api2.ViewerRole))
	v3.POST("/user/password/reset", httpserver.AuthorizeHandler(r.ResetUserPassword, api2.ViewerRole))
	v3.DELETE("/user/:email_address/delete", httpserver.AuthorizeHandler(r.DeleteUser, api2.AdminRole))
}

func bindValidate(ctx echo.Context, i interface{}) error {
	if err := ctx.Bind(i); err != nil {
		return err
	}

	if err := ctx.Validate(i); err != nil {
		return err
	}

	return nil
}

func (r *httpRoutes) Check(ctx echo.Context) error {
	checkRequest := envoyauth.CheckRequest{
		Attributes: &envoyauth.AttributeContext{
			Request: &envoyauth.AttributeContext_Request{
				Http: &envoyauth.AttributeContext_HttpRequest{
					Headers: make(map[string]string),
				},
			},
		},
	}

	for k, v := range ctx.Request().Header {
		if len(v) == 0 {
			checkRequest.Attributes.Request.Http.Headers[k] = ""
		} else {
			checkRequest.Attributes.Request.Http.Headers[k] = v[0]
		}
	}
	originalUri, err := url.Parse(ctx.Request().Header.Get("X-Original-URI"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid original uri")
	}
	checkRequest.Attributes.Request.Http.Path = originalUri.Path
	checkRequest.Attributes.Request.Http.Method = ctx.Request().Header.Get("X-Original-Method")

	res, err := r.authServer.Check(ctx.Request().Context(), &checkRequest)
	if err != nil {
		return err
	}

	if res.Status.Code != int32(codes.OK) {
		return echo.NewHTTPError(http.StatusForbidden, res.Status.Message)
	}

	if res.GetOkResponse() == nil {
		return echo.NewHTTPError(http.StatusForbidden, "no ok response")
	}

	for _, header := range res.GetOkResponse().GetHeaders() {
		if header == nil || header.Header == nil {
			continue
		}
		ctx.Response().Header().Set(header.Header.Key, header.Header.Value)
	}

	return ctx.NoContent(http.StatusOK)
}

// PutRoleBinding godoc
//
//	@Summary		Update User Role
//	@Description	Updates the role of a user in the workspace.
//	@Security		BearerToken
//	@Tags			users
//	@Produce		json
//	@Param			request	body		api.PutRoleBindingRequest	true	"Request Body"
//	@Success		200		{object}	nil
//	@Router			/auth/api/v1/user/role/binding [put]
func (r *httpRoutes) PutRoleBinding(ctx echo.Context) error {
	var req api.PutRoleBindingRequest
	if err := bindValidate(ctx, &req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	workspaceID := httpserver.GetWorkspaceID(ctx)

	if httpserver.GetUserID(ctx) == req.UserID {
		return echo.NewHTTPError(http.StatusBadRequest, "admin user permission can't be modified by self")
	}

	// The WorkspaceManager service will call this API to set the AdminRole
	// for the admin user on behalf of him. Allow for the Admin to only set its
	// role to admin for that user case
	auth0User, err := r.auth0Service.GetUser(req.UserID)
	if err != nil {
		return err
	}

	if _, ok := auth0User.AppMetadata.WorkspaceAccess[workspaceID]; !ok {
		hctx := httpclient.FromEchoContext(ctx)
		metadataService := metadataClient.NewMetadataServiceClient(fmt.Sprintf(metadataBaseUrl, workspaceID))
		cnf, err := metadataService.GetConfigMetadata(hctx, models.MetadataKeyUserLimit)
		if err != nil {
			return err
		}
		maxUsers := cnf.GetValue().(int)

		users, err := r.auth0Service.SearchUsers(workspaceID, nil, nil, nil)
		if err != nil {
			return err
		}

		if len(users)+1 > maxUsers {
			return echo.NewHTTPError(http.StatusNotAcceptable, "cannot invite new user, max users reached")
		}
	}

	auth0User.AppMetadata.WorkspaceAccess[workspaceID] = req.RoleName

	if auth0User.AppMetadata.ConnectionIDs == nil {
		auth0User.AppMetadata.ConnectionIDs = map[string][]string{}
	}
	auth0User.AppMetadata.ConnectionIDs[workspaceID] = req.ConnectionIDs
	err = r.auth0Service.PatchUserAppMetadata(req.UserID, auth0User.AppMetadata, nil)
	if err != nil {
		return err
	}
	return ctx.NoContent(http.StatusOK)
}

// DeleteRoleBinding godoc
//
//	@Summary		Revoke User Access
//	@Description	Revokes a user's access to the workspace
//	@Security		BearerToken
//	@Tags			users
//	@Produce		json
//	@Param			userId	query		string	true	"User ID"
//	@Success		200		{object}	nil
//	@Router			/auth/api/v1/user/role/binding [delete]
func (r *httpRoutes) DeleteRoleBinding(ctx echo.Context) error {
	userId := ctx.QueryParam("userId")
	if httpserver.GetUserID(ctx) == userId {
		return echo.NewHTTPError(http.StatusBadRequest, "admin user permission can't be modified by self")
	}

	workspaceID := httpserver.GetWorkspaceID(ctx)
	auth0User, err := r.auth0Service.GetUser(userId)
	if err != nil {
		return err
	}

	delete(auth0User.AppMetadata.WorkspaceAccess, workspaceID)
	if len(auth0User.AppMetadata.WorkspaceAccess) == 0 {
		auth0User.AppMetadata.WorkspaceAccess = nil
	}

	err = r.auth0Service.PatchUserAppMetadata(userId, auth0User.AppMetadata, nil)
	if err != nil {
		return err
	}
	return ctx.NoContent(http.StatusOK)
}

// GetRoleBindings godoc
//
//	@Summary		Get User Roles
//	@Description	Retrieves the roles that the user who sent the request has in all workspaces they are a member of.
//	@Security		BearerToken
//	@Tags			users
//	@Produce		json
//	@Success		200	{object}	api.GetRoleBindingsResponse
//	@Router			/auth/api/v1/user/role/bindings [get]
func (r *httpRoutes) GetRoleBindings(ctx echo.Context) error {
	userID := httpserver.GetUserID(ctx)

	var resp api.GetRoleBindingsResponse
	usr, err := r.auth0Service.GetUser(userID)
	if err != nil {
		r.logger.Warn("failed to get user from auth0 due to", zap.Error(err))
		return err
	}

	if usr != nil {
		for wsID, role := range usr.AppMetadata.WorkspaceAccess {
			resp.RoleBindings = append(resp.RoleBindings, api.UserRoleBinding{
				WorkspaceID: wsID,
				RoleName:    role,
			})
		}
		resp.GlobalRoles = usr.AppMetadata.GlobalAccess

		timeNow := time.Now().Format("2006-01-02 15:00:00 MST")
		doUpdate := false
		if usr.AppMetadata.MemberSince == nil {
			usr.AppMetadata.MemberSince = &timeNow
			doUpdate = true
		}
		if usr.AppMetadata.LastLogin == nil || *usr.AppMetadata.LastLogin != timeNow {
			usr.AppMetadata.LastLogin = &timeNow
			usr.LastLogin = time.Now()
			doUpdate = true
		}

		if doUpdate {
			err = r.auth0Service.PatchUserAppMetadata(usr.UserId, usr.AppMetadata, &usr.LastLogin)
			if err != nil {
				r.logger.Error("failed to update user metadata", zap.String("userId", userID), zap.Error(err))
			}
		}
	} else {
		r.logger.Warn("user not found in auth0", zap.String("externalID", userID))
	}
	return ctx.JSON(http.StatusOK, resp)
}

// GetWorkspaceRoleBindings godoc
//
//	@Summary		Workspace user roleBindings.
//	@Description	Get all the RoleBindings of the workspace. RoleBinding defines the roles and actions a user can perform. There are currently three roles (admin, editor, viewer). The workspace path is based on the DNS such as (workspace1.app.opengovernance.io)
//	@Security		BearerToken
//	@Tags			users
//	@Produce		json
//	@Success		200	{object}	api.GetWorkspaceRoleBindingResponse
//	@Router			/auth/api/v1/workspace/role/bindings [get]
func (r *httpRoutes) GetWorkspaceRoleBindings(ctx echo.Context) error {
	userID := httpserver.GetUserID(ctx)
	workspaceID := httpserver.GetWorkspaceID(ctx)
	users, err := r.auth0Service.SearchUsersByWorkspace(workspaceID)
	if err != nil {
		return err
	}

	var resp api.GetWorkspaceRoleBindingResponse
	userHasAccess := false
	for _, u := range users {
		status := api.InviteStatus_PENDING
		if u.EmailVerified {
			status = api.InviteStatus_ACCEPTED
		}
		if u.UserId == userID {
			userHasAccess = true
		}

		resp = append(resp, api.WorkspaceRoleBinding{
			UserID:              u.UserId,
			UserName:            u.Name,
			Email:               u.Email,
			RoleName:            u.AppMetadata.WorkspaceAccess[workspaceID],
			Status:              status,
			LastActivity:        u.AppMetadata.LastLogin,
			CreatedAt:           u.AppMetadata.MemberSince,
			ScopedConnectionIDs: u.AppMetadata.ConnectionIDs[workspaceID],
		})
	}

	if !userHasAccess && userID != api2.GodUserID {
		//TODO-Saleh
		r.logger.Error("access denied!!!", zap.String("userID", userID), zap.String("workspaceID", workspaceID))
	}
	return ctx.JSON(http.StatusOK, resp)
}

// GetUsers godoc
//
//	@Summary		List Users
//	@Description	Retrieves a list of users who are members of the workspace.
//	@Security		BearerToken
//	@Tags			users
//	@Produce		json
//	@Param			request	body	api.GetUsersRequest	false	"Request Body"
//	@Success		200		{array}	api.GetUsersResponse
//	@Router			/auth/api/v1/users [get]
func (r *httpRoutes) GetUsers(ctx echo.Context) error {
	workspaceID := httpserver.GetWorkspaceID(ctx)
	var req api.GetUsersRequest
	if err := ctx.Bind(&req); err != nil {
		ctx.Logger().Errorf("bind the request: %v", err)
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}
	users, err := r.auth0Service.SearchUsers(workspaceID, req.Email, req.EmailVerified, req.RoleName)
	if err != nil {
		return err
	}
	var resp []api.GetUsersResponse
	for _, u := range users {

		resp = append(resp, api.GetUsersResponse{
			UserID:        u.UserId,
			UserName:      u.Name,
			Email:         u.Email,
			EmailVerified: u.EmailVerified,
			RoleName:      u.AppMetadata.WorkspaceAccess[workspaceID],
		})
	}
	return ctx.JSON(http.StatusOK, resp)
}

// GetUserDetails godoc
//
//	@Summary		Get User details
//	@Description	Returns user details by specified user id.
//	@Security		BearerToken
//	@Tags			users
//	@Produce		json
//	@Param			userId	path		string	true	"User ID"
//	@Success		200		{object}	api.GetUserResponse
//	@Router			/auth/api/v1/user/{userId} [get]
func (r *httpRoutes) GetUserDetails(ctx echo.Context) error {
	workspaceID := httpserver.GetWorkspaceID(ctx)
	userID := ctx.Param("user_id")
	userID, err := url.QueryUnescape(userID)
	if err != nil {
		return err
	}
	user, err := r.auth0Service.GetUser(userID)
	if err != nil {
		return err
	}
	hasARole := false
	for ws, _ := range user.AppMetadata.WorkspaceAccess {
		if ws == workspaceID {
			hasARole = true
			break
		}
	}
	if hasARole == false {
		return echo.NewHTTPError(http.StatusBadRequest, "The user is not in the specified workspace.")
	}
	status := api.InviteStatus_PENDING
	if user.EmailVerified {
		status = api.InviteStatus_ACCEPTED
	}
	resp := api.GetUserResponse{
		UserID:        user.UserId,
		UserName:      user.Name,
		Email:         user.Email,
		EmailVerified: user.EmailVerified,
		Status:        status,
		LastActivity:  user.LastLogin,
		CreatedAt:     user.CreatedAt,
		Blocked:       user.Blocked,
		RoleName:      user.AppMetadata.WorkspaceAccess[workspaceID],
	}

	return ctx.JSON(http.StatusOK, resp)

}

// GetMe godoc
//
//	@Summary		Get Me
//	@Description	Returns my user details
//	@Security		BearerToken
//	@Tags			users
//	@Produce		json
//	@Success		200	{object}	api.GetMeResponse
//	@Router			/auth/api/v1/me [get]
func (r *httpRoutes) GetMe(ctx echo.Context) error {
	userID := httpserver.GetUserID(ctx)

	user, err := r.auth0Service.GetUser(userID)
	if err != nil {
		return err
	}

	status := api.InviteStatus_PENDING
	if user.EmailVerified {
		status = api.InviteStatus_ACCEPTED
	}
	resp := api.GetMeResponse{
		UserID:          user.UserId,
		UserName:        user.Name,
		Email:           user.Email,
		EmailVerified:   user.EmailVerified,
		Status:          status,
		LastActivity:    user.LastLogin,
		CreatedAt:       user.CreatedAt,
		Blocked:         user.Blocked,
		Theme:           user.AppMetadata.Theme,
		ColorBlindMode:  user.AppMetadata.ColorBlindMode,
		WorkspaceAccess: user.AppMetadata.WorkspaceAccess,
		MemberSince:     user.AppMetadata.MemberSince,
		LastLogin:       user.AppMetadata.LastLogin,
	}

	return ctx.JSON(http.StatusOK, resp)

}

// Invite godoc
//
//	@Summary		Invite User
//	@Description	Sends an invitation to a user to join the workspace with a designated role.
//	@Security		BearerToken
//	@Tags			users
//	@Produce		json
//	@Param			request	body		api.InviteRequest	true	"Request Body"
//	@Success		200		{object}	nil
//	@Router			/auth/api/v1/user/invite [post]
func (r *httpRoutes) Invite(ctx echo.Context) error {
	var req api.InviteRequest
	if err := bindValidate(ctx, &req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}
	workspaceID := httpserver.GetWorkspaceID(ctx)

	hctx := httpclient.FromEchoContext(ctx)

	metadataService := metadataClient.NewMetadataServiceClient(fmt.Sprintf(metadataBaseUrl, workspaceID))
	cnf, err := metadataService.GetConfigMetadata(hctx, models.MetadataKeyAllowInvite)
	if err != nil {
		return err
	}

	allowInvite := cnf.GetValue().(bool)
	if !allowInvite {
		return echo.NewHTTPError(http.StatusNotAcceptable, "invite not allowed")
	}

	cnf, err = metadataService.GetConfigMetadata(hctx, models.MetadataKeyUserLimit)
	if err != nil {
		return err
	}
	maxUsers := cnf.GetValue().(int)

	users, err := r.auth0Service.SearchUsers(workspaceID, nil, nil, nil)
	if err != nil {
		return err
	}
	if len(users)+1 > maxUsers {
		return echo.NewHTTPError(http.StatusNotAcceptable, "cannot invite new user, max users reached")
	}

	cnf, err = metadataService.GetConfigMetadata(hctx, models.MetadataKeyAllowedEmailDomains)
	if err != nil {
		return err
	}

	if allowedEmailDomains, ok := cnf.GetValue().([]string); ok {
		passed := false
		if len(allowedEmailDomains) > 0 {
			for _, domain := range allowedEmailDomains {
				if strings.HasSuffix(req.Email, domain) {
					passed = true
				}
			}
		} else {
			passed = true
		}

		if !passed {
			return echo.NewHTTPError(http.StatusNotAcceptable, "email domain not allowed")
		}
	} else {
		fmt.Printf("failed to parse allowed domains, type: %s, value: %v", reflect.TypeOf(cnf.GetValue()).Name(), cnf.GetValue())
	}

	us, err := r.auth0Service.SearchByEmail(req.Email)
	if err != nil {
		return err
	}

	if len(us) > 0 {
		auth0User := us[0]
		if auth0User.AppMetadata.WorkspaceAccess == nil {
			auth0User.AppMetadata.WorkspaceAccess = map[string]api2.Role{}
		}
		auth0User.AppMetadata.WorkspaceAccess[workspaceID] = req.RoleName
		err = r.auth0Service.PatchUserAppMetadata(auth0User.UserId, auth0User.AppMetadata, nil)
		if err != nil {
			return err
		}

		//emailContent := inviteEmailTemplate
		//err = r.emailService.SendEmail(ctx.Request().Context(), req.Email, emailContent)
		//if err != nil {
		//	return err
		//}
	} else {
		_, err := r.auth0Service.CreateUser(req.Email, workspaceID, req.RoleName)
		if err != nil {
			return err
		}

		//emailContent := inviteEmailTemplate
		//err = r.emailService.SendEmail(ctx.Request().Context(), req.Email, emailContent)
		//if err != nil {
		//	return err
		//}
	}

	return ctx.NoContent(http.StatusOK)
}

// ChangeUserPreferences godoc
//
//	@Summary		Change User Preferences
//	@Description	Changes user color blind mode and color mode
//	@Security		BearerToken
//	@Tags			users
//	@Produce		json
//	@Param			request	body		api.ChangeUserPreferencesRequest	true	"Request Body"
//	@Success		200		{object}	nil
//	@Router			/auth/api/v1/user/preferences [put]
func (r *httpRoutes) ChangeUserPreferences(ctx echo.Context) error {
	var req api.ChangeUserPreferencesRequest
	if err := bindValidate(ctx, &req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	userId := httpserver.GetUserID(ctx)
	auth0User, err := r.auth0Service.GetUser(userId)
	if err != nil {
		return err
	}

	auth0User.AppMetadata.ColorBlindMode = &req.EnableColorBlindMode
	auth0User.AppMetadata.Theme = &req.Theme

	err = r.auth0Service.PatchUserAppMetadata(auth0User.UserId, auth0User.AppMetadata, nil)
	if err != nil {
		return err
	}

	return ctx.NoContent(http.StatusOK)
}

// CreateAPIKey godoc
//
//	@Summary		Create Workspace Key
//	@Description	Creates workspace key for the defined role with the defined name in the workspace.
//	@Security		BearerToken
//	@Tags			keys
//	@Produce		json
//	@Param			request	body		api.CreateAPIKeyRequest	true	"Request Body"
//	@Success		200		{object}	api.CreateAPIKeyResponse
//	@Failure		406		{object}	echo.HTTPError
//	@Router			/auth/api/v1/key/create [post]
func (r *httpRoutes) CreateAPIKey(ctx echo.Context) error {
	userID := httpserver.GetUserID(ctx)
	var req api.CreateAPIKeyRequest
	if err := bindValidate(ctx, &req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	usr, err := r.auth0Service.GetUser(userID)
	if err != nil {
		r.logger.Error("failed to get user", zap.Error(err))
		return err
	}

	if usr == nil {
		return errors.New("failed to find user in auth0")
	}

	u := userClaim{
		WorkspaceAccess: map[string]api2.Role{
			"opengovernance": api2.EditorRole,
		},
		GlobalAccess:   nil,
		Email:          usr.Email,
		ExternalUserID: usr.UserId,
	}

	if r.kaytuPrivateKey == nil {
		return echo.NewHTTPError(http.StatusBadRequest, "opengovernance api key is disabled")
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodRS256, &u).SignedString(r.kaytuPrivateKey)
	if err != nil {
		r.logger.Error("failed to create token", zap.Error(err))
		return err
	}

	masked := fmt.Sprintf("%s...%s", token[:3], token[len(token)-2:])

	hash := sha512.New()
	_, err = hash.Write([]byte(token))
	if err != nil {
		r.logger.Error("failed to hash token", zap.Error(err))
		return err
	}
	keyHash := hex.EncodeToString(hash.Sum(nil))
	r.logger.Info("hashed token")

	currentKeyCount, err := r.db.CountApiKeysForUser(userID)
	if err != nil {
		r.logger.Error("failed to get user API Keys count", zap.Error(err))
		return err
	}
	if currentKeyCount > 5 {
		return echo.NewHTTPError(http.StatusNotAcceptable, "maximum number of keys for user reached")
	}
	r.logger.Info("creating API Key")
	apikey := db.ApiKey{
		Name:          req.Name,
		Role:          api2.EditorRole,
		CreatorUserID: userID,
		WorkspaceID:   "opengovernance",
		Active:        true,
		Revoked:       false,
		MaskedKey:     masked,
		KeyHash:       keyHash,
	}

	r.logger.Info("adding API Key")
	err = r.db.AddApiKey(&apikey)
	if err != nil {
		r.logger.Error("failed to add API Key", zap.Error(err))
		return err
	}

	return ctx.JSON(http.StatusOK, api.CreateAPIKeyResponse{
		ID:        apikey.ID,
		Name:      apikey.Name,
		Active:    apikey.Active,
		CreatedAt: apikey.CreatedAt,
		RoleName:  apikey.Role,
		Token:     token,
	})
}

// DeleteAPIKey godoc
//
//	@Summary		Delete Workspace Key
//	@Description	Deletes the specified workspace key by ID.
//	@Security		BearerToken
//	@Tags			keys
//	@Produce		json
//	@Param			id	path		string	true	"Key ID"
//	@Success		200	{object}	nil
//	@Router			/auth/api/v1/key/{id}/delete [delete]
func (r *httpRoutes) DeleteAPIKey(ctx echo.Context) error {
	userId := httpserver.GetUserID(ctx)
	name := ctx.Param("name")

	keys, err := r.db.ListApiKeysForUser(userId)
	if err != nil {
		return err
	}

	var keyId uint
	exists := false
	for _, key := range keys {
		if key.Name == name {
			keyId = key.ID
			exists = true
		}
	}

	if !exists {
		return echo.NewHTTPError(http.StatusBadRequest, "key not found")
	}

	err = r.db.RevokeUserAPIKey(userId, keyId)
	if err != nil {
		return err
	}

	return ctx.NoContent(http.StatusOK)
}

// ListAPIKeys godoc
//
//	@Summary		Get Workspace Keys
//	@Description	Gets list of all keys in the workspace.
//	@Security		BearerToken
//	@Tags			keys
//	@Produce		json
//	@Success		200	{object}	[]api.WorkspaceApiKey
//	@Router			/auth/api/v1/keys [get]
func (r *httpRoutes) ListAPIKeys(ctx echo.Context) error {
	userID := httpserver.GetUserID(ctx)
	keys, err := r.db.ListApiKeysForUser(userID)
	if err != nil {
		return err
	}

	var resp []api.WorkspaceApiKey
	for _, key := range keys {
		resp = append(resp, api.WorkspaceApiKey{
			ID:            key.ID,
			CreatedAt:     key.CreatedAt,
			Name:          key.Name,
			RoleName:      key.Role,
			CreatorUserID: key.CreatorUserID,
			Active:        key.Active,
			MaskedKey:     key.MaskedKey,
		})
	}

	return ctx.JSON(http.StatusOK, resp)
}

// CreateUser godoc
//
//	@Summary		Create User
//	@Description	Creates User.
//	@Security		BearerToken
//	@Tags			keys
//	@Produce		json
//	@Param			request	body	api.CreateUserRequest	true	"Request Body"
//	@Success		200
//	@Router			/auth/api/v3/user/create [post]
func (r *httpRoutes) CreateUser(ctx echo.Context) error {
	var req api.CreateUserRequest
	if err := bindValidate(ctx, &req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	err := r.DoCreateUser(req)
	if err != nil {
		return err
	}

	return ctx.NoContent(http.StatusOK)
}

func (r *httpRoutes) DoCreateUser(req api.CreateUserRequest) error {
	if req.EmailAddress == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "email address is required")
	}

	user, err := r.db.GetUserByEmail(req.EmailAddress)
	if user != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "email already used")
	}

	count, err := r.db.GetUsersCount()
	if err != nil {
		r.logger.Error("failed to get users count", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to get users count")
	}
	adminAccount := false
	var firstUser *db.User
	if count == 1 {
		firstUser, err = r.db.GetFirstUser()
		if err != nil {
			r.logger.Error("failed to get first user", zap.Error(err))
			return echo.NewHTTPError(http.StatusInternalServerError, "failed to get first user")
		}
		if firstUser.StaticOwner {
			adminAccount = true
		}
	} else if count == 0 {
		adminAccount = true
	}
	if adminAccount && (req.Role == nil || *req.Role != api2.AdminRole) {
		return echo.NewHTTPError(http.StatusBadRequest, "You should define admin role")
	}

	if adminAccount && firstUser != nil {
		err = r.DoDeleteUser(firstUser.Email)
		if err != nil {
			r.logger.Error("failed to delete first user", zap.Error(err))
			return echo.NewHTTPError(http.StatusInternalServerError, "failed to delete first user")
		}
	}

	connector := ""
	userId := fmt.Sprintf("dex|%s", req.EmailAddress)
	if req.Password != nil {
		connector = "local"
		dexClient, err := newDexClient(dexGrpcAddress)
		if err != nil {
			r.logger.Error("failed to create dex client", zap.Error(err))
			return echo.NewHTTPError(http.StatusBadRequest, "failed to create dex client")
		}
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(*req.Password), bcrypt.DefaultCost)
		if err != nil {
			r.logger.Error("failed to hash token", zap.Error(err))
			return err
		}

		dexReq := &dexApi.CreatePasswordReq{
			Password: &dexApi.Password{
				UserId: userId,
				Email:  req.EmailAddress,
				Hash:   hashedPassword,
			},
		}

		resp, err := dexClient.CreatePassword(context.TODO(), dexReq)
		if err != nil {
			r.logger.Error("failed to create dex password", zap.Error(err))
			return echo.NewHTTPError(http.StatusBadRequest, "failed to create dex password")
		}
		if resp.AlreadyExists {
			dexReq := &dexApi.UpdatePasswordReq{
				Email:   req.EmailAddress,
				NewHash: hashedPassword,
			}

			_, err = dexClient.UpdatePassword(context.TODO(), dexReq)
			if err != nil {
				r.logger.Error("failed to update dex password", zap.Error(err))
				return echo.NewHTTPError(http.StatusBadRequest, "failed to create dex password")
			}
		}
	}

	wm, err := r.db.GetWorkspaceMapByName("main")
	if err != nil {
		r.logger.Error("failed to get main workspace", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to get workspace")
	}

	role := api2.ViewerRole
	if req.Role != nil {
		role = *req.Role
	}

	var appMetadata auth0.Metadata
	appMetadata.WorkspaceAccess = map[string]api2.Role{
		wm.ID: role,
	}

	timeNow := time.Now().Format("2006-01-02 15:00:00 MST")
	appMetadata.MemberSince = &timeNow
	appMetadataJson, err := json.Marshal(appMetadata)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "failed to marshal app metadata")
	}

	appMetadataJsonb := pgtype.JSONB{}
	err = appMetadataJsonb.Set(appMetadataJson)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "failed to marshal app metadata")
	}

	userMetadataJsonb := pgtype.JSONB{}
	err = userMetadataJsonb.Set([]byte(""))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "failed to marshal user metadata")
	}

	requirePasswordChange := true
	if adminAccount {
		requirePasswordChange = false
	}

	newUser := &db.User{
		UserUuid:              uuid.New(),
		Email:                 req.EmailAddress,
		Username:              req.EmailAddress,
		Name:                  req.EmailAddress,
		IdLifecycle:           db.UserLifecycleActive,
		Role:                  role,
		UserId:                userId,
		AppMetadata:           appMetadataJsonb,
		UserMetadata:          userMetadataJsonb,
		Connector:             connector,
		RequirePasswordChange: requirePasswordChange,
	}
	err = r.db.CreateUser(newUser)
	if err != nil {
		r.logger.Error("failed to create user", zap.Error(err))
		return echo.NewHTTPError(http.StatusBadRequest, "failed to create user")
	}
	return nil
}

// UpdateUser godoc
//
//	@Summary		Update User
//	@Description	Updates User.
//	@Security		BearerToken
//	@Tags			keys
//	@Produce		json
//	@Param			request	body	api.UpdateUserRequest	true	"Request Body"
//	@Success		200
//	@Router			/auth/api/v3/user/update [post]
func (r *httpRoutes) UpdateUser(ctx echo.Context) error {
	var req api.UpdateUserRequest
	if err := bindValidate(ctx, &req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	if req.EmailAddress == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "email address is required")
	}

	user, err := r.db.GetUserByEmail(req.EmailAddress)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to get user")
	}
	if user == nil {
		return echo.NewHTTPError(http.StatusBadRequest, "user not found")
	}

	if req.Password != nil && user.Connector == "local" {
		dexClient, err := newDexClient(dexGrpcAddress)
		if err != nil {
			r.logger.Error("failed to create dex client", zap.Error(err))
			return echo.NewHTTPError(http.StatusBadRequest, "failed to create dex client")
		}
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(*req.Password), bcrypt.DefaultCost)
		if err != nil {
			r.logger.Error("failed to hash token", zap.Error(err))
			return err
		}

		dexReq := &dexApi.UpdatePasswordReq{
			Email:   req.EmailAddress,
			NewHash: hashedPassword,
		}

		resp, err := dexClient.UpdatePassword(context.TODO(), dexReq)
		if err != nil {
			r.logger.Error("failed to update dex password", zap.Error(err))
			return echo.NewHTTPError(http.StatusBadRequest, "failed to create dex password")
		}
		if resp.NotFound {
			dexReq := &dexApi.CreatePasswordReq{
				Password: &dexApi.Password{
					UserId: fmt.Sprintf("dex|%s", req.EmailAddress),
					Email:  req.EmailAddress,
					Hash:   hashedPassword,
				},
			}

			_, err = dexClient.CreatePassword(context.TODO(), dexReq)
			if err != nil {
				r.logger.Error("failed to create dex password", zap.Error(err))
				return echo.NewHTTPError(http.StatusBadRequest, "failed to create dex password")
			}
		}

		err = r.db.UserPasswordUpdatedByEmail(req.EmailAddress)
		if err != nil {
			r.logger.Error("failed to update user", zap.Error(err))
			return echo.NewHTTPError(http.StatusBadRequest, "failed to update user")
		}
	}

	wm, err := r.db.GetWorkspaceMapByName("main")
	if err != nil {
		r.logger.Error("failed to get main workspace", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to get workspace")
	}

	if req.Role != nil {
		auth0User, err := r.auth0Service.GetUser(user.UserId)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "failed to get user")
		}
		auth0User.AppMetadata.WorkspaceAccess[wm.ID] = *req.Role

		if auth0User.AppMetadata.ConnectionIDs == nil {
			auth0User.AppMetadata.ConnectionIDs = map[string][]string{}
		}
		err = r.auth0Service.PatchUserAppMetadata(user.UserId, auth0User.AppMetadata, nil)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "failed to update user role")
		}
	}

	return ctx.NoContent(http.StatusOK)
}

// DeleteUser godoc
//
//	@Summary		Delete User
//	@Description	Delete User.
//	@Security		BearerToken
//	@Tags			keys
//	@Produce		json
//	@Param			email_address	path	string	true	"Request Body"
//	@Success		200
//	@Router			/auth/api/v3/user/{email_address}/delete [delete]
func (r *httpRoutes) DeleteUser(ctx echo.Context) error {
	emailAddress := ctx.Param("email_address")
	if emailAddress == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "email address is required")
	}

	err := r.DoDeleteUser(emailAddress)
	if err != nil {
		return err
	}

	return ctx.NoContent(http.StatusOK)
}

func (r *httpRoutes) DoDeleteUser(emailAddress string) error {
	dexClient, err := newDexClient(dexGrpcAddress)
	if err != nil {
		r.logger.Error("failed to create dex client", zap.Error(err))
		return echo.NewHTTPError(http.StatusBadRequest, "failed to create dex client")
	}

	dexReq := &dexApi.DeletePasswordReq{
		Email: emailAddress,
	}

	_, err = dexClient.DeletePassword(context.TODO(), dexReq)
	if err != nil {
		r.logger.Error("failed to create dex password", zap.Error(err))
		return echo.NewHTTPError(http.StatusBadRequest, "failed to create dex password")
	}

	user, err := r.db.GetUserByEmail(emailAddress)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "user does not exist")
	}
	if user == nil {
		return echo.NewHTTPError(http.StatusBadRequest, "user does not exist")
	}

	err = r.db.DeleteUserWithEmail(emailAddress)
	if err != nil {
		r.logger.Error("failed to delete user", zap.Error(err))
		return echo.NewHTTPError(http.StatusBadRequest, "failed to create user")
	}
	err = r.db.DeleteUser(user.UserId)
	if err != nil {
		r.logger.Error("failed to delete user", zap.Error(err))
		return echo.NewHTTPError(http.StatusBadRequest, "failed to create user")
	}
	return nil
}

func newDexClient(hostAndPort string) (dexApi.DexClient, error) {
	conn, err := grpc.NewClient(hostAndPort, grpc.WithInsecure())
	if err != nil {
		return nil, fmt.Errorf("dial: %v", err)
	}
	return dexApi.NewDexClient(conn), nil
}

// CheckUserPasswordChangeRequired godoc
//
//	@Summary		Delete User
//	@Description	Delete User.
//	@Security		BearerToken
//	@Tags			keys
//	@Produce		json
//	@Success		200
//	@Router			/auth/api/v3/user/password/check [get]
func (r *httpRoutes) CheckUserPasswordChangeRequired(ctx echo.Context) error {
	userId := httpserver.GetUserID(ctx)

	user, err := r.db.GetUser(userId)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to get user")
	}
	if user == nil {
		return echo.NewHTTPError(http.StatusBadRequest, "user not found")
	}

	if user.RequirePasswordChange {
		return ctx.String(http.StatusOK, "CHANGE_REQUIRED")
	} else {
		return ctx.String(http.StatusOK, "CHANGE_NOT_REQUIRED")
	}
}

// ResetUserPassword godoc
//
//	@Summary		Reset current user password
//	@Description	Reset current user password
//	@Security		BearerToken
//	@Tags			keys
//	@Produce		json
//	@Success		200
//	@Router			/auth/api/v3/user/password/reset [post]
func (r *httpRoutes) ResetUserPassword(ctx echo.Context) error {
	userId := httpserver.GetUserID(ctx)

	user, err := r.db.GetUser(userId)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to get user")
	}
	if user == nil {
		return echo.NewHTTPError(http.StatusBadRequest, "user not found")
	}

	var req api.ResetUserPasswordRequest
	if err := bindValidate(ctx, &req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	if user.Connector != "local" {
		return echo.NewHTTPError(http.StatusBadRequest, "user connector should be local")
	}

	dexClient, err := newDexClient(dexGrpcAddress)
	if err != nil {
		r.logger.Error("failed to create dex client", zap.Error(err))
		return echo.NewHTTPError(http.StatusBadRequest, "failed to create dex client")
	}

	dexReq := &dexApi.VerifyPasswordReq{
		Email:    user.Email,
		Password: req.CurrentPassword,
	}

	resp, err := dexClient.VerifyPassword(context.TODO(), dexReq)
	if err != nil {
		r.logger.Error("failed to validate dex password", zap.Error(err))
		return echo.NewHTTPError(http.StatusBadRequest, "failed to validate dex password")
	}
	if resp.NotFound {
		return echo.NewHTTPError(http.StatusNotFound, "user not found")
	}
	if !resp.Verified {
		return echo.NewHTTPError(http.StatusUnauthorized, "current password is not correct")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		r.logger.Error("failed to hash token", zap.Error(err))
		return err
	}

	passwordUpdateReq := &dexApi.UpdatePasswordReq{
		Email:   user.Email,
		NewHash: hashedPassword,
	}

	passwordUpdateResp, err := dexClient.UpdatePassword(context.TODO(), passwordUpdateReq)
	if err != nil {
		r.logger.Error("failed to update dex password", zap.Error(err))
		return echo.NewHTTPError(http.StatusBadRequest, "failed to update dex password")
	}
	if passwordUpdateResp.NotFound {
		dexReq := &dexApi.CreatePasswordReq{
			Password: &dexApi.Password{
				UserId: fmt.Sprintf("dex|%s", user.Email),
				Email:  user.Email,
				Hash:   hashedPassword,
			},
		}

		_, err = dexClient.CreatePassword(context.TODO(), dexReq)
		if err != nil {
			r.logger.Error("failed to create dex password", zap.Error(err))
			return echo.NewHTTPError(http.StatusBadRequest, "failed to create dex password")
		}
	}

	err = r.db.UserPasswordUpdatedByEmail(user.Email)
	if err != nil {
		r.logger.Error("failed to update user", zap.Error(err))
		return echo.NewHTTPError(http.StatusBadRequest, "failed to update user")
	}

	return ctx.NoContent(http.StatusOK)
}
