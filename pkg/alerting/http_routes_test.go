package alerting

import (
	"bytes"
	"encoding/json"
	"github.com/kaytu-io/kaytu-engine/pkg/internal/httpserver"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"io"
	"net/http"
	"testing"
)

//type HttpHandlerTest struct {
//	suite.Suite
//
//	handler *HttpHandler
//	router  *echo.Echo
//	orm     *gorm.DB
//}

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

func TestSetup(t *testing.T) {
	logger, err := zap.NewProduction()
	if err != nil {
		t.Errorf("new logger : %v", err)
	}

	dsn := "host=localhost user=user_1 password=qwertyPostgres dbname=test-database port=5432 sslmode=disable"
	orm, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Errorf("error in connecting to postgres , err : %v", err)
	}
	db, err := orm.DB()
	if err != nil {
		t.Errorf("raw db : %v", err)
	}
	if err := db.Ping(); err != nil {
		t.Errorf("ping db: %w", err)
	}

	testEmptyListRule(t)
	testCreateRule(t)
	testUpdateRule(t)
	testDeleteRule(t)

	handler := HttpHandler{db: Database{orm: orm}}
	err = httpserver.RegisterAndStart(logger, "http://localhost:8081", &handler)
	if err != nil {
		t.Errorf("error in register and start : %v", err)
	}
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

