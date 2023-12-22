package types

type ConformanceStatus string

const (
	ComplianceResultOK    ConformanceStatus = "ok"
	ComplianceResultALARM ConformanceStatus = "alarm"
	ComplianceResultINFO  ConformanceStatus = "info"
	ComplianceResultSKIP  ConformanceStatus = "skip"
	ComplianceResultERROR ConformanceStatus = "error"
)

func (r ConformanceStatus) IsPassed() bool {
	return r == ComplianceResultOK
}

type ConformanceStatusSummary struct {
	OkCount    int `json:"okCount" example:"1"`
	AlarmCount int `json:"alarmCount" example:"1"`
	InfoCount  int `json:"infoCount" example:"1"`
	SkipCount  int `json:"skipCount" example:"1"`
	ErrorCount int `json:"errorCount" example:"1"`
}

func (c *ConformanceStatusSummary) AddConformanceStatusSummary(summary ConformanceStatusSummary) {
	c.OkCount += summary.OkCount
	c.AlarmCount += summary.AlarmCount
	c.InfoCount += summary.InfoCount
	c.SkipCount += summary.SkipCount
	c.ErrorCount += summary.ErrorCount
}

func (c *ConformanceStatusSummary) AddConformanceStatusMap(summary map[ConformanceStatus]int) {
	c.OkCount += summary[ComplianceResultOK]
	c.AlarmCount += summary[ComplianceResultALARM]
	c.InfoCount += summary[ComplianceResultINFO]
	c.SkipCount += summary[ComplianceResultSKIP]
	c.ErrorCount += summary[ComplianceResultERROR]
}

type ComplianceResultShortSummary struct {
	Passed int `json:"passed"`
	Failed int `json:"failed"`
}
