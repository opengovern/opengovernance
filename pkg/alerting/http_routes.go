package alerting

import (
	"github.com/go-errors/errors"
	authapi "github.com/kaytu-io/kaytu-engine/pkg/auth/api"
	"github.com/kaytu-io/kaytu-engine/pkg/internal/httpserver"
	"github.com/labstack/echo/v4"
	"strconv"
)

func (h *HttpHandler) Register(e *echo.Echo) {
	ruleGroup := e.Group("/api/rule")
	ruleGroup.GET("/list", httpserver.AuthorizeHandler(h.ListRules, authapi.ViewerRole))
	ruleGroup.POST("/create/", httpserver.AuthorizeHandler(h.CreateRule, authapi.ViewerRole))
	ruleGroup.DELETE("/delete/:ruleId", httpserver.AuthorizeHandler(h.DeleteRule, authapi.ViewerRole))
	ruleGroup.GET("/update/:ruleId", httpserver.AuthorizeHandler(h.UpdateRule, authapi.ViewerRole))

	actionGroup := e.Group("/api/action")
	actionGroup.GET("/list", httpserver.AuthorizeHandler(h.ListActions, authapi.ViewerRole))
	actionGroup.POST("/create/", httpserver.AuthorizeHandler(h.CreateAction, authapi.ViewerRole))
	actionGroup.DELETE("/delete/:ActionID", httpserver.AuthorizeHandler(h.DeleteAction, authapi.ViewerRole))
	actionGroup.GET("/update/:ActionID", httpserver.AuthorizeHandler(h.UpdateAction, authapi.ViewerRole))
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

func (h *HttpHandler) ListRules(ctx echo.Context) error {
	rules, err := h.db.ListRules()
	if err != nil {
		return err
	}
	return ctx.JSON(200, rules)
}

func (h *HttpHandler) CreateRule(ctx echo.Context) error {
	var req Rule
	if err := bindValidate(ctx, &req); err != nil {
		return err
	}

	if err := h.db.CreateRule(req); err != nil {
		return err
	}

	return ctx.JSON(200, "Rule successfully added")
}

func (h *HttpHandler) DeleteRule(ctx echo.Context) error {
	idS := ctx.Param("ruleId")
	id, err := strconv.ParseUint(idS, 10, 64)
	if err != nil {
		return err
	}
	if id == 0 {
		return errors.New("The ruleId input must be set")
	}

	if err = h.db.DeleteRule(uint(id)); err != nil {
		return err
	}

	return ctx.JSON(200, "Rule successfully deleted")
}

func (h *HttpHandler) UpdateRule(ctx echo.Context) error {
	var value int64
	var operator string

	idS := ctx.Param("ruleId")
	id, err := strconv.ParseUint(idS, 10, 64)
	if err != nil {
		return err
	}
	if id == 0 {
		return errors.New("The ruleId input must be set")
	}

	err = echo.QueryParamsBinder(ctx).
		String("operator", &operator).
		Int64("value", &value).
		BindError()
	if err != nil {
		return err
	}

	if id == 0 {
		return errors.New("The id input must be set")
	}

	if value == 0 && operator == "" {
		return errors.New("The operator or value inputs must be set")
	}

	err = h.db.UpdateRule(uint(id), &operator, &value)
	if err != nil {
		return err
	}
	return ctx.JSON(200, "Rule successfully updated")
}

func (h *HttpHandler) ListActions(ctx echo.Context) error {
	actions, err := h.db.ListAction()
	if err != nil {
		return err
	}

	return ctx.JSON(200, actions)
}

func (h *HttpHandler) CreateAction(ctx echo.Context) error {
	var req Action
	err := bindValidate(ctx, &req)
	if err != nil {
		return err
	}

	err = h.db.CreateAction(req)
	if err != nil {
		return err
	}

	return ctx.JSON(200, "Action created successfully ")
}

func (h *HttpHandler) DeleteAction(ctx echo.Context) error {
	idS := ctx.Param("actionID")
	id, err := strconv.ParseUint(idS, 10, 64)
	if err != nil {
		return err
	}
	if id == 0 {
		return errors.New("The actionID must be set")
	}

	err = h.db.DeleteAction(uint(id))
	if err != nil {
		return err
	}

	return ctx.JSON(200, "Action deleted successfully")
}

func (h *HttpHandler) UpdateAction(ctx echo.Context) error {
	idS := ctx.Param("actionID")
	id, err := strconv.ParseUint(idS, 10, 64)
	if err != nil {
		return err
	}
	if id == 0 {
		return errors.New("The actionID must be set")
	}

	var method string
	var url string
	var body string

	err = echo.QueryParamsBinder(ctx).
		String("method", &method).
		String("url", &url).
		String("body", &body).
		BindError()
	if err != nil {
		return err
	}
	if url == "" || body == "" {
		return errors.New("The url and body inputs must be set")
	}

	err = h.db.UpdateAction(uint(id), &method, &url, &body)
	if err != nil {
		return err
	}

	return ctx.JSON(200, "Action updated successfully")
}
