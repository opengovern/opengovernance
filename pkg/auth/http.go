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
	"strconv"
	"strings"
	"time"

	"gitlab.com/keibiengine/keibi-engine/pkg/auth/db"

	"github.com/golang-jwt/jwt"
	"gitlab.com/keibiengine/keibi-engine/pkg/internal/httpclient"

	"gitlab.com/keibiengine/keibi-engine/pkg/auth/auth0"

	"gitlab.com/keibiengine/keibi-engine/pkg/workspace/client"

	"gitlab.com/keibiengine/keibi-engine/pkg/internal/email"

	"github.com/labstack/echo/v4"
	"gitlab.com/keibiengine/keibi-engine/pkg/auth/api"
	"gitlab.com/keibiengine/keibi-engine/pkg/internal/httpserver"
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
	keibiPrivateKey *rsa.PrivateKey
	db              db.Database
}

func (r *httpRoutes) Register(e *echo.Echo) {
	v1 := e.Group("/api/v1")

	v1.PUT("/user/role/binding", httpserver.AuthorizeHandler(r.PutRoleBinding, api.AdminRole))
	v1.DELETE("/user/role/binding", httpserver.AuthorizeHandler(r.DeleteRoleBinding, api.AdminRole))
	v1.GET("/user/role/bindings", httpserver.AuthorizeHandler(r.GetRoleBindings, api.EditorRole))
	v1.GET("/user/:user_id/workspace/membership", httpserver.AuthorizeHandler(r.GetWorkspaceMembership, api.AdminRole))
	v1.GET("/workspace/role/bindings", httpserver.AuthorizeHandler(r.GetWorkspaceRoleBindings, api.AdminRole))
	v1.GET("/user/:user_id", httpserver.AuthorizeHandler(r.GetUserDetails, api.AdminRole))
	v1.POST("/invite", httpserver.AuthorizeHandler(r.Invite, api.AdminRole))
	v1.DELETE("/invite/:user_id", httpserver.AuthorizeHandler(r.DeleteInvitation, api.AdminRole))

	v1.POST("/apikey/create", httpserver.AuthorizeHandler(r.CreateAPIKey, api.AdminRole))
	v1.GET("/apikey", httpserver.AuthorizeHandler(r.ListAPIKeys, api.AdminRole))
	v1.POST("/apikey/:id/suspend", httpserver.AuthorizeHandler(r.SuspendAPIKey, api.AdminRole))
	v1.POST("/apikey/:id/activate", httpserver.AuthorizeHandler(r.ActivateAPIKey, api.AdminRole))
	v1.DELETE("/apikey/:id/delete", httpserver.AuthorizeHandler(r.DeleteAPIKey, api.AdminRole))
	v1.GET("/apikey/:id", httpserver.AuthorizeHandler(r.GetAPIKey, api.AdminRole))

	v1.GET("/role/users", httpserver.AuthorizeHandler(r.GetRoleUsers, api.AdminRole))
	v1.GET("/role/apikeys", httpserver.AuthorizeHandler(r.GetRoleKeys, api.AdminRole))
	v1.POST("/apikey/role", httpserver.AuthorizeHandler(r.UpdateKeyRole, api.AdminRole))
	v1.DELETE("apikey/:id/role", httpserver.AuthorizeHandler(r.DeleteKeyRole, api.AdminRole))
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
//	@Summary		Update RoleBinding for a user.
//	@Description	RoleBinding defines the roles and actions a user can perform.
//	@Description	There are currently three roles (ADMIN, EDITOR, VIEWER).
//	@Description	User must exist before you can update its RoleBinding.
//	@Description	If you want to add a role binding for a user given the email address, call invite first to get a user id. Then call this endpoint.
//	@Tags			auth
//	@Produce		json
//	@Param			request		body		api.PutRoleBindingRequest	true	"Request Body"
//	@Param			workspaceId	query		string						true	"workspaceId"
//	@Success		200			{object}	nil
//	@Router			/auth/api/v1/user/role/binding [put]
func (r httpRoutes) PutRoleBinding(ctx echo.Context) error {
	var req api.PutRoleBindingRequest
	if err := bindValidate(ctx, &req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	// The WorkspaceManager service will call this API to set the AdminRole
	// for the admin user on behalf of him. Allow for the Admin to only set its
	// role to admin for that user case
	if httpserver.GetUserID(ctx) == req.UserID &&
		req.Role != api.AdminRole {
		return echo.NewHTTPError(http.StatusBadRequest, "admin user permission can't be modified by self")
	}

	workspaceID := httpserver.GetWorkspaceID(ctx)
	auth0User, err := r.auth0Service.GetUser(req.UserID)
	if err != nil {
		return err
	}

	auth0User.AppMetadata.WorkspaceAccess[workspaceID] = req.Role
	err = r.auth0Service.PatchUserAppMetadata(req.UserID, auth0User.AppMetadata)
	if err != nil {
		return err
	}
	return ctx.NoContent(http.StatusOK)
}

// DeleteRoleBinding godoc
//
//	@Summary	Delete RoleBinding for the defined user in the defined workspace.
//	@Tags		auth
//	@Produce	json
//	@Param		userId		query		string	true	"userId"
//	@Param		workspaceId	query		string	true	"workspaceId"
//	@Success	200			{object}	nil
//	@Router		/auth/api/v1/user/role/binding [delete]
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
//	@Summary		Get RoleBindings for the calling user
//	@Description	RoleBinding defines the roles and actions a user can perform. There are currently three roles (ADMIN, EDITOR, VIEWER).
//	@Tags			auth
//	@Produce		json
//	@Param			userId	query		string	true	"userId"
//	@Success		200		{object}	api.GetRoleBindingsResponse
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
				Role:        role,
			})
		}
		resp.GlobalRoles = usr.AppMetadata.GlobalAccess
	} else {
		r.logger.Warn("user not found in auth0", zap.String("externalID", userID))
	}
	return ctx.JSON(http.StatusOK, resp)
}

