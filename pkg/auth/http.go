package auth

import (
	"fmt"
	"net/http"
	"time"

	"gitlab.com/keibiengine/keibi-engine/pkg/auth/auth0"

	"gitlab.com/keibiengine/keibi-engine/pkg/workspace/client"

	"gitlab.com/keibiengine/keibi-engine/pkg/internal/email"

	"github.com/labstack/echo/v4"
	"gitlab.com/keibiengine/keibi-engine/pkg/auth/api"
	"gitlab.com/keibiengine/keibi-engine/pkg/internal/httpserver"
	"go.uber.org/zap"
)

const inviteDuration = time.Hour * 24 * 7

type httpRoutes struct {
	logger             *zap.Logger
	emailService       email.Service
	workspaceClient    client.WorkspaceServiceClient
	auth0Service       *auth0.Service
	inviteLinkTemplate string
}

func (r *httpRoutes) Register(e *echo.Echo) {
	v1 := e.Group("/api/v1")

	v1.PUT("/user/role/binding", httpserver.AuthorizeHandler(r.PutRoleBinding, api.AdminRole))
	v1.DELETE("/user/role/binding", httpserver.AuthorizeHandler(r.DeleteRoleBinding, api.AdminRole))
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
//
//	@Summary		Update RoleBinding for a user.
//	@Description	RoleBinding defines the roles and actions a user can perform. There are currently three roles (ADMIN, EDITOR, VIEWER). User must exist before you can update its RoleBinding. If you want to add a role binding for a user given the email address, call invite first to get a user id. Then call this endpoint.
//	@Tags			auth
//	@Produce		json
//	@Success		200		{object}	nil
//	@Param			userId	body		string	true	"userId"
//	@Param			role	body		string	true	"role"
//	@Router			/auth/api/v1/role/binding [put]
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

	workspaceName := httpserver.GetWorkspaceName(ctx)
	auth0User, err := r.auth0Service.GetUser(req.UserID)
	if err != nil {
		return err
	}

	auth0User.AppMetadata.WorkspaceAccess[workspaceName] = req.Role
	err = r.auth0Service.PatchUserAppMetadata(req.UserID, auth0User.AppMetadata)
	if err != nil {
		return err
	}
	return ctx.NoContent(http.StatusOK)
}

