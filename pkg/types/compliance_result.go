package types

import "strings"

type ConformanceStatus string

const (
	ConformanceStatusOK    ConformanceStatus = "ok"
	ConformanceStatusALARM ConformanceStatus = "alarm"
	ConformanceStatusINFO  ConformanceStatus = "info"
	ConformanceStatusSKIP  ConformanceStatus = "skip"
	ConformanceStatusERROR ConformanceStatus = "error"
)

func GetConformanceStatuses() []ConformanceStatus {
	return conformanceStatuses
}

func GetPassedConformanceStatuses() []ConformanceStatus {
	passed := make([]ConformanceStatus, 0)
	for _, status := range conformanceStatuses {
		if status.IsPassed() {
			passed = append(passed, status)
		}
	}
	return passed
}

func GetFailedConformanceStatuses() []ConformanceStatus {
	failed := make([]ConformanceStatus, 0)
	for _, status := range conformanceStatuses {
		if !status.IsPassed() {
			failed = append(failed, status)
		}
	}
	return failed
}

func (r ConformanceStatus) IsPassed() bool {
	return r == ConformanceStatusOK || r == ConformanceStatusINFO || r == ConformanceStatusSKIP
}

type ConformanceStatusSummaryWithTotal struct {
	ConformanceStatusSummary
	TotalCount int `json:"totalCount" example:"5"`
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

var conformanceStatuses = []ConformanceStatus{
	ConformanceStatusOK,
	ConformanceStatusALARM,
	ConformanceStatusINFO,
	ConformanceStatusSKIP,
	ConformanceStatusERROR,
}

func ParseConformanceStatus(s string) ConformanceStatus {
	s = strings.ToLower(s)
	for _, status := range conformanceStatuses {
		if s == strings.ToLower(string(status)) {
			return status
		}
	}
	return ""
}

func ParseConformanceStatuses(list []string) []ConformanceStatus {
	result := make([]ConformanceStatus, 0, len(list))
	for _, s := range list {
		result = append(result, ParseConformanceStatus(s))
	}
	return result
}
