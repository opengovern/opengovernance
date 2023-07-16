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

func (r *SeverityResult) AddSeverityResult(severity SeverityResult) {
	r.UnknownCount += severity.UnknownCount
	r.PassedCount += severity.PassedCount
	r.LowCount += severity.LowCount
	r.MediumCount += severity.MediumCount
	r.HighCount += severity.HighCount
	r.CriticalCount += severity.CriticalCount
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

func (r *SeverityResult) IncreaseBySeverityByAmount(severity Severity, amount int) {
	switch severity {
	case SeverityCritical:
		r.CriticalCount += amount
	case SeverityHigh:
		r.HighCount += amount
	case SeverityMedium:
		r.MediumCount += amount
	case SeverityLow:
		r.LowCount += amount
	case SeverityNone:
		r.LowCount += amount
	}
}