// GetWorkspaceMembership godoc
//
//	@Summary		List of workspaces which the user is member of
//	@Description	Returns a list of workspaces and the user role in it for the specified user
//	@Tags			auth
//	@Produce		json
//	@Param			userId	path		string	true	"userId"
//	@Success		200		{object}	api.GetRoleBindingsResponse
//	@Router			/auth/api/v1/user/{user_id}/workspace/membership [get]
func (r *httpRoutes) GetWorkspaceMembership(ctx echo.Context) error {
	hctx := httpclient.FromEchoContext(ctx)
	userID := ctx.Param("user_id")

	var resp []api.Membership
	usr, err := r.auth0Service.GetUser(userID)
	if err != nil {
		r.logger.Warn("failed to get user from auth0 due to", zap.Error(err))
		return err
	}

	if usr != nil {
		for wsID, role := range usr.AppMetadata.WorkspaceAccess {
			ws, err := r.workspaceClient.GetByID(hctx, wsID)
			if err != nil {
				r.logger.Warn("failed to get workspace due to", zap.Error(err))
				return err
			}

			resp = append(resp, api.Membership{
				WorkspaceID:   wsID,
				WorkspaceName: ws.Name,
				Role:          role,
				AssignedAt:    time.Time{}, //TODO- add assigned at
				LastActivity:  time.Time{}, //TODO- add assigned at
			})
		}
	} else {
		r.logger.Warn("user not found in auth0", zap.String("externalID", userID))
	}
	return ctx.JSON(http.StatusOK, resp)
}

// GetWorkspaceRoleBindings godoc
//
//	@Summary		Get all the user RoleBindings for the given workspace.
//	@Description	RoleBinding defines the roles and actions a user can perform. There are currently three roles (ADMIN, EDITOR, VIEWER). The workspace path is based on the DNS such as (workspace1.app.keibi.io)
//	@Tags			auth
//	@Produce		json
//	@Success		200	{object}	api.GetWorkspaceRoleBindingResponse
//	@Router			/auth/api/v1/workspace/role/bindings [get]
func (r *httpRoutes) GetWorkspaceRoleBindings(ctx echo.Context) error {
	workspaceID := httpserver.GetWorkspaceID(ctx)
	users, err := r.auth0Service.SearchUsersByWorkspace(workspaceID)
	if err != nil {
		return err
	}
	tenant, err := r.auth0Service.GetClientTenant()
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
			UserID:        u.UserId,
			UserName:      u.Name,
			TenantId:      tenant,
			Email:         u.Email,
			EmailVerified: u.EmailVerified,
			Role:          u.AppMetadata.WorkspaceAccess[workspaceID],
			Status:        status,
			LastActivity:  u.LastLogin,
			CreatedAt:     u.CreatedAt,
			Blocked:       u.Blocked,
		})
	}
	return ctx.JSON(http.StatusOK, resp)
}

