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

			diff := 24 * time.Hour
			oneDayAgo := time.Now().Add(-diff)
			timeNow := time.Now()
			insightID := strconv.Itoa(int(eventType.InsightId))
			insight, err := h.complianceClient.GetInsight(&httpclient.Context{UserRole: api2.InternalRole}, insightID, []string{scope.ConnectionId}, &oneDayAgo, &timeNow)
			if err != nil {
				fmt.Printf("error in getting GetInsight , error  equal to : %v", err)
				return
			}
			if insight.TotalResultValue == nil {
				continue
			}
			stat := compareValue(rule.Operator, int(rule.Value), int(*insight.TotalResultValue))
			if !stat {
				continue
			}
			var action Action
			action, err = h.db.GetAction(rule.ActionID)
			if err != nil {
				fmt.Printf("error in getting action , error equal to : %v", err)
			}

			req, err := http.NewRequest(action.Method, action.Url, bytes.NewBuffer([]byte(action.Body)))
			if err != nil {
				fmt.Printf("error in sending the request , error equal to : %v", err)
				return
			}
			var headers map[string]string
			err = json.Unmarshal(action.Headers, &headers)
			if err != nil {
				fmt.Printf("error in unmarshaling the headers  , error : %v", err)
				return
			}

			for k, v := range headers {
				req.Header.Add(k, v)
			}

			res, err := http.DefaultClient.Do(req)
			if err != nil {
				fmt.Printf("error equal to : %v", err)
				return
			}

			err = res.Body.Close()
			if err != nil {
				fmt.Printf("error equal to : %v", err)
				return
			}
		}
	}
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
