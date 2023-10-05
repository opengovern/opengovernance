package alerting

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/kaytu-io/kaytu-engine/pkg/alerting/api"
	api2 "github.com/kaytu-io/kaytu-engine/pkg/auth/api"
	compliance "github.com/kaytu-io/kaytu-engine/pkg/compliance/api"
	"github.com/kaytu-io/kaytu-engine/pkg/compliance/client"
	"github.com/kaytu-io/kaytu-engine/pkg/internal/httpserver"
	api3 "github.com/kaytu-io/kaytu-engine/pkg/onboard/api"
	onboardClient "github.com/kaytu-io/kaytu-engine/pkg/onboard/client"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"
)

var (
	isCallAction bool

	server  *httptest.Server
	com     client.ComplianceServiceClient
	onboard onboardClient.OnboardServiceClient
)

func setupSuite(tb testing.TB) (func(tb testing.TB), *HttpHandler) {
	logger, err := zap.NewProduction()
	if err != nil {
		tb.Errorf("new logger : %v", err)
	}

	mux := http.NewServeMux()
	s := http.Server{Addr: "localhost:8082", Handler: mux}
	mux.HandleFunc("/call", func(writer http.ResponseWriter, request *http.Request) {
		isCallAction = true
	})
	go s.ListenAndServe()

	//mocking server
	server = httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/api/v1/insight/123123":
			mockGetInsightEndpoint(writer, request)
		case "/api/v1/findings":
			mockGetFindingsEndpoint(writer, request)
		case "/api/v1/connection-groups/testConnectionId":
			mockGetConnectionGroupEndpoint(writer, request)
		default:
			http.NotFoundHandler().ServeHTTP(writer, request)
		}
	}))

	handler, err := InitializeHttpHandler("127.0.0.1", "5432", "test-database",
		"user_1", "qwertyPostgres", "disable", server.URL, server.URL, logger)
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
		err = s.Shutdown(context.Background())
		if err != nil {
			tb.Errorf("error in Shutdown the action server , err : %v ", err)
		}

		err = tp.Shutdown(context.Background())
		if err != nil {
			tb.Errorf("error stopping the server ,err : %v ", err)
		}
		err = e.Shutdown(context.Background())
		if err != nil {
			tb.Errorf("error stopping the server ,err : %v ", err)
		}

		server.Close()
	}, handler
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
	operatorInfo := api.OperatorInformation{OperatorType: "<", Value: 1000}
	operator := api.OperatorStruct{
		OperatorInfo: &operatorInfo,
		Condition:    nil,
	}

	benchmarkId := "CIS v1.4.0"
	connectionId := "testConnectionID"
	req := api.ApiRule{
		ID:        12,
		EventType: api.EventType{BenchmarkId: &benchmarkId},
		Scope:     api.Scope{ConnectionId: &connectionId},
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

	operatorInfo := api.OperatorInformation{OperatorType: "<", Value: 100}
	operator := api.OperatorStruct{
		OperatorInfo: &operatorInfo,
		Condition:    nil,
	}

	var id uint = 123
	var insightId int64 = 123123
	connectionId := "testConnectionId"
	req := api.ApiRule{
		ID:        id,
		EventType: api.EventType{InsightId: &insightId},
		Scope:     api.Scope{ConnectionId: &connectionId},
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

	operatorInfo := api.OperatorInformation{OperatorType: "<", Value: 110}
	operator := api.OperatorStruct{
		OperatorInfo: &operatorInfo,
		Condition:    nil,
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

	OperatorInfo := api.OperatorInformation{OperatorType: ">", Value: 100}
	operatorInformation2 := api.OperatorInformation{OperatorType: "<", Value: 230}

	operator = append(operator, api.OperatorStruct{
		OperatorInfo: &OperatorInfo,
	})
	operator = append(operator, api.OperatorStruct{
		OperatorInfo: &operatorInformation2,
	})

	conditionStruct.ConditionType = "AND"
	conditionStruct.Operator = operator
	stat, err := calculationOperations(api.OperatorStruct{Condition: &conditionStruct}, 200)
	if err != nil {
		t.Errorf("Error calculationOperations: %v ", err)
	}
	if !stat {
		t.Errorf("Error in calculate the calculationOperations")
	}
}

func TestCalculationOperationsInCombination(t *testing.T) {
	var conditionStruct api.ConditionStruct
	conditionStruct.ConditionType = "OR"

	var newCondition api.ConditionStruct
	newCondition.ConditionType = "AND"
	number1 := api.OperatorInformation{OperatorType: ">", Value: 700}
	number2 := api.OperatorInformation{OperatorType: ">", Value: 750}
	newCondition.Operator = append(newCondition.Operator, api.OperatorStruct{
		OperatorInfo: &number2,
	})
	newCondition.Operator = append(newCondition.Operator, api.OperatorStruct{
		OperatorInfo: &number1,
	})

	OperatorInfo := api.OperatorInformation{OperatorType: "<", Value: 600}
	conditionStruct.Operator = append(conditionStruct.Operator, api.OperatorStruct{
		OperatorInfo: &OperatorInfo,
	})
	conditionStruct.Operator = append(conditionStruct.Operator, api.OperatorStruct{
		Condition: &newCondition,
	})

	stat, err := calculationOperations(api.OperatorStruct{OperatorInfo: nil, Condition: &conditionStruct}, 1000)
	if err != nil {
		t.Errorf("Error calculationOperations: %v ", err)
	}
	if !stat {
		t.Errorf("error : state is false")
	}
}

func mockGetConnectionGroupEndpoint(w http.ResponseWriter, r *http.Request) {
	response := api3.ConnectionGroup{
		ConnectionIds: []string{"connectionGroupTest"},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func mockGetFindingsEndpoint(w http.ResponseWriter, r *http.Request) {
	response := compliance.GetFindingsResponse{
		Findings:   nil,
		TotalCount: 2000,
	}
	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(response)
	if err != nil {
		panic(fmt.Errorf("error in encode the response of the Finding , error equal to : %v ", err))
	}
}

func mockGetInsightEndpoint(w http.ResponseWriter, r *http.Request) {
	var TotalResultValue int64 = 2000
	insight := compliance.Insight{
		TotalResultValue: &TotalResultValue,
	}

	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(insight)
	if err != nil {
		panic(fmt.Errorf("error in encode the response of the insight , error equal to : %v ", err))
	}
}

func TestTrigger(t *testing.T) {
	teardownSuite, h := setupSuite(t)
	defer teardownSuite(t)

	operatorInfo := api.OperatorInformation{OperatorType: ">", Value: 100}
	operator := api.OperatorStruct{
		OperatorInfo: &operatorInfo,
		Condition:    nil,
	}

	var id uint = 123
	//var insightId int64 = 123123
	var benchmarkId string = "testBenchmarkId"
	connectionId := "testConnectionId"
	req := api.ApiRule{
		ID:        id,
		EventType: api.EventType{BenchmarkId: &benchmarkId},
		Scope:     api.Scope{ConnectionId: &connectionId},
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
		Url:     "http://localhost:8082/call",
		Headers: map[string]string{"insightId": "123123"},
		Body:    "testBody",
	}
	_, err = doSimpleJSONRequest("POST", "/api/v1/action/create", action, nil)
	require.NoError(t, err)

	// trigger :
	h.complianceClient = com
	h.onboardClient = onboard
	_ = h.Trigger

	if !isCallAction {
		t.Errorf("isCall equal to : %v", isCallAction)
	}
}