// GetUserDetails godoc
//
//	@Summary		Get user details by user id
//	@Description	Get user details by user id
//	@Tags			auth
//	@Produce		json
//	@Param			userId	path		string	true	"userId"
//	@Success		200		{object}	api.WorkspaceRoleBinding
//	@Router			/auth/api/v1/user/{user_id} [get]
func (r *httpRoutes) GetUserDetails(ctx echo.Context) error {
	userID := ctx.Param("user_id")
	user, err := r.auth0Service.GetUser(userID)
	if err != nil {
		return err
	}
	tenant, err := r.auth0Service.GetClientTenant()
	if err != nil {
		return err
	}
	status := api.InviteStatus_PENDING
	if user.EmailVerified {
		status = api.InviteStatus_ACCEPTED
	}
	resp := api.WorkspaceRoleBinding{
		UserID:        user.UserId,
		UserName:      user.Name,
		TenantId:      tenant,
		Email:         user.Email,
		EmailVerified: user.EmailVerified,
		Status:        status,
		LastActivity:  user.LastLogin,
		CreatedAt:     user.CreatedAt,
		Blocked:       user.Blocked,
	}

	return ctx.JSON(http.StatusOK, resp)

}

// Invite godoc
//
//	@Summary		Invites a user to a workspace with defined role.
//	@Description	Invites a user to a workspace with defined role
//	@Description	by sending an email to the specified email address.
//	@Description	The user will be found by the email address.
//	@Tags			auth
//	@Produce		json
//	@Param			request	body		api.InviteRequest	true	"Request Body"
//	@Success		200		{object}	nil
//	@Router			/auth/api/v1/invite [post]
func (r *httpRoutes) Invite(ctx echo.Context) error {
	var req api.InviteRequest
	if err := bindValidate(ctx, &req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}
	workspaceID := httpserver.GetWorkspaceID(ctx)

	us, err := r.auth0Service.SearchByEmail(req.Email)
	if err != nil {
		return err
	}

	if len(us) > 0 {
		auth0User := us[0]
		if auth0User.AppMetadata.WorkspaceAccess == nil {
			auth0User.AppMetadata.WorkspaceAccess = map[string]api.Role{}
		}
		auth0User.AppMetadata.WorkspaceAccess[workspaceID] = req.Role
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
		user, err := r.auth0Service.CreateUser(req.Email, workspaceID, req.Role)
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

// DeleteInvitation godoc
//
//	@Summary	Deletes an Invitation
//	@Tags		auth
//	@Produce	json
//	@Success	200	{object}	nil
//	@Router		/auth/api/v1/invite/{user_id} [delete]
func (r *httpRoutes) DeleteInvitation(ctx echo.Context) error {
	userID := ctx.Param("user_id")
	err := r.auth0Service.DeleteUser(userID)
	if err != nil {
		return err
	}
	return ctx.NoContent(http.StatusOK)
}

// CreateAPIKey godoc
//
//	@Summary	Creates an API Key
//	@Tags		auth
//	@Produce	json
//	@Param		request	body	api.CreateAPIKeyRequest	true	"Request Body"
//	@Router		/auth/api/v1/apikey/create [post]
func (r *httpRoutes) CreateAPIKey(ctx echo.Context) error {
	userID := httpserver.GetUserID(ctx)
	workspaceID := httpserver.GetWorkspaceID(ctx)

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
			workspaceID: req.Role,
		},
		GlobalAccess:   nil,
		Email:          usr.Email,
		ExternalUserID: usr.UserId,
	}

	token, err := jwt.NewWithClaims(jwt.SigningMethodRS256, &u).SignedString(r.keibiPrivateKey)
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

	apikey := db.ApiKey{
		Name:          req.Name,
		Role:          req.Role,
		CreatorUserID: userID,
		WorkspaceID:   workspaceID,
		Active:        true,
		Revoked:       false,
		MaskedKey:     masked,
		KeyHash:       keyHash,
	}
	err = r.db.AddApiKey(&apikey)
	if err != nil {
		return err
	}

	return ctx.JSON(http.StatusOK, api.CreateAPIKeyResponse{
		ID:    apikey.ID,
		Token: token,
	})
}