func doSimpleJSONRequest(router *echo.Echo, method string, path string, request, response interface{}) (*http.Response, error) {
	var r io.Reader
	if request != nil {
		out, err := json.Marshal(request)
		if err != nil {
			return nil, err
		}

		r = bytes.NewReader(out)
	}

	req, err := http.NewRequest(method, path, r)
	if err != nil {
		return nil, err
	}
	if response != nil {
		req.Header.Add("Content-Type", "application/json")
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	if response != nil {
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

func testEmptyListRule(t *testing.T) {
	var rules []Rule
	var router *echo.Echo
	res, err := doSimpleJSONRequest(router, "GET", "/api/rule/list", nil, &rules)
	if err == nil {
		t.Errorf("Expect to give error because it is empty but it got : %v", res)
	}
}

func testCreateRule(t *testing.T) {
	var router *echo.Echo
	eventTypeReq := EventType{InsightId: 123123}
	eventTypeM, err := json.Marshal(eventTypeReq)
	if err != nil {
		t.Errorf("error in marshaling the event type , err : %v", err)
	}

	scopeReq := Scope{ConnectionId: "testConnectionId"}
	scopeM, err := json.Marshal(scopeReq)
	if err != nil {
		t.Errorf("error in marshaling the scope , err : %v ", err)
	}

	req := rule{
		ID:        123,
		EventType: eventTypeM,
		Scope:     scopeM,
		Operator:  Operator_GreaterThan,
		Value:     100,
		ActionID:  1231,
	}
	_, err = doSimpleJSONRequest(router, "POST", "http://localhost:8081/api/rule/create", req, nil)
	if err != nil {
		t.Errorf("error in create rule , err : %v", err)
	}

	var ruleFind rule
	_, err = doSimpleJSONRequest(router, "GET", "http://localhost:8081/api/rule/get/123", nil, &ruleFind)
	if err != nil {
		t.Errorf("error in get rule , err : %v", err)
	}

	if ruleFind.Operator != Operator_GreaterThan {
		t.Errorf("Expected rule operator to be '>' . Got  : %v ", ruleFind.Operator)
	}
	if ruleFind.Value != 100 {
		t.Errorf("Expected rule value to be '100' . Got : %v ", req.Value)
	}

	var scope Scope
	err = json.Unmarshal(ruleFind.Scope, &scope)
	if err != nil {
		t.Errorf("error in unmarshaling the scope , err : %v", err)
	}
	if scope.ConnectionId != "testConnectionId" {
		t.Errorf("Expected rule scope to be 'testConnectionId' . Got : %v", scope.ConnectionId)
	}

	var eventType EventType
	err = json.Unmarshal(ruleFind.EventType, &eventType)
	if err != nil {
		t.Errorf("error in unmarshaling the event type , err : %v", err)
	}
	if eventType.InsightId != 123123 {
		t.Errorf("Expected rule event type to be '123123' , Got %v", eventType.InsightId)
	}
	if ruleFind.ActionID != 1231 {
		t.Errorf("Expected rule actionID to be '1231' . Got %d ", ruleFind.ActionID)
	}
}

func addUsers(t *testing.T) {
	var router *echo.Echo
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
	_, err := doSimpleJSONRequest(router, "POST", "/api/rule/create", req, nil)
	if err != nil {
		t.Errorf("error in create rule : %v", err)
	}
}

func testUpdateRule(t *testing.T) {
	addUsers(t)
	req := rule{
		ID:       12,
		Value:    110,
		Operator: Operator_LessThan,
		ActionID: 34567,
	}
	var router *echo.Echo
	_, err := doSimpleJSONRequest(router, "GET", "/api/rule/update", req, nil)
	if err != nil {
		t.Errorf("error in update rule : %v", err)
	}

	var ruleNew rule
	_, err = doSimpleJSONRequest(router, "GET", "/api/rule/get/12", nil, &ruleNew)
	if err != nil {
		t.Errorf("error in get rule : %v", err)
	}

	if ruleNew.Value != 110 {
		t.Errorf("Expect from rule value to be '110' , Got : %d ", ruleNew.Value)
	}
	if ruleNew.ActionID != 34567 {
		t.Errorf("Expect from rule actionID to be '34567' , Got : %d ", ruleNew.ActionID)
	}
	if ruleNew.Operator != Operator_LessThan {
		t.Errorf("Expect from rule operator to be '<' , Got : %s ", ruleNew.Operator)
	}
}

func testDeleteRule(t *testing.T) {
	var router *echo.Echo
	_, err := doSimpleJSONRequest(router, "DELETE", "/api/rule/delete/12", nil, nil)
	if err != nil {
		t.Errorf("error in deleting the rule : %v", err)
	}

	responseGet, err := doSimpleJSONRequest(router, "GET", "/api/rule/get/12", nil, nil)
	if err == nil {
		t.Errorf("Expect from get rule to give err because it was deleted  , Got : %v ", responseGet)
	}
}

// -------------------------------------------------- action test --------------------------------------------------
//func (h *HttpHandlerTest) TestListAction() {
//	require := h.Require()
//	var action Action
//	res, err := doSimpleJSONRequest(router, "GET", "/api/action/list", nil, &action)
//	if err == nil {
//		require.NoError(nil, "Expect to give an error because the list of the actions is empty but it get : %s", res)
//	}
//}
//
//func (h *HttpHandlerTest) TestCreateAction() {
//	require := h.Require()
//	header := map[string]string{"insightId": "123123"}
//	headerM, err := json.Marshal(header)
//	if err != nil {
//		require.NoError(err, "error in marshaling the headers ")
//	}
//	action := Action{
//		ID:      12,
//		Method:  "GET",
//		Url:     "https://kaytu.dev/company",
//		Headers: headerM,
//		Body:    "",
//	}
//
//	resC, err := doSimpleJSONRequest(router, "POST", "/api/action/create", action, nil)
//	if err != nil {
//		require.NoError(err, "error in create the row ")
//	}
//	require.Equal(resC.StatusCode, http.StatusOK)
//	var actionG Action
//	resG, err := doSimpleJSONRequest(router, "GET", "api/action/get/12", nil, &actionG)
//	if err != nil {
//		require.Errorf(err, "error in get action ")
//	}
//	require.Equal(resG.StatusCode, http.StatusOK)
//
//	if actionG.Url != "https://kaytu.dev/company" {
//		require.Errorf(err, "Expect the url action to be 'https://kaytu.dev/company', got : ", action.Url)
//	}
//	if actionG.Body != "" {
//		require.Errorf(err, "Expect the body action to be '', got : ", action.Body)
//	}
//	if actionG.Method != "GET" {
//		require.Errorf(err, "Expect the Method action to be 'GET', got : ", action.Method)
//	}
//	require.Equal(actionG.Headers, headerM)
//}
//
//func (h *HttpHandlerTest) addUsersForAction() {
//	require := h.Require()
//	header := map[string]string{"insight": "teatInsight"}
//
//	headerM, err := json.Marshal(header)
//	if err != nil {
//		require.NoError(err, "error in marshaling the header")
//	}
//
//	req := Action{
//		ID:      12,
//		Method:  "GET",
//		Url:     "https://kaytu.dev/",
//		Headers: headerM,
//		Body:    "",
//	}
//	res, err := doSimpleJSONRequest(router, "POST", "/api/action/create", req, nil)
//	if err != nil {
//		require.Errorf(nil, "error in create action ", res)
//	}
//}
//
//func (h *HttpHandlerTest) TestUpdateAction() {
//	require := h.Require()
//
//	header := map[string]string{"insightId": "newTestInsight"}
//	headerM, err := json.Marshal(header)
//	if err != nil {
//		require.NoError(err, "error in marshaling ")
//	}
//
//	req := Action{
//		ID:      12,
//		Method:  "POST",
//		Headers: headerM,
//		Url:     "https://kaytu.dev/use-cases",
//	}
//
//	_, err = doSimpleJSONRequest(router, "UPDATE", "/api/action/Update", req, nil)
//	if err != nil {
//		require.Errorf(err, "error in update the action")
//	}
//
//	var actionG Action
//	resG, err := doSimpleJSONRequest(router, "GET", "/api/action/get/12", nil, actionG)
//	if err != nil {
//		require.Errorf(err, "error in get the action")
//	}
//	require.Equal(resG.StatusCode, http.StatusOK)
//	require.NotEqual(actionG.Headers, headerM)
//	if actionG.Method != "POST" {
//		require.NoError(nil, "Expect to be the action method 'POST' , but it got : ", actionG.Method)
//	}
//	if actionG.Url != "https://kaytu.dev/use-cases" {
//		require.NoError(nil, "Expect to be the action url 'https://kaytu.dev/use-cases' , but it got : ", actionG.Url)
//	}
//}
//
//func (h *HttpHandlerTest) TestDeleteAction() {
//	require := h.Require()
//
//	_, err := doSimpleJSONRequest(router, "DELETE", "/api/action/delete/12", nil, nil)
//	if err != nil {
//		require.Errorf(err, "error in deleting the action")
//	}
//
//	responseGet, err := doSimpleJSONRequest(router, "GET", "/api/action/get/12", nil, nil)
//	if err == nil {
//		require.Errorf(nil, "Expect from get action to give err because it was deleted  , Got : %v ", responseGet)
//	}
//}
