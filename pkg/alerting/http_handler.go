package alerting

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/kaytu-io/kaytu-engine/pkg/alerting/api"
	api2 "github.com/kaytu-io/kaytu-engine/pkg/auth/api"
	authApi "github.com/kaytu-io/kaytu-engine/pkg/auth/api"
	apiCompliance "github.com/kaytu-io/kaytu-engine/pkg/compliance/api"
	"github.com/kaytu-io/kaytu-engine/pkg/compliance/client"
	"github.com/kaytu-io/kaytu-engine/pkg/internal/httpclient"
	onboardClient "github.com/kaytu-io/kaytu-engine/pkg/onboard/client"
	"github.com/kaytu-io/kaytu-util/pkg/postgres"
	"github.com/kaytu-io/kaytu-util/pkg/source"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
	"net/http"
	"strconv"
	"time"
)

type HttpHandler struct {
	db               Database
	onboardClient    onboardClient.OnboardServiceClient
	complianceClient client.ComplianceServiceClient
}

func InitializeHttpHandler(
	postgresHost string, postgresPort string, postgresDb string, postgresUsername string, postgresPassword string, postgresSSLMode string,
	logger *zap.Logger,
) (h *HttpHandler, err error) {

	fmt.Println("Initializing http handler")

	cfg := postgres.Config{
		Host:    postgresHost,
		Port:    postgresPort,
		User:    postgresUsername,
		Passwd:  postgresPassword,
		DB:      postgresDb,
		SSLMode: postgresSSLMode,
	}
	orm, err := postgres.NewClient(&cfg, logger)
	if err != nil {
		return nil, fmt.Errorf("new postgres client: %w", err)
	}
	fmt.Println("Connected to the postgres database: ", postgresDb)

	db := NewDatabase(orm)
	err = db.Initialize()
	if err != nil {
		return nil, err
	}
	fmt.Println("Initialized postgres database: ", postgresDb)

	return &HttpHandler{
		db: db,
	}, nil
}

func (h HttpHandler) TriggerLoop() {
	timer := time.NewTicker(30 * time.Minute)
	defer timer.Stop()

	for ; ; <-timer.C {
		Trigger(h)
	}
}

func Trigger(h HttpHandler) {
	rules, err := h.db.ListRules()
	if err != nil {
		fmt.Printf("Error in giving list rules error equal to : %v", err)
		return
	}

	for _, rule := range rules {
		var scope api.Scope
		err := json.Unmarshal(rule.Scope, &scope)
		if err != nil {
			fmt.Printf("Error in unmarshaling scope , error  equal to : %v", err)
			return
		}

		var eventType api.EventType
		err = json.Unmarshal(rule.EventType, &eventType)
		if err != nil {
			fmt.Printf("Error in unmarshaling event type , error equal to : %v ", err)
			return
		}

		var operator api.OperatorStruct
		err = json.Unmarshal(rule.Operator, &operator)
		if err != nil {
			fmt.Printf("Error in unmarshaling operator , error equal to : %v ", err)
			return
		}

		if eventType.InsightId != nil {
			statInsight, err := triggerInsight(h, operator, eventType, scope)
			if err != nil {
				fmt.Printf("Error in trigger insight : %v", err)
			}
			if statInsight {
				err = sendAlert(h, rule)
				if err != nil {
					fmt.Printf("Error in send alert for insigh , err : %v ", err)
					return
				}
			}
		} else if eventType.BenchmarkId != nil {
			statCompliance, err := triggerCompliance(h, operator, scope, eventType)
			if err != nil {
				fmt.Printf("Error in trigger compliance : %v ", err)
			}
			if statCompliance {
				err = sendAlert(h, rule)
				if err != nil {
					fmt.Printf("Error in send alert for compliance , err : %v ", err)
					return
				}
			}
		} else {
			fmt.Printf("Error: insighId or complianceId not entered ")
		}
	}
}

func getConnectionIdFilter(h HttpHandler, connectionIds []string, connectionGroup []string) ([]string, error) {
	if len(connectionIds[0]) == 0 && len(connectionGroup[0]) == 0 {
		return nil, nil
	}

	if len(connectionIds[0]) > 0 && len(connectionGroup[0]) > 0 {
		return nil, echo.NewHTTPError(http.StatusBadRequest, "connectionId and connectionGroup cannot be used together")
	}

	if len(connectionIds) > 0 {
		return connectionIds, nil
	}
	check := make(map[string]bool)
	var connectionIDSChecked []string

	for i := 0; i < len(connectionGroup); i++ {
		connectionGroupObj, err := h.onboardClient.GetConnectionGroup(&httpclient.Context{UserRole: authApi.KaytuAdminRole}, connectionGroup[i])
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
	}
	connectionIds = connectionIDSChecked

	return connectionIds, nil
}

