package api

type OperatorType = string

const (
	Operator_GreaterThan        OperatorType = ">"
	Operator_LessThan           OperatorType = "<"
	Operator_LessThanOrEqual    OperatorType = "<="
	Operator_GreaterThanOrEqual OperatorType = ">="
	Operator_Equal              OperatorType = "="
	Operator_DoesNotEqual       OperatorType = "!="
)

type EventType struct {
	InsightId   *int64
	BenchmarkId *string
}

type Scope struct {
	ConnectionId string
}

type OperatorStruct struct {
	OperatorInfo *OperatorInformation
	ConditionStr *ConditionStruct
}

type OperatorInformation struct {
	Operator OperatorType
	Value    int64
}

type ConditionStruct struct {
	ConditionType string
	OperatorStr   []OperatorStruct
}

type ApiRule struct {
	ID        uint           `json:"id"`
	EventType EventType      `json:"event_type"`
	Scope     Scope          `json:"scope"`
	Operator  OperatorStruct `json:"operator"`
	ActionID  uint           `json:"action_id"`
}

type UpdateRuleRequest struct {
	ID        uint            `json:"id"`
	EventType *EventType      `json:"event_type"`
	Scope     *Scope          `json:"scope"`
	Operator  *OperatorStruct `json:"operator"`
	ActionID  *uint           `json:"action_id"`
}
