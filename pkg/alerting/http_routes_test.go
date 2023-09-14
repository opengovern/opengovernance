package alerting

import (
	"encoding/json"
	"fmt"
	"github.com/kaytu-io/kaytu-util/pkg/postgres"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
	"gorm.io/gorm"
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
	require := h.Require()

	cfg := postgres.Config{
		Host:    "localhost",
		Port:    "5432",
		User:    "user_1",
		Passwd:  "qwertyPostgres",
		DB:      "test-database",
		SSLMode: "verify-full",
	}

	logger, err := zap.NewProduction()
	require.NoError(err, "new zap logger")

	h.orm, err = postgres.NewClient(&cfg, logger)
	if err != nil {
		require.NoError(err, "new postgres client")
	}
	fmt.Println("Connected to the postgres database: ", "test-database")

	d, err := h.orm.DB()
	if err != nil {
		require.NoError(err)
	}

	err = d.Ping()
	if err != nil {
		require.NoError(err)
	}
}

func (h *HttpHandlerTest) ensureTableExists() {
	//h.handler.db.orm.Exec(tableCreationActions)
	h.handler.db.orm.Exec(tableCreationRule)
}

func (h *HttpHandlerTest) clearTable() {
	h.handler.db.orm.Exec("DELETE FROM rule")
	h.handler.db.orm.Exec("ALTER TABLE rule AUTO_INCREMENT = 1")

	//h.handler.db.orm.Exec("DELETE FROM actions")
	//h.handler.db.orm.Exec("ALTER TABLE action AUTO_INCREMENT = 1")
}

//const tableCreationActions = `CREATE TABLE IF NOT EXISTS actions
//(
//    id INT ,
//    method TEXT,
//    url TEXT,
//    headers JSON,
//    body JSON
//)`

const tableCreationRule = `CREATE TABLE IF NOT EXISTS rule
(
    id INT ,
	event_type JSON,
	scope JSON ,
	operator CHAR ,
	value INT,
	actionID INT
)`

func (h *HttpHandlerTest) BeforeTest(suiteName, testName string) {
	require := h.Require()

	err := h.handler.db.Initialize()
	if err != nil {
		require.NoError(err)
	}
	fmt.Println("Initialized postgres database: ", "test-database")
	h.ensureTableExists()
}

func (h *HttpHandlerTest) AfterTest(suiteName, testName string) {
	h.clearTable()
}

func TestHttpHandlerSuite(t *testing.T) {
	suite.Run(t, &HttpHandlerTest{})
}

func (h *HttpHandlerTest) TestEmptyListRule() {
	require := h.Require()
	h.clearTable()
	var ctx echo.Context
	res := h.handler.ListRules(ctx)
	if res != nil {
		if res.Error() == "" {
			require.Errorf(res, "Expected an empty array. but i got it :")
		}
	}
	require.NoError(res)
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

	var ctx echo.Context
	ctx.Set("request", req)
	res := h.handler.CreateRule(ctx)
	if res.Error() != "" {
		require.Errorf(res, "error in create rule ")
	}

	ruleFind, err := h.handler.db.GetRule(req.ID)
	if err != nil {
		require.Errorf(nil, "error in find rule \n the create func don't work completely ")
	}

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
	err := h.handler.db.CreateRule(12, eventTypeM, scopeM, Operator_GreaterThan, 1000, 123123)
	if err != nil {
		require.Errorf(nil, "error in run create rule ")
	}
}

func (h *HttpHandlerTest) TestUpdateRule() {
	require := h.Require()
	h.addUsers()

	req := rule{
		ID:       12,
		Value:    110,
		Operator: Operator_LessThan,
		ActionID: 34567,
	}
	var ctx echo.Context
	ctx.Set("request", req)
	err := h.handler.UpdateRule(ctx)
	if err != nil {
		require.Errorf(err, "error in update rule , error : %s ")
	}

	res, err := h.handler.db.GetRule(req.ID)
	if err != nil {
		require.Errorf(err, "error in giving data from database , error : ")
	}
	if res.Value != 110 {
		require.Errorf(nil, "Expect from rule value to be '110' , Got : %d ", res.Value)
	}
	if res.ActionID != 34567 {
		require.Errorf(nil, "Expect from rule actionID to be '34567' , Got : %d ", res.ActionID)
	}
	if res.Operator != Operator_LessThan {
		require.Errorf(nil, "Expect from rule operator to be '<' , Got : %s ", res.Operator)
	}
}

func (h *HttpHandlerTest) TestDeleteRule() {
	require := h.Require()
	var ctx echo.Context
	h.clearTable()
	h.addUsers()
	ctx.Set("ruleID", 12)
	err := h.handler.DeleteRule(ctx)
	if err != nil {
		require.Errorf(err, "error in deleting the rule , error :")
	}

	res, err := h.handler.db.GetRule(12)
	if err == nil {
		require.Errorf(nil, "Expect from get rule to give err because it was deleted  , Got : %v ", res)
	}
}
