package alerting

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/kaytu-io/kaytu-engine/pkg/alerting/api"
	authApi "github.com/kaytu-io/kaytu-engine/pkg/auth/api"
	apiCompliance "github.com/kaytu-io/kaytu-engine/pkg/compliance/api"
	"github.com/kaytu-io/kaytu-engine/pkg/internal/httpclient"
	"github.com/kaytu-io/kaytu-util/pkg/source"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
	"net/http"
	"strconv"
	"time"
)

func (h *HttpHandler) TriggerRulesJobCycle() {
	timer := time.NewTicker(30 * time.Minute)
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

	stat := false
	var totalCount int64
	if eventType.InsightId != nil {
		h.logger.Info("triggering insight", zap.String("rule", fmt.Sprintf("%v", rule.Id)))
		stat, totalCount, err = h.triggerInsight(operator, eventType, scope)
		if err != nil {
			return fmt.Errorf("error triggering the insight : %v ", err.Error())
		}
	} else if eventType.BenchmarkId != nil {
		h.logger.Info("triggering compliance", zap.String("rule", fmt.Sprintf("%v", rule.Id)))
		stat, totalCount, err = h.triggerCompliance(operator, scope, eventType)
		if err != nil {
			return fmt.Errorf("Error in trigger compliance : %v ", err.Error())
		}
	} else {
		return fmt.Errorf("Error: insighId or complianceId not entered ")
	}
	if stat {
		err = h.sendAlert(rule, totalCount)
		h.logger.Info("Sending alert", zap.String("rule", fmt.Sprintf("%v", rule.Id)),
			zap.String("action", fmt.Sprintf("%v", rule.ActionID)))
		if err != nil {
			return fmt.Errorf("Error sending alert : %v ", err.Error())
		}
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

	filters := apiCompliance.FindingFilters{BenchmarkID: []string{*eventType.BenchmarkId}}
	if len(connectionIds) > 0 {
		filters.ConnectionID = connectionIds
	}
	if scope.Connector != nil {
		filters.Connector = []source.Type{*scope.Connector}
	}
	reqCompliance := apiCompliance.GetFindingsRequest{
		Filters: filters,
		Sorts: []apiCompliance.FindingSortItem{{Field: apiCompliance.FieldResourceID,
			Direction: apiCompliance.DirectionAscending}},
		Page: apiCompliance.Page{No: 100, Size: 100},
	}
	h.logger.Info("sending finding request",
		zap.String("request", fmt.Sprintf("%v", reqCompliance)))
	compliance, err := h.complianceClient.GetFindings(&httpclient.Context{UserRole: authApi.AdminRole}, reqCompliance)
	if err != nil {
		return false, 0, fmt.Errorf("error getting finding : %v ", err.Error())
	}
	h.logger.Info("received findings")
	stat, err := calculationOperations(operator, compliance.TotalCount)
	if err != nil {
		return false, 0, fmt.Errorf("error in rule operations : %v ", err.Error())
	}

	h.logger.Info("Compliance rule operation done",
		zap.Bool("result", stat),
		zap.Int64("totalCount", compliance.TotalCount))
	return stat, compliance.TotalCount, nil
}

func (h HttpHandler) sendAlert(rule Rule, TotalCount int64) error {
	action, err := h.db.GetAction(rule.ActionID)
	if err != nil {
		return fmt.Errorf("error getting action : %v", err.Error())
	}

	var eventType api.EventType
	err = json.Unmarshal(rule.EventType, &eventType)
	if err != nil {
		return fmt.Errorf("error unmarshalling the eventType : %v", err.Error())
	}

	var scope api.Scope
	err = json.Unmarshal(rule.Scope, &scope)
	if err != nil {
		return fmt.Errorf("error unmarshalling the scope : %v ", err.Error())
	}

	req, err := http.NewRequest(action.Method, action.Url, bytes.NewBuffer([]byte(action.Body)))
	if err != nil {
		return fmt.Errorf("error sending the request : %v", err.Error())
	}

	var headers map[string]string
	err = json.Unmarshal(action.Headers, &headers)
	if err != nil {
		return fmt.Errorf("error unmarshalling the headers : %v ", err.Error())
	}
	for k, v := range headers {
		req.Header.Add(k, v)
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("error sending the alert request : %v ", err.Error())
	}

	if eventType.BenchmarkId != nil {
		err = h.db.CreateComplianceTrigger(time.Now(), *eventType.BenchmarkId, *scope.ConnectionId, TotalCount, res.StatusCode)
		if err != nil {
			return fmt.Errorf("error in create compliance trigger : %v ", err)
		}
	} else if eventType.InsightId != nil {
		err = h.db.CreateInsightTrigger(time.Now(), *eventType.InsightId, *scope.ConnectionId, TotalCount, res.StatusCode)
		if err != nil {
			return fmt.Errorf("error in create compliance trigger : %v ", err)
		}
	}

	err = res.Body.Close()
	if err != nil {
		return fmt.Errorf("error closing the response body : %v ", err.Error())
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

func calculationOperations(operator api.OperatorStruct, totalValue int64) (bool, error) {
	if oneCondition := operator.OperatorInfo; oneCondition != nil {
		stat := compareValue(oneCondition.OperatorType, oneCondition.Value, totalValue)
		return stat, nil
	} else if operator.Condition != nil {
		stat, err := calculationConditionStr(operator, totalValue)
		if err != nil {
			return false, fmt.Errorf("error in calculation operator : %v ", err.Error())
		}
		return stat, nil
	}
	return false, fmt.Errorf("error entering the operation")
}

func calculationConditionStrAND(operator api.OperatorStruct, totalValue int64) (bool, error) {
	// AND condition
	numberOperatorStr := len(operator.Condition.Operator)
	for i := 0; i < numberOperatorStr; i++ {
		newOperator := operator.Condition.Operator[i]

		if newOperator.OperatorInfo != nil {
			stat := compareValue(newOperator.OperatorInfo.OperatorType, newOperator.OperatorInfo.Value, totalValue)
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
				stat, err := calculationOperations(newOperator2.Operator[j], totalValue)
				if err != nil {
					return false, fmt.Errorf("error in calculationOperations : %v ", err.Error())
				}

				if conditionType2 == api.ConditionAnd {
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
				} else if conditionType2 == api.ConditionOr {
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
			return false, fmt.Errorf("error condition is impty")
		}
	}
	return false, fmt.Errorf("error")
}

func calculationConditionStr(operator api.OperatorStruct, totalValue int64) (bool, error) {
	conditionType := operator.Condition.ConditionType

	if conditionType == api.ConditionAnd {
		stat, err := calculationConditionStrAND(operator, totalValue)
		if err != nil {
			return false, err
		}
		return stat, nil

	} else if conditionType == api.ConditionOr {
		stat, err := calculationConditionStrOr(operator, totalValue)
		if err != nil {
			return false, err
		}
		return stat, nil
	}

	return false, fmt.Errorf("please enter right condition")
}

func calculationConditionStrOr(operator api.OperatorStruct, totalValue int64) (bool, error) {
	// OR condition
	numberOperatorStr := len(operator.Condition.Operator)

	for i := 0; i < numberOperatorStr; i++ {
		newOperator := operator.Condition.Operator[i]

		if newOperator.OperatorInfo != nil {
			stat := compareValue(newOperator.OperatorInfo.OperatorType, newOperator.OperatorInfo.Value, totalValue)
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
				stat, err := calculationOperations(newOperator2.Operator[j], totalValue)
				if err != nil {
					return false, fmt.Errorf("error in calculationOperations : %v ", err)
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
				} else if conditionType2 == api.ConditionOr {
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

func compareValue(operator api.OperatorType, value int64, totalValue int64) bool {
	switch operator {
	case api.OperatorGreaterThan:
		if totalValue > value {
			return true
		}
	case api.OperatorLessThan:
		if totalValue < value {
			return true
		}
	case api.OperatorGreaterThanOrEqual:
		if totalValue >= value {
			return true
		}
	case api.OperatorLessThanOrEqual:
		if totalValue <= value {
			return true
		}
	case api.OperatorEqual:
		if totalValue == value {
			return true
		}
	case api.OperatorDoesNotEqual:
		if totalValue != value {
			return true
		}
	default:
		return false
	}
	return false
}
