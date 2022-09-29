package types

type ComplianceResult string

const (
	ComplianceResultOK    ComplianceResult = "ok"
	ComplianceResultALARM ComplianceResult = "alarm"
	ComplianceResultINFO  ComplianceResult = "info"
	ComplianceResultSKIP  ComplianceResult = "skip"
	ComplianceResultERROR ComplianceResult = "error"
)

type ComplianceResultSummary struct {
	OkCount    int `json:"okCount"`
	AlarmCount int `json:"alarmCount"`
	InfoCount  int `json:"infoCount"`
	SkipCount  int `json:"skipCount"`
	ErrorCount int `json:"errorCount"`
}

type ComplianceResultShortSummary struct {
	Passed int `json:"passed"`
	Failed int `json:"failed"`
}
