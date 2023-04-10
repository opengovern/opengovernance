package types

type Severity = string

const (
	SeverityNone     = "none"
	SeverityLow      = "low"
	SeverityMedium   = "medium"
	SeverityHigh     = "high"
	SeverityCritical = "critical"
)

type SeverityResult struct {
	UnknownCount  int `json:"unknownCount"`
	PassedCount   int `json:"passedCount"`
	LowCount      int `json:"lowCount"`
	MediumCount   int `json:"mediumCount"`
	HighCount     int `json:"highCount"`
	CriticalCount int `json:"criticalCount"`
}

func (r *SeverityResult) IncreaseBySeverity(severity Severity) {
	switch severity {
	case SeverityCritical:
		r.CriticalCount++
	case SeverityHigh:
		r.HighCount++
	case SeverityMedium:
		r.MediumCount++
	case SeverityLow:
		r.LowCount++
	case SeverityNone:
		r.LowCount++
	}
}
