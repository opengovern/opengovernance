package types

type ComplianceResult string

const (
	ComplianceResultOK    ComplianceResult = "ok"
	ComplianceResultALARM ComplianceResult = "alarm"
	ComplianceResultINFO  ComplianceResult = "info"
	ComplianceResultSKIP  ComplianceResult = "skip"
	ComplianceResultERROR ComplianceResult = "error"
)

func (r ComplianceResult) IsPassed() bool {
	return r == ComplianceResultOK
}

type ComplianceResultSummary struct {
	OkCount    int `json:"okCount" example:"1"`
	AlarmCount int `json:"alarmCount" example:"1"`
	InfoCount  int `json:"infoCount" example:"1"`
	SkipCount  int `json:"skipCount" example:"1"`
	ErrorCount int `json:"errorCount" example:"1"`
}

func (c *ComplianceResultSummary) AddComplianceResultSummary(summary ComplianceResultSummary) {
	c.OkCount += summary.OkCount
	c.AlarmCount += summary.AlarmCount
	c.InfoCount += summary.InfoCount
	c.SkipCount += summary.SkipCount
	c.ErrorCount += summary.ErrorCount
}

type ComplianceResultShortSummary struct {
	Passed int `json:"passed"`
	Failed int `json:"failed"`
}
