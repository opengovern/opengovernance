package alerting

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/kaytu-io/kaytu-engine/pkg/alerting/api"
	api2 "github.com/kaytu-io/kaytu-engine/pkg/auth/api"
	compliance "github.com/kaytu-io/kaytu-engine/pkg/compliance/api"
	"github.com/kaytu-io/kaytu-engine/pkg/internal/httpserver"
	api3 "github.com/kaytu-io/kaytu-engine/pkg/onboard/api"
	"github.com/kaytu-io/kaytu-util/pkg/source"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strconv"
	"testing"
	"time"
)

var (
	isCallAction bool

	server *httptest.Server
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

		body, err := io.ReadAll(request.Body)
		if err != nil {
			fmt.Printf("error in read response call addres of http server  : %v ", err)
		}
		fmt.Printf("response Call : %v \n", string(body))

	})
	go s.ListenAndServe()

	//mocking server
	server = httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/api/v1/insight/1231":
			mockGetInsightEndpoint(writer, request)
		case "/api/v1/findings/CIS v1.4.0/accounts":
			mockGetFindingsEndpoint(writer, request)
		case "/api/v1/connection-groups/testConnectionId":
			mockGetConnectionGroupEndpoint(writer, request)
		default:
			http.NotFoundHandler().ServeHTTP(writer, request)
		}
	}))

	handler, err := InitializeHttpHandler("127.0.0.1", "5432", "postgres",
		"testdatabase", "qwertyPostgres", "disable", server.URL, server.URL, logger)
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
		body, _ := io.ReadAll(res.Body)
		return nil, fmt.Errorf("invalid status code : %d, body : %v ", res.StatusCode, string(body))
	}

	if response != nil {
		b, err := io.ReadAll(res.Body)
		if err != nil {
			return nil, err
		}

		if reflect.ValueOf(response).Kind() == reflect.String {
			response = string(b)
		} else {
			if err := json.Unmarshal(b, response); err != nil {
				return nil, fmt.Errorf("error unmarshalling the response ,err : %v", err)
			}
		}
	}
	return res, nil
}

func addRule(t *testing.T) uint {
	operator := api.OperatorStruct{
		Condition: &api.ConditionStruct{
			ConditionType: "OR",
			Operator: []api.OperatorStruct{
				{
					OperatorType: "<", Value: 100},
				{
					OperatorType: ">", Value: 200},
			},
		},
	}

	var id uint
	benchmarkId := "CIS v1.4.0"
	connectionId := "testConnectionID"
	connector := source.CloudAWS
	req := api.Rule{
		EventType: api.EventType{BenchmarkId: &benchmarkId},
		Scope:     api.Scope{ConnectionId: &connectionId, Connector: &connector},
		Metadata: api.Metadata{
			Name:        "test metadata name ",
			Description: "test metadata description",
			Label:       []string{"test label"},
		},
		Operator: operator,
		ActionID: 123123,
	}

	_, err := doSimpleJSONRequest("POST", "/api/v1/rule/create", req, &id)
	require.NoError(t, err, "error creating rule")
	return id
}