// DeleteAPIKey godoc
//
//	@Summary	Deletes an API Key
//	@Tags		auth
//	@Produce	json
//	@Param		id	path	string	true	"ID"
//	@Router		/auth/api/v1/apikey/{id}/delete [delete]
func (r *httpRoutes) DeleteAPIKey(ctx echo.Context) error {
	workspaceID := httpserver.GetWorkspaceID(ctx)
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		return err
	}

	err = r.db.RevokeAPIKey(workspaceID, uint(id))
	if err != nil {
		return err
	}

	return ctx.NoContent(http.StatusOK)
}

// ListAPIKeys godoc
//
//	@Summary	Lists all API Keys
//	@Tags		auth
//	@Produce	json
//	@Success	200	{object}	[]api.WorkspaceApiKey
//	@Router		/auth/api/v1/apikey [get]
func (r *httpRoutes) ListAPIKeys(ctx echo.Context) error {
	workspaceID := httpserver.GetWorkspaceID(ctx)
	keys, err := r.db.ListApiKeys(workspaceID)
	if err != nil {
		return err
	}

	var resp []api.WorkspaceApiKey
	for _, key := range keys {
		resp = append(resp, api.WorkspaceApiKey{
			ID:            key.ID,
			CreatedAt:     key.CreatedAt,
			UpdatedAt:     key.UpdatedAt,
			Name:          key.Name,
			Role:          key.Role,
			CreatorUserID: key.CreatorUserID,
			Active:        key.Active,
			MaskedKey:     key.MaskedKey,
		})
	}

	return ctx.JSON(http.StatusOK, resp)
}

// GetAPIKey godoc
//
//	@Summary	Fetches an API Key
//	@Tags		auth
//	@Produce	json
//	@Param		id	path		string	true	"ID"
//	@Success	200	{object}	api.WorkspaceApiKey
//	@Router		/auth/api/v1/apikey/{id} [get]
func (r *httpRoutes) GetAPIKey(ctx echo.Context) error {
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		return err
	}

	workspaceID := httpserver.GetWorkspaceID(ctx)
	key, err := r.db.GetApiKeys(workspaceID, uint(id))
	if err != nil {
		return err
	}
	if key.ID == 0 {
		return echo.NewHTTPError(http.StatusNotFound, "api key not found")
	}

	resp := api.WorkspaceApiKey{
		ID:            key.ID,
		CreatedAt:     key.CreatedAt,
		UpdatedAt:     key.UpdatedAt,
		Name:          key.Name,
		Role:          key.Role,
		CreatorUserID: key.CreatorUserID,
		Active:        key.Active,
		MaskedKey:     key.MaskedKey,
	}

	return ctx.JSON(http.StatusOK, resp)
}

// SuspendAPIKey godoc
//
//	@Summary	Suspend an API Key
//	@Tags		auth
//	@Produce	json
//	@Param		id	path	string	true	"ID"
//	@Router		/auth/api/v1/apikey/{id}/suspend [post]
func (r *httpRoutes) SuspendAPIKey(ctx echo.Context) error {
	workspaceID := httpserver.GetWorkspaceID(ctx)
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		return err
	}

	err = r.db.UpdateActiveAPIKey(workspaceID, uint(id), false)
	if err != nil {
		return err
	}

	return ctx.NoContent(http.StatusOK)
}

// ActivateAPIKey godoc
//
//	@Summary	Suspend an API Key
//	@Tags		auth
//	@Produce	json
//	@Param		id	path	string	true	"ID"
//	@Router		/auth/api/v1/apikey/{id}/activate [post]
func (r *httpRoutes) ActivateAPIKey(ctx echo.Context) error {
	workspaceID := httpserver.GetWorkspaceID(ctx)
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		return err
	}

	err = r.db.UpdateActiveAPIKey(workspaceID, uint(id), true)
	if err != nil {
		return err
	}

	return ctx.NoContent(http.StatusOK)
}

