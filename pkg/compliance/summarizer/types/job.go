package types

import "time"

type Job struct {
	ID          uint
	BenchmarkID string
	CreatedAt   time.Time
}
