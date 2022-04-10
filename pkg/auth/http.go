package auth

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"gitlab.com/keibiengine/keibi-engine/pkg/auth/api"
)

type httpRoutes struct {
	db Database
}

func (r httpRoutes) Register(e *echo.Echo) {
	v1 := e.Group("/api/v1")

	v1.GET("/role/binding", r.GetRoleBinding)
	v1.PUT("/role/binding", r.PutRoleBinding)
	v1.GET("/role/bindings", r.GetRoleBindings)
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
// @Description  RoleBinding defines the roles and actions a user can perform. There are currently three roles (ADMIN, EDITOR, VIEWER). User must exist before you can update its RoleBinding
// @Tags         auth
// @Produce      json
// @Success      200
// @Failure      400     {object}  api.ErrorResponse
// @Param        userId  body      string  true  "userId"
// @Param        role    body      string  true  "role"
// @Router       /auth/api/v1/role/bindings [put]
func (r httpRoutes) PutRoleBinding(ctx echo.Context) error {
	var req api.PutRoleBindingRequest
	if err := bindValidate(ctx, &req); err != nil {
		return ctx.JSON(http.StatusBadRequest, api.ErrorResponse{
			Message: err.Error(),
		})
	}

	err := r.db.UpdateRoleBinding(&RoleBinding{
		UserID:     req.UserID,
		Role:       req.Role,
		AssignedAt: time.Now(),
	})
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, api.ErrorResponse{
			Message: err.Error(),
		})
	}

	return ctx.NoContent(http.StatusOK)
}

// GetRoleBinding godoc
// @Summary      Get RoleBinding for a single user.
// @Description  RoleBinding defines the roles and actions a user can perform. There are currently three roles (ADMIN, EDITOR, VIEWER).
// @Tags         auth
// @Accept       json
// @Produce      json
// @Success      200     {object}  api.GetRoleBindingResponse
// @Failure      400     {object}  api.ErrorResponse
// @Param        userId  body      string  true  "userId"
// @Router       /auth/api/v1/role/binding [get]
func (r httpRoutes) GetRoleBinding(ctx echo.Context) error {
	var req api.GetRoleBindingRequest
	if err := bindValidate(ctx, &req); err != nil {
		return ctx.JSON(http.StatusBadRequest, api.ErrorResponse{
			Message: err.Error(),
		})
	}

	rb, err := r.db.GetUserRoleBinding(req.UserID)
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, api.ErrorResponse{
			Message: err.Error(),
		})
	}

	return ctx.JSON(http.StatusOK, api.GetRoleBindingResponse{
		UserID:     rb.UserID,
		Role:       rb.Role,
		Name:       rb.Name,
		Emails:     rb.Emails,
		AssignedAt: rb.AssignedAt,
	})
}

// GetRoleBindings godoc
// @Summary      Get RoleBinding for all users.
// @Description  RoleBinding defines the roles and actions a user can perform. There are currently three roles (ADMIN, EDITOR, VIEWER).
// @Tags         auth
// @Produce      json
// @Success      200  {object}  api.GetRoleBindingsResponse
// @Failure      400  {object}  api.ErrorResponse
// @Router       /auth/api/v1/role/bindings [get]
func (r httpRoutes) GetRoleBindings(ctx echo.Context) error {
	rbs, err := r.db.GetAllUserRoleBindings()
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, api.ErrorResponse{
			Message: err.Error(),
		})
	}

	resp := make(api.GetRoleBindingsResponse, 0, len(rbs))
	for _, rb := range rbs {
		resp = append(resp, api.RoleBinding{
			UserID:     rb.UserID,
			Role:       rb.Role,
			Name:       rb.Name,
			Emails:     rb.Emails,
			AssignedAt: rb.AssignedAt,
		})
	}

	return ctx.JSON(http.StatusOK, resp)
}
