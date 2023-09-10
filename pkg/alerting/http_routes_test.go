package alerting

import (
	"encoding/json"
	"fmt"
	"testing"
)

type Operator = string

const (
	Operator_GreaterThan        Operator = ">"
	Operator_LessThan           Operator = "<"
	Operator_LessThanOrEqual    Operator = "<="
	Operator_GreaterThanOrEqual Operator = ">="
	Operator_Equal              Operator = "="
	Operator_DoesNotEqual       Operator = "!="
)

type EventType struct {
	InsightId int `json:"insight_id"`
	FolanId   int `json:"folan_id"`
}

type Scope struct {
	ConnectionId string `json:"connection_id"`
}

type object struct {
	ID        uint
	EventType json.RawMessage
	Scope     json.RawMessage
	Operator  Operator
	Value     int64
	ActionID  uint
}

type objectResponse struct {
	ID        uint      `json:"id"`
	EventType EventType `json:"event_type"`
	Scope     Scope     `json:"scope"`
	Operator  Operator  `json:"operator"`
	Value     int64     `json:"value"`
	ActionID  uint      `json:"action_id"`
}

func Test_ListRule(t *testing.T) {
	var response objectResponse

	var req object
	req.ID = 123
	req.Value = 100
	req.Scope = json.RawMessage(`{ "connection_id": "testScope" }`)
	req.ActionID = 123
	req.Operator = ">"
	req.EventType = json.RawMessage(`{ "insight_id": 12312 , "folan_id" : 122223}`)

	var eventType EventType
	err := json.Unmarshal(req.EventType, &eventType)
	if err != nil {
		t.Errorf("error in unmarshaling error equal to : %s", err)
		return
	}

	var newScope Scope
	err = json.Unmarshal(req.Scope, &newScope)
	if err != nil {
		t.Errorf("error in unmarshaling error equal to : %s", err)
		return
	}

	response.ID = req.ID
	response.EventType = eventType
	response.Scope = newScope
	response.Value = req.Value
	response.ActionID = req.ActionID
	response.Operator = req.Operator

	fmt.Printf("Id : %v, Value : %v , ActionId : %v, Operator : %s , Scope : %v , EvenType : %v \n", response.ID, response.Value, response.ActionID, response.Operator, response.Scope, response.EventType)
}
