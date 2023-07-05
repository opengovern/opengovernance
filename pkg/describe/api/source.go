package api

import (
	"time"

	"github.com/kaytu-io/kaytu-util/pkg/source"
)

type SourceType string

const (
	SourceCloudAWS   SourceType = "AWS"
	SourceCloudAzure SourceType = "Azure"
)

func IsValidSourceType(t SourceType) bool {
	switch t {
	case SourceCloudAWS, SourceCloudAzure:
		return true
	default:
		return false
	}
}

type Source struct {
	ID                     string      `json:"id" example:"8e0f8e7a-1b1c-4e6f-b7e4-9c6af9d2b1c8"`
	AccountID              string      `json:"accountId" example:"123456789012"`
	Type                   source.Type `json:"type" example:"Azure"`
	LastDescribedAt        time.Time   `json:"lastDescribedAt" example:"2021-01-01T00:00:00Z"`
	LastDescribeJobStatus  string      `json:"lastDescribeJobStatus" example:"COMPLETED"`
	LastComplianceReportAt time.Time   `json:"lastComplianceReportAt" example:"2021-01-01T00:00:00Z"`
}

type DescribeSource struct {
	DescribeResourceJobs []DescribeResource      `json:"describeResourceJobs"`
	Status               DescribeSourceJobStatus `json:"status" example:"IN_PROGRESS"` // CREATED, QUEUED, IN_PROGRESS, TIMEOUT, FAILED, SUCCEEDED
}

type DescribeResource struct {
	ResourceType   string                    `json:"resourceType" example:"Microsoft.Compute/virtualMachines"`
	Status         DescribeResourceJobStatus `json:"status" example:"IN_PROGRESS"` // CREATED, QUEUED, IN_PROGRESS, TIMEOUT, FAILED, SUCCEEDED
	FailureMessage string                    `json:"failureMessage"`
}

type ErrorResponse struct {
	Message string
}

type DescribeSourceJobStatus string

const (
	DescribeSourceJobCreated              DescribeSourceJobStatus = "CREATED"
	DescribeSourceJobInProgress           DescribeSourceJobStatus = "IN_PROGRESS"
	DescribeSourceJobCompletedWithFailure DescribeSourceJobStatus = "COMPLETED_WITH_FAILURE"
	DescribeSourceJobCompleted            DescribeSourceJobStatus = "COMPLETED"
)

type DescribeResourceJobStatus string

const (
	DescribeResourceJobCreated    DescribeResourceJobStatus = "CREATED"
	DescribeResourceJobQueued     DescribeResourceJobStatus = "QUEUED"
	DescribeResourceJobInProgress DescribeResourceJobStatus = "IN_PROGRESS"
	DescribeResourceJobTimeout    DescribeResourceJobStatus = "TIMEOUT"
	DescribeResourceJobFailed     DescribeResourceJobStatus = "FAILED"
	DescribeResourceJobSucceeded  DescribeResourceJobStatus = "SUCCEEDED"
)
