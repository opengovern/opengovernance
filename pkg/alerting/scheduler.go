package alerting

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/kaytu-io/kaytu-engine/pkg/alerting/api"
	authApi "github.com/kaytu-io/kaytu-engine/pkg/auth/api"
	"github.com/kaytu-io/kaytu-engine/pkg/httpclient"
	"github.com/kaytu-io/kaytu-util/pkg/source"
	"github.com/labstack/echo/v4"
	_ "github.com/labstack/echo/v4"
	"go.uber.org/zap"

	"net/http"
	"strconv"
	"text/template"
	"time"
)

func (h *HttpHandler) TriggerRulesJobCycle() error {
	timer := time.NewTicker(5 * time.Minute)
	defer timer.Stop()

	for ; ; <-timer.C {
		err := h.TriggerRulesList()
		if err != nil {
			h.logger.Error(err.Error())
		}
	}
}

func (h *HttpHandler) TriggerRulesList() error {
	rules, err := h.db.ListRules()
	if err != nil {
		return fmt.Errorf("Error giving list rules : %v ", err.Error())
	}

	for _, rule := range rules {
		err := h.TriggerRule(rule)
		if err != nil {
			h.logger.Error("Rule trigger has been failed",
				zap.String("rule id", fmt.Sprintf("%v", rule.Id)),
				zap.Error(err))
			return err
		}
	}
	return nil
}

func (h *HttpHandler) TriggerRule(rule Rule) error {
	var scope api.Scope
	err := json.Unmarshal(rule.Scope, &scope)
	if err != nil {
		return fmt.Errorf("error unmarshalling the scope : %v ", err.Error())
	}

	var eventType api.EventType
	err = json.Unmarshal(rule.EventType, &eventType)
	if err != nil {
		return fmt.Errorf("error unmarshalling the eventType : %v", err.Error())
	}

	var operator api.Condition
	err = json.Unmarshal(rule.Operator, &operator)
	if err != nil {
		return fmt.Errorf("error unmarshalling the operator : %v ", err.Error())
	}

	var metadata api.Metadata
	err = json.Unmarshal(rule.Metadata, &metadata)
	if err != nil {
		return fmt.Errorf("error unmarshalling the metadata : %v ", err.Error())
	}

	var averageSecurityScorePercentage int64
	status := rule.TriggerStatus
	stat := false

	if eventType.InsightId != nil {
		h.logger.Info("triggering insight", zap.String("rule", fmt.Sprintf("%v", rule.Id)))
		stat, averageSecurityScorePercentage, err = h.triggerInsight(operator, eventType, scope)
		if err != nil {
			return fmt.Errorf("error triggering the insight : %v ", err.Error())
		}
	} else if eventType.BenchmarkId != nil {
		h.logger.Info("triggering compliance", zap.String("rule", fmt.Sprintf("%v", rule.Id)))
		stat, averageSecurityScorePercentage, err = h.triggerCompliance(operator, scope, eventType)
		if err != nil {
			return fmt.Errorf("Error in trigger compliance : %v ", err.Error())
		}
	} else {
		return fmt.Errorf("Error: insighId or complianceId not entered ")
	}

	if stat == true && status == api.TriggerStatus_NotActive {
		err = h.sendAlert(rule, metadata, averageSecurityScorePercentage)
		h.logger.Info("Sending alert", zap.String("rule", fmt.Sprintf("%v", rule.Id)),
			zap.String("action", fmt.Sprintf("%v", rule.ActionID)))
		if err != nil {
			return fmt.Errorf("Error sending alert : %v ", err.Error())
		}
		err = h.db.UpdateRule(rule.Id, nil, nil, nil, nil, nil, api.TriggerStatus_Active)
		if err != nil {
			return fmt.Errorf("Error updating rule : %v ", err.Error())
		}
	} else if stat == false && status == api.TriggerStatus_Active {
		err = h.db.UpdateRule(rule.Id, nil, nil, nil, nil, nil, api.TriggerStatus_NotActive)
		if err != nil {
			return fmt.Errorf("Error updating rule : %v ", err.Error())
		}
	} else if status == api.Nil {
		if stat == true {
			err = h.db.UpdateRule(rule.Id, nil, nil, nil, nil, nil, api.TriggerStatus_Active)
			if err != nil {
				return fmt.Errorf("Error updating rule : %v ", err.Error())
			}
		} else if stat == false {
			err = h.db.UpdateRule(rule.Id, nil, nil, nil, nil, nil, api.TriggerStatus_NotActive)
			if err != nil {
				return fmt.Errorf("Error updating rule : %v ", err.Error())
			}
		}
	}
	return nil
}

