package alerting

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/kaytu-io/kaytu-engine/pkg/alerting/api"
	api2 "github.com/kaytu-io/kaytu-engine/pkg/auth/api"
	"github.com/kaytu-io/kaytu-engine/pkg/internal/httpserver"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"io"
	"net/http"
	"strconv"
	"testing"
	"time"
)

func setupSuite(tb testing.TB) (func(tb testing.TB), *HttpHandler) {
	logger, err := zap.NewProduction()
	if err != nil {
		tb.Errorf("new logger : %v", err)
	}

	handler, err := InitializeHttpHandler("127.0.0.1", "5432", "test-database", "user_1", "qwertyPostgres", "disable", logger)
	if err != nil {
		tb.Errorf("error connecting to postgres , err : %v", err)
	}
	handler.db.orm.Exec("DELETE FROM rules")
	handler.db.orm.Exec("DELETE FROM actions")

	e, tp := httpserver.Register(logger, handler)

	go e.Start("localhost:8081")
	time.Sleep(500 * time.Millisecond)

	// Return a function to teardown the test
	return func(tb testing.TB) {
		err = tp.Shutdown(context.Background())
		if err != nil {
			tb.Errorf("error stopping the server ,err : %v ", err)
		}
		err = e.Shutdown(context.Background())
		if err != nil {
			tb.Errorf("error stopping the server ,err : %v ", err)
		}
	}, handler
}

func setupActionRequests(tb testing.TB) (func(), bool) {
	var isCall bool
	mux := http.NewServeMux()
	s := http.Server{Addr: "localhost:8082", Handler: mux}
	mux.HandleFunc("/call", func(writer http.ResponseWriter, request *http.Request) {
		isCall = true
	})
	go s.ListenAndServe()

	return func() {
		err := s.Shutdown(context.Background())
		if err != nil {
			tb.Errorf("error in Shutdown the server , err : %v ", err)
		}
	}, isCall
}

func doSimpleJSONRequest(method string, path string, request, response interface{}) (*http.Response, error) {
	var r io.Reader
	if request != nil {
		out, err := json.Marshal(request)
		if err != nil {
			return nil, fmt.Errorf("error marshalling the request , error : %v ", err)
		}

		r = bytes.NewReader(out)
	}
	completeAddress := fmt.Sprintf("http://localhost:8081" + path)
	req, err := http.NewRequest(method, completeAddress, r)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add(httpserver.XKaytuUserRoleHeader, string(api2.AdminRole))
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error sending the request ,err : %v", err)
	}
	if res.StatusCode != 200 {
		return nil, fmt.Errorf("invalid status code : %d", res.StatusCode)
	}

	if response != nil {
		b, err := io.ReadAll(res.Body)
		if err != nil {
			return nil, err
		}

		if err := json.Unmarshal(b, response); err != nil {
			return nil, fmt.Errorf("error unmarshalling the response ,err : %v", err)
		}
	}
	return res, nil
}

func addRule(t *testing.T) uint {
	operatorInfo := api.OperatorInformation{Operator: "<", Value: 1000}
	operator := api.OperatorStruct{
		OperatorInfo: &operatorInfo,
		ConditionStr: nil,
	}

	benchmarkId := "CIS v1.4.0"
	req := api.ApiRule{
		ID:        12,
		EventType: api.EventType{BenchmarkId: &benchmarkId},
		Scope:     api.Scope{ConnectionId: "testConnectionID"},
		Operator:  operator,
		ActionID:  123123,
	}
	_, err := doSimpleJSONRequest("POST", "/api/v1/rule/create", req, nil)
	require.NoError(t, err, "error creating rule")
	return 12
}

func getRule(h *HttpHandler, id uint) (api.ApiRule, error) {
	var rule Rule
	err := h.db.orm.Model(&Rule{}).Where("id = ? ", id).Find(&rule).Error
	if err != nil {
		return api.ApiRule{}, err
	}

	var eventType api.EventType
	err = json.Unmarshal(rule.EventType, &eventType)
	if err != nil {
		return api.ApiRule{}, fmt.Errorf("error unmarshalling the eventType , error : %v", err)
	}

	var scope api.Scope
	err = json.Unmarshal(rule.Scope, &scope)
	if err != nil {
		return api.ApiRule{}, fmt.Errorf("error unmarshalling the scope , error : %v", err)
	}

	var operator api.OperatorStruct
	err = json.Unmarshal(rule.Operator, &operator)
	if err != nil {
		return api.ApiRule{}, fmt.Errorf("error unmarshalling the operator , error : %v", err)
	}

	response := api.ApiRule{
		ID:        rule.ID,
		EventType: eventType,
		Scope:     scope,
		Operator:  operator,
		ActionID:  rule.ActionID,
	}
	return response, nil
}

func TestEmptyListRule(t *testing.T) {
	teardownSuite, _ := setupSuite(t)
	defer teardownSuite(t)

	var rules []Rule
	_, err := doSimpleJSONRequest("GET", "/api/v1/rule/list", nil, &rules)
	require.NoError(t, err, "error in getting rules")

	require.Empty(t, rules)
}

