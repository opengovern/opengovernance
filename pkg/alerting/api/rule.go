package api

import "github.com/kaytu-io/kaytu-util/pkg/source"

type OperatorType string

const (
	OperatorGreaterThan        OperatorType = ">"
	OperatorLessThan           OperatorType = "<"
	OperatorLessThanOrEqual    OperatorType = "<="
	OperatorGreaterThanOrEqual OperatorType = ">="
	OperatorEqual              OperatorType = "="
	OperatorDoesNotEqual       OperatorType = "!="
)

type ConditionType string

const (
	ConditionAnd ConditionType = "AND"
	ConditionOr  ConditionType = "OR"
)

type EventType struct {
	InsightId   *int64  `json:"insight_id,omitempty"`
	BenchmarkId *string `json:"benchmark_id,omitempty"`
}

type Scope struct {
	ConnectionId    *string      `json:"connection_id,omitempty"`
	ConnectionGroup *string      `json:"connection_group,omitempty"`
	Connector       *source.Type `json:"connector,omitempty"`
}

type OperatorStruct struct {
	OperatorInfo *OperatorInformation `json:"operator_info,omitempty"`
	Condition    *ConditionStruct     `json:"condition,omitempty"`
}

type OperatorInformation struct {
	OperatorType OperatorType `json:"operator_type"`
	Value        int64        `json:"value"`
}

type ConditionStruct struct {
	ConditionType ConditionType    `json:"condition_type"`
	Operator      []OperatorStruct `json:"operator"`
}

type Rule struct {
	Id        uint           `json:"id"`
	EventType EventType      `json:"event_type"`
	Scope     Scope          `json:"scope"`
	Operator  OperatorStruct `json:"operator"`
	ActionID  uint           `json:"action_id"`
}

type CreateRuleRequest struct {
	EventType EventType      `json:"event_type"`
	Scope     Scope          `json:"scope"`
	Operator  OperatorStruct `json:"operator"`
	ActionID  uint           `json:"action_id"`
}

type UpdateRuleRequest struct {
	Id        uint            `json:"id"`
	EventType *EventType      `json:"event_type"`
	Scope     *Scope          `json:"scope"`
	Operator  *OperatorStruct `json:"operator"`
	ActionID  *uint           `json:"action_id"`
}
