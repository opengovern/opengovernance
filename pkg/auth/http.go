package auth

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/labstack/echo/v4"
	"gitlab.com/keibiengine/keibi-engine/pkg/auth/api"
	"gitlab.com/keibiengine/keibi-engine/pkg/auth/extauth"
	"gitlab.com/keibiengine/keibi-engine/pkg/internal/email"
	"gitlab.com/keibiengine/keibi-engine/pkg/internal/httpserver"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type httpRoutes struct {
	logger       *zap.Logger
	db           Database
	verifier     *oidc.IDTokenVerifier
	authProvider extauth.Provider
	emailService email.Service
}

func (r *httpRoutes) Register(e *echo.Echo) {
	v1 := e.Group("/api/v1")

	v1.PUT("/user/role/binding", httpserver.AuthorizeHandler(r.PutRoleBinding, api.AdminRole))
	v1.GET("/user/role/bindings", httpserver.AuthorizeHandler(r.GetRoleBindings, api.ViewerRole))
	v1.POST("/user/invite", httpserver.AuthorizeHandler(r.InviteUser, api.AdminRole))
	v1.GET("/host/role/bindings", httpserver.AuthorizeHandler(r.GetWorkspaceRoleBindings, api.AdminRole))
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
// @Summary      Update RoleBinding for a user.
// @Description  RoleBinding defines the roles and actions a user can perform. There are currently three roles (ADMIN, EDITOR, VIEWER). User must exist before you can update its RoleBinding. If you want to add a role binding for a user given the email address, call invite first to get a user id. Then call this endpoint.
// @Tags         auth
// @Produce      json
// @Success      200
// @Param        userId  body  string  true  "userId"
// @Param        role    body  string  true  "role"
// @Router       /auth/api/v1/role/bindings [put]
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

	usr, err := r.db.GetUserByID(req.UserID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return echo.NewHTTPError(http.StatusBadRequest, "user not found")
		}

		return err
	}

	err = r.db.CreateOrUpdateRoleBinding(&RoleBinding{
		UserID:        req.UserID,
		ExternalID:    usr.ExternalID,
		WorkspaceName: httpserver.GetWorkspaceName(ctx),
		Role:          req.Role,
		AssignedAt:    time.Now(),
	})
	if err != nil {
		return err
	}

	return ctx.NoContent(http.StatusOK)
}

// GetRoleBindings godoc
// @Summary      Get RoleBindings for the calling user
// @Description  RoleBinding defines the roles and actions a user can perform. There are currently three roles (ADMIN, EDITOR, VIEWER).
// @Tags         auth
// @Produce      json
// @Success      200  {object}  api.GetRoleBindingsResponse
// @Router       /auth/api/v1/user/role/bindings [get]
func (r *httpRoutes) GetRoleBindings(ctx echo.Context) error {
	rbs, err := r.db.GetRoleBindingsOfUser(httpserver.GetUserID(ctx))
	if err != nil {
		return err
	}

	resp := make(api.GetRoleBindingsResponse, 0, len(rbs))
	for _, rb := range rbs {
		resp = append(resp, api.RoleBinding{
			WorkspaceName: rb.WorkspaceName,
			Role:          rb.Role,
			AssignedAt:    rb.AssignedAt,
		})
	}

	return ctx.JSON(http.StatusOK, resp)
}

// GetWorkspaceRoleBindings godoc
// @Summary      Get all the user RoleBindings for the given workspace.
// @Description  RoleBinding defines the roles and actions a user can perform. There are currently three roles (ADMIN, EDITOR, VIEWER). The workspace path is based on the DNS such as (workspace1.app.keibi.io)
// @Tags         auth
// @Produce      json
// @Success      200  {object}  api.GetWorkspaceRoleBindingResponse
// @Router       /auth/api/v1/workspace/role/bindings [get]
func (r *httpRoutes) GetWorkspaceRoleBindings(ctx echo.Context) error {
	rbs, err := r.db.GetRoleBindingsOfWorkspace(httpserver.GetWorkspaceName(ctx))
	if err != nil {
		return err
	}

	resp := make(api.GetWorkspaceRoleBindingResponse, 0, len(rbs))
	for _, rb := range rbs {
		resp = append(resp, api.WorkspaceRoleBinding{
			UserID:     rb.UserID,
			Role:       rb.Role,
			AssignedAt: rb.AssignedAt,
		})
	}

	return ctx.JSON(http.StatusOK, resp)
}

// InviteUser godoc
// @Summary      Invites a user by sending them an email and creating an internal ID for that user.
// @Description  RoleBinding defines the roles and actions a user can perform. There are currently three roles (ADMIN, EDITOR, VIEWER). The workspace path is based on the DNS such as (workspace1.app.keibi.io)
// @Tags         auth
// @Produce      json
// @Success      200  {object}  api.InviteUserResponse
// @Router       /auth/api/v1/user/invite [post]
func (r *httpRoutes) InviteUser(ctx echo.Context) error {
	var req api.InviteUserRequest
	if err := bindValidate(ctx, &req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	usr, err := r.inviteUser(ctx.Request().Context(), req.Email, httpserver.GetWorkspaceName(ctx))
	if err != nil {
		r.logger.Error("inviting user",
			zap.String("path", ctx.Path()),
			zap.String("method", ctx.Request().Method),
			zap.Error(err))
		return err
	}

	return ctx.JSON(http.StatusOK, api.InviteUserResponse{
		UserID: usr.ID,
	})
}

func (r *httpRoutes) inviteUser(ctx context.Context, email string, workspace string) (User, error) {
	usr, err := r.db.GetUserByEmail(email)
	if err == nil {
		return usr, nil
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return User{}, err
	}

	newAzureUser, err := r.authProvider.FetchUser(ctx, email)
	if err != nil {
		if !errors.Is(err, extauth.ErrUserNotExists) {
			return User{}, err
		}

		newAzureUser, err = r.authProvider.CreateUser(ctx, email)
		if err != nil {
			return User{}, err
		}

		// This is awful. If we lose the password, the user can never sign-in and we have to reset the password manually
		err := r.emailService.SendEmail(ctx, email, newAzureUser.PasswordProfile.Password, workspace)
		if err != nil {
			return User{}, err
		}
	}

	usr = User{
		Email:      email,
		ExternalID: newAzureUser.ID,
	}
	err = r.db.CreateUser(&usr)
	if err != nil {
		return User{}, err
	}

	return usr, nil
}
