package api

import (
	"time"
)

type ComplianceTrigger struct {
	ComplianceId   string    `json:"compliance_id"`
	Hour           time.Time `json:"hour"`
	ConnectionId   string    `json:"connection_id"`
	Value          int64     `json:"value"`
	ResponseStatus int       `json:"response_status"`
}

type InsightTrigger struct {
	InsightId      int64     `json:"insight_id"`
	Hour           time.Time `json:"hour"`
	ConnectionId   string    `json:"connection_id"`
	Value          int64     `json:"value"`
	ResponseStatus int       `json:"response_status"`
}
