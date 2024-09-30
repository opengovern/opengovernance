package types

import (
	"strings"
)

type FindingSeverity string

const (
	FindingSeverityNone     FindingSeverity = "none"
	FindingSeverityLow      FindingSeverity = "low"
	FindingSeverityMedium   FindingSeverity = "medium"
	FindingSeverityHigh     FindingSeverity = "high"
	FindingSeverityCritical FindingSeverity = "critical"
)

func (s FindingSeverity) Level() int {
	switch s {
	case FindingSeverityNone:
		return 1
	case FindingSeverityLow:
		return 2
	case FindingSeverityMedium:
		return 3
	case FindingSeverityHigh:
		return 4
	case FindingSeverityCritical:
		return 5
	default:
		return 0
	}
}

func (s FindingSeverity) String() string {
	return string(s)
}

var findingSeverities = []FindingSeverity{
	FindingSeverityNone,
	FindingSeverityLow,
	FindingSeverityMedium,
	FindingSeverityHigh,
	FindingSeverityCritical,
}

func ParseFindingSeverity(s string) FindingSeverity {
	s = strings.ToLower(s)
	for _, sev := range findingSeverities {
		if s == strings.ToLower(sev.String()) {
			return sev
		}
	}
	return ""
}

func ParseFindingSeverities(list []string) []FindingSeverity {
	result := make([]FindingSeverity, 0, len(list))
	for _, s := range list {
		result = append(result, ParseFindingSeverity(s))
	}
	return result
}

type SeverityResultWithTotal struct {
	SeverityResult
	TotalCount int `json:"totalCount" example:"5"`
}

type SeverityResult struct {
	NoneCount     int `json:"noneCount" example:"1"`
	LowCount      int `json:"lowCount" example:"1"`
	MediumCount   int `json:"mediumCount" example:"1"`
	HighCount     int `json:"highCount" example:"1"`
	CriticalCount int `json:"criticalCount" example:"1"`
}

type SeverityResultV2 struct {
	None     int `json:"none"`
	Low      int `json:"low"`
	Medium   int `json:"medium"`
	High     int `json:"high"`
	Critical int `json:"critical"`
	Total    int `json:"total"`
}

func (r *SeverityResultV2) AddSeverityResult(severity SeverityResult) {
	r.None += severity.NoneCount
	r.Low += severity.LowCount
	r.Medium += severity.MediumCount
	r.High += severity.HighCount
	r.Critical += severity.CriticalCount
	r.Total += severity.NoneCount + severity.LowCount + severity.MediumCount + severity.HighCount + severity.CriticalCount
}

func (r *SeverityResultV2) AddResultMap(result map[FindingSeverity]int) {
	r.None += result[FindingSeverityNone]
	r.Low += result[FindingSeverityLow]
	r.Medium += result[FindingSeverityMedium]
	r.High += result[FindingSeverityHigh]
	r.Critical += result[FindingSeverityCritical]
	r.Total += result[FindingSeverityNone] + result[FindingSeverityLow] + result[FindingSeverityMedium] + result[FindingSeverityHigh] + result[FindingSeverityCritical]
}

func (r *SeverityResult) AddSeverityResult(severity SeverityResult) {
	r.NoneCount += severity.NoneCount
	r.LowCount += severity.LowCount
	r.MediumCount += severity.MediumCount
	r.HighCount += severity.HighCount
	r.CriticalCount += severity.CriticalCount
}

func (r *SeverityResult) AddResultMap(result map[FindingSeverity]int) {
	r.NoneCount += result[FindingSeverityNone]
	r.LowCount += result[FindingSeverityLow]
	r.MediumCount += result[FindingSeverityMedium]
	r.HighCount += result[FindingSeverityHigh]
	r.CriticalCount += result[FindingSeverityCritical]
}

func (r *SeverityResult) IncreaseBySeverity(severity FindingSeverity) {
	switch severity {
	case FindingSeverityCritical:
		r.CriticalCount++
	case FindingSeverityHigh:
		r.HighCount++
	case FindingSeverityMedium:
		r.MediumCount++
	case FindingSeverityLow:
		r.LowCount++
	case FindingSeverityNone:
		r.LowCount++
	}
}

func (r *SeverityResult) IncreaseBySeverityByAmount(severity FindingSeverity, amount int) {
	switch severity {
	case FindingSeverityCritical:
		r.CriticalCount += amount
	case FindingSeverityHigh:
		r.HighCount += amount
	case FindingSeverityMedium:
		r.MediumCount += amount
	case FindingSeverityLow:
		r.LowCount += amount
	case FindingSeverityNone:
		r.LowCount += amount
	}
}
