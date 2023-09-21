package api

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
	InsightId   int64
	BenchmarkId string
}

type Scope struct {
	ConnectionId string
}

type ApiRule struct {
	ID        uint      `json:"id"`
	EventType EventType `json:"event_type"`
	Scope     Scope     `json:"scope"`
	Operator  Operator  `json:"operator"`
	Value     int64     `json:"value"`
	ActionID  uint      `json:"action_id"`
}

type UpdateRuleRequest struct {
	ID        uint       `json:"id"`
	EventType *EventType `json:"event_type"`
	Scope     *Scope     `json:"scope"`
	Operator  *Operator  `json:"operator"`
	Value     *int64     `json:"value"`
	ActionID  *uint      `json:"action_id"`
}