func (h HttpHandler) sendAlert(rule Rule, metadata api.Metadata, averageSecurityScorePercentage int64) error {
	action, err := h.db.GetAction(rule.ActionID)
	if err != nil {
		return fmt.Errorf("error getting action : %v", err.Error())
	}

	tmpl, err := template.New("Metadata.Name").Parse(action.Body)
	if err != nil {
		return fmt.Errorf("error create template : %v", err)
	}
	var outputExecute bytes.Buffer
	err = tmpl.Execute(&outputExecute, metadata)
	if err != nil {
		return fmt.Errorf("error executing template : %v ", err)
	}

	action.Body = outputExecute.String()

	req, err := http.NewRequest(action.Method, action.Url, bytes.NewBuffer([]byte(action.Body)))
	if err != nil {
		return fmt.Errorf("error sending the request : %v", err.Error())
	}

	if len(action.Headers) > 0 {
		var headers map[string]string
		err = json.Unmarshal(action.Headers, &headers)
		if err != nil {
			return fmt.Errorf("error unmarshalling the headers : %v ", err.Error())
		}
		for k, v := range headers {
			req.Header.Add(k, v)
		}
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("error sending the alert request : %v ", err.Error())
	}
	defer res.Body.Close()

	err = h.addTriggerToDatabase(rule, averageSecurityScorePercentage, res.StatusCode)
	if err != nil {
		return err
	}

	err = res.Body.Close()
	if err != nil {
		return fmt.Errorf("error closing the response body : %v ", err.Error())
	}
	return nil
}

func (h HttpHandler) triggerInsight(operator api.Condition, eventType api.EventType, scope api.Scope) (bool, int64, error) {
	diff := 24 * time.Hour
	oneDayAgo := time.Now().Add(-diff)
	timeNow := time.Now()
	insightID := strconv.Itoa(int(*eventType.InsightId))
	connectionIds, err := h.getConnectionIdFilter(scope)
	if err != nil {
		return false, 0, fmt.Errorf("error getting connectionId : %v ", err.Error())
	}

	insight, _ := h.complianceClient.GetInsight(&httpclient.Context{UserRole: authApi.InternalRole}, insightID, connectionIds, &oneDayAgo, &timeNow)
	if err != nil {
		return false, 0, fmt.Errorf("error getting Insight : %v", err.Error())
	}
	if insight.TotalResultValue == nil {
		return false, 0, nil
	}

	fieldValue := map[string]int64{
		"insight_value": *insight.TotalResultValue,
	}

	stat, err := checkCondition(operator, fieldValue)
	if err != nil {
		return false, 0, fmt.Errorf("error calculating operator : %v", err.Error())
	}
	h.logger.Info("Insight rule operation done",
		zap.Bool("result", stat),
		zap.Int64("totalCount", *insight.TotalResultValue))
	return stat, *insight.TotalResultValue, nil
}

