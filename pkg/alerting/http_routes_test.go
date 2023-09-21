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
		tb.Errorf("error in connecting to postgres , err : %v", err)
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
			tb.Errorf("error in stopping the server ,err : %v ", err)
		}
		err = e.Shutdown(context.Background())
		if err != nil {
			tb.Errorf("error in stopping the server ,err : %v ", err)
		}
	}, handler
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
	completeAddress := fmt.Sprintf("http://localhost:8081" + path)
	req, err := http.NewRequest(method, completeAddress, r)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add(httpserver.XKaytuUserRoleHeader, string(api2.AdminRole))
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
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
			return nil, err
		}
	}
	return res, nil
}

func addRule(t *testing.T) uint {
	req := api.ApiRule{
		ID:        12,
		EventType: api.EventType{InsightId: 1231},
		Scope:     api.Scope{ConnectionId: "testConnectionID"},
		Operator:  ">",
		Value:     1000,
		ActionID:  123123,
	}
	_, err := doSimpleJSONRequest("POST", "/api/v1/rule/create", req, nil)
	require.NoError(t, err, "error creating rule")
	return 12
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
	teardownSuite, _ := setupSuite(t)
	defer teardownSuite(t)

	var id uint = 123
	req := api.ApiRule{
		ID:        id,
		EventType: api.EventType{InsightId: 123123},
		Scope:     api.Scope{ConnectionId: "testConnectionId"},
		Operator:  ">",
		Value:     100,
		ActionID:  1231,
	}
	_, err := doSimpleJSONRequest("POST", "/api/v1/rule/create", req, nil)
	require.NoError(t, err, "error creating rule")

	idS := strconv.FormatUint(uint64(id), 10)
	var foundRule api.ApiRule

	_, err = doSimpleJSONRequest("GET", "/api/v1/rule/get/"+idS, nil, &foundRule)
	require.NoError(t, err, "error getting rule")

	require.Equal(t, api.Operator_GreaterThan, foundRule.Operator)
	require.Equal(t, 100, int(foundRule.Value))
	require.Equal(t, "testConnectionId", foundRule.Scope.ConnectionId)
	require.Equal(t, 123123, int(foundRule.EventType.InsightId))
	require.Equal(t, 1231, int(foundRule.ActionID))
}

func TestUpdateRule(t *testing.T) {
	teardownSuite, _ := setupSuite(t)
	defer teardownSuite(t)

	id := addRule(t)

	req := api.ApiRule{
		ID:       id,
		Value:    110,
		Operator: api.Operator_LessThan,
		ActionID: 34567,
	}
	reqUpdate := api.UpdateRuleRequest{
		ID:       id,
		Value:    &req.Value,
		Operator: &req.Operator,
		ActionID: &req.ActionID,
	}
	_, err := doSimpleJSONRequest("GET", "/api/v1/rule/update", reqUpdate, nil)
	require.NoError(t, err, "error updating rule")

	var ruleNew api.ApiRule
	idS := strconv.FormatUint(uint64(id), 10)
	_, err = doSimpleJSONRequest("GET", "/api/v1/rule/get/"+idS, nil, &ruleNew)
	require.NoError(t, err, "error getting rule")

	require.Equal(t, 110, int(ruleNew.Value))
	require.Equal(t, 34567, int(ruleNew.ActionID))
	require.Equal(t, api.Operator_LessThan, ruleNew.Operator)
}

func TestDeleteRule(t *testing.T) {
	teardownSuite, _ := setupSuite(t)
	defer teardownSuite(t)
	id := addRule(t)
	idS := strconv.FormatUint(uint64(id), 10)
	_, err := doSimpleJSONRequest("DELETE", "/api/v1/rule/delete/"+idS, nil, nil)
	require.NoError(t, err, "error deleting rule")

	var rule api.ApiRule
	_, err = doSimpleJSONRequest("GET", "/api/v1/rule/get/"+idS, nil, &rule)
	require.Empty(t, rule)
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

func TestListAction(t *testing.T) {
	teardownSuite, _ := setupSuite(t)
	defer teardownSuite(t)

	var actions []api.ApiAction
	_, err := doSimpleJSONRequest("GET", "/api/v1/action/list", nil, &actions)
	require.NoError(t, err)
	require.Empty(t, actions)
}

func TestCreateAction(t *testing.T) {
	teardownSuite, _ := setupSuite(t)
	defer teardownSuite(t)
	//var id uint  = 12
	action := api.ApiAction{
		ID:      12,
		Method:  "GET",
		Url:     "https://kaytu.dev/company",
		Headers: map[string]string{"insightId": "123123"},
		Body:    "testBody",
	}
	_, err := doSimpleJSONRequest("POST", "/api/v1/action/create", action, nil)
	require.NoError(t, err)

	var foundAction api.ApiAction
	//idS := strconv.FormatUint(uint64(id), 10)
	_, err = doSimpleJSONRequest("GET", "/api/v1/action/get/12", nil, &foundAction)
	require.NoError(t, err, "error getting rule")

	require.Equal(t, "https://kaytu.dev/company", foundAction.Url)
	require.Equal(t, "testBody", foundAction.Body)
	require.Equal(t, "GET", foundAction.Method)
	require.Equal(t, map[string]string{"insightId": "123123"}, foundAction.Headers)
}

func TestUpdateAction(t *testing.T) {
	teardownSuite, _ := setupSuite(t)
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

	var actionG api.ApiAction
	idS := strconv.FormatUint(uint64(id), 10)
	_, err = doSimpleJSONRequest("GET", "/api/v1/action/get/"+idS, nil, &actionG)
	require.NoError(t, err, "error getting action")

	require.Equal(t, map[string]string{"insightId": "newTestInsight"}, actionG.Headers)
	require.Equal(t, "POST", actionG.Method)
	require.Equal(t, "https://kaytu.dev/use-cases", actionG.Url)
}

func TestDeleteAction(t *testing.T) {
	teardownSuite, _ := setupSuite(t)
	defer teardownSuite(t)

	id := addAction(t)
	idS := strconv.FormatUint(uint64(id), 10)
	_, err := doSimpleJSONRequest("DELETE", "/api/v1/action/delete/"+idS, nil, nil)
	require.NoError(t, err, "error deleting action")

	var action Action
	_, err = doSimpleJSONRequest("GET", "/api/v1/action/get/"+idS, nil, &action)
	require.Error(t, err)
}
