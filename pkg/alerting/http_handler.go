package alerting

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/kaytu-io/kaytu-engine/pkg/alerting/api"
	api2 "github.com/kaytu-io/kaytu-engine/pkg/auth/api"
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
			fmt.Errorf("error in giving list rules error equal to : %s", err)
			return
		}

		for _, rule := range rules {

			var scope api.Scope
			err := json.Unmarshal(rule.Scope, &scope)
			if err != nil {
				fmt.Errorf("error in unmarshaling scope , error  equal to : %s", err)
				return
			}

			var eventType api.EventType
			err = json.Unmarshal(rule.EventType, &eventType)
			if err != nil {
				fmt.Errorf("error in unmarshaling event type , error equal to : %s ", err)
				return
			}

			diff := 24 * time.Hour
			oneDayAgo := time.Now().Add(-diff)
			timeNow := time.Now()
			number := strconv.Itoa(int(eventType.InsightId))
			insight, err := h.complianceClient.GetInsight(&httpclient.Context{UserRole: api2.InternalRole}, number, []string{scope.ConnectionId}, &oneDayAgo, &timeNow)
			if err != nil {
				fmt.Errorf("error in getting GetInsight , error  equal to : %s", err)
				return
			}

			stat := compareValue(rule.Operator, int(rule.Value), int(*insight.TotalResultValue))
			if stat == true {
				var action Action
				err = h.db.orm.Model(Rule{}).Where("id = ?", rule.ActionID).Find(&action).Error
				if err != nil {
					fmt.Errorf("error in getting action , error equal to : %s", err)
					return
				}

				reqBody, err := json.Marshal(action.Body)
				if err != nil {
					fmt.Errorf("error in marshaling the body ,error equal to : %s", err)
					return
				}

				req, err := http.NewRequest(action.Method, action.Url, bytes.NewBuffer(reqBody))
				if err != nil {
					fmt.Errorf("error in sending the request , error equal to : %s", err)
					return
				}
				req.Header.Add("Content-Type", "application/json")

				res, err := http.DefaultClient.Do(req)
				if err != nil {
					fmt.Errorf("error equal to : %s", err)
					return
				}

				err = res.Body.Close()
				if err != nil {
					fmt.Errorf("error equal to : %s", err)
					return
				}
			} else {
				continue
			}
		}
	}
}

func compareValue(operator string, value int, totalValue int) bool {
	switch operator {
	case ">":
		if totalValue > value {
			return true
		} else {
			return false
		}
	case "<":
		if totalValue < value {
			return true
		} else {
			return false
		}
	case ">=":
		if totalValue >= value {
			return true
		} else {
			return false
		}
	case "<=":
		if totalValue <= value {
			return true
		} else {
			return false
		}
	case "=":
		if totalValue == value {
			return true
		} else {
			return false
		}
	case "!=":
		if totalValue != value {
			return true
		} else {
			return false
		}
	default:
		return false
	}
}