// GetRoleUsers godoc
//
//	@Summary	Get the list of users having the specefic role
//	@Tags		auth
//	@Produce	json
//	@Param		request	body		api.Role	true	"Request Body"
//	@Success	200		{object}	api.GetRoleUsersResponse
//	@Router		/auth/api/v1/role/users [get]
func (r *httpRoutes) GetRoleUsers(ctx echo.Context) error {
	var role api.Role
	if err := bindValidate(ctx, &role); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}
	users, err := r.auth0Service.SearchUsersByRole(role)
	if err != nil {
		return err
	}
	tenant, err := r.auth0Service.GetClientTenant()
	if err != nil {
		return err
	}
	var resp api.GetRoleUsersResponse
	for _, u := range users {
		status := api.InviteStatus_PENDING
		if u.EmailVerified {
			status = api.InviteStatus_ACCEPTED
		}
		var workspaces []string
		for ws, r := range u.AppMetadata.WorkspaceAccess {
			if r == role {
				workspaces = append(workspaces, ws)
			}
		}

		resp = append(resp, api.RoleUser{
			UserID:        u.UserId,
			UserName:      u.Name,
			TenantId:      tenant,
			Email:         u.Email,
			EmailVerified: u.EmailVerified,
			Role:          role,
			Workspaces:    workspaces,
			Status:        status,
			LastActivity:  u.LastLogin,
			CreatedAt:     u.CreatedAt,
			Blocked:       u.Blocked,
		})
	}
	return ctx.JSON(http.StatusOK, resp)
}

// GetRoleKeys godoc
//
//	@Summary	Lists all API Keys
//	@Tags		auth
//	@Produce	json
//	@Param		request	body		api.Role	true	"Request Body"
//	@Success	200		{object}	[]api.WorkspaceApiKey
//	@Router		/auth/api/v1/role/apikeys [get]
func (r *httpRoutes) GetRoleKeys(ctx echo.Context) error {
	var role api.Role
	if err := bindValidate(ctx, &role); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}
	keys, err := r.db.GetAPIKeysByRole(role)
	if err != nil {
		return err
	}

	var resp []api.WorkspaceApiKey
	for _, key := range keys {
		resp = append(resp, api.WorkspaceApiKey{
			ID:            key.ID,
			CreatedAt:     key.CreatedAt,
			UpdatedAt:     key.UpdatedAt,
			Name:          key.Name,
			Role:          key.Role,
			CreatorUserID: key.CreatorUserID,
			Active:        key.Active,
			MaskedKey:     key.MaskedKey,
		})
	}

	return ctx.JSON(http.StatusOK, resp)
}

// UpdateKeyRole godoc
//
//	@Summary	Fetches an API Key
//	@Tags		auth
//	@Produce	json
//	@Param		request	body		api.UpdateKeyRoleRequest	true	"Request Body"
//	@Success	200		{object}	api.WorkspaceApiKey
//	@Router		/auth/api/v1/apikey/role [post]
func (r *httpRoutes) UpdateKeyRole(ctx echo.Context) error {
	workspaceID := httpserver.GetWorkspaceID(ctx)
	var req api.UpdateKeyRoleRequest
	if err := bindValidate(ctx, &req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	err := r.db.UpdateRoleAPIKey(workspaceID, uint(req.ID), req.Role)
	if err != nil {
		return err
	}

	return ctx.NoContent(http.StatusOK)
}

// DeleteKeyRole godoc
//
//	@Summary	Fetches an API Key
//	@Tags		auth
//	@Produce	json
//	@Success	200	{object}	api.WorkspaceApiKey
//	@Router		/auth/api/v1/apikey/{id}/role [delete]
func (r *httpRoutes) DeleteKeyRole(ctx echo.Context) error {
	workspaceID := httpserver.GetWorkspaceID(ctx)
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		return err
	}

	err = r.db.UpdateRoleAPIKey(workspaceID, uint(id), api.Role(""))
	if err != nil {
		return err
	}

	return ctx.NoContent(http.StatusOK)
}
