package alerting

import (
	"encoding/json"
	"fmt"
	"github.com/go-errors/errors"
	"github.com/kaytu-io/kaytu-engine/pkg/alerting/api"
	authapi "github.com/kaytu-io/kaytu-engine/pkg/auth/api"
	"github.com/kaytu-io/kaytu-engine/pkg/internal/httpserver"
	"github.com/labstack/echo/v4"
	"net/http"
	"strconv"
)

func (h *HttpHandler) Register(e *echo.Echo) {
	v1 := e.Group("/api/v1")
	ruleGroup := v1.Group("/rule")
	ruleGroup.GET("/list", httpserver.AuthorizeHandler(h.ListRules, authapi.ViewerRole))
	ruleGroup.POST("/create", httpserver.AuthorizeHandler(h.CreateRule, authapi.EditorRole))
	ruleGroup.DELETE("/delete/:ruleId", httpserver.AuthorizeHandler(h.DeleteRule, authapi.EditorRole))
	ruleGroup.PUT("/update", httpserver.AuthorizeHandler(h.UpdateRule, authapi.EditorRole))
	ruleGroup.GET("/:ruleId/trigger", httpserver.AuthorizeHandler(h.TriggerRuleAPI, authapi.EditorRole))

	actionGroup := v1.Group("/action")
	actionGroup.GET("/list", httpserver.AuthorizeHandler(h.ListActions, authapi.ViewerRole))
	actionGroup.POST("/create", httpserver.AuthorizeHandler(h.CreateAction, authapi.EditorRole))
	actionGroup.DELETE("/delete/:actionId", httpserver.AuthorizeHandler(h.DeleteAction, authapi.EditorRole))
	actionGroup.PUT("/update", httpserver.AuthorizeHandler(h.UpdateAction, authapi.EditorRole))
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

// TriggerRuleAPI godoc
//
//	@Summary		Trigger one rule
//	@Description	Trigger one rule manually
//	@Security		BearerToken
//	@Tags			alerting
//	@Produce		json
//	@Param			ruleId	path		string	true	"RuleID"
//	@Success		200		{object}	string
//	@Router			/alerting/api/v1/rule/{ruleId}/trigger [get]
func (h *HttpHandler) TriggerRuleAPI(ctx echo.Context) error {
	ruleIdStr := ctx.Param("ruleId")
	ruleId, err := strconv.ParseUint(ruleIdStr, 10, 32)
	if err != nil {
		return ctx.String(http.StatusInternalServerError, fmt.Sprintf("error parsing the ruleId : %v ", err))
	}

	rule, err := h.db.GetRule(uint(ruleId))
	if err != nil {
		return ctx.String(http.StatusBadRequest, fmt.Sprintf("Couldn't get rule , %v ", err))
	}
	err = h.TriggerRule(rule)
	if err != nil {
		return ctx.String(http.StatusInternalServerError, err.Error())
	}
	return ctx.JSON(http.StatusOK, "trigger executed successfully")
}

// ListRules godoc
//
//	@Summary		List rules
//	@Description	returns list of all rules
//	@Security		BearerToken
//	@Tags			alerting
//	@Produce		json
//	@Success		200	{object}	[]api.Rule
//	@Router			/alerting/api/v1/rule/list [get]
func (h *HttpHandler) ListRules(ctx echo.Context) error {
	rules, err := h.db.ListRules()
	if err != nil {
		return ctx.String(http.StatusBadRequest, fmt.Sprintf("error getting the list of the rules : %v ", err))
	}

	var response []api.Rule
	for _, rule := range rules {

		var eventType api.EventType
		err := json.Unmarshal(rule.EventType, &eventType)
		if err != nil {
			return ctx.String(http.StatusBadRequest, fmt.Sprintf("error unmarshalling eventType : %v ", err))
		}

		var scope api.Scope
		err = json.Unmarshal(rule.Scope, &scope)
		if err != nil {
			return ctx.String(http.StatusBadRequest, fmt.Sprintf("error unmarshalling scope : %v ", err))
		}

		var operator api.OperatorStruct
		err = json.Unmarshal(rule.Operator, &operator)
		if err != nil {
			return ctx.String(http.StatusBadRequest, fmt.Sprintf("error unmarshalling operator : %v ", err))
		}

		response = append(response, api.Rule{
			Id:        rule.Id,
			EventType: eventType,
			Scope:     scope,
			Operator:  operator,
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
//	@Param			request	body		api.CreateRuleRequest	true	"Request Body"
//	@Success		200		{object}	string
//	@Router			/alerting/api/v1/rule/create [post]
func (h *HttpHandler) CreateRule(ctx echo.Context) error {
	var req api.CreateRuleRequest
	if err := bindValidate(ctx, &req); err != nil {
		return ctx.String(http.StatusBadRequest, fmt.Sprintf("error getting the inputs : %v ", err))
	}

	EmptyFields := api.CreateRuleRequest{}
	if req.Scope == EmptyFields.Scope ||
		req.ActionID == EmptyFields.ActionID || req.Operator == EmptyFields.Operator || req.EventType == EmptyFields.EventType {
		return errors.New("All the fields in struct must be set")
	}

	scope, err := json.Marshal(req.Scope)
	if err != nil {
		return ctx.String(http.StatusBadRequest, fmt.Sprintf("error marshalling scope : %v ", err))
	}

	event, err := json.Marshal(req.EventType)
	if err != nil {
		return ctx.String(http.StatusBadRequest, fmt.Sprintf("error marshalling eventType : %v ", err))
	}

	operator, err := json.Marshal(req.Operator)
	if err != nil {
		return ctx.String(http.StatusBadRequest, fmt.Sprintf("error marshalling operator : %v ", err))
	}

	if err := h.db.CreateRule(event, scope, operator, req.ActionID); err != nil {
		return ctx.String(http.StatusInternalServerError, fmt.Sprintf("error creating rule : %v ", err))
	}

	return ctx.JSON(200, "Rule successfully created")
}

// DeleteRule godoc
//
//	@Summary		Delete rule
//	@Description	Deleting a single rule for the given rule id
//	@Security		BearerToken
//	@Tags			alerting
//	@Param			ruleId	path		string	true	"ruleId"
//	@Success		200		{object}	string
//	@Router			/alerting/api/v1/rule/delete/{ruleId} [delete]
func (h *HttpHandler) DeleteRule(ctx echo.Context) error {
	idS := ctx.Param("ruleId")
	if idS == "" {
		return errors.New("ruleId is required")
	}
	id, err := strconv.ParseUint(idS, 10, 64)
	if err != nil {
		return ctx.String(http.StatusInternalServerError, fmt.Sprintf("error parsing the ruleId : %v", err))
	}

	if err = h.db.DeleteRule(uint(id)); err != nil {
		return ctx.String(http.StatusInternalServerError, fmt.Sprintf("error deleting the rule : %v ", err))
	}

	return ctx.JSON(200, "Rule successfully deleted")
}

// UpdateRule godoc
//
//	@Summary		Update rule
//	@Description	Retrieving a rule by the specified input
//	@Security		BearerToken
//	@Tags			alerting
//	@Param			request	body		api.UpdateRuleRequest	true	"Request Body"
//	@Success		200		{object}	string
//	@Router			/alerting/api/v1/rule/update [put]
func (h *HttpHandler) UpdateRule(ctx echo.Context) error {
	var req api.UpdateRuleRequest
	if err := bindValidate(ctx, &req); err != nil {
		return ctx.String(http.StatusBadRequest, fmt.Sprintf("error getting the inputs : %v ", err))
	}
	if req.Id == 0 {
		return errors.New("ruleId is required")
	}

	var scope []byte
	var eventType []byte
	var operator []byte
	var err error

	if req.Scope != nil {
		scope, err = json.Marshal(req.Scope)
		if err != nil {
			return ctx.String(http.StatusInternalServerError, fmt.Sprintf("error marshalling the scope : %v ", err))
		}
	} else {
		scope = nil
	}

	if req.EventType != nil {
		eventType, err = json.Marshal(req.EventType)
		if err != nil {
			return ctx.String(http.StatusInternalServerError, fmt.Sprintf("error marshalling the eventType : %v ", err))
		}
	} else {
		eventType = nil
	}

	if req.Operator != nil {
		operator, err = json.Marshal(req.Operator)
		if err != nil {
			return ctx.String(http.StatusInternalServerError, fmt.Sprintf("error marshalling the operator : %v ", err))
		}
	} else {
		operator = nil
	}

	err = h.db.UpdateRule(req.Id, &eventType, &scope, &operator, req.ActionID)
	if err != nil {
		return ctx.String(http.StatusInternalServerError, fmt.Sprintf("error updating the rule : %v ", err))
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
//	@Success		200	{object}	[]api.Action
//	@Router			/alerting/api/v1/action/list [get]
func (h *HttpHandler) ListActions(ctx echo.Context) error {
	actions, err := h.db.ListAction()
	if err != nil {
		return ctx.String(http.StatusInternalServerError, fmt.Sprintf("error getting the actions : %v ", err))
	}

	var response []api.Action
	for _, action := range actions {

		var headers map[string]string
		err = json.Unmarshal(action.Headers, &headers)
		if err != nil {
			return ctx.String(http.StatusBadRequest, fmt.Sprintf("error unmarshalling the action : %v ", err))
		}

		response = append(response, api.Action{
			Id:      action.Id,
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
//	@Param			request	body		api.CreateActionReq	true	"Request Body"
//	@Success		200		{object}	string
//	@Router			/alerting/api/v1/action/create [post]
func (h *HttpHandler) CreateAction(ctx echo.Context) error {
	var req api.CreateActionReq
	err := bindValidate(ctx, &req)
	if err != nil {
		return ctx.String(http.StatusBadRequest, fmt.Sprintf("error getting the inputs : %v ", err))
	}

	testEmptyFields := api.CreateActionReq{}
	if req.Url == testEmptyFields.Url || req.Body == testEmptyFields.Body ||
		req.Method == testEmptyFields.Method || req.Headers == nil {
		return errors.New("All the fields in struct must be set")
	}

	headers, err := json.Marshal(req.Headers)
	if err != nil {
		return ctx.String(http.StatusInternalServerError, fmt.Sprintf("error marshalling the headers : %v ", err))
	}

	err = h.db.CreateAction(req.Method, req.Url, headers, req.Body)
	if err != nil {
		return ctx.String(http.StatusInternalServerError, fmt.Sprintf("error creating the action : %v ", err))
	}

	return ctx.JSON(200, "Action created successfully ")
}

// DeleteAction godoc
//
//	@Summary		Delete action
//	@Description	Deleting a single action for the given action id
//	@Security		BearerToken
//	@Tags			alerting
//	@Param			actionId	path		string	true	"actionId"
//	@Success		200			{object}	string
//	@Router			/alerting/api/v1/action/delete/{actionId} [delete]
func (h *HttpHandler) DeleteAction(ctx echo.Context) error {
	idS := ctx.Param("actionId")
	if idS == "" {
		return errors.New("actionId is required")
	}
	id, err := strconv.ParseUint(idS, 10, 64)
	if err != nil {
		return ctx.String(http.StatusInternalServerError, fmt.Sprintf("error parsing the actionId : %v", err))
	}

	err = h.db.DeleteAction(uint(id))
	if err != nil {
		return ctx.String(http.StatusInternalServerError, fmt.Sprintf("error deleting the action : %v ", err))
	}

	return ctx.JSON(200, "Action deleted successfully")
}

// UpdateAction godoc
//
//	@Summary		Update action
//	@Description	Retrieving an action by the specified input
//	@Security		BearerToken
//	@Tags			alerting
//	@Param			request	body		api.UpdateActionRequest	true	"Request Body"
//	@Success		200		{object}	string
//	@Router			/alerting/api/v1/action/update [put]
func (h *HttpHandler) UpdateAction(ctx echo.Context) error {
	var req api.UpdateActionRequest
	if err := bindValidate(ctx, &req); err != nil {
		return ctx.String(http.StatusBadRequest, fmt.Sprintf("error getting the inputs : %v ", err))
	}
	if req.Id == 0 {
		return errors.New("actionId is required")
	}

	var MarshalHeader []byte
	var err error
	if req.Headers != nil {
		MarshalHeader, err = json.Marshal(req.Headers)
		if err != nil {
			return ctx.String(http.StatusInternalServerError, fmt.Sprintf("error marshalling the headers : %v ", err))
		}
	} else {
		MarshalHeader = nil
	}

	err = h.db.UpdateAction(req.Id, &MarshalHeader, req.Url, req.Body, req.Method)
	if err != nil {
		return ctx.String(http.StatusInternalServerError, fmt.Sprintf("error updating the action : %v ", err))
	}
	return ctx.JSON(200, "Action updated successfully")
}
