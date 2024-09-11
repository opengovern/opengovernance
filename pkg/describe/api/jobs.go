package api

import (
	"github.com/kaytu-io/kaytu-engine/pkg/analytics/api"
	"github.com/kaytu-io/kaytu-util/pkg/source"
	"time"
)

type GetCredsForJobRequest struct {
	SourceID string `json:"sourceId"`
}

type GetCredsForJobResponse struct {
	Credentials string `json:"creds"`
}

type GetDataResponse struct {
	Data string `json:"data"`
}

type TriggerBenchmarkEvaluationRequest struct {
	BenchmarkID  string   `json:"benchmarkID" example:"azure_cis_v1"`                                                                          // Benchmark ID to evaluate
	ConnectionID *string  `json:"connectionID" example:"8e0f8e7a-1b1c-4e6f-b7e4-9c6af9d2b1c8"`                                                 // Connection ID to evaluate
	ResourceIDs  []string `json:"resourceIDs" example:"/subscriptions/123/resourceGroups/rg1/providers/Microsoft.Compute/virtualMachines/vm1"` // Resource IDs to evaluate
}

type ListBenchmarkEvaluationsRequest struct {
	EvaluatedAtAfter  *int64       `json:"evaluatedAtAfter" example:"1619510400"`                       // Filter evaluations created after this timestamp
	EvaluatedAtBefore *int64       `json:"evaluatedAtBefore" example:"1619610400"`                      // Filter evaluations created before this timestamp
	ConnectionID      *string      `json:"connectionID" example:"8e0f8e7a-1b1c-4e6f-b7e4-9c6af9d2b1c8"` // Filter evaluations for this connection
	Connector         *source.Type `json:"connector" example:"Azure"`                                   // Filter evaluations for this connector
	BenchmarkID       *string      `json:"benchmarkID" example:"azure_cis_v1"`                          // Filter evaluations for this benchmark
}

type JobType string

const (
	JobType_Discovery  JobType = "discovery"
	JobType_Analytics  JobType = "analytics"
	JobType_Compliance JobType = "compliance"
)

type JobStatus string

const (
	JobStatus_Created    JobStatus = "created"
	JobStatus_Queued     JobStatus = "queued"
	JobStatus_InProgress JobStatus = "in_progress"
	JobStatus_Successful JobStatus = "successful"
	JobStatus_Failure    JobStatus = "failure"
	JobStatus_Timeout    JobStatus = "timeout"
)

type JobSort string

const (
	JobSort_ByJobID        = "id"
	JobSort_ByJobType      = "job_type"
	JobSort_ByConnectionID = "connection_id"
	JobSort_ByStatus       = "status"
)

type JobSortOrder string

const (
	JobSortOrder_ASC  = "ASC"
	JobSortOrder_DESC = "DESC"
)

type ListJobsRequest struct {
	Hours        int      `json:"hours"`
	PageStart    int      `json:"pageStart"`
	PageEnd      int      `json:"pageEnd"`
	TypeFilters  []string `json:"typeFilters"`
	StatusFilter []string `json:"statusFilter"`

	SortBy    JobSort      `json:"sortBy"`
	SortOrder JobSortOrder `json:"sortOrder"`
}

type Job struct {
	ID                     uint      `json:"id"`
	CreatedAt              time.Time `json:"createdAt"`
	UpdatedAt              time.Time `json:"updatedAt"`
	Type                   JobType   `json:"type"`
	ConnectionID           string    `json:"connectionID"`
	ConnectionProviderID   string    `json:"connectionProviderID"`
	ConnectionProviderName string    `json:"connectionProviderName"`
	Title                  string    `json:"title"`
	Status                 string    `json:"status"`
	FailureReason          string    `json:"failureReason"`
}

type JobSummary struct {
	Type   JobType `json:"type"`
	Status string  `json:"status"`
	Count  int64   `json:"count"`
}

type ListJobsResponse struct {
	Jobs      []Job        `json:"jobs"`
	Summaries []JobSummary `json:"summaries"`
}

type ListDiscoveryResourceTypes struct {
	AWSResourceTypes   []string `json:"awsResourceTypes"`
	AzureResourceTypes []string `json:"azureResourceTypes"`
}

type JobSeqCheckResponse struct {
	IsRunning bool `json:"isRunning"`
}

type GetDescribeJobsHistoryRequest struct {
	ConnectionId  *string    `json:"connectionId"`
	AccountId     *string    `json:"accountId"`
	ResourceType  []string   `json:"resourceType"`
	DiscoveryType []string   `json:"discoveryType"`
	JobStatus     []string   `json:"jobStatus"`
	StartTime     time.Time  `json:"startTime"`
	EndTime       *time.Time `json:"endTime"`
	SortBy        *string    `json:"sortBy"`
	PageNumber    *int64     `json:"pageNumber"`
	PageSize      *int64     `json:"pageSize"`
}

type GetDescribeJobsHistoryResponse struct {
	JobId         uint                      `json:"jobId"`
	DiscoveryType string                    `json:"discoveryType"`
	ResourceType  string                    `json:"resourceType"`
	JobStatus     DescribeResourceJobStatus `json:"jobStatus"`
	DateTime      time.Time                 `json:"dateTime"`
}

type GetComplianceJobsHistoryRequest struct {
	ConnectionId *string    `json:"connectionId"`
	AccountId    *string    `json:"accountId"`
	BenchmarkId  []string   `json:"benchmarkId"`
	JobStatus    []string   `json:"jobStatus"`
	StartTime    time.Time  `json:"startTime"`
	EndTime      *time.Time `json:"endTime"`
	SortBy       *string    `json:"sortBy"`
	PageNumber   *int64     `json:"pageNumber"`
	PageSize     *int64     `json:"pageSize"`
}

type GetComplianceJobsHistoryResponse struct {
	JobId       uint                `json:"jobId"`
	BenchmarkId string              `json:"benchmarkId"`
	JobStatus   ComplianceJobStatus `json:"jobStatus"`
	DateTime    time.Time           `json:"dateTime"`
}

type GetAnalyticsJobsHistoryRequest struct {
	Type       []string   `json:"type"`
	JobStatus  []string   `json:"jobStatus"`
	StartTime  time.Time  `json:"startTime"`
	EndTime    *time.Time `json:"endTime"`
	SortBy     *string    `json:"sortBy"`
	PageNumber *int64     `json:"pageNumber"`
	PageSize   *int64     `json:"pageSize"`
}

type GetAnalyticsJobsHistoryResponse struct {
	JobId     uint          `json:"jobId"`
	Type      string        `json:"type"`
	JobStatus api.JobStatus `json:"jobStatus"`
	DateTime  time.Time     `json:"dateTime"`
}
