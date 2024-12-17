package types

import "time"

type Job struct {
	ID              uint
	ComplianceJobID uint
	RetryCount      int
	BenchmarkID     string
	CreatedAt       time.Time
}
