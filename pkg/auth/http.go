package auth

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"gitlab.com/keibiengine/keibi-engine/pkg/internal/httpclient"
	"gitlab.com/keibiengine/keibi-engine/pkg/workspace/client"

	"gitlab.com/keibiengine/keibi-engine/pkg/internal/email"

	"gitlab.com/keibiengine/keibi-engine/pkg/auth/emails"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"gitlab.com/keibiengine/keibi-engine/pkg/auth/api"
	"gitlab.com/keibiengine/keibi-engine/pkg/internal/httpserver"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

const inviteDuration = time.Hour * 24 * 7

type httpRoutes struct {
	logger             *zap.Logger
	db                 Database
	emailService       email.Service
	workspaceClient    client.WorkspaceServiceClient
	inviteLinkTemplate string
}

func (r *httpRoutes) Register(e *echo.Echo) {
	v1 := e.Group("/api/v1")

	v1.PUT("/user/role/binding", httpserver.AuthorizeHandler(r.PutRoleBinding, api.AdminRole))
	v1.GET("/user/role/bindings", httpserver.AuthorizeHandler(r.GetRoleBindings, api.ViewerRole))
	v1.GET("/workspace/role/bindings", httpserver.AuthorizeHandler(r.GetWorkspaceRoleBindings, api.AdminRole))
	v1.GET("/invites", httpserver.AuthorizeHandler(r.ListInvites, api.AdminRole))
	v1.POST("/invite", httpserver.AuthorizeHandler(r.Invite, api.AdminRole))
	v1.GET("/invite/:invite_id", httpserver.AuthorizeHandler(r.AcceptInvitation, api.ViewerRole))
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
// @Summary     Update RoleBinding for a user.
// @Description RoleBinding defines the roles and actions a user can perform. There are currently three roles (ADMIN, EDITOR, VIEWER). User must exist before you can update its RoleBinding. If you want to add a role binding for a user given the email address, call invite first to get a user id. Then call this endpoint.
// @Tags        auth
// @Produce     json
// @Success     200    {object} nil
// @Param       userId body     string true "userId"
// @Param       role   body     string true "role"
// @Router      /auth/api/v1/role/binding [put]
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

	workspaceName := httpserver.GetWorkspaceName(ctx)
	count, err := r.db.CountRoleBindings(workspaceName)
	if err != nil {
		return err
	}

	limits, err := r.workspaceClient.GetLimits(httpclient.FromEchoContext(ctx))
	if err != nil {
		return err
	}

	if count >= limits.MaxUsers {
		err = r.db.UpdateRoleBinding(&RoleBinding{
			UserID:        req.UserID,
			ExternalID:    usr.ExternalID,
			WorkspaceName: workspaceName,
			Role:          req.Role,
			AssignedAt:    time.Now(),
		})
	} else {
		err = r.db.CreateOrUpdateRoleBinding(&RoleBinding{
			UserID:        req.UserID,
			ExternalID:    usr.ExternalID,
			WorkspaceName: workspaceName,
			Role:          req.Role,
			AssignedAt:    time.Now(),
		})
	}
	if err != nil {
		return err
	}

	return ctx.NoContent(http.StatusOK)
}

// GetRoleBindings godoc
// @Summary     Get RoleBindings for the calling user
// @Description RoleBinding defines the roles and actions a user can perform. There are currently three roles (ADMIN, EDITOR, VIEWER).
// @Tags        auth
// @Produce     json
// @Success     200 {object} api.GetRoleBindingsResponse
// @Router      /auth/api/v1/user/role/bindings [get]
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
// @Summary     Get all the user RoleBindings for the given workspace.
// @Description RoleBinding defines the roles and actions a user can perform. There are currently three roles (ADMIN, EDITOR, VIEWER). The workspace path is based on the DNS such as (workspace1.app.keibi.io)
// @Tags        auth
// @Produce     json
// @Success     200 {object} api.GetWorkspaceRoleBindingResponse
// @Router      /auth/api/v1/workspace/role/bindings [get]
func (r *httpRoutes) GetWorkspaceRoleBindings(ctx echo.Context) error {
	rbs, err := r.db.GetRoleBindingsOfWorkspace(httpserver.GetWorkspaceName(ctx))
	if err != nil {
		return err
	}

	resp := make(api.GetWorkspaceRoleBindingResponse, 0, len(rbs))
	for _, rb := range rbs {
		u, err := r.db.GetUserByID(rb.UserID)
		if err != nil {
			return err
		}

		resp = append(resp, api.WorkspaceRoleBinding{
			UserID:     rb.UserID,
			Email:      u.Email,
			Role:       rb.Role,
			AssignedAt: rb.AssignedAt,
		})
	}

	return ctx.JSON(http.StatusOK, resp)
}

