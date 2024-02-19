package types

import "time"

type Job struct {
	ID          uint
	RetryCount  int
	BenchmarkID string
	CreatedAt   time.Time
}