func (h HttpHandler) triggerCompliance(operator api.Condition, scope api.Scope, eventType api.EventType) (bool, int64, error) {
	connectionIds, err := h.getConnectionIdFilter(scope)
	if err != nil {
		return false, 0, fmt.Errorf("error getting connectionId : %v ", err.Error())
	}

	h.logger.Info("sending finding request",
		zap.String("request", fmt.Sprintf("benchmarkId : %v , connectionId : %v , connection group : %v , connector : %v  ", eventType.BenchmarkId, connectionIds, scope.ConnectionGroup, scope.Connector)))

	var connector []source.Type
	if scope.Connector == nil {
		connector = nil
	} else {
		connector = append(connector, *scope.Connector)
	}

	compliance, err := h.complianceClient.GetAccountsFindingsSummary(&httpclient.Context{UserRole: authApi.AdminRole}, *eventType.BenchmarkId, connectionIds, connector)
	if err != nil {
		return false, 0, fmt.Errorf("error getting AccountsFindingsSummary : %v", err)
	}

	var securityScore float64
	for _, account := range compliance.Accounts {
		securityScore += account.SecurityScore
	}

	h.logger.Info("received compliance account ")
	averageSecurityScore := securityScore / float64(len(compliance.Accounts))
	fmt.Printf("averageSecurityScore : %v \n ", int64(averageSecurityScore*100))

	fieldValue := map[string]int64{
		"security_score": int64(averageSecurityScore * 100),
	}

	stat, err := checkCondition(operator, fieldValue)
	fmt.Printf("stat : %v \n ", stat)

	if err != nil {
		return false, 0, fmt.Errorf("error in operation : %v ", err.Error())
	}
	h.logger.Info("Compliance rule operation done",
		zap.Bool("Result", stat),
		zap.Int64("Average security score percentage ", int64(averageSecurityScore*100)))

	return stat, int64(averageSecurityScore * 100), nil
}

func (h HttpHandler) addTriggerToDatabase(rule Rule, averageSecurityScorePercentage int64, responseStatusCode int) error {
	err := h.db.CreateTrigger(time.Now(), rule.Id, averageSecurityScorePercentage, responseStatusCode)
	if err != nil {
		return fmt.Errorf("error in add trigger to the database : %v ", err)
	}
	return nil
}

func (h HttpHandler) getConnectionIdFilter(scope api.Scope) ([]string, error) {
	if scope.ConnectionId == nil && scope.ConnectionGroup == nil && scope.Connector == nil {
		return nil, nil
	}

	if scope.ConnectionId != nil && scope.ConnectionGroup != nil {
		return nil, echo.NewHTTPError(http.StatusBadRequest, "connectionId and connectionGroup cannot be used together")
	}

	if scope.ConnectionId != nil {
		return []string{*scope.ConnectionId}, nil
	}
	check := make(map[string]bool)
	var connectionIDSChecked []string
	if scope.ConnectionGroup != nil {
		connectionGroupObj, err := h.onboardClient.GetConnectionGroup(&httpclient.Context{UserRole: authApi.InternalRole}, *scope.ConnectionGroup)
		if err != nil {
			return nil, fmt.Errorf("error getting connectionId : %v ", err.Error())
		}
		if len(connectionGroupObj.ConnectionIds) == 0 {
			return nil, nil
		}

		// Check for duplicate connection groups
		for _, entry := range connectionGroupObj.ConnectionIds {
			if _, value := check[entry]; !value {
				check[entry] = true
				connectionIDSChecked = append(connectionIDSChecked, entry)
			}
		}
	}

	return connectionIDSChecked, nil
}

func runOperation(field, operator, value string, fieldValue map[string]int64) (bool, error) {
	currValue, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return false, err
	}

	switch operator {
	case ">":
		return fieldValue[field] > currValue, nil
	case ">=":
		return fieldValue[field] >= currValue, nil
	case "<":
		return fieldValue[field] < currValue, nil
	case "<=":
		return fieldValue[field] <= currValue, nil
	case "=":
		return fieldValue[field] == currValue, nil
	case "!=":
		return fieldValue[field] != currValue, nil
	default:
		return false, fmt.Errorf("invalid operator %s", operator)
	}
}

func checkCondition(condition api.Condition, fieldValue map[string]int64) (bool, error) {
	if condition.Combinator == nil {
		return runOperation(condition.Field, condition.Operator, condition.Value, fieldValue)
	} else if *condition.Combinator == "and" {
		for _, rule := range condition.Rules {
			res, err := checkCondition(rule, fieldValue)
			if err != nil {
				return false, err
			}

			if res == false {
				return false, nil
			}
		}
	} else if *condition.Combinator == "or" {
		for _, rule := range condition.Rules {
			res, err := checkCondition(rule, fieldValue)
			if err != nil {
				return false, err
			}

			if res == true {
				return true, nil
			}
		}
	}
	return true, nil
}
