package alerting

import (
	"encoding/json"
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
	ruleGroup.POST("/create", httpserver.AuthorizeHandler(h.CreateRule, authapi.EditorRole))
	ruleGroup.DELETE("/delete/:ruleID", httpserver.AuthorizeHandler(h.DeleteRule, authapi.EditorRole))
	ruleGroup.GET("/update", httpserver.AuthorizeHandler(h.UpdateRule, authapi.EditorRole))

	actionGroup := e.Group("/api/action")
	actionGroup.GET("/list", httpserver.AuthorizeHandler(h.ListActions, authapi.ViewerRole))
	actionGroup.POST("/create", httpserver.AuthorizeHandler(h.CreateAction, authapi.EditorRole))
	actionGroup.DELETE("/delete/:actionID", httpserver.AuthorizeHandler(h.DeleteAction, authapi.EditorRole))
	actionGroup.GET("/update", httpserver.AuthorizeHandler(h.UpdateAction, authapi.EditorRole))
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
//	@Success		200			{object}	[]api.ApiRule
//	@Router			/alerting/api/rule/list [get]
func (h *HttpHandler) ListRules(ctx echo.Context) error {
	rules, err := h.db.ListRules()
	if err != nil {
		return err
	}

	var response []api.ApiRule
	for _, rule := range rules {

		var eventType api.EventType
		err := json.Unmarshal(rule.EventType, &eventType)
		if err != nil {
			return err
		}

		var scope api.Scope
		err = json.Unmarshal(rule.Scope, &scope)
		if err != nil {
			return err
		}

		response = append(response, api.ApiRule{
			ID:        rule.ID,
			EventType: eventType,
			Scope:     scope,
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
//	@Param			request		body		api.ApiRule			true	"Request Body"
//	@Success		200
//	@Router			/alerting/api/rule/create [post]
func (h *HttpHandler) CreateRule(ctx echo.Context) error {
	var req api.ApiRule
	if err := bindValidate(ctx, &req); err != nil {
		return err
	}

	EmptyFields := api.ApiRule{}
	if req.ID == EmptyFields.ID || req.Value == EmptyFields.Value || req.Scope == EmptyFields.Scope ||
		req.ActionID == EmptyFields.ActionID || req.Operator == EmptyFields.Operator || req.EventType == EmptyFields.EventType {
		return errors.New("All the fields in struct must be set")
	}

	scope, err := json.Marshal(req.Scope)
	if err != nil {
		return err
	}

	event, err := json.Marshal(req.EventType)
	if err != nil {
		return err
	}

	request := Rule{
		ID:        req.ID,
		EventType: event,
		Scope:     scope,
		Operator:  req.Operator,
		Value:     req.Value,
		ActionID:  req.ActionID,
	}

	if err := h.db.CreateRule(&request); err != nil {
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
		return errors.New("ruleID is required")
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
//	@Param			request		body		api.ApiRule			true	"Request Body"
//	@Success		200
//	@Router			/alerting/api/rule/update [get]
func (h *HttpHandler) UpdateRule(ctx echo.Context) error {
	var req api.ApiRule
	if err := bindValidate(ctx, &req); err != nil {
		return err
	}
	if req.ID == 0 {
		return errors.New("ruleID is required")
	}

	scope, err := json.Marshal(req.Scope)
	if err != nil {
		return err
	}

	event, err := json.Marshal(req.EventType)
	if err != nil {
		return err
	}

	request := Rule{
		ID:        req.ID,
		EventType: event,
		Scope:     scope,
		Operator:  req.Operator,
		Value:     req.Value,
		ActionID:  req.ActionID,
	}

	err = h.db.UpdateRule(&request)
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
//	@Success		200		{object}	[]api.ApiAction
//	@Router			/alerting/api/action/list [get]
func (h *HttpHandler) ListActions(ctx echo.Context) error {
	actions, err := h.db.ListAction()
	if err != nil {
		return err
	}

	var response []api.ApiAction
	for _, action := range actions {

		var headers api.Headers
		err := json.Unmarshal(action.Headers, headers)
		if err != nil {
			return err
		}

		response = append(response, api.ApiAction{
			ID:      action.ID,
			Method:  action.Method,
			Url:     action.Url,
			Headers: headers,
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
//	@Param			request		body	api.ApiAction		true	"Request Body"
//	@Success		200
//	@Router			/alerting/api/action/create [post]
func (h *HttpHandler) CreateAction(ctx echo.Context) error {
	var req api.ApiAction
	var testEmptyFields api.ApiAction
	err := bindValidate(ctx, &req)
	if err != nil {
		return err
	}

	if req.ID == testEmptyFields.ID || req.Url == testEmptyFields.Url || req.Body == testEmptyFields.Body ||
		req.Method == testEmptyFields.Method || req.Headers == testEmptyFields.Headers {
		return errors.New("All the fields in struct must be set")
	}
	headers, err := json.Marshal(req.Headers)
	if err != nil {
		return err
	}

	request := Action{
		ID:      req.ID,
		Method:  req.Method,
		Url:     req.Url,
		Headers: headers,
		Body:    req.Body,
	}

	err = h.db.CreateAction(&request)
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
		return errors.New("actionID is required")
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
//	@Param			request		body	api.ApiAction		true	"Request Body"
//	@Success		200
//	@Router			/alerting/api/action/update [get]
func (h *HttpHandler) UpdateAction(ctx echo.Context) error {
	var req api.ApiAction
	if err := bindValidate(ctx, &req); err != nil {
		return err
	}
	if req.ID == 0 {
		return errors.New("actionID is required")
	}

	headers, err := json.Marshal(req.Headers)
	if err != nil {
		return err
	}

	request := Action{
		ID:      req.ID,
		Method:  req.Method,
		Url:     req.Url,
		Headers: headers,
		Body:    req.Body,
	}

	err = h.db.UpdateAction(&request)
	if err != nil {
		return err
	}
	return ctx.JSON(200, "Action updated successfully")
}