func getRule(h *HttpHandler, id uint) (api.Rule, error) {
	var rule Rule
	err := h.db.orm.Model(&Rule{}).Where("id = ? ", id).Find(&rule).Error
	if err != nil {
		return api.Rule{}, err
	}

	fmt.Println("================", rule)

	var eventType api.EventType
	err = json.Unmarshal(rule.EventType, &eventType)
	if err != nil {
		return api.Rule{}, fmt.Errorf("error unmarshalling the eventType, error : %v", err)
	}

	var scope api.Scope
	err = json.Unmarshal(rule.Scope, &scope)
	if err != nil {
		return api.Rule{}, fmt.Errorf("error unmarshalling the scope , error : %v", err)
	}

	var operator api.OperatorStruct
	err = json.Unmarshal(rule.Operator, &operator)
	if err != nil {
		return api.Rule{}, fmt.Errorf("error unmarshalling the operator , error : %v", err)
	}

	var metadata api.Metadata
	err = json.Unmarshal(rule.Metadata, &metadata)
	if err != nil {
		return api.Rule{}, fmt.Errorf("error unmarshalling the metadata , error : %v", err)
	}

	response := api.Rule{
		Id:        rule.Id,
		EventType: eventType,
		Scope:     scope,
		Operator:  operator,
		Metadata:  metadata,
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

	operator := api.OperatorStruct{
		OperatorType: "<", Value: 100,
		Condition: nil,
	}

	var id uint
	var insightId int64 = 123123
	connectionId := "testConnectionId"
	connector := source.CloudAWS
	req := api.Rule{
		Id:        id,
		EventType: api.EventType{InsightId: &insightId},
		Scope:     api.Scope{ConnectionId: &connectionId, Connector: &connector},
		Metadata: api.Metadata{
			Name:        "test metadata name",
			Description: "test metadata description",
			Label:       []string{"test label"},
		},
		Operator: operator,
		ActionID: 1231,
	}

	_, err := doSimpleJSONRequest("POST", "/api/v1/rule/create", req, &id)
	require.NoError(t, err, "error creating rule")

	foundRule, err := getRule(h, id)
	require.NoErrorf(t, err, "error getting the rule")

	require.Equal(t, operator, foundRule.Operator)
	require.Equal(t, 100, int(foundRule.Operator.Value))
	require.Equal(t, "test metadata name", foundRule.Metadata.Name)
	require.Equal(t, "test metadata description", foundRule.Metadata.Description)
	require.Equal(t, []string{"test label"}, foundRule.Metadata.Label)

	require.Equal(t, "testConnectionId", *foundRule.Scope.ConnectionId)
	require.Equal(t, &connector, foundRule.Scope.Connector)
	require.Equal(t, 123123, int(*foundRule.EventType.InsightId))
	require.Equal(t, 1231, int(foundRule.ActionID))

	operator = api.OperatorStruct{
		Condition: &api.ConditionStruct{
			ConditionType: "OR",
			Operator: []api.OperatorStruct{
				{
					OperatorType: "<", Value: 100},
				{
					OperatorType: ">", Value: 200},
			},
		},
	}
	req.Operator = operator

	_, err = doSimpleJSONRequest("POST", "/api/v1/rule/create", req, &id)
	require.NoError(t, err, "error creating rule")

	foundRule, err = getRule(h, id)
	require.NoErrorf(t, err, "error getting the rule")

	require.Equal(t, operator, foundRule.Operator)

	operator = api.OperatorStruct{
		Condition: &api.ConditionStruct{
			ConditionType: "AND",
			Operator: []api.OperatorStruct{
				{
					OperatorType: ">", Value: 50},
				{
					OperatorType: "<", Value: 200},
			},
		},
	}
	req.Operator = operator

	_, err = doSimpleJSONRequest("POST", "/api/v1/rule/create", req, &id)
	require.NoError(t, err, "error creating rule")

	foundRule, err = getRule(h, id)
	require.NoErrorf(t, err, "error getting the rule")

	require.Equal(t, operator, foundRule.Operator)

	operator = api.OperatorStruct{
		Condition: &api.ConditionStruct{
			ConditionType: "AND",
			Operator: []api.OperatorStruct{
				{
					OperatorType: ">", Value: 50},
				{
					Condition: &api.ConditionStruct{
						ConditionType: "OR",
						Operator: []api.OperatorStruct{
							{
								OperatorType: "<", Value: 100},
							{
								OperatorType: ">", Value: 200,
							},
						},
					},
				},
			},
		},
	}
	req.Operator = operator

	_, err = doSimpleJSONRequest("POST", "/api/v1/rule/create", req, &id)
	require.NoError(t, err, "error creating rule")

	foundRule, err = getRule(h, id)
	require.NoErrorf(t, err, "error getting the rule")

	require.Equal(t, operator, foundRule.Operator)
}

func TestUpdateRule(t *testing.T) {
	teardownSuite, h := setupSuite(t)
	defer teardownSuite(t)
	id := addRule(t)

	operator := api.OperatorStruct{
		Condition: &api.ConditionStruct{
			ConditionType: "OR",
			Operator: []api.OperatorStruct{
				{
					OperatorType: "<", Value: 100},
				{
					OperatorType: ">", Value: 200},
			},
		},
	}

	req := api.Rule{
		Id:       id,
		Operator: operator,
		ActionID: 34567,
	}

	reqUpdate := api.UpdateRuleRequest{
		Operator: &req.Operator,
		ActionID: &req.ActionID,
	}

	idString := strconv.FormatUint(uint64(id), 10)
	_, err := doSimpleJSONRequest("PUT", "/api/v1/rule/update/"+idString, reqUpdate, nil)
	require.NoError(t, err, "error updating rule")

	ruleNew, err := getRule(h, id)
	require.NoErrorf(t, err, "error getting the rule ")

	require.Equal(t, operator, ruleNew.Operator)
	require.Equal(t, 34567, int(ruleNew.ActionID))
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
	var id uint
	req := api.Action{
		Method:  "GET",
		Url:     "https://kaytu.dev/",
		Headers: map[string]string{"insight": "teatInsight"},
		Body:    "testBody",
	}

	_, err := doSimpleJSONRequest("POST", "/api/v1/action/create", req, &id)
	require.NoError(t, err)
	return id
}

func getAction(h *HttpHandler, id uint) (api.Action, error) {
	var action Action
	err := h.db.orm.Model(&Action{}).Where("id = ?", id).Find(&action).Error
	if err != nil {
		return api.Action{}, err
	}

	var header map[string]string
	err = json.Unmarshal(action.Headers, &header)
	if err != nil {
		return api.Action{}, fmt.Errorf("error unmarshalling the header , error : %v ", err)
	}

	response := api.Action{
		Id:      action.Id,
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

	var actions []api.Action
	_, err := doSimpleJSONRequest("GET", "/api/v1/action/list", nil, &actions)
	require.NoError(t, err)
	require.Empty(t, actions)
}

func TestCreateAction(t *testing.T) {
	teardownSuite, h := setupSuite(t)
	defer teardownSuite(t)

	var id uint
	action := api.Action{
		Id:      id,
		Method:  "GET",
		Url:     "https://kaytu.dev/company",
		Headers: map[string]string{"insightId": "123123"},
		Body:    "testBody",
	}
	_, err := doSimpleJSONRequest("POST", "/api/v1/action/create", action, &id)
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
	newMethod := "POST"
	newUrl := "https://kaytu.dev/use-cases"
	header := map[string]string{"insight": "teatInsight22"}
	req := api.UpdateActionRequest{
		Method:  &newMethod,
		Headers: header,
		Url:     &newUrl,
	}

	idString := strconv.FormatUint(uint64(id), 10)
	_, err := doSimpleJSONRequest("PUT", "/api/v1/action/update/"+idString, req, nil)
	require.NoError(t, err, "error updating action")

	newAction, err := getAction(h, id)
	require.NoErrorf(t, err, "error getting the action")

	require.Equal(t, header, newAction.Headers)
	require.Equal(t, "POST", newAction.Method)
	require.Equal(t, "https://kaytu.dev/use-cases", newAction.Url)
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

	operator = append(operator, api.OperatorStruct{
		OperatorType: ">", Value: 100,
	})
	operator = append(operator, api.OperatorStruct{
		OperatorType: "<", Value: 230,
	})

	conditionStruct.ConditionType = api.ConditionAnd
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

	operator := api.OperatorStruct{
		Condition: &api.ConditionStruct{
			ConditionType: "AND",
			Operator: []api.OperatorStruct{
				{
					OperatorType: "GreaterThan", Value: 50},
				{
					Condition: &api.ConditionStruct{
						ConditionType: "OR",
						Operator: []api.OperatorStruct{
							{
								OperatorType: "LessThan", Value: 100},
							{
								OperatorType: "GreaterThan", Value: 200,
							},
						},
					},
				},
			},
		},
	}

	stat, err := calculationOperations(operator, 1000)
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
	var Accounts []compliance.AccountsFindingsSummary
	Accounts = append(Accounts, compliance.AccountsFindingsSummary{SecurityScore: 5.5})
	var response compliance.GetAccountsFindingsSummaryResponse
	response.Accounts = Accounts

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

	operator := api.OperatorStruct{
		OperatorType: "GreaterThan",
		Value:        100,
		Condition:    nil,
	}

	var actionId uint
	// create Action:
	action := api.Action{
		Id:      actionId,
		Method:  "POST",
		Url:     "http://localhost:8082/call",
		Headers: map[string]string{"insightId": "123123"},
		Body:    "{{.Name}} rule triggered successfully",
	}
	_, err := doSimpleJSONRequest("POST", "/api/v1/action/create", action, &actionId)
	require.NoError(t, err)

	var ruleId uint
	var benchmarkId string = "CIS v1.4.0"
	//connectionId := "testConnectionId"
	//connector := source.CloudAWS
	req := api.Rule{
		Id:        ruleId,
		EventType: api.EventType{BenchmarkId: &benchmarkId},
		Metadata:  api.Metadata{Name: "testMetadataName"},
		Operator:  operator,
		ActionID:  actionId,
	}

	_, err = doSimpleJSONRequest("POST", "/api/v1/rule/create", req, &ruleId)
	require.NoError(t, err, "error creating rule")

	// trigger :
	err = h.TriggerRulesList()
	if err != nil {
		t.Errorf("%v", err)
	}

	if !isCallAction {
		t.Errorf("isCall equal to : %v", isCallAction)
	}
}
