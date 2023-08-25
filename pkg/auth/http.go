package auth

import (
	"context"
	"crypto/rsa"
	"crypto/sha512"
	_ "embed"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"

	"github.com/kaytu-io/kaytu-engine/pkg/internal/httpclient"
	"github.com/kaytu-io/kaytu-engine/pkg/internal/httpserver"
	metadataClient "github.com/kaytu-io/kaytu-engine/pkg/metadata/client"
	"github.com/kaytu-io/kaytu-util/pkg/email"

	"github.com/kaytu-io/kaytu-engine/pkg/metadata/models"

	"github.com/kaytu-io/kaytu-engine/pkg/auth/db"

	"github.com/golang-jwt/jwt"
	"github.com/kaytu-io/kaytu-engine/pkg/auth/auth0"

	"github.com/kaytu-io/kaytu-engine/pkg/workspace/client"

	"github.com/kaytu-io/kaytu-engine/pkg/auth/api"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

var (
	//go:embed email/invite.html
	inviteEmailTemplate string
)

type httpRoutes struct {
	logger          *zap.Logger
	emailService    email.Service
	workspaceClient client.WorkspaceServiceClient
	auth0Service    *auth0.Service
	metadataBaseUrl string
	kaytuPrivateKey *rsa.PrivateKey
	db              db.Database
}

func (r *httpRoutes) Register(e *echo.Echo) {
	v1 := e.Group("/api/v1")

	v1.PUT("/user/role/binding", httpserver.AuthorizeHandler(r.PutRoleBinding, api.AdminRole))
	v1.DELETE("/user/role/binding", httpserver.AuthorizeHandler(r.DeleteRoleBinding, api.AdminRole))
	v1.GET("/user/role/bindings", httpserver.AuthorizeHandler(r.GetRoleBindings, api.EditorRole))
	v1.GET("/workspace/role/bindings", httpserver.AuthorizeHandler(r.GetWorkspaceRoleBindings, api.AdminRole))
	v1.GET("/users", httpserver.AuthorizeHandler(r.GetUsers, api.EditorRole))
	v1.GET("/user/:user_id", httpserver.AuthorizeHandler(r.GetUserDetails, api.EditorRole))
	v1.POST("/user/invite", httpserver.AuthorizeHandler(r.Invite, api.AdminRole))

	v1.POST("/key/create", httpserver.AuthorizeHandler(r.CreateAPIKey, api.AdminRole))
	v1.GET("/keys", httpserver.AuthorizeHandler(r.ListAPIKeys, api.EditorRole))
	v1.DELETE("/key/:id/delete", httpserver.AuthorizeHandler(r.DeleteAPIKey, api.AdminRole))
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
func (r httpRoutes) PutRoleBinding(ctx echo.Context) error {
	var req api.PutRoleBindingRequest
	if err := bindValidate(ctx, &req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	workspaceID := httpserver.GetWorkspaceID(ctx)

	if httpserver.GetUserID(ctx) == req.UserID &&
		req.RoleName != api.AdminRole {
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
	err = r.auth0Service.PatchUserAppMetadata(req.UserID, auth0User.AppMetadata)
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
func (r httpRoutes) DeleteRoleBinding(ctx echo.Context) error {
	userId := ctx.QueryParam("userId")
	// The WorkspaceManager service will call this API to set the AdminRole
	// for the admin user on behalf of him. Allow for the Admin to only set its
	// role to admin for that user case
	if httpserver.GetUserID(ctx) == userId {
		return echo.NewHTTPError(http.StatusBadRequest, "admin user permission can't be modified by self")
	}

	workspaceID := httpserver.GetWorkspaceID(ctx)
	auth0User, err := r.auth0Service.GetUser(userId)
	if err != nil {
		return err
	}

	delete(auth0User.AppMetadata.WorkspaceAccess, workspaceID)
	err = r.auth0Service.PatchUserAppMetadata(userId, auth0User.AppMetadata)
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
	} else {
		r.logger.Warn("user not found in auth0", zap.String("externalID", userID))
	}
	return ctx.JSON(http.StatusOK, resp)
}

// GetWorkspaceRoleBindings godoc
//
//	@Summary		Workspace user roleBindings.
//	@Description	Get all the RoleBindings of the workspace. RoleBinding defines the roles and actions a user can perform. There are currently three roles (admin, editor, viewer). The workspace path is based on the DNS such as (workspace1.app.kaytu.io)
//	@Security		BearerToken
//	@Tags			users
//	@Produce		json
//	@Success		200	{object}	api.GetWorkspaceRoleBindingResponse
//	@Router			/auth/api/v1/workspace/role/bindings [get]
func (r *httpRoutes) GetWorkspaceRoleBindings(ctx echo.Context) error {
	workspaceID := httpserver.GetWorkspaceID(ctx)
	users, err := r.auth0Service.SearchUsersByWorkspace(workspaceID)
	if err != nil {
		return err
	}

	var resp api.GetWorkspaceRoleBindingResponse
	for _, u := range users {
		status := api.InviteStatus_PENDING
		if u.EmailVerified {
			status = api.InviteStatus_ACCEPTED
		}

		resp = append(resp, api.WorkspaceRoleBinding{
			UserID:       u.UserId,
			UserName:     u.Name,
			Email:        u.Email,
			RoleName:     u.AppMetadata.WorkspaceAccess[workspaceID],
			Status:       status,
			LastActivity: u.LastLogin,
			CreatedAt:    u.CreatedAt,
		})
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
			auth0User.AppMetadata.WorkspaceAccess = map[string]api.Role{}
		}
		auth0User.AppMetadata.WorkspaceAccess[workspaceID] = req.RoleName
		err = r.auth0Service.PatchUserAppMetadata(auth0User.UserId, auth0User.AppMetadata)
		if err != nil {
			return err
		}

		emailContent := inviteEmailTemplate
		err = r.emailService.SendEmail(context.Background(), req.Email, emailContent)
		if err != nil {
			return err
		}
	} else {
		user, err := r.auth0Service.CreateUser(req.Email, workspaceID, req.RoleName)
		if err != nil {
			return err
		}

		resp, err := r.auth0Service.CreatePasswordChangeTicket(user.UserId)
		if err != nil {
			return err
		}

		emailContent := inviteEmailTemplate
		emailContent = strings.ReplaceAll(emailContent, "{{ url }}", resp.Ticket)
		err = r.emailService.SendEmail(context.Background(), req.Email, emailContent)
		if err != nil {
			return err
		}
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
	workspaceID := httpserver.GetWorkspaceID(ctx)
	hctx := httpclient.FromEchoContext(ctx)
	metadataService := metadataClient.NewMetadataServiceClient(fmt.Sprintf(metadataBaseUrl, workspaceID))

	cnf, err := metadataService.GetConfigMetadata(hctx, models.MetadataKeyWorkspaceKeySupport)
	if err != nil {
		return err
	}
	keySupport := cnf.GetValue().(bool)
	if !keySupport {
		return echo.NewHTTPError(http.StatusNotAcceptable, "keys are not supported in this workspace")
	}

	var req api.CreateAPIKeyRequest
	if err := bindValidate(ctx, &req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	usr, err := r.auth0Service.GetUser(userID)
	if err != nil {
		return err
	}

	if usr == nil {
		return errors.New("failed to find user in auth0")
	}

	u := userClaim{
		WorkspaceAccess: map[string]api.Role{
			workspaceID: req.RoleName,
		},
		GlobalAccess:   nil,
		Email:          usr.Email,
		ExternalUserID: usr.UserId,
	}

	token, err := jwt.NewWithClaims(jwt.SigningMethodRS256, &u).SignedString(r.kaytuPrivateKey)
	if err != nil {
		return err
	}

	masked := fmt.Sprintf("%s...%s", token[:3], token[len(token)-2:])

	hash := sha512.New()
	_, err = hash.Write([]byte(token))
	if err != nil {
		return err
	}
	keyHash := hex.EncodeToString(hash.Sum(nil))
	// trace :
	//span := jaegertracing.CreateChildSpan(ctx, "CountApiKeys")
	//span.SetBaggageItem("auth", "CreateAPIKey")

	currentKeyCount, err := r.db.CountApiKeys(workspaceID)
	if err != nil {
		return err
	}
	//span.Finish()

	cnf, err = metadataService.GetConfigMetadata(hctx, models.MetadataKeyWorkspaceMaxKeys)
	if err != nil {
		return err
	}
	maxKeys := cnf.GetValue().(int)
	if currentKeyCount+1 > int64(maxKeys) {
		return echo.NewHTTPError(http.StatusNotAcceptable, "maximum number of keys in workspace reached")
	}

	apikey := db.ApiKey{
		Name:          req.Name,
		Role:          req.RoleName,
		CreatorUserID: userID,
		WorkspaceID:   workspaceID,
		Active:        true,
		Revoked:       false,
		MaskedKey:     masked,
		KeyHash:       keyHash,
	}
	// trace :
	//spanAAK := jaegertracing.CreateChildSpan(ctx, "AddApiKey")
	//spanAAK.SetBaggageItem("auth", "CreateAPIKey")
	err = r.db.AddApiKey(&apikey)
	if err != nil {
		return err
	}
	//spanAAK.Finish()

	// trace :
	//spanGAK := jaegertracing.CreateChildSpan(ctx, "GetApiKey")
	//spanGAK.SetBaggageItem("auth", "CreateAPIKey")

	key, err := r.db.GetApiKey(workspaceID, uint(apikey.ID))
	if err != nil {
		return err
	}
	//spanGAK.Finish()

	return ctx.JSON(http.StatusOK, api.CreateAPIKeyResponse{
		ID:        apikey.ID,
		Name:      key.Name,
		Active:    key.Active,
		CreatedAt: key.CreatedAt,
		RoleName:  key.Role,
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
	workspaceID := httpserver.GetWorkspaceID(ctx)
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		return err
	}
	// trace :
	//span := jaegertracing.CreateChildSpan(ctx, "RevokeAPIKey")
	//span.SetBaggageItem("auth", "DeleteAPIKey")

	err = r.db.RevokeAPIKey(workspaceID, uint(id))
	if err != nil {
		return err
	}
	//span.Finish()

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
	workspaceID := httpserver.GetWorkspaceID(ctx)
	// trace :
	//span := jaegertracing.CreateChildSpan(ctx, "ListApiKeys")
	//span.SetBaggageItem("auth", "ListAPIKeys")

	keys, err := r.db.ListApiKeys(workspaceID)
	if err != nil {
		return err
	}
	//span.Finish()

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