// Invite godoc
// @Summary Invites a user by sending them an email and registering invitation.
// @Tags    auth
// @Produce json
// @Success 200 {object} api.InviteResponse
// @Router  /auth/api/v1/invite [post]
func (r *httpRoutes) Invite(ctx echo.Context) error {
	var req api.InviteRequest
	if err := bindValidate(ctx, &req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	workspaceName := httpserver.GetWorkspaceName(ctx)

	inv := Invitation{
		Email:         req.Email,
		ExpiredAt:     time.Now().UTC().Add(inviteDuration),
		WorkspaceName: workspaceName,
	}

	count, err := r.db.CountRoleBindings(workspaceName)
	if err != nil {
		return err
	}

	limits, err := r.workspaceClient.GetLimits(httpclient.FromEchoContext(ctx))
	if err != nil {
		return err
	}

	if count >= limits.MaxUsers {
		return echo.NewHTTPError(http.StatusBadRequest, "user limit reached")
	}

	err = r.db.CreateInvitation(&inv)
	if err != nil {
		return err
	}

	invLink := fmt.Sprintf(r.inviteLinkTemplate, inv.ID.String())
	mBody, err := emails.GetInviteMailBody(invLink, workspaceName)
	if err != nil {
		return err
	}

	err = r.emailService.SendEmail(ctx.Request().Context(), req.Email, mBody)
	if err != nil {
		return err
	}

	return ctx.JSON(http.StatusOK, api.InviteResponse{
		InviteID: inv.ID,
	})
}

// ListInvites godoc
// @Summary lists all invites
// @Tags    auth
// @Produce json
// @Success 200 {object} []api.InviteItem
// @Router  /auth/api/v1/invites [get]
func (r *httpRoutes) ListInvites(ctx echo.Context) error {
	workspaceName := httpserver.GetWorkspaceName(ctx)
	invs, err := r.db.ListInvitesByWorkspaceName(workspaceName)
	if err != nil {
		return err
	}

	var resp []api.InviteItem
	for _, inv := range invs {
		if inv.ExpiredAt.Before(time.Now()) {
			continue
		}

		resp = append(resp, api.InviteItem{
			Email: inv.Email,
		})
	}

	return ctx.JSON(http.StatusOK, resp)
}

// AcceptInvitation godoc
// @Summary Accepts users invitation and creates default (VIEW) role in invited workspace.
// @Tags    auth
// @Produce json
// @Success 200 {object} nil
// @Router  /auth/api/v1/invite/invite_id [get]
func (r *httpRoutes) AcceptInvitation(ctx echo.Context) error {
	invIDPrm := ctx.Param("invite_id")
	if invIDPrm == "" {
		return echo.NewHTTPError(http.StatusBadRequest, errors.New("empty invitation id"))
	}

	invID, err := uuid.Parse(invIDPrm)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Errorf("bad invitation id: %w", err))
	}

	// check that invitation exists
	inv, err := r.db.GetInvitationByID(invID)
	if err != nil {
		r.logger.Error("invitation not found",
			zap.String("path", ctx.Path()),
			zap.String("method", ctx.Request().Method),
			zap.Error(err))

		if errors.Is(err, gorm.ErrRecordNotFound) {
			return echo.NewHTTPError(http.StatusBadRequest, "invitation not found")
		}

		return err
	}

	if inv.ExpiredAt.Before(time.Now()) {
		r.logger.Error("invitation expired",
			zap.String("path", ctx.Path()),
			zap.String("method", ctx.Request().Method),
			zap.String("invitationID", invIDPrm))
		return echo.NewHTTPError(http.StatusBadRequest, "invitation expired")
	}

	userID := httpserver.GetUserID(ctx)

	// check that invited user exists
	usr, err := r.db.GetUserByID(userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return echo.NewHTTPError(http.StatusBadRequest, "user not found")
		}

		return err
	}

	// if binding exists do not change Role
	err = r.db.CreateBindingIfNotExists(&RoleBinding{
		UserID:        userID,
		ExternalID:    usr.ExternalID,
		WorkspaceName: inv.WorkspaceName,
		Role:          api.ViewerRole,
		AssignedAt:    time.Now(),
	})
	if err != nil {
		return err
	}

	err = r.db.DeleteInvitation(inv.ID)
	if err != nil {
		return err
	}

	return nil
}
