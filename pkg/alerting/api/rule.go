package api

import "github.com/kaytu-io/kaytu-util/pkg/source"

type OperatorType string

const (
	OperatorGreaterThan        OperatorType = "GreaterThan"
	OperatorLessThan           OperatorType = "LessThan"
	OperatorLessThanOrEqual    OperatorType = "LessThanOrEqual"
	OperatorGreaterThanOrEqual OperatorType = "GreaterThanOrEqual"
	OperatorEqual              OperatorType = "Equal"
	OperatorDoesNotEqual       OperatorType = "DoesNotEqual"
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
	OperatorType OperatorType `json:"operator_type"`
	Value        int64        `json:"value"`

	Condition *ConditionStruct `json:"condition,omitempty"`
}

type ConditionStruct struct {
	ConditionType ConditionType    `json:"condition_type"`
	Operator      []OperatorStruct `json:"operator"`
}

type TriggerStatus string

const (
	TriggerStatus_Active    = "Active"
	TriggerStatus_NotActive = "Not Active"
)

type Rule struct {
	Id            uint           `json:"id"`
	EventType     EventType      `json:"event_type"`
	Scope         Scope          `json:"scope"`
	Operator      OperatorStruct `json:"operator"`
	Metadata      Metadata       `json:"metadata"`
	TriggerStatus TriggerStatus  `json:"trigger_status"`
	ActionID      uint           `json:"action_id"`
}

type CreateRuleRequest struct {
	EventType EventType      `json:"event_type"`
	Scope     Scope          `json:"scope"`
	Operator  OperatorStruct `json:"operator"`
	Metadata  Metadata       `json:"metadata"`
	ActionID  uint           `json:"action_id"`
}

type UpdateRuleRequest struct {
	EventType *EventType      `json:"event_type"`
	Scope     *Scope          `json:"scope"`
	Operator  *OperatorStruct `json:"operator"`
	Metadata  *Metadata       `json:"metadata"`
	ActionID  *uint           `json:"action_id"`
}

type Metadata struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Label       []string `json:"label"`
}
