package alerting

import (
	"bytes"
	"encoding/json"
	"github.com/kaytu-io/kaytu-engine/pkg/internal/httpserver"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

type HttpHandlerTest struct {
	suite.Suite

	handler *HttpHandler
	router  *echo.Echo
	orm     *gorm.DB
}

type Operator = string

const (
	Operator_GreaterThan Operator = ">"
	Operator_LessThan    Operator = "<"
)

type EventType struct {
	InsightId int64
}

type Scope struct {
	ConnectionId string
}

type rule struct {
	ID        uint
	EventType json.RawMessage
	Scope     json.RawMessage
	Operator  string
	Value     int64
	ActionID  uint
}

func (h *HttpHandlerTest) setup() {
	required := h.Require()
	logger, err := zap.NewProduction()
	if err != nil {
		required.NoError(err, "new logger : ")
	}

	handler, err := InitializeHttpHandler(
		"localhost",
		"5432",
		"test-database",
		"user_1",
		"qwertyPostgres",
		"verify-full",
		logger,
	)
	if err != nil {
		required.NoError(err, "init http handler: : ")
	}
	h.handler = handler
	//address := "postgresql://user_1:qwertyPostgres@127.0.0.1:5432/test-database"

	err = httpserver.RegisterAndStart(logger, "http", handler)
	if err != nil {
		required.NoError(err, "error in register and start:")
	}
}

func (h *HttpHandlerTest) ensureTableExists() {
	//h.handler.db.orm.Exec(tableCreationActions)
	//h.handler.db.orm.Exec(tableCreationRule)
}

func (h *HttpHandlerTest) clearTable() {
	//h.handler.db.orm.Exec("DELETE FROM rule")

	//h.handler.db.orm.Exec("DELETE FROM actions")
}

//const tableCreationActions = `CREATE TABLE IF NOT EXISTS actions
//(
//   id INT ,
//   method TEXT,
//   url TEXT,
//   headers JSON,
//   body JSON
//)`

//const tableCreationRule = `CREATE TABLE IF NOT EXISTS rule
//(
//   id INT ,
//	event_type JSON,
//	scope JSON ,
//	operator CHAR ,
//	value INT,
//	actionID INT
//)`

func (h *HttpHandlerTest) AfterTest(suiteName, testName string) {
}

func TestHttpHandlerSuite(t *testing.T) {
	suite.Run(t, &HttpHandlerTest{})
}

func doSimpleJSONRequest(method string, path string, request, response interface{}) (*http.Response, error) {
	var r io.Reader
	if request != nil {
		out, err := json.Marshal(request)
		if err != nil {
			return nil, err
		}

		r = bytes.NewReader(out)
	}

	req := httptest.NewRequest(method, path, r)
	req.Header.Add("Content-Type", "application/json")
	//req.Header.Add(httpserver.XKaytuUserRoleHeader, string(api2.AdminRole))

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	if response != nil {
		// Wrap in NopCloser in case the calling method wants to also read the body
		b, err := io.ReadAll(res.Body)
		if err != nil {
			return nil, err
		}

		if err := json.Unmarshal(b, response); err != nil {
			return nil, err
		}
	}
	return res, nil
}

func (h *HttpHandlerTest) TestEmptyListRule() {
	require := h.Require()
	//h.handler.db.orm.Exec("DELETE FROM rule")

	var rules []Rule
	res, err := doSimpleJSONRequest("GET", "/api/rule/list", nil, &rules)
	if err == nil {
		require.NoError(nil, "Expect to give error because it is empty but it got : ", res)
	}
}

func (h *HttpHandlerTest) TestCreateRule() {
	require := h.Require()
	eventTypeReq := EventType{InsightId: 123123}
	eventTypeM, err := json.Marshal(eventTypeReq)
	if err != nil {
		require.Errorf(err, "error in marshaling the event type")
	}

	scopeReq := Scope{ConnectionId: "testConnectionId"}
	scopeM, err := json.Marshal(scopeReq)
	if err != nil {
		require.Errorf(err, "error in marshaling the scope")
	}

	req := rule{
		ID:        123,
		EventType: eventTypeM,
		Scope:     scopeM,
		Operator:  Operator_GreaterThan,
		Value:     100,
		ActionID:  1231,
	}
	var responseCreate string
	res, err := doSimpleJSONRequest("POST", "/api/rule/create", req, &responseCreate)
	if err != nil {
		require.Errorf(nil, "error in create rule ", res)
	}
	var ruleFind rule
	response, err := doSimpleJSONRequest("GET", "/api/rule/get/123", nil, &ruleFind)
	if err != nil {
		require.Errorf(err, "error in get rule")
	}
	require.Equal(http.StatusOK, response.StatusCode)

	if ruleFind.Operator != Operator_GreaterThan {
		require.Errorf(nil, "Expected rule operator to be '>' . Got  :", ruleFind.Operator)
	}
	if ruleFind.Value != 100 {
		require.Errorf(nil, "Expected rule value to be '100' . Got :", req.Value)
	}
	var scope Scope
	err = json.Unmarshal(ruleFind.Scope, &scope)
	if err != nil {
		require.Errorf(nil, "error in unmarshaling the scope")
	}
	if scope.ConnectionId != "testConnectionId" {
		require.Errorf(nil, "Expected rule scope to be 'testConnectionId' . Got ", scope.ConnectionId)
	}

	var eventType EventType
	err = json.Unmarshal(ruleFind.EventType, &eventType)
	if err != nil {
		require.Errorf(nil, "error in unmarshaling the event type")
	}
	if eventType.InsightId != 123123 {
		require.Errorf(nil, "Expected rule event type to be '123123' . Got '%v'", eventType.InsightId)
	}
	if ruleFind.ActionID != 1231 {
		require.Errorf(nil, "Expected rule actionID to be '1231' . Got '%d'", ruleFind.ActionID)
	}
}

func (h *HttpHandlerTest) addUsers() {
	require := h.Require()
	eventType := EventType{
		InsightId: 1231,
	}
	eventTypeM, _ := json.Marshal(eventType)

	scope := Scope{
		ConnectionId: "testConnectionID",
	}
	scopeM, _ := json.Marshal(scope)

	req := rule{
		ID:        12,
		EventType: eventTypeM,
		Scope:     scopeM,
		Operator:  Operator_GreaterThan,
		Value:     1000,
		ActionID:  123123,
	}
	res, err := doSimpleJSONRequest("POST", "/api/rule/create", req, nil)
	if err != nil {
		require.Errorf(nil, "error in create rule ", res)
	}
}

func (h *HttpHandlerTest) TestUpdateRule() {
	require := h.Require()

	req := rule{
		ID:       12,
		Value:    110,
		Operator: Operator_LessThan,
		ActionID: 34567,
	}

	_, err := doSimpleJSONRequest("GET", "/api/rule/update", req, nil)
	if err != nil {
		require.Errorf(err, "error in update rule ")
	}

	var ruleNew rule
	_, err = doSimpleJSONRequest("GET", "/api/rule/get/12", nil, &ruleNew)
	if err != nil {
		require.Errorf(err, "error in get rule")
	}

	if ruleNew.Value != 110 {
		require.Errorf(nil, "Expect from rule value to be '110' , Got : %d ", ruleNew.Value)
	}
	if ruleNew.ActionID != 34567 {
		require.Errorf(nil, "Expect from rule actionID to be '34567' , Got : %d ", ruleNew.ActionID)
	}
	if ruleNew.Operator != Operator_LessThan {
		require.Errorf(nil, "Expect from rule operator to be '<' , Got : %s ", ruleNew.Operator)
	}
}

func (h *HttpHandlerTest) TestDeleteRule() {
	require := h.Require()
	//h.handler.db.orm.Exec("DELETE FROM rule")
	//h.addUsers()

	_, err := doSimpleJSONRequest("DELETE", "/api/rule/delete/12", nil, nil)
	if err != nil {
		require.Errorf(err, "error in deleting the rule")
	}

	responseGet, err := doSimpleJSONRequest("GET", "/api/rule/get/12", nil, nil)
	if err == nil {
		require.Errorf(nil, "Expect from get rule to give err because it was deleted  , Got : %v ", responseGet)
	}
}

// -------------------------------------------------- action test --------------------------------------------------
func (h *HttpHandlerTest) TestListAction() {
	require := h.Require()
	var action Action
	res, err := doSimpleJSONRequest("GET", "/api/action/list", nil, &action)
	if err == nil {
		require.NoError(nil, "Expect to give an error because the list of the actions is empty but it get : %s", res)
	}
}

func (h *HttpHandlerTest) TestCreateAction() {
	require := h.Require()
	header := map[string]string{"insightId": "123123"}
	headerM, err := json.Marshal(header)
	if err != nil {
		require.NoError(err, "error in marshaling the headers ")
	}
	action := Action{
		ID:      12,
		Method:  "GET",
		Url:     "https://kaytu.dev/company",
		Headers: headerM,
		Body:    "",
	}

	resC, err := doSimpleJSONRequest("POST", "/api/action/create", action, nil)
	if err != nil {
		require.NoError(err, "error in create the row ")
	}
	require.Equal(resC.StatusCode, http.StatusOK)
	var actionG Action
	resG, err := doSimpleJSONRequest("GET", "api/action/get/12", nil, &actionG)
	if err != nil {
		require.Errorf(err, "error in get action ")
	}
	require.Equal(resG.StatusCode, http.StatusOK)

	if actionG.Url != "https://kaytu.dev/company" {
		require.Errorf(err, "Expect the url action to be 'https://kaytu.dev/company', got : ", action.Url)
	}
	if actionG.Body != "" {
		require.Errorf(err, "Expect the body action to be '', got : ", action.Body)
	}
	if actionG.Method != "GET" {
		require.Errorf(err, "Expect the Method action to be 'GET', got : ", action.Method)
	}
	require.Equal(actionG.Headers, headerM)
}

func (h *HttpHandlerTest) addUsersForAction() {
	require := h.Require()
	header := map[string]string{"insight": "teatInsight"}

	headerM, err := json.Marshal(header)
	if err != nil {
		require.NoError(err, "error in marshaling the header")
	}

	req := Action{
		ID:      12,
		Method:  "GET",
		Url:     "https://kaytu.dev/",
		Headers: headerM,
		Body:    "",
	}
	res, err := doSimpleJSONRequest("POST", "/api/action/create", req, nil)
	if err != nil {
		require.Errorf(nil, "error in create action ", res)
	}
}

func (h *HttpHandlerTest) TestUpdateAction() {
	require := h.Require()

	header := map[string]string{"insightId": "newTestInsight"}
	headerM, err := json.Marshal(header)
	if err != nil {
		require.NoError(err, "error in marshaling ")
	}

	req := Action{
		ID:      12,
		Method:  "POST",
		Headers: headerM,
		Url:     "https://kaytu.dev/use-cases",
	}

	resU, err := doSimpleJSONRequest("UPDATE", "/api/action/Update", req, nil)
	if err != nil {
		require.Errorf(err, "error in update the action")
	}
	require.Equal(resU.StatusCode, http.StatusOK)

	var actionG Action
	resG, err := doSimpleJSONRequest("GET", "/api/action/get/12", nil, actionG)
	if err != nil {
		require.Errorf(err, "error in get the action")
	}
	require.Equal(resG.StatusCode, http.StatusOK)
	require.NotEqual(actionG.Headers, headerM)
	if actionG.Method != "POST" {
		require.NoError(nil, "Expect to be the action method 'POST' , but it got : ", actionG.Method)
	}
	if actionG.Url != "https://kaytu.dev/use-cases" {
		require.NoError(nil, "Expect to be the action url 'https://kaytu.dev/use-cases' , but it got : ", actionG.Url)
	}
}

func (h *HttpHandlerTest) TestDeleteAction() {
	require := h.Require()

	response, err := doSimpleJSONRequest("DELETE", "/api/action/delete/12", nil, nil)
	if err != nil {
		require.Errorf(err, "error in deleting the action")
	}
	require.Equal(http.StatusOK, response.StatusCode)

	responseGet, err := doSimpleJSONRequest("GET", "/api/action/get/12", nil, nil)
	if err == nil {
		require.Errorf(nil, "Expect from get action to give err because it was deleted  , Got : %v ", responseGet)
	}
}
