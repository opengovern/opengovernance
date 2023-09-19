package alerting

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/kaytu-io/kaytu-engine/pkg/alerting/api"
	"github.com/kaytu-io/kaytu-engine/pkg/internal/httpserver"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"io"
	"net/http"
	"strconv"
	"testing"
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
	handler.db.orm.Exec("DELETE FROM rule")
	handler.db.orm.Exec("DELETE FROM actions")

	e, tp := httpserver.Register(logger, handler)
	err = e.Start("http://localhost:8081")
	if err != nil {
		tb.Errorf("error in uploading the server , err : %v ", err)
	}

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

func TestEmptyListRule(t *testing.T) {
	teardownSuite, handler := setupSuite(t)
	defer teardownSuite(t)
	handler.db.orm.Exec("DELETE from rule")

	var rules []Rule
	res, err := doSimpleJSONRequest("GET", "/api/v1/rule/list", nil, &rules)
	if err != nil {
		t.Errorf("error getting list of the rules , err : %v ", err)
	}

	require.Equal(t, http.StatusOK, res.Status)
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
	resC, err := doSimpleJSONRequest("POST", "/api/v1/rule/create", req, nil)
	if err != nil {
		t.Errorf("error creating rule , err : %v", err)
	}
	require.Equal(t, http.StatusOK, resC.Status)

	var foundRule api.ApiRule
	idS := strconv.FormatUint(uint64(id), 10)
	resG, err := doSimpleJSONRequest("GET", "/api/v1/rule/get/"+idS, nil, &foundRule)
	if err != nil {
		t.Errorf("error getting rule , err : %v", err)
	}
	require.Equal(t, http.StatusOK, resG.Status)

	require.Equal(t, api.Operator_GreaterThan, foundRule.Operator)
	require.Equal(t, 100, foundRule.Value)
	require.Equal(t, "testConnectionId", foundRule.Scope.ConnectionId)
	require.Equal(t, 123123, foundRule.EventType.InsightId)
	require.Equal(t, 1231, foundRule.ActionID)
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
	res, err := doSimpleJSONRequest("POST", "/api/v1/rule/create", req, nil)
	if err != nil {
		t.Errorf("error creating rule : %v", err)
	}
	require.Equal(t, http.StatusOK, res.Status)
	return 12
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
		ID:       req.ID,
		Value:    &req.Value,
		Operator: &req.Operator,
		ActionID: &req.ActionID,
	}
	resU, err := doSimpleJSONRequest("GET", "/api/v1/rule/update", reqUpdate, nil)
	if err != nil {
		t.Errorf("error in update rule : %v", err)
	}
	require.Equal(t, http.StatusOK, resU.Status)

	var ruleNew api.ApiRule
	idS := strconv.FormatUint(uint64(id), 10)
	resG, err := doSimpleJSONRequest("GET", "/api/v1/rule/get/"+idS, nil, &ruleNew)
	if err != nil {
		t.Errorf("error in get rule : %v", err)
	}
	require.Equal(t, http.StatusOK, resG)

	require.Equal(t, 110, ruleNew.Value)
	require.Equal(t, 34567, ruleNew.ActionID)
	require.Equal(t, api.Operator_LessThan, ruleNew.Operator)
}

func TestDeleteRule(t *testing.T) {
	teardownSuite, _ := setupSuite(t)
	defer teardownSuite(t)
	id := addRule(t)
	idS := strconv.FormatUint(uint64(id), 10)
	resD, err := doSimpleJSONRequest("DELETE", "/api/v1/rule/delete/"+idS, nil, nil)
	if err != nil {
		t.Errorf("error in deleting the rule : %v", err)
	}
	require.Equal(t, http.StatusOK, resD.Status)

	responseGet, _ := doSimpleJSONRequest("GET", "/api/v1/rule/get/"+idS, nil, nil)
	require.NotEqual(t, http.StatusOK, responseGet.Status)
}

// -------------------------------------------------- action test --------------------------------------------------

func TestListAction(t *testing.T) {
	teardownSuite, handler := setupSuite(t)
	defer teardownSuite(t)
	handler.db.orm.Exec("DELETE FROM actions")

	var actions []api.ApiAction
	res, _ := doSimpleJSONRequest("GET", "/api/v1/action/list", nil, &actions)

	require.Equal(t, http.StatusOK, res.Status)
	require.Empty(t, actions)
}

func TestCreateAction(t *testing.T) {
	teardownSuite, _ := setupSuite(t)
	defer teardownSuite(t)

	action := api.ApiAction{
		ID:      12,
		Method:  "GET",
		Url:     "https://kaytu.dev/company",
		Headers: map[string]string{"insightId": "123123"},
		Body:    "",
	}

	resC, err := doSimpleJSONRequest("POST", "/api/v1/action/create", action, nil)
	if err != nil {
		t.Errorf("error creating action ,err : %v", err)
	}
	require.Equal(t, http.StatusOK, resC.Status)

	var actionG api.ApiAction
	resG, err := doSimpleJSONRequest("GET", "api/v1/action/get/12", nil, &actionG)
	if err != nil {
		t.Errorf("error geting action ,err : %v", err)
	}
	require.Equal(t, http.StatusOK, resG.Status)

	require.Equal(t, "https://kaytu.dev/company", action.Url)
	require.Equal(t, "", action.Body)
	require.Equal(t, "GET", action.Method)
	require.Equal(t, map[string]string{"insightId": "123123"}, action.Headers)
}

func addAction(t *testing.T) uint {
	req := api.ApiAction{
		ID:      12,
		Method:  "GET",
		Url:     "https://kaytu.dev/",
		Headers: map[string]string{"insight": "teatInsight"},
		Body:    "",
	}
	res, err := doSimpleJSONRequest("POST", "/api/v1/action/create", req, nil)
	if err != nil {
		t.Errorf("error creating action, err : %v ", err)
	}
	require.Equal(t, http.StatusOK, res.Status)
	return 12
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

	resU, err := doSimpleJSONRequest("UPDATE", "/api/v1/action/Update", req, nil)
	if err != nil {
		t.Errorf("error updating action , error : %v", err)
	}
	require.Equal(t, http.StatusOK, resU.Status)

	var actionG api.ApiAction
	idS := strconv.FormatUint(uint64(id), 10)
	resG, err := doSimpleJSONRequest("GET", "/api/v1/action/get/"+idS, nil, actionG)
	if err != nil {
		t.Errorf("error getting action , err : %v", err)
	}
	require.Equal(t, http.StatusOK, resG.Status)

	require.Equal(t, map[string]string{"insightId": "newTestInsight"}, actionG.Headers)
	require.Equal(t, "POST", actionG.Method)
	require.Equal(t, "https://kaytu.dev/use-cases", actionG.Url)
}

func TestDeleteAction(t *testing.T) {
	teardownSuite, _ := setupSuite(t)
	defer teardownSuite(t)

	id := addAction(t)
	idS := strconv.FormatUint(uint64(id), 10)
	resD, err := doSimpleJSONRequest("DELETE", "/api/v1/action/delete/"+idS, nil, nil)
	if err != nil {
		t.Errorf("error deleting action , err : %v ", err)
	}
	require.Equal(t, http.StatusOK, resD.Status)

	resG, err := doSimpleJSONRequest("GET", "/api/v1/action/get/"+idS, nil, nil)
	require.NotEqual(t, http.StatusOK, resG.Status)
}
