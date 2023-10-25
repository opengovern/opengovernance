package model

import (
	"github.com/lib/pq"
	"gorm.io/gorm"
)

type JobSequencerStatus string

const (
	JobSequencerWaitingForDependencies JobSequencerStatus = "WaitingForDependencies"
	JobSequencerFinished               JobSequencerStatus = "FINISHED"
	JobSequencerFailed                 JobSequencerStatus = "Failed"
)

type JobSequencerJobType string

const (
	JobSequencerJobTypeBenchmark           JobSequencerJobType = "Benchmark"
	JobSequencerJobTypeBenchmarkSummarizer JobSequencerJobType = "BenchmarkSummarizer"
	JobSequencerJobTypeDescribe            JobSequencerJobType = "Describe"
	JobSequencerJobTypeAnalytics           JobSequencerJobType = "Analytics"
)

type JobSequencer struct {
	gorm.Model
	DependencyList   pq.Int64Array `gorm:"type:bigint[]"`
	DependencySource string
	NextJob          string
	Status           JobSequencerStatus
}
