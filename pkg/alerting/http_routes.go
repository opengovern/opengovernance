package alerting

import (
	"github.com/go-errors/errors"
	"github.com/kaytu-io/kaytu-engine/pkg/alerting/api"
	authapi "github.com/kaytu-io/kaytu-engine/pkg/auth/api"
	"github.com/kaytu-io/kaytu-engine/pkg/internal/httpserver"

	"github.com/labstack/echo/v4"
	"strconv"
)

func (h *HttpHandler) Register(e *echo.Echo) {
	ruleGroup := e.Group("/api/rule")
	ruleGroup.GET("/list", httpserver.AuthorizeHandler(h.ListRules, authapi.ViewerRole))
	ruleGroup.POST("/create", httpserver.AuthorizeHandler(h.CreateRule, authapi.ViewerRole))
	ruleGroup.DELETE("/delete/:ruleID", httpserver.AuthorizeHandler(h.DeleteRule, authapi.ViewerRole))
	ruleGroup.GET("/update", httpserver.AuthorizeHandler(h.UpdateRule, authapi.ViewerRole))

	actionGroup := e.Group("/api/action")
	actionGroup.GET("/list", httpserver.AuthorizeHandler(h.ListActions, authapi.ViewerRole))
	actionGroup.POST("/create", httpserver.AuthorizeHandler(h.CreateAction, authapi.ViewerRole))
	actionGroup.DELETE("/delete/:actionID", httpserver.AuthorizeHandler(h.DeleteAction, authapi.ViewerRole))
	actionGroup.GET("/update", httpserver.AuthorizeHandler(h.UpdateAction, authapi.ViewerRole))
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

// ListRules godoc
//
//	@Summary		List rules
//	@Description	returns list of all rules
//	@Security		BearerToken
//	@Tags			alerting
//	@Produce		json
//	@Success		200			{object}	[]api.ResponseRule
//	@Router			/alerting/api/rule/list [get]
func (h *HttpHandler) ListRules(ctx echo.Context) error {
	rules, err := h.db.ListRules()
	if err != nil {
		return err
	}

	var response []api.ResponseRule
	for _, rule := range rules {
		response = append(response, api.ResponseRule{
			ID:        rule.ID,
			EventType: rule.EventType,
			Scope:     rule.Scope,
			Operator:  rule.Operator,
			Value:     rule.Value,
			ActionID:  rule.ActionID,
		})
	}

	return ctx.JSON(200, response)
}

// CreateRule godoc
//
//	@Summary		Create rule
//	@Description	create a rule by the specified input
//	@Security		BearerToken
//	@Tags			alerting
//	@Param			request		body		api.RequestRule			true	"Request Body"
//	@Success		200
//	@Router			/alerting/api/rule/create [post]
func (h *HttpHandler) CreateRule(ctx echo.Context) error {
	var req api.RequestRule
	if err := bindValidate(ctx, &req); err != nil {
		return err
	}

	if &req.ID == nil || &req.Value == nil || &req.Scope == nil || &req.ActionID == nil || &req.Operator == nil || &req.EventType == nil {
		return errors.New("All the fields in struct must be set")
	}

	request := Rule{
		ID:        req.ID,
		EventType: req.EventType,
		Scope:     req.Scope,
		Operator:  req.Operator,
		Value:     req.Value,
		ActionID:  req.ActionID,
	}

	if err := h.db.CreateRule(request); err != nil {
		return err
	}

	return ctx.JSON(200, "Rule successfully created")
}

// DeleteRule godoc
//
//	@Summary		Delete rule
//	@Description	Deleting a single rule for the given rule id
//	@Security		BearerToken
//	@Tags			alerting
//	@Param			ruleID		path		string			true	"Rule ID"
//	@Success		200
//	@Router			/alerting/api/rule/Delete/{ruleID} [delete]
func (h *HttpHandler) DeleteRule(ctx echo.Context) error {
	idS := ctx.Param("ruleID")
	if idS == "" {
		return errors.New("The ruleID input must be set")
	}
	id, err := strconv.ParseUint(idS, 10, 64)
	if err != nil {
		return err
	}

	if err = h.db.DeleteRule(uint(id)); err != nil {
		return err
	}

	return ctx.JSON(200, "Rule successfully deleted")
}

// UpdateRule godoc
//
//	@Summary		Update rule
//	@Description	Retrieving a rule by the specified input
//	@Security		BearerToken
//	@Tags			alerting
//	@Param			request		body		api.RequestRule			true	"Request Body"
//	@Success		200
//	@Router			/alerting/api/rule/update [get]
func (h *HttpHandler) UpdateRule(ctx echo.Context) error {
	var req api.RequestRule
	if err := bindValidate(ctx, &req); err != nil {
		return err
	}
	if req.ID == 0 {
		return errors.New("The ruleID must be set")
	}

	request := Rule{
		ID:        req.ID,
		EventType: req.EventType,
		Scope:     req.Scope,
		Operator:  req.Operator,
		Value:     req.Value,
		ActionID:  req.ActionID,
	}

	err := h.db.UpdateRule(request)
	if err != nil {
		return err
	}
	return ctx.JSON(200, "Rule successfully updated")
}

// ListActions godoc
//
//	@Summary		List actions
//	@Description	returns list of all actions
//	@Security		BearerToken
//	@Tags			alerting
//	@Produce		json
//	@Success		200		{object}	[]api.ResponseAction
//	@Router			/alerting/api/action/list [get]
func (h *HttpHandler) ListActions(ctx echo.Context) error {
	actions, err := h.db.ListAction()
	if err != nil {
		return err
	}

	var response []api.ResponseAction
	for _, action := range actions {
		response = append(response, api.ResponseAction{
			ID:      action.ID,
			Method:  action.Method,
			Url:     action.Url,
			Headers: action.Headers,
			Body:    action.Body,
		})
	}

	return ctx.JSON(200, response)
}

// CreateAction godoc
//
//	@Summary		Create action
//	@Description	create an action by the specified input
//	@Security		BearerToken
//	@Tags			alerting
//	@Param			request		body	api.RequestAction		true	"Request Body"
//	@Success		200
//	@Router			/alerting/api/action/create [post]
func (h *HttpHandler) CreateAction(ctx echo.Context) error {
	var req api.RequestAction
	err := bindValidate(ctx, &req)
	if err != nil {
		return err
	}

	if &req.ID == nil || &req.Url == nil || &req.Body == nil || &req.Method == nil || req.Headers == nil {
		return errors.New("All the fields in struct must be set")
	}

	request := Action{
		ID:      req.ID,
		Method:  req.Method,
		Url:     req.Url,
		Headers: req.Headers,
		Body:    req.Body,
	}

	err = h.db.CreateAction(request)
	if err != nil {
		return err
	}

	return ctx.JSON(200, "Action created successfully ")
}

// DeleteAction godoc
//
//	@Summary		Delete action
//	@Description	Deleting a single action for the given action id
//	@Security		BearerToken
//	@Tags			alerting
//	@Param			actionID		path	string		true	"ActionID"
//	@Success		200
//	@Router			/alerting/api/action/delete/{actionID} [delete]
func (h *HttpHandler) DeleteAction(ctx echo.Context) error {
	idS := ctx.Param("actionID")
	if idS == "" {
		return errors.New("The actionID must be set")
	}
	id, err := strconv.ParseUint(idS, 10, 64)
	if err != nil {
		return err
	}

	err = h.db.DeleteAction(uint(id))
	if err != nil {
		return err
	}

	return ctx.JSON(200, "Action deleted successfully")
}

// UpdateAction godoc
//
//	@Summary		Update action
//	@Description	Retrieving an action by the specified input
//	@Security		BearerToken
//	@Tags			alerting
//	@Param			request		body	api.RequestAction		true	"Request Body"
//	@Success		200
//	@Router			/alerting/api/action/update [get]
func (h *HttpHandler) UpdateAction(ctx echo.Context) error {
	var req api.RequestAction
	if err := bindValidate(ctx, &req); err != nil {
		return err
	}
	if req.ID == 0 {
		return errors.New("ActionID in struct must be set")
	}

	request := Action{
		ID:      req.ID,
		Method:  req.Method,
		Url:     req.Url,
		Headers: req.Headers,
		Body:    req.Body,
	}

	err := h.db.UpdateAction(request)
	if err != nil {
		return err
	}
	return ctx.JSON(200, "Action updated successfully")
}
