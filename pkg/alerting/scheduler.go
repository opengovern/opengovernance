package alerting

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/kaytu-io/kaytu-engine/pkg/alerting/api"
	authApi "github.com/kaytu-io/kaytu-engine/pkg/auth/api"
	"github.com/kaytu-io/kaytu-engine/pkg/internal/httpclient"
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

	var operator api.OperatorStruct
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

func (h HttpHandler) triggerInsight(operator api.OperatorStruct, eventType api.EventType, scope api.Scope) (bool, int64, error) {
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

	stat, err := calculationOperations(operator, *insight.TotalResultValue)
	if err != nil {
		return false, 0, fmt.Errorf("error calculating operator : %v", err.Error())
	}
	h.logger.Info("Insight rule operation done",
		zap.Bool("result", stat),
		zap.Int64("totalCount", *insight.TotalResultValue))
	return stat, *insight.TotalResultValue, nil
}

func (h HttpHandler) triggerCompliance(operator api.OperatorStruct, scope api.Scope, eventType api.EventType) (bool, int64, error) {
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
	stat, err := calculationOperations(operator, int64(averageSecurityScore*100))
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
	eventTypeM, err := json.Marshal(rule.EventType)
	if err != nil {
		return fmt.Errorf("error in marshalling eventType : %v ", err)
	}

	scopeM, err := json.Marshal(rule.Scope)
	if err != nil {
		return fmt.Errorf("error in marshalling scope : %v ", err)
	}

	err = h.db.CreateTrigger(time.Now(), eventTypeM, scopeM, averageSecurityScorePercentage, responseStatusCode)
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
		connectionGroupObj, err := h.onboardClient.GetConnectionGroup(&httpclient.Context{UserRole: authApi.KaytuAdminRole}, *scope.ConnectionGroup)
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

func calculationOperations(operator api.OperatorStruct, averageSecurityScorePercentage int64) (bool, error) {
	if operator.Condition == nil {
		fmt.Println("test operation 1 ")
		if operator.OperatorType == ">" {
			operator.OperatorType = api.OperatorGreaterThan
		} else if operator.OperatorType == "<" {
			operator.OperatorType = api.OperatorLessThan
		} else if operator.OperatorType == ">=" {
			operator.OperatorType = api.OperatorGreaterThanOrEqual
		} else if operator.OperatorType == "<=" {
			operator.OperatorType = api.OperatorLessThanOrEqual
		} else if operator.OperatorType == "=" {
			operator.OperatorType = api.OperatorEqual
		} else if operator.OperatorType == "!=" {
			operator.OperatorType = api.OperatorDoesNotEqual
		} else {
			return false, fmt.Errorf("Error : Your operator sign is wrong , please enter the correct operator ")
		}
		stat := compareValue(operator.OperatorType, operator.Value, averageSecurityScorePercentage)
		return stat, nil
	} else if operator.Condition != nil {

		fmt.Println("test operation 2 ")
		stat, err := calculationConditionStr(operator, averageSecurityScorePercentage)
		if err != nil {
			return false, fmt.Errorf("error in calculation operator : %v ", err.Error())
		}
		return stat, nil
	}
	return false, fmt.Errorf("error entering the operation")
}

func calculationConditionStr(operator api.OperatorStruct, averageSecurityScorePercentage int64) (bool, error) {
	conditionType := operator.Condition.ConditionType

	if conditionType == api.ConditionAnd || conditionType == api.ConditionAndLowerCase {
		stat, err := calculationConditionStrAND(operator, averageSecurityScorePercentage)
		if err != nil {
			return false, fmt.Errorf("error in AND condition type : %v", err)
		}
		return stat, nil

	} else if conditionType == api.ConditionOr || conditionType == api.ConditionOrLowerCase {
		stat, err := calculationConditionStrOr(operator, averageSecurityScorePercentage)
		if err != nil {
			return false, fmt.Errorf("error in OR condition type : %v", err)
		}
		return stat, nil
	}

	return false, fmt.Errorf("please enter right condition type ")
}

func calculationConditionStrAND(operator api.OperatorStruct, averageSecurityScorePercentage int64) (bool, error) {
	// AND condition
	numberOperatorStr := len(operator.Condition.Operator)
	for i := 0; i < numberOperatorStr; i++ {
		newOperator := operator.Condition.Operator[i]

		if newOperator.Condition == nil {
			if newOperator.OperatorType == ">" {
				newOperator.OperatorType = api.OperatorGreaterThan
			} else if newOperator.OperatorType == "<" {
				newOperator.OperatorType = api.OperatorLessThan
			} else if operator.OperatorType == ">=" {
				newOperator.OperatorType = api.OperatorGreaterThanOrEqual
			} else if operator.OperatorType == "<=" {
				newOperator.OperatorType = api.OperatorLessThanOrEqual
			} else if newOperator.OperatorType == "=" {
				newOperator.OperatorType = api.OperatorEqual
			} else if newOperator.OperatorType == "!=" {
				newOperator.OperatorType = api.OperatorDoesNotEqual
			} else {
				return false, fmt.Errorf("Error : Your operator type is wrong , please enter the correct operator ")
			}
			stat := compareValue(newOperator.OperatorType, newOperator.Value, averageSecurityScorePercentage)
			if !stat {
				return false, nil
			} else {
				if i == numberOperatorStr-1 {
					return true, nil
				}
				continue
			}

		} else if newOperator.Condition.ConditionType != "" {
			newOperator2 := newOperator.Condition
			conditionType2 := newOperator2.ConditionType
			numberOperatorStr2 := len(newOperator2.Operator)

			for j := 0; j < numberOperatorStr2; j++ {
				stat, err := calculationOperations(newOperator2.Operator[j], averageSecurityScorePercentage)
				if err != nil {
					return false, fmt.Errorf("error in calculation operations : %v ", err.Error())
				}

				if conditionType2 == api.ConditionAnd || conditionType2 == api.ConditionAndLowerCase {
					if !stat {
						return false, nil
					} else {
						if j == numberOperatorStr2-1 {
							if i == numberOperatorStr-1 {
								return true, nil
							}
							break
						}
						continue
					}
				} else if conditionType2 == api.ConditionOr || conditionType2 == api.ConditionOrLowerCase {
					if stat {
						return true, nil
					} else {
						if j == numberOperatorStr2-1 {
							if i == numberOperatorStr-1 {
								return false, nil
							}
							break
						}
						continue
					}
				} else {
					return false, fmt.Errorf("error: condition type is invalid")
				}

			}
			continue
		} else {
			return false, fmt.Errorf("error : condition is is invalid")
		}
	}
	return false, fmt.Errorf("error")
}

func calculationConditionStrOr(operator api.OperatorStruct, averageSecurityScorePercentage int64) (bool, error) {
	// OR condition
	numberOperatorStr := len(operator.Condition.Operator)

	for i := 0; i < numberOperatorStr; i++ {
		newOperator := operator.Condition.Operator[i]

		if newOperator.Condition == nil {
			if newOperator.OperatorType == ">" {
				newOperator.OperatorType = api.OperatorGreaterThan
			} else if newOperator.OperatorType == "<" {
				newOperator.OperatorType = api.OperatorLessThan
			} else if newOperator.OperatorType == ">=" {
				newOperator.OperatorType = api.OperatorGreaterThanOrEqual
			} else if newOperator.OperatorType == "<=" {
				newOperator.OperatorType = api.OperatorLessThanOrEqual
			} else if newOperator.OperatorType == "=" {
				newOperator.OperatorType = api.OperatorEqual
			} else if newOperator.OperatorType == "!=" {
				newOperator.OperatorType = api.OperatorDoesNotEqual
			} else {
				return false, fmt.Errorf("Error : Your operator type is wrong , please enter the correct operator type ")
			}
			stat := compareValue(newOperator.OperatorType, newOperator.Value, averageSecurityScorePercentage)
			if stat {
				return true, nil
			} else {
				if i == numberOperatorStr-1 {
					return false, nil
				}
				continue
			}

		} else if newOperator.Condition.Operator != nil {
			newOperator2 := newOperator.Condition
			conditionType2 := newOperator2.ConditionType
			numberConditionStr2 := len(newOperator2.ConditionType)

			for j := 0; j < numberConditionStr2; j++ {
				stat, err := calculationOperations(newOperator2.Operator[j], averageSecurityScorePercentage)
				if err != nil {
					return false, fmt.Errorf("error in calculation operations : %v ", err)
				}

				if conditionType2 == api.ConditionAnd {
					if !stat {
						return false, nil
					} else {
						if j == numberConditionStr2-1 {
							if i == numberOperatorStr-1 {
								return true, nil
							}
							break
						}
						continue
					}
				} else if conditionType2 == api.ConditionOr || conditionType2 == api.ConditionOrLowerCase {
					if stat {
						return true, nil
					} else {
						if j == numberConditionStr2-1 {
							if i == numberOperatorStr-1 {
								return false, nil
							}
							break
						}
						continue
					}
				} else {
					return false, fmt.Errorf("error: condition type is invalid")
				}

			}
			continue
		} else {
			return false, fmt.Errorf("error : condition is invalid ")
		}

	}
	return false, fmt.Errorf("error")
}

func compareValue(operator api.OperatorType, value int64, averageSecurityScorePercentage int64) bool {
	switch operator {
	case api.OperatorGreaterThan:
		if averageSecurityScorePercentage > value {
			return true
		}
	case api.OperatorLessThan:
		if averageSecurityScorePercentage < value {
			return true
		}
	case api.OperatorGreaterThanOrEqual:
		if averageSecurityScorePercentage >= value {
			return true
		}
	case api.OperatorLessThanOrEqual:
		if averageSecurityScorePercentage <= value {
			return true
		}
	case api.OperatorEqual:
		if averageSecurityScorePercentage == value {
			return true
		}
	case api.OperatorDoesNotEqual:
		if averageSecurityScorePercentage != value {
			return true
		}
	default:
		return false
	}
	return false
}
