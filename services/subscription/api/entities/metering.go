package entities

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

func (t MeterType) IsTotal() bool {
	switch t {
	case MeterType_TotalFindings,
		MeterType_TotalResource,
		MeterType_TotalUsers,
		MeterType_TotalApiKeys,
		MeterType_TotalRules:
		return true
	default:
		return false
	}
}

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

type GetMetersRequest struct {
	StartTimeEpochMillis int64 `json:"start_time_epoch_millis"`
	EndTimeEpochMillis   int64 `json:"end_time_epoch_millis"`
}

type GetMetersResponse struct {
	Meters []Meter `json:"meters"`
}

type Meter struct {
	Type  MeterType `json:"type"`
	Value float64   `json:"value"`
}