// DeleteRoleBinding godoc
//
//	@Summary		Delete RoleBinding for a user.
//	@Tags			auth
//	@Produce		json
//	@Success		200		{object}	nil
//	@Param			userId	body		string	true	"userId"
//	@Param			role	body		string	true	"role"
//	@Router			/auth/api/v1/user/role/binding [delete]
func (r httpRoutes) DeleteRoleBinding(ctx echo.Context) error {
	var req api.DeleteRoleBindingRequest
	if err := bindValidate(ctx, &req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	// The WorkspaceManager service will call this API to set the AdminRole
	// for the admin user on behalf of him. Allow for the Admin to only set its
	// role to admin for that user case
	if httpserver.GetUserID(ctx) == req.UserID {
		return echo.NewHTTPError(http.StatusBadRequest, "admin user permission can't be modified by self")
	}

	workspaceName := httpserver.GetWorkspaceName(ctx)
	auth0User, err := r.auth0Service.GetUser(req.UserID)
	if err != nil {
		return err
	}

	delete(auth0User.AppMetadata.WorkspaceAccess, workspaceName)
	err = r.auth0Service.PatchUserAppMetadata(req.UserID, auth0User.AppMetadata)
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
		userStr := fmt.Sprintf("%v", usr.AppMetadata)
		r.logger.Warn("user found", zap.String("user", userStr))
		for wsID, role := range usr.AppMetadata.WorkspaceAccess {
			resp.RoleBindings = append(resp.RoleBindings, api.RoleBinding{
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

// GetWorkspaceRoleBindings godoc
//
//	@Summary		Get all the user RoleBindings for the given workspace.
//	@Description	RoleBinding defines the roles and actions a user can perform. There are currently three roles (ADMIN, EDITOR, VIEWER). The workspace path is based on the DNS such as (workspace1.app.keibi.io)
//	@Tags			auth
//	@Produce		json
//	@Success		200	{object}	api.GetWorkspaceRoleBindingResponse
//	@Router			/auth/api/v1/workspace/role/bindings [get]
func (r *httpRoutes) GetWorkspaceRoleBindings(ctx echo.Context) error {
	workspaceName := httpserver.GetWorkspaceName(ctx)
	users, err := r.auth0Service.SearchUsersByWorkspace(workspaceName)
	if err != nil {
		return err
	}

	var resp api.GetWorkspaceRoleBindingResponse
	for _, u := range users {
		resp = append(resp, api.WorkspaceRoleBinding{
			UserID: u.UserId,
			Email:  u.Email,
			Role:   u.AppMetadata.WorkspaceAccess[workspaceName],
		})
	}
	return ctx.JSON(http.StatusOK, resp)
}

// Invite godoc
//
//	@Summary	Invites a user by sending them an email and registering invitation.
//	@Tags		auth
//	@Produce	json
//	@Success	200	{object}	api.InviteResponse
//	@Router		/auth/api/v1/invite [post]
func (r *httpRoutes) Invite(ctx echo.Context) error {
	var req api.InviteRequest
	if err := bindValidate(ctx, &req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}
	workspaceName := httpserver.GetWorkspaceName(ctx)

	us, err := r.auth0Service.SearchByEmail(req.Email)
	if err != nil {
		return err
	}

	if len(us) > 0 {
		auth0User := us[0]
		auth0User.AppMetadata.WorkspaceAccess[workspaceName] = req.Role
		err = r.auth0Service.PatchUserAppMetadata(auth0User.UserId, auth0User.AppMetadata)
		if err != nil {
			return err
		}
	} else {
		err = r.auth0Service.CreateUser(req.Email, workspaceName, req.Role)
		if err != nil {
			return err
		}

		resp, err := r.auth0Service.CreatePasswordChangeTicket(req.Email)
		if err != nil {
			return err
		}

		fmt.Println("Ticket:", resp.Ticket)
	}

	return echo.NewHTTPError(http.StatusOK)

	//req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	//
	//workspaceName := httpserver.GetWorkspaceName(ctx)
	//
	//inv := Invitation{
	//	Email:         req.Email,
	//	ExpiredAt:     time.Now().UTC().Add(inviteDuration),
	//	WorkspaceName: workspaceName,
	//}
	//
	//count, err := r.db.CountRoleBindings(workspaceName)
	//if err != nil {
	//	return err
	//}
	//
	//limits, err := r.workspaceClient.GetLimits(httpclient.FromEchoContext(ctx), true)
	//if err != nil {
	//	return err
	//}
	//
	//if count >= limits.MaxUsers {
	//	return echo.NewHTTPError(http.StatusBadRequest, "user limit reached")
	//}
	//
	//err = r.db.CreateInvitation(&inv)
	//if err != nil {
	//	return err
	//}
	//
	//invLink := fmt.Sprintf(r.inviteLinkTemplate, inv.ID.String())
	//mBody, err := emails.GetInviteMailBody(invLink, workspaceName)
	//if err != nil {
	//	return err
	//}
	//
	//err = r.emailService.SendEmail(ctx.Request().Context(), req.Email, mBody)
	//if err != nil {
	//	return err
	//}
	//
	//return ctx.JSON(http.StatusOK, api.InviteResponse{
	//	InviteID: inv.ID,
	//})
}

// ListInvites godoc
//
//	@Summary	lists all invites
//	@Tags		auth
//	@Produce	json
//	@Success	200	{object}	[]api.InviteItem
//	@Router		/auth/api/v1/invites [get]
func (r *httpRoutes) ListInvites(ctx echo.Context) error {
	//workspaceName := httpserver.GetWorkspaceName(ctx)
	//invs, err := r.db.ListInvitesByWorkspaceName(workspaceName)
	//if err != nil {
	//	return err
	//}

	var resp []api.InviteItem
	//for _, inv := range invs {
	//	if inv.ExpiredAt.Before(time.Now()) {
	//		continue
	//	}
	//
	//	resp = append(resp, api.InviteItem{
	//		Email: inv.Email,
	//	})
	//}
	//
	return ctx.JSON(http.StatusOK, resp)
}

// AcceptInvitation godoc
//
//	@Summary	Accepts users invitation and creates default (VIEW) role in invited workspace.
//	@Tags		auth
//	@Produce	json
//	@Success	200	{object}	nil
//	@Router		/auth/api/v1/invite/invite_id [get]
func (r *httpRoutes) AcceptInvitation(ctx echo.Context) error {
	return echo.NewHTTPError(http.StatusNotImplemented)
	//
	//invIDPrm := ctx.Param("invite_id")
	//if invIDPrm == "" {
	//	return echo.NewHTTPError(http.StatusBadRequest, errors.New("empty invitation id"))
	//}
	//
	//invID, err := uuid.Parse(invIDPrm)
	//if err != nil {
	//	return echo.NewHTTPError(http.StatusBadRequest, fmt.Errorf("bad invitation id: %w", err))
	//}
	//
	//// check that invitation exists
	//inv, err := r.db.GetInvitationByID(invID)
	//if err != nil {
	//	r.logger.Error("invitation not found",
	//		zap.String("path", ctx.Path()),
	//		zap.String("method", ctx.Request().Method),
	//		zap.Error(err))
	//
	//	if errors.Is(err, gorm.ErrRecordNotFound) {
	//		return echo.NewHTTPError(http.StatusBadRequest, "invitation not found")
	//	}
	//
	//	return err
	//}
	//
	//if inv.ExpiredAt.Before(time.Now()) {
	//	r.logger.Error("invitation expired",
	//		zap.String("path", ctx.Path()),
	//		zap.String("method", ctx.Request().Method),
	//		zap.String("invitationID", invIDPrm))
	//	return echo.NewHTTPError(http.StatusBadRequest, "invitation expired")
	//}
	//
	//userID := httpserver.GetUserID(ctx)
	//
	//// if binding exists do not change Role
	//err = r.db.CreateBindingIfNotExists(&RoleBinding{
	//	UserID:        userID,
	//	WorkspaceName: inv.WorkspaceName,
	//	Role:          api.ViewerRole,
	//	AssignedAt:    time.Now(),
	//})
	//if err != nil {
	//	return err
	//}
	//
	//err = r.db.DeleteInvitation(inv.ID)
	//if err != nil {
	//	return err
	//}
	//
	//return nil
}
