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
		_ = h.TriggerRulesList
	}
}

func (h *HttpHandler) TriggerRulesList() error {
	rules, err := h.db.ListRules()
	if err != nil {
		fmt.Printf("Error in giving list rules error equal to : %v", err)
		return err
	}

	for _, rule := range rules {
		err := h.TriggerRule(rule)
		if err != nil {
			h.logger.Error("Rule trigger has been failed",
				zap.String("rule id", fmt.Sprintf("%v", rule.ID)),
				zap.Error(err))
		}
	}
	return nil
}

func (h *HttpHandler) TriggerRule(rule Rule) error {
	var scope api.Scope
	err := json.Unmarshal(rule.Scope, &scope)
	if err != nil {
		return err
	}

	var eventType api.EventType
	err = json.Unmarshal(rule.EventType, &eventType)
	if err != nil {
		return err
	}

	var operator api.OperatorStruct
	err = json.Unmarshal(rule.Operator, &operator)
	if err != nil {
		return err
	}

	if eventType.InsightId != nil {
		statInsight, err := h.triggerInsight(operator, eventType, scope)
		if err != nil {
			return err
		}
		if statInsight {
			err = h.sendAlert(rule)
			h.logger.Info("Sending alert", zap.String("rule", fmt.Sprintf("%v", rule.ID)),
				zap.String("action", fmt.Sprintf("%v", rule.ActionID)))
			if err != nil {
				return err
			}
		}
	} else if eventType.BenchmarkId != nil {
		statCompliance, err := h.triggerCompliance(operator, scope, eventType)
		if err != nil {
			fmt.Printf("Error in trigger compliance : %v ", err)
		}
		if statCompliance {
			err = h.sendAlert(rule)
			h.logger.Info("Sending alert", zap.String("rule", fmt.Sprintf("%v", rule.ID)),
				zap.String("action", fmt.Sprintf("%v", rule.ActionID)))
			if err != nil {
				return err
			}
		}
	} else {
		return fmt.Errorf("Error: insighId or complianceId not entered ")
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

	connectionGroupObj, err := h.onboardClient.GetConnectionGroup(&httpclient.Context{UserRole: authApi.KaytuAdminRole}, *scope.ConnectionGroup)
	if err != nil {
		return nil, err
	}
	if len(connectionGroupObj.ConnectionIds) == 0 {
		return nil, err
	}

	// Check for duplicate connection groups
	for _, entry := range connectionGroupObj.ConnectionIds {
		if _, value := check[entry]; !value {
			check[entry] = true
			connectionIDSChecked = append(connectionIDSChecked, entry)
		}
	}

	return connectionIDSChecked, nil
}

func (h HttpHandler) sendAlert(rule Rule) error {
	action, err := h.db.GetAction(rule.ActionID)
	if err != nil {
		return fmt.Errorf("error getting action , error equal to : %v", err)
	}
	req, err := http.NewRequest(action.Method, action.Url, bytes.NewBuffer([]byte(action.Body)))
	if err != nil {
		return fmt.Errorf("error sending the request , error equal to : %v", err)
	}

	var headers map[string]string
	err = json.Unmarshal(action.Headers, &headers)
	if err != nil {
		return fmt.Errorf("error unmarshalling the headers  , error : %v", err)
	}
	for k, v := range headers {
		req.Header.Add(k, v)
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	err = res.Body.Close()
	if err != nil {
		return err
	}
	return nil
}

func (h HttpHandler) triggerInsight(operator api.OperatorStruct, eventType api.EventType, scope api.Scope) (bool, error) {
	diff := 24 * time.Hour
	oneDayAgo := time.Now().Add(-diff)
	timeNow := time.Now()
	insightID := strconv.Itoa(int(*eventType.InsightId))
	connectionIds, err := h.getConnectionIdFilter(scope)
	if err != nil {
		return false, err
	}

	insight, _ := h.complianceClient.GetInsight(&httpclient.Context{UserRole: authApi.InternalRole}, insightID, connectionIds, &oneDayAgo, &timeNow)
	if err != nil {
		return false, fmt.Errorf("error in getting GetInsight , error  equal to : %v", err)
	}
	if insight.TotalResultValue == nil {
		return false, nil
	}

	stat, err := calculationOperations(operator, *insight.TotalResultValue)
	if err != nil {
		return false, err
	}
	h.logger.Info("Insight rule operation done",
		zap.Bool("result", stat),
		zap.Int64("totalCount", *insight.TotalResultValue))
	return stat, nil
}

func (h HttpHandler) triggerCompliance(operator api.OperatorStruct, scope api.Scope, eventType api.EventType) (bool, error) {
	connectionIds, err := h.getConnectionIdFilter(scope)
	if err != nil {
		return false, err
	}
	filters := apiCompliance.FindingFilters{ConnectionID: connectionIds, BenchmarkID: []string{*eventType.BenchmarkId}}
	if scope.Connector != nil {
		filters.Connector = []source.Type{*scope.Connector}
	}
	reqCompliance := apiCompliance.GetFindingsRequest{
		Filters: filters,
		Page:    apiCompliance.Page{No: 1, Size: 1},
	}
	compliance, err := h.complianceClient.GetFindings(&httpclient.Context{UserRole: authApi.InternalRole}, reqCompliance)
	if err != nil {
		return false, fmt.Errorf("error getting compliance , err : %v ", err)
	}
	stat, err := calculationOperations(operator, compliance.TotalCount)
	if err != nil {
		return false, err
	}
	h.logger.Info("Insight rule operation done",
		zap.Bool("result", stat),
		zap.Int64("totalCount", compliance.TotalCount))
	return stat, nil
}

func calculationOperations(operator api.OperatorStruct, totalValue int64) (bool, error) {
	if oneCondition := operator.OperatorInfo; oneCondition != nil {
		stat := compareValue(oneCondition.OperatorType, oneCondition.Value, totalValue)
		return stat, nil

	} else if operator.Condition != nil {
		stat, err := calculationConditionStr(operator, totalValue)
		if err != nil {
			return false, err
		}
		return stat, nil
	}
	return false, fmt.Errorf("error entering the operation")
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
					return false, fmt.Errorf("error in calculationOperations : %v ", err)
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
