package api

import (
	"time"

	"github.com/google/uuid"
	"gitlab.com/keibiengine/keibi-engine/pkg/source"
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
	ID                     uuid.UUID   `json:"id"`
	Type                   source.Type `json:"type"`
	LastDescribedAt        time.Time   `json:"lastDescribedAt"`
	LastDescribeJobStatus  string      `json:"lastDescribeJobStatus"`
	LastComplianceReportAt time.Time   `json:"lastComplianceReportAt"`
}

type DescribeSource struct {
	DescribeResourceJobs []DescribeResource      `json:"describeResourceJobs"`
	Status               DescribeSourceJobStatus `json:"status"`
}

type DescribeResource struct {
	ResourceType   string                    `json:"resourceType"`
	Status         DescribeResourceJobStatus `json:"status"`
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
	DescribeResourceJobCreated      DescribeResourceJobStatus = "CREATED"
	DescribeResourceJobQueued       DescribeResourceJobStatus = "QUEUED"
	DescribeResourceJobCloudTimeout DescribeResourceJobStatus = "CLOUD_TIMEOUT"
	DescribeResourceJobFailed       DescribeResourceJobStatus = "FAILED"
	DescribeResourceJobSucceeded    DescribeResourceJobStatus = "SUCCEEDED"
)
