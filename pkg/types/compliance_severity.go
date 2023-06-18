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
	UnknownCount  int `json:"unknownCount" example:"1"`
	PassedCount   int `json:"passedCount" example:"1"`
	LowCount      int `json:"lowCount" example:"1"`
	MediumCount   int `json:"mediumCount" example:"1"`
	HighCount     int `json:"highCount" example:"1"`
	CriticalCount int `json:"criticalCount" example:"1"`
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
