package alerting

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/kaytu-io/kaytu-engine/pkg/alerting/api"
	api2 "github.com/kaytu-io/kaytu-engine/pkg/auth/api"
	apiCompliance "github.com/kaytu-io/kaytu-engine/pkg/compliance/api"
	"github.com/kaytu-io/kaytu-engine/pkg/compliance/client"
	"github.com/kaytu-io/kaytu-engine/pkg/internal/httpclient"
	"github.com/kaytu-io/kaytu-util/pkg/postgres"
	"go.uber.org/zap"
	"net/http"
	"strconv"
	"time"
)

type HttpHandler struct {
	db               Database
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
		rules, err := h.db.ListRules()
		if err != nil {
			fmt.Printf("error in giving list rules error equal to : %v", err)
			return
		}

		for _, rule := range rules {
			var scope api.Scope
			err := json.Unmarshal(rule.Scope, &scope)
			if err != nil {
				fmt.Printf("error in unmarshaling scope , error  equal to : %v", err)
				return
			}

			var eventType api.EventType
			err = json.Unmarshal(rule.EventType, &eventType)
			if err != nil {
				fmt.Printf("error in unmarshaling event type , error equal to : %v ", err)
				return
			}
			var operator api.OperatorStruct
			err = json.Unmarshal(rule.Operator, &operator)
			if err != nil {
				fmt.Printf("error in unmarshaling operator , error equal to : %v ", err)
				return
			}

			statInsight, err := triggerInsight(h, rule, operator, eventType, scope)
			if err != nil {
				fmt.Printf("error in trigger insight : %v", err)
			}
			if statInsight {
				err = sendAlert(h, rule)
				if err != nil {
					fmt.Printf("error in send alert for insigh , err : %v ", err)
					return
				}
			}

			statCompliance, err := triggerCompliance(h, rule, operator, scope, eventType)
			if err != nil {
				fmt.Printf("error in trigger compliance : %v ", err)
			}
			if statCompliance {
				err = sendAlert(h, rule)
				if err != nil {
					fmt.Printf("error in send alert for compliance , err : %v ", err)
					return
				}
			}
		}
	}
}

func sendAlert(h HttpHandler, rule Rule) error {
	var action Action
	action, err := h.db.GetAction(rule.ActionID)
	if err != nil {
		fmt.Printf("error in getting action , error equal to : %v", err)
	}

	req, err := http.NewRequest(action.Method, action.Url, bytes.NewBuffer([]byte(action.Body)))
	if err != nil {
		return fmt.Errorf("error in sending the request , error equal to : %v", err)
	}
	var headers map[string]string
	err = json.Unmarshal(action.Headers, &headers)
	if err != nil {
		return fmt.Errorf("error in unmarshaling the headers  , error : %v", err)
	}
	for k, v := range headers {
		req.Header.Add(k, v)
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("error equal to : %v", err)
	}

	err = res.Body.Close()
	if err != nil {
		return fmt.Errorf("error equal to : %v", err)
	}
	return nil
}

func triggerInsight(h HttpHandler, rule Rule, operator api.OperatorStruct, eventType api.EventType, scope api.Scope) (bool, error) {
	diff := 24 * time.Hour
	oneDayAgo := time.Now().Add(-diff)
	timeNow := time.Now()
	insightID := strconv.Itoa(int(eventType.InsightId))
	insight, err := h.complianceClient.GetInsight(&httpclient.Context{UserRole: api2.InternalRole}, insightID, []string{scope.ConnectionId}, &oneDayAgo, &timeNow)
	if err != nil {
		return false, fmt.Errorf("error in getting GetInsight , error  equal to : %v", err)
	}
	if insight.TotalResultValue == nil {
		return false, nil
	}

	stat, err := calculationOperations(operator, int(rule.Value), int(*insight.TotalResultValue))
	if err != nil {
		return false, err
	}
	if !stat {
		return false, nil
	}
	return true, nil
}

