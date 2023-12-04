package model

import (
	"time"
)

type MeterType string

const (
	MeterType_InventoryDiscoveryJobCount MeterType = "InventoryDiscoveryJobCount"
	MeterType_CostDiscoveryJobCount      MeterType = "CostDiscoveryJobCount"
	MeterType_MetricEvaluationCount      MeterType = "MetricEvaluationCount"
	MeterType_InsightEvaluationCount     MeterType = "InsightEvaluationCount"
	MeterType_BenchmarkEvaluationCount   MeterType = "BenchmarkEvaluationCount"
	MeterType_TotalFindings              MeterType = "TotalFindings"
	MeterType_TotalResource              MeterType = "TotalResource"
	MeterType_TotalUsers                 MeterType = "TotalUsers"
	MeterType_TotalApiKeys               MeterType = "TotalApiKeys"
	MeterType_TotalRules                 MeterType = "TotalRules"
	MeterType_AlertCount                 MeterType = "AlertCount"
)

func ListAllMeterTypes() []MeterType {
	return []MeterType{
		MeterType_InventoryDiscoveryJobCount,
		MeterType_CostDiscoveryJobCount,
		MeterType_MetricEvaluationCount,
		MeterType_InsightEvaluationCount,
		MeterType_BenchmarkEvaluationCount,
		MeterType_TotalFindings,
		MeterType_TotalResource,
		MeterType_TotalUsers,
		MeterType_TotalApiKeys,
		MeterType_TotalRules,
		MeterType_AlertCount,
	}
}

type Meter struct {
	WorkspaceID string    `gorm:"primarykey"`
	DateHour    string    `gorm:"primarykey"`
	MeterType   MeterType `gorm:"primarykey"`

	CreatedAt time.Time
	Value     int64
	Published bool
}
