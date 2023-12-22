package types

type ConformanceStatus string

const (
	ConformanceStatusOK    ConformanceStatus = "ok"
	ConformanceStatusALARM ConformanceStatus = "alarm"
	ConformanceStatusINFO  ConformanceStatus = "info"
	ConformanceStatusSKIP  ConformanceStatus = "skip"
	ConformanceStatusERROR ConformanceStatus = "error"
)

func (r ConformanceStatus) IsPassed() bool {
	return r == ConformanceStatusOK
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
	c.OkCount += summary[ConformanceStatusOK]
	c.AlarmCount += summary[ConformanceStatusALARM]
	c.InfoCount += summary[ConformanceStatusINFO]
	c.SkipCount += summary[ConformanceStatusSKIP]
	c.ErrorCount += summary[ConformanceStatusERROR]
}

type ComplianceResultShortSummary struct {
	Passed int `json:"passed"`
	Failed int `json:"failed"`
}