func triggerCompliance(h HttpHandler, rule Rule, operator api.OperatorStruct, scope api.Scope, eventType api.EventType) (bool, error) {
	reqCompliance := apiCompliance.GetFindingsRequest{
		Filters: apiCompliance.FindingFilters{ConnectionID: []string{scope.ConnectionId}, BenchmarkID: []string{eventType.BenchmarkId}},
		Page:    apiCompliance.Page{No: 1, Size: 1},
	}
	compliance, err := h.complianceClient.GetFindings(&httpclient.Context{UserRole: api2.InternalRole}, reqCompliance)
	if err != nil {
		return false, fmt.Errorf("error getting compliance , err : %v ", err)
	}

	stat, err := calculationOperations(operator, int(rule.Value), int(compliance.TotalCount))
	if err != nil {
		return false, err
	}
	if !stat {
		return false, nil
	}
	return true, nil
}

func calculationOperations(operator api.OperatorStruct, value int, totalValue int) (bool, error) {
	for {
		if oneCondition := operator.OperatorInfo; oneCondition != nil {
			stat := compareValue(oneCondition.Operator, value, totalValue)
			return stat, nil
		} else if operator.ConditionStr != nil {
			stat, err := calculationConditionStr(operator, value, totalValue)
			if err != nil {
				return false, err
			}
			return stat, nil
		} else {
			break
		}
	}
	return false, fmt.Errorf("error entering the operation")
}

func calculationConditionStr(operator api.OperatorStruct, value int, totalValue int) (bool, error) {
	conditionType := operator.ConditionStr.ConditionType
	if conditionType == "AND" {
		stat, err := calculationConditionStrAND(operator, value, totalValue)
		if err != nil {
			return false, err
		}
		return stat, nil
	} else if conditionType == "OR" {
		stat, err := calculationConditionStrOr(operator, value, totalValue)
		if err != nil {
			return false, err
		}
		return stat, nil
	}
	return false, fmt.Errorf("please enter right condition")
}

func calculationConditionStrAND(operator api.OperatorStruct, value int, totalValue int) (bool, error) {
	// AND condition
	for i := 0; i < len(operator.ConditionStr.OperatorStr); i++ {
		operator = operator.ConditionStr.OperatorStr[i]
		if operator.OperatorInfo != nil {
			stat := compareValue(operator.OperatorInfo.Operator, value, totalValue)
			if !stat {
				return false, nil
			} else {
				if i == len(operator.ConditionStr.OperatorStr) {
					return true, nil
				}
				continue
			}
		} else if operator.ConditionStr.OperatorStr != nil {
			for j := 0; j < len(operator.ConditionStr.OperatorStr); j++ {
				stat, _ := calculationConditionStr(operator.ConditionStr.OperatorStr[j], value, totalValue)
				if !stat {
					return false, nil
				} else {
					if i == len(operator.ConditionStr.OperatorStr) {
						return true, nil
					}
					continue
				}
			}
		} else {
			return false, fmt.Errorf("error condition is impty")
		}
	}
	return false, nil
}

func calculationConditionStrOr(operator api.OperatorStruct, value int, totalValue int) (bool, error) {
	for i := 0; i < len(operator.ConditionStr.OperatorStr); i++ {
		operator = operator.ConditionStr.OperatorStr[i]
		if operator.OperatorInfo != nil {
			stat := compareValue(operator.OperatorInfo.Operator, value, totalValue)
			if stat {
				return true, nil
			} else {
				if i == len(operator.ConditionStr.OperatorStr) {
					return false, nil
				}
				continue
			}
		} else if operator.ConditionStr.OperatorStr != nil {
			for j := 0; j < len(operator.ConditionStr.OperatorStr); j++ {
				stat, _ := calculationConditionStr(operator.ConditionStr.OperatorStr[j], value, totalValue)
				if stat {
					return true, nil
				} else {
					if i == len(operator.ConditionStr.OperatorStr) {
						return false, nil
					}
					continue
				}
			}
		} else {
			return false, fmt.Errorf("error condition is impty ")
		}
	}
	return false, fmt.Errorf("error")
}

func compareValue(operator string, value int, totalValue int) bool {
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
