package alerting

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	jira "github.com/andygrunwald/go-jira"
	"github.com/go-errors/errors"
	"github.com/kaytu-io/kaytu-engine/pkg/alerting/api"
	authapi "github.com/kaytu-io/kaytu-engine/pkg/auth/api"
	"github.com/kaytu-io/kaytu-engine/pkg/httpserver"
	"github.com/labstack/echo/v4"
	"net/http"
	"strconv"
	"time"
)

func (h *HttpHandler) Register(e *echo.Echo) {
	v1 := e.Group("/api/v1")

	ruleGroup := v1.Group("/rule")
	ruleGroup.GET("/list", httpserver.AuthorizeHandler(h.ListRules, authapi.ViewerRole))
	ruleGroup.POST("/create", httpserver.AuthorizeHandler(h.CreateRule, authapi.EditorRole))
	ruleGroup.DELETE("/delete/:ruleId", httpserver.AuthorizeHandler(h.DeleteRule, authapi.EditorRole))
	ruleGroup.PUT("/update/:ruleId", httpserver.AuthorizeHandler(h.UpdateRule, authapi.EditorRole))
	ruleGroup.GET("/:ruleId/trigger", httpserver.AuthorizeHandler(h.TriggerRuleAPI, authapi.EditorRole))

	actionGroup := v1.Group("/action")
	actionGroup.GET("/list", httpserver.AuthorizeHandler(h.ListActions, authapi.ViewerRole))
	actionGroup.POST("/create", httpserver.AuthorizeHandler(h.CreateAction, authapi.EditorRole))
	actionGroup.POST("/jira", httpserver.AuthorizeHandler(h.CreateJiraAction, authapi.EditorRole))
	actionGroup.POST("/slack", httpserver.AuthorizeHandler(h.CreateSlackAction, authapi.EditorRole))
	actionGroup.DELETE("/delete/:actionId", httpserver.AuthorizeHandler(h.DeleteAction, authapi.EditorRole))
	actionGroup.PUT("/update/:actionId", httpserver.AuthorizeHandler(h.UpdateAction, authapi.EditorRole))

	trigger := v1.Group("/trigger")
	trigger.GET("/list", httpserver.AuthorizeHandler(h.ListTriggers, authapi.ViewerRole))
	trigger.GET("/bydate", httpserver.AuthorizeHandler(h.CountTriggersByDate, authapi.ViewerRole))
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

// ListTriggers godoc
//
//	@Summary		List triggers
//	@Description	returns list of all the triggers
//	@Security		BearerToken
//	@Tags			alerting
//	@Produce		json
//	@Success		200	{object}	[]api.Triggers
//	@Router			/alerting/api/v1/trigger/list [get]
func (h *HttpHandler) ListTriggers(ctx echo.Context) error {
	listTriggers, err := h.db.ListTriggers()
	if err != nil {
		return ctx.String(http.StatusInternalServerError, fmt.Sprintf("error getting the list of the triggers : %v ", err))
	}
	var resListTrigger []api.Triggers
	for _, trigger := range listTriggers {
		rule, err := h.db.GetRule(trigger.RuleID)
		if err != nil {
			return err
		}

		action, err := h.db.GetAction(rule.ActionID)
		if err != nil {
			return err
		}

		var eventType api.EventType
		err = json.Unmarshal(rule.EventType, &eventType)
		if err != nil {
			return ctx.String(http.StatusBadRequest, fmt.Sprintf("error unmarshalling eventType : %v ", err))
		}

		var scope api.Scope
		err = json.Unmarshal(rule.Scope, &scope)
		if err != nil {
			return ctx.String(http.StatusBadRequest, fmt.Sprintf("error unmarshalling scope : %v ", err))
		}

		var condition api.Condition
		err = json.Unmarshal(rule.Operator, &condition)
		if err != nil {
			return ctx.String(http.StatusBadRequest, fmt.Sprintf("error unmarshalling operator : %v ", err))
		}

		var metadata api.Metadata
		err = json.Unmarshal(rule.Metadata, &metadata)
		if err != nil {
			return ctx.String(http.StatusBadRequest, fmt.Sprintf("error unmarshalling metadata : %v ", err))
		}

		apiRule := api.Rule{
			Id:            rule.Id,
			EventType:     eventType,
			Scope:         scope,
			Condition:     condition,
			Metadata:      metadata,
			TriggerStatus: api.TriggerStatus(rule.TriggerStatus),
			ActionID:      rule.ActionID,
		}

		var headers map[string]string
		if len(action.Headers) > 0 {
			err = json.Unmarshal(action.Headers, &headers)
			if err != nil {
				return ctx.String(http.StatusBadRequest, fmt.Sprintf("error unmarshalling the action : %v ", err))
			}
		}

		apiAction := api.Action{
			Id:      action.Id,
			Method:  action.Method,
			Url:     action.Url,
			Headers: headers,
			Body:    action.Body,
		}

		complianceT := api.Triggers{
			Rule:           apiRule,
			Action:         apiAction,
			TriggeredAt:    trigger.TriggeredAt,
			Value:          trigger.Value,
			ResponseStatus: trigger.ResponseStatus,
		}
		resListTrigger = append(resListTrigger, complianceT)
	}
	return ctx.JSON(http.StatusOK, resListTrigger)
}

func (h *HttpHandler) CountTriggersByDate(ctx echo.Context) error {
	startDate, err := strconv.ParseInt(ctx.QueryParam("startDate"), 10, 64)
	if err != nil {
		return err
	}
	endDate, err := strconv.ParseInt(ctx.QueryParam("endDate"), 10, 64)
	if err != nil {
		return err
	}

	count, err := h.db.CountTriggersJobsByDate(time.UnixMilli(startDate), time.UnixMilli(endDate))
	if err != nil {
		return ctx.String(http.StatusInternalServerError, fmt.Sprintf("error getting the list of the triggers : %v ", err))
	}
	return ctx.JSON(http.StatusOK, count)
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

		var condition api.Condition
		err = json.Unmarshal(rule.Operator, &condition)
		if err != nil {
			return ctx.String(http.StatusBadRequest, fmt.Sprintf("error unmarshalling operator : %v ", err))
		}

		var metadata api.Metadata
		err = json.Unmarshal(rule.Metadata, &metadata)
		if err != nil {
			return ctx.String(http.StatusBadRequest, fmt.Sprintf("error unmarshalling metadata : %v ", err))
		}

		response = append(response, api.Rule{
			Id:            rule.Id,
			EventType:     eventType,
			Scope:         scope,
			Condition:     condition,
			Metadata:      metadata,
			TriggerStatus: api.TriggerStatus(rule.TriggerStatus),
			ActionID:      rule.ActionID,
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
//	@Success		200		{object}	uint
//	@Router			/alerting/api/v1/rule/create [post]
func (h *HttpHandler) CreateRule(ctx echo.Context) error {
	var req api.CreateRuleRequest
	if err := bindValidate(ctx, &req); err != nil {
		return ctx.String(http.StatusBadRequest, fmt.Sprintf("error getting the inputs : %v ", err))
	}

	//EmptyFields := api.CreateRuleRequest{}
	//if req.ActionID == EmptyFields.ActionID || req.Operator == EmptyFields.Operator || req.EventType == EmptyFields.EventType || req.Metadata.Name == req.Metadata.Name {
	//	return errors.New("All the fields in struct must be set")
	//}

	scope, err := json.Marshal(req.Scope)
	if err != nil {
		return ctx.String(http.StatusBadRequest, fmt.Sprintf("error marshalling scope : %v ", err))
	}

	event, err := json.Marshal(req.EventType)
	if err != nil {
		return ctx.String(http.StatusBadRequest, fmt.Sprintf("error marshalling eventType : %v ", err))
	}

	operator, err := json.Marshal(req.Condition)
	if err != nil {
		return ctx.String(http.StatusBadRequest, fmt.Sprintf("error marshalling operator : %v ", err))
	}

	metadata, err := json.Marshal(req.Metadata)
	if err != nil {
		return ctx.String(http.StatusBadRequest, fmt.Sprintf("error marshalling metadata : %v ", err))
	}

	ruleID, err := h.db.CreateRule(event, scope, operator, req.ActionID, metadata)
	if err != nil {
		return ctx.String(http.StatusInternalServerError, fmt.Sprintf("error creating rule : %v ", err))
	}

	return ctx.JSON(200, ruleID)
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
//	@Param			ruleId	path		string					true	"ruleId"
//	@Param			request	body		api.UpdateRuleRequest	false	"Request Body"
//	@Success		200		{object}	string
//	@Router			/alerting/api/v1/rule/update/{ruleId} [put]
func (h *HttpHandler) UpdateRule(ctx echo.Context) error {
	idString := ctx.Param("ruleId")
	if idString == "" {
		return errors.New("ruleId is required")
	}
	id, err := strconv.ParseUint(idString, 10, 64)
	if err != nil {
		return ctx.String(http.StatusInternalServerError, fmt.Sprintf("error parsing the ruleId : %v", err))
	}

	var req api.UpdateRuleRequest
	if err := bindValidate(ctx, &req); err != nil {
		return ctx.String(http.StatusBadRequest, fmt.Sprintf("error getting the inputs : %v ", err))
	}

	var scope []byte
	var eventType []byte
	var operator []byte
	var metadata []byte

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

	if req.Condition != nil {
		operator, err = json.Marshal(req.Condition)
		if err != nil {
			return ctx.String(http.StatusInternalServerError, fmt.Sprintf("error marshalling the operator : %v ", err))
		}
	} else {
		operator = nil
	}

	if req.Metadata != nil {
		metadata, err = json.Marshal(req.Metadata)
		if err != nil {
			return ctx.String(http.StatusInternalServerError, fmt.Sprintf("error marshalling the metadata : %v ", err))
		}
	} else {
		metadata = nil
	}

	err = h.db.UpdateRule(uint(id), &eventType, &scope, &metadata, &operator, req.ActionID, api.TriggerStatus_NotActive)
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
		if len(action.Headers) > 0 {
			err = json.Unmarshal(action.Headers, &headers)
			if err != nil {
				return ctx.String(http.StatusBadRequest, fmt.Sprintf("error unmarshalling the action : %v ", err))
			}
		}

		response = append(response, api.Action{
			Id:      action.Id,
			Name:    action.Name,
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
//	@Success		200		{object}	uint
//	@Router			/alerting/api/v1/action/create [post]
func (h *HttpHandler) CreateAction(ctx echo.Context) error {
	var req api.CreateActionReq
	err := bindValidate(ctx, &req)
	if err != nil {
		return ctx.String(http.StatusBadRequest, fmt.Sprintf("error getting the inputs : %v ", err))
	}

	testEmptyFields := api.CreateActionReq{}
	if req.Name == "" || req.Url == testEmptyFields.Url || req.Method == testEmptyFields.Method {
		return errors.New("All the fields in struct must be set")
	}

	headers, err := json.Marshal(req.Headers)
	if err != nil {
		return ctx.String(http.StatusInternalServerError, fmt.Sprintf("error marshalling the headers : %v ", err))
	}

	id, err := h.db.CreateAction(req.Name, req.Method, req.Url, headers, req.Body, ActionType_Webhook)
	if err != nil {
		return ctx.String(http.StatusInternalServerError, fmt.Sprintf("error creating the action : %v ", err))
	}

	return ctx.JSON(200, id)
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
//	@Param			actionId	path		string					true	"actionId"
//	@Param			request		body		api.UpdateActionRequest	false	"Request Body"
//	@Success		200			{object}	string
//	@Router			/alerting/api/v1/action/update/{actionId} [put]
func (h *HttpHandler) UpdateAction(ctx echo.Context) error {
	idString := ctx.Param("actionId")
	if idString == "" {
		return errors.New("actionId is required")
	}
	id, err := strconv.ParseUint(idString, 10, 64)
	if err != nil {
		return ctx.String(http.StatusInternalServerError, fmt.Sprintf("error parsing the actionId : %v", err))
	}

	var req api.UpdateActionRequest
	if err := bindValidate(ctx, &req); err != nil {
		return ctx.String(http.StatusBadRequest, fmt.Sprintf("error getting the inputs : %v ", err))
	}

	var MarshalHeader []byte
	if req.Headers != nil {
		MarshalHeader, err = json.Marshal(req.Headers)
		if err != nil {
			return ctx.String(http.StatusInternalServerError, fmt.Sprintf("error marshalling the headers : %v ", err))
		}
	} else {
		MarshalHeader = nil
	}

	err = h.db.UpdateAction(uint(id), req.Name, &MarshalHeader, req.Url, req.Body, req.Method)
	if err != nil {
		return ctx.String(http.StatusInternalServerError, fmt.Sprintf("error updating the action : %v ", err))
	}

	return ctx.JSON(200, "Action updated successfully")
}

// CreateSlackAction godoc
//
//	@Summary		Create Slack Action
//	@Description	Create action with slack url and body
//	@Security		BearerToken
//	@Tags			alerting
//	@Param			request	body		api.SlackInputs	true	"Request Body"
//	@Success		200		{object}	api.JiraAndStackResponse
//	@Router			/alerting/api/v1/action/slack [post]
func (h *HttpHandler) CreateSlackAction(ctx echo.Context) error {
	var inputs api.SlackInputs
	if err := bindValidate(ctx, &inputs); err != nil {
		return ctx.String(http.StatusBadRequest, fmt.Sprintf("error getting the channelName : %v ", err))
	}

	reqStr := api.SlackRequest{
		ChannelName: inputs.ChannelName,
		Text:        "{{.Name}} alert is failing",
	}

	reqStrMarshalled, err := json.Marshal(&reqStr)
	if err != nil {
		return fmt.Errorf("error in marshalling the request : %v", err)
	}
	id, err := h.db.CreateAction(inputs.Name, "POST", inputs.SlackUrl, nil, string(reqStrMarshalled), ActionType_Slack)
	if err != nil {
		return fmt.Errorf("error creating action : %v ", err)
	}

	res := api.JiraAndStackResponse{
		ActionId: id,
	}
	return ctx.JSON(http.StatusOK, res)
}

// CreateJiraAction godoc
//
//	@Summary		Create Jira Action
//	@Description	Create action with jira url and header and body
//	@Security		BearerToken
//	@Tags			alerting
//	@Param			request	body		api.JiraInputs	true	"Request Body"
//	@Success		200		{object}	api.JiraAndStackResponse
//	@Router			/alerting/api/v1/action/jira [post]
func (h *HttpHandler) CreateJiraAction(ctx echo.Context) error {
	var inputs api.JiraInputs
	if err := bindValidate(ctx, &inputs); err != nil {
		return ctx.JSON(http.StatusBadRequest, fmt.Sprintf("error getting the channelName : %v ", err))
	}

	requestBody := jira.Issue{
		Fields: &jira.IssueFields{
			Type:    jira.IssueType{ID: inputs.IssueTypeId},
			Project: jira.Project{ID: inputs.ProjectId},
			Summary: "{{.Name}} rule triggered successfully",
			Duedate: jira.Date(time.Now()),
		},
	}

	auth := fmt.Sprintf("%s:%s", inputs.Email, inputs.AtlassianApiToken)
	authEncoded := base64.StdEncoding.EncodeToString([]byte(auth))
	headers := map[string]string{
		"Content-Type":  "application/json",
		"Authorization": "Basic " + authEncoded,
	}

	requestMarshalled, err := json.Marshal(requestBody)
	if err != nil {
		return err
	}

	headerMarshalled, err := json.Marshal(headers)
	if err != nil {
		return err
	}

	id, err := h.db.CreateAction(inputs.Name, "POST", fmt.Sprintf("https://%s/rest/api/3/issue", inputs.AtlassianDomain), headerMarshalled, string(requestMarshalled), ActionType_Jira)
	if err != nil {
		return err
	}

	res := api.JiraAndStackResponse{
		ActionId: id,
	}
	return ctx.JSON(http.StatusOK, res)
}
