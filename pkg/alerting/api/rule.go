package api

//type OperatorI = string
//
//const (
//	Operator_GreaterThan        OperatorI = ">"
//	Operator_LessThan           OperatorI = "<"
//	Operator_LessThanOrEqual    OperatorI = "<="
//	Operator_GreaterThanOrEqual OperatorI = ">="
//	Operator_Equal              OperatorI = "="
//	Operator_DoesNotEqual       OperatorI = "!="
//)

type EventType struct {
	InsightId   int64
	BenchmarkId string
}

type Scope struct {
	ConnectionId string
}

type OperatorStruct struct {
	OperatorInfo *OperatorInformation
	ConditionStr *ConditionStruct
}

type OperatorInformation struct {
	Field    string
	Operator string
	Value    string
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
	Value     int64          `json:"value"`
	ActionID  uint           `json:"action_id"`
}

type UpdateRuleRequest struct {
	ID        uint            `json:"id"`
	EventType *EventType      `json:"event_type"`
	Scope     *Scope          `json:"scope"`
	Operator  *OperatorStruct `json:"operator"`
	Value     *int64          `json:"value"`
	ActionID  *uint           `json:"action_id"`
}