func sendAlert(h HttpHandler, rule Rule) error {
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

func triggerInsight(h HttpHandler, operator api.OperatorStruct, eventType api.EventType, scope api.Scope) (bool, error) {
	diff := 24 * time.Hour
	oneDayAgo := time.Now().Add(-diff)
	timeNow := time.Now()
	insightID := strconv.Itoa(int(*eventType.InsightId))
	connectionIds, err := getConnectionIdFilter(h, []string{scope.ConnectionId}, []string{scope.ConnectionGroup})
	if err != nil {
		return false, err
	}

	insight, _ := h.complianceClient.GetInsight(&httpclient.Context{UserRole: api2.InternalRole}, insightID, connectionIds, &oneDayAgo, &timeNow)
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
	if !stat {
		return false, nil
	}
	return true, nil
}

func triggerCompliance(h HttpHandler, operator api.OperatorStruct, scope api.Scope, eventType api.EventType) (bool, error) {
	connectionIds, err := getConnectionIdFilter(h, []string{scope.ConnectionId}, []string{scope.ConnectionGroup})
	if err != nil {
		return false, err
	}
	reqCompliance := apiCompliance.GetFindingsRequest{
		Filters: apiCompliance.FindingFilters{ConnectionID: connectionIds, BenchmarkID: []string{*eventType.BenchmarkId}, Connector: []source.Type{scope.ConnectorName}},
		Page:    apiCompliance.Page{No: 1, Size: 1},
	}
	compliance, err := h.complianceClient.GetFindings(&httpclient.Context{UserRole: api2.InternalRole}, reqCompliance)
	if err != nil {
		return false, fmt.Errorf("error getting compliance , err : %v ", err)
	}

	stat, err := calculationOperations(operator, compliance.TotalCount)
	if err != nil {
		return false, err
	}
	if !stat {
		return false, nil
	}
	return true, nil
}

func calculationOperations(operator api.OperatorStruct, totalValue int64) (bool, error) {
	if oneCondition := operator.OperatorInfo; oneCondition != nil {
		stat := compareValue(oneCondition.Operator, oneCondition.Value, totalValue)
		return stat, nil

	} else if operator.ConditionStr != nil {
		stat, err := calculationConditionStr(operator, totalValue)
		if err != nil {
			return false, err
		}
		return stat, nil
	}
	return false, fmt.Errorf("error entering the operation")
}

func calculationConditionStr(operator api.OperatorStruct, totalValue int64) (bool, error) {
	conditionType := operator.ConditionStr.ConditionType

	if conditionType == "AND" {
		stat, err := calculationConditionStrAND(operator, totalValue)
		if err != nil {
			return false, err
		}
		return stat, nil

	} else if conditionType == "OR" {
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
	numberOperatorStr := len(operator.ConditionStr.OperatorStr)
	for i := 0; i < numberOperatorStr; i++ {
		newOperator := operator.ConditionStr.OperatorStr[i]

		if newOperator.OperatorInfo != nil {
			stat := compareValue(newOperator.OperatorInfo.Operator, newOperator.OperatorInfo.Value, totalValue)
			if !stat {
				return false, nil
			} else {
				if i == numberOperatorStr-1 {
					return true, nil
				}
				continue
			}

		} else if newOperator.ConditionStr.ConditionType != "" {
			newOperator2 := newOperator.ConditionStr
			conditionType2 := newOperator2.ConditionType
			numberOperatorStr2 := len(newOperator2.OperatorStr)

			for j := 0; j < numberOperatorStr2; j++ {
				stat, err := calculationOperations(newOperator2.OperatorStr[j], totalValue)
				if err != nil {
					return false, fmt.Errorf("error in calculationOperations : %v ", err)
				}

				if conditionType2 == "AND" {
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
				} else if conditionType2 == "OR" {
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
	numberOperatorStr := len(operator.ConditionStr.OperatorStr)

	for i := 0; i < numberOperatorStr; i++ {
		newOperator := operator.ConditionStr.OperatorStr[i]

		if newOperator.OperatorInfo != nil {
			stat := compareValue(newOperator.OperatorInfo.Operator, newOperator.OperatorInfo.Value, totalValue)
			if stat {
				return true, nil
			} else {
				if i == numberOperatorStr-1 {
					return false, nil
				}
				continue
			}

		} else if newOperator.ConditionStr.OperatorStr != nil {
			newOperator2 := newOperator.ConditionStr
			conditionType2 := newOperator2.ConditionType
			numberConditionStr2 := len(newOperator2.OperatorStr)

			for j := 0; j < numberConditionStr2; j++ {
				stat, err := calculationOperations(newOperator2.OperatorStr[j], totalValue)
				if err != nil {
					return false, fmt.Errorf("error in calculationOperations : %v ", err)
				}

				if conditionType2 == "AND" {
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
				} else if conditionType2 == "OR" {
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

func compareValue(operator string, value int64, totalValue int64) bool {
	switch operator {
	case ">":
		if totalValue > value {
			return true
		}
	case "<":
		if totalValue < value {
			return true
		}
	case ">=":
		if totalValue >= value {
			return true
		}
	case "<=":
		if totalValue <= value {
			return true
		}
	case "=":
		if totalValue == value {
			return true
		}
	case "!=":
		if totalValue != value {
			return true
		}
	default:
		return false
	}
	return false
}