func TestCreateRule(t *testing.T) {
	teardownSuite, h := setupSuite(t)
	defer teardownSuite(t)

	operatorInfo := api.OperatorInformation{Operator: "<", Value: 100}
	operator := api.OperatorStruct{
		OperatorInfo: &operatorInfo,
		ConditionStr: nil,
	}

	var id uint = 123
	var insightId int64 = 123123
	req := api.ApiRule{
		ID:        id,
		EventType: api.EventType{InsightId: &insightId},
		Scope:     api.Scope{ConnectionId: "testConnectionId"},
		Operator:  operator,
		ActionID:  1231,
	}
	_, err := doSimpleJSONRequest("POST", "/api/v1/rule/create", req, nil)
	require.NoError(t, err, "error creating rule")

	foundRule, err := getRule(h, id)
	require.NoErrorf(t, err, "error getting the rule")

	require.Equal(t, operator, foundRule.Operator)
	require.Equal(t, 100, int(foundRule.Operator.OperatorInfo.Value))
	require.Equal(t, "testConnectionId", foundRule.Scope.ConnectionId)
	require.Equal(t, 123123, int(*foundRule.EventType.InsightId))
	require.Equal(t, 1231, int(foundRule.ActionID))
}

func TestUpdateRule(t *testing.T) {
	teardownSuite, h := setupSuite(t)
	defer teardownSuite(t)
	id := addRule(t)

	operatorInfo := api.OperatorInformation{Operator: "<", Value: 110}
	operator := api.OperatorStruct{
		OperatorInfo: &operatorInfo,
		ConditionStr: nil,
	}

	req := api.ApiRule{
		ID:       id,
		Operator: operator,
		ActionID: 34567,
	}

	reqUpdate := api.UpdateRuleRequest{
		ID:       id,
		Operator: &req.Operator,
		ActionID: &req.ActionID,
	}
	_, err := doSimpleJSONRequest("GET", "/api/v1/rule/update", reqUpdate, nil)
	require.NoError(t, err, "error updating rule")

	ruleNew, err := getRule(h, id)
	require.NoErrorf(t, err, "error getting the rule ")

	require.Equal(t, 110, int(ruleNew.Operator.OperatorInfo.Value))
	require.Equal(t, 34567, int(ruleNew.ActionID))
	require.Equal(t, operator, ruleNew.Operator)
}

func TestDeleteRule(t *testing.T) {
	teardownSuite, h := setupSuite(t)
	defer teardownSuite(t)
	id := addRule(t)
	idS := strconv.FormatUint(uint64(id), 10)
	_, err := doSimpleJSONRequest("DELETE", "/api/v1/rule/delete/"+idS, nil, nil)
	require.NoError(t, err, "error deleting rule")

	_, err = getRule(h, id)
	require.Error(t, err)
}

// -------------------------------------------------- action test --------------------------------------------------
func addAction(t *testing.T) uint {
	req := api.ApiAction{
		ID:      12,
		Method:  "GET",
		Url:     "https://kaytu.dev/",
		Headers: map[string]string{"insight": "teatInsight"},
		Body:    "testBody",
	}

	_, err := doSimpleJSONRequest("POST", "/api/v1/action/create", req, nil)
	require.NoError(t, err)
	return 12
}

func getAction(h *HttpHandler, id uint) (api.ApiAction, error) {
	var action Action
	err := h.db.orm.Model(&Action{}).Where("id = ?", id).Find(&action).Error
	if err != nil {
		return api.ApiAction{}, err
	}

	var header map[string]string
	err = json.Unmarshal(action.Headers, &header)
	if err != nil {
		return api.ApiAction{}, fmt.Errorf("error unmarshalling the header , error : %v ", err)
	}

	response := api.ApiAction{
		ID:      action.ID,
		Method:  action.Method,
		Url:     action.Url,
		Headers: header,
		Body:    action.Body,
	}
	return response, nil
}

func TestListAction(t *testing.T) {
	teardownSuite, _ := setupSuite(t)
	defer teardownSuite(t)

	var actions []api.ApiAction
	_, err := doSimpleJSONRequest("GET", "/api/v1/action/list", nil, &actions)
	require.NoError(t, err)
	require.Empty(t, actions)
}

func TestCreateAction(t *testing.T) {
	teardownSuite, h := setupSuite(t)
	defer teardownSuite(t)
	var id uint = 12
	action := api.ApiAction{
		ID:      id,
		Method:  "GET",
		Url:     "https://kaytu.dev/company",
		Headers: map[string]string{"insightId": "123123"},
		Body:    "testBody",
	}
	_, err := doSimpleJSONRequest("POST", "/api/v1/action/create", action, nil)
	require.NoError(t, err)

	foundAction, err := getAction(h, id)
	require.NoErrorf(t, err, "error getting the action")

	require.Equal(t, "https://kaytu.dev/company", foundAction.Url)
	require.Equal(t, "testBody", foundAction.Body)
	require.Equal(t, "GET", foundAction.Method)
	require.Equal(t, map[string]string{"insightId": "123123"}, foundAction.Headers)
}

func TestUpdateAction(t *testing.T) {
	teardownSuite, h := setupSuite(t)
	defer teardownSuite(t)

	id := addAction(t)
	req := api.ApiAction{
		ID:      id,
		Method:  "POST",
		Headers: map[string]string{"insightId": "newTestInsight"},
		Url:     "https://kaytu.dev/use-cases",
	}

	_, err := doSimpleJSONRequest("GET", "/api/v1/action/update", req, nil)
	require.NoError(t, err, "error updating action")

	actionG, err := getAction(h, id)
	require.NoErrorf(t, err, "error getting the action")

	require.Equal(t, map[string]string{"insightId": "newTestInsight"}, actionG.Headers)
	require.Equal(t, "POST", actionG.Method)
	require.Equal(t, "https://kaytu.dev/use-cases", actionG.Url)
}

func TestDeleteAction(t *testing.T) {
	teardownSuite, h := setupSuite(t)
	defer teardownSuite(t)

	id := addAction(t)
	idS := strconv.FormatUint(uint64(id), 10)
	_, err := doSimpleJSONRequest("DELETE", "/api/v1/action/delete/"+idS, nil, nil)
	require.NoError(t, err, "error deleting action")

	_, err = getAction(h, id)
	require.Error(t, err)
}

// ------------------------------------------------ trigger test ----------------------------------------------

func TestCalculationOperationsWithAnd(t *testing.T) {
	var conditionStruct api.ConditionStruct
	var operator []api.OperatorStruct

	OperatorInfo := api.OperatorInformation{Operator: ">", Value: 100}
	operatorInformation2 := api.OperatorInformation{Operator: "<", Value: 230}

	operator = append(operator, api.OperatorStruct{
		OperatorInfo: &OperatorInfo,
	})
	operator = append(operator, api.OperatorStruct{
		OperatorInfo: &operatorInformation2,
	})

	conditionStruct.ConditionType = "AND"
	conditionStruct.OperatorStr = operator
	stat, err := calculationOperations(api.OperatorStruct{ConditionStr: &conditionStruct}, 200)
	if err != nil {
		t.Errorf("Error calculationOperations: %v ", err)
	}
	if !stat {
		t.Errorf("Error in calculate the calculationOperations")
	}
}

func TestCalculationOperationsInCombination(t *testing.T) {
	var conditionStruct api.ConditionStruct
	conditionStruct.ConditionType = "AND"

	var newCondition api.ConditionStruct
	newCondition.ConditionType = "OR"
	number1 := api.OperatorInformation{Operator: "<", Value: 250}
	number2 := api.OperatorInformation{Operator: ">", Value: 220}
	newCondition.OperatorStr = append(newCondition.OperatorStr, api.OperatorStruct{
		OperatorInfo: &number2,
	})
	newCondition.OperatorStr = append(newCondition.OperatorStr, api.OperatorStruct{
		OperatorInfo: &number1,
	})

	OperatorInfo := api.OperatorInformation{Operator: "<", Value: 300}
	conditionStruct.OperatorStr = append(conditionStruct.OperatorStr, api.OperatorStruct{
		OperatorInfo: &OperatorInfo, ConditionStr: &newCondition,
	})

	stat, err := calculationOperations(api.OperatorStruct{OperatorInfo: nil, ConditionStr: &conditionStruct}, 400)
	if err != nil {
		t.Errorf("Error calculationOperations: %v ", err)
	}
	if !stat {
		t.Errorf("Error in calculate the calculationOperations")
	}
}

func TestTrigger(t *testing.T) {
	teardownSuite, h := setupSuite(t)
	defer teardownSuite(t)

	actionServer, isCall := setupActionRequests(t)
	defer actionServer()

	//create Rule:

	operatorInfo := api.OperatorInformation{Operator: "<", Value: 100}
	operator := api.OperatorStruct{
		OperatorInfo: &operatorInfo,
		ConditionStr: nil,
	}

	var id uint = 123
	var insightId int64 = 123123
	req := api.ApiRule{
		ID:        id,
		EventType: api.EventType{InsightId: &insightId},
		Scope:     api.Scope{ConnectionId: "testConnectionId"},
		Operator:  operator,
		ActionID:  1231,
	}
	_, err := doSimpleJSONRequest("POST", "/api/v1/rule/create", req, nil)
	require.NoError(t, err, "error creating rule")

	// create Action:

	var idAction uint = 1231
	action := api.ApiAction{
		ID:      idAction,
		Method:  "GET",
		Url:     "https://kaytu.dev/company",
		Headers: map[string]string{"insightId": "123123"},
		Body:    "testBody",
	}
	_, err = doSimpleJSONRequest("POST", "/api/v1/action/create", action, nil)
	require.NoError(t, err)

	// trigger :
	Trigger(*h)
	t.Errorf("isCall equal to : %v", isCall)
}
