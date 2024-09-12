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
	ResourceType  []string   `json:"resource_type"`
	DiscoveryType []string   `json:"discovery_type"`
	JobStatus     []string   `json:"job_status"`
	StartTime     time.Time  `json:"start_time"`
	EndTime       *time.Time `json:"end_time"`
	SortBy        *string    `json:"sort_by"`
	Cursor        *int64     `json:"cursor"`
	PerPage       *int64     `json:"per_page"`
}

type GetDescribeJobsHistoryResponse struct {
	JobId           uint                      `json:"job_id"`
	DiscoveryType   string                    `json:"discovery_type"`
	ResourceType    string                    `json:"resource_type"`
	JobStatus       DescribeResourceJobStatus `json:"job_status"`
	DateTime        time.Time                 `json:"date_time"`
	IntegrationInfo *IntegrationInfo          `json:"integration_info"`
}

type GetComplianceJobsHistoryRequest struct {
	ConnectionId *string    `json:"connection_id"`
	AccountId    *string    `json:"account_id"`
	BenchmarkId  []string   `json:"benchmark_id"`
	JobStatus    []string   `json:"job_status"`
	StartTime    time.Time  `json:"start_time"`
	EndTime      *time.Time `json:"end_time"`
	SortBy       *string    `json:"sort_by"`
	Cursor       *int64     `json:"cursor"`
	PerPage      *int64     `json:"per_page"`
}

type GetComplianceJobsHistoryResponse struct {
	JobId           uint                `json:"job_id"`
	BenchmarkId     string              `json:"benchmark_id"`
	JobStatus       ComplianceJobStatus `json:"job_status"`
	DateTime        time.Time           `json:"date_time"`
	IntegrationInfo []IntegrationInfo   `json:"integration_info"`
}

type GetAnalyticsJobsHistoryRequest struct {
	Type      []string   `json:"type"`
	JobStatus []string   `json:"job_status"`
	StartTime time.Time  `json:"start_time"`
	EndTime   *time.Time `json:"end_time"`
	SortBy    *string    `json:"sort_by"`
	Cursor    *int64     `json:"cursor"`
	PerPage   *int64     `json:"per_page"`
}

type GetAnalyticsJobsHistoryResponse struct {
	JobId     uint          `json:"job_id"`
	Type      string        `json:"type"`
	JobStatus api.JobStatus `json:"job_status"`
	DateTime  time.Time     `json:"date_time"`
}

type RunBenchmarkByIdRequest struct {
	ConnectionInfo []struct {
		ConnectionId      *string `json:"connection_id"`
		Connector         *string `json:"connector"`
		ProviderIdRegex   *string `json:"provider_id_regex"`
		ProviderNameRegex *string `json:"provider_name_regex"`
	} `json:"connection_info"`
}

type RunBenchmarkRequest struct {
	BenchmarkIds    []string `json:"benchmark_ids"`
	IntegrationInfo []struct {
		Integration        *string `json:"integration"`
		Type               *string `json:"type"`
		ID                 *string `json:"id"`
		IDName             *string `json:"id_name"`
		IntegrationTracker *string `json:"integration_tracker"`
	} `json:"integration_info"`
}

type IntegrationInfo struct {
	Integration        string `json:"integration"`
	Type               string `json:"type"`
	ID                 string `json:"id"`
	IDName             string `json:"id_name"`
	IntegrationTracker string `json:"integration_tracker"`
}

type RunBenchmarkResponse struct {
	JobId           uint              `json:"job_id"`
	BenchmarkId     string            `json:"benchmark_id"`
	IntegrationInfo []IntegrationInfo `json:"integration_info"`
}

type RunDiscoveryRequest struct {
	ResourceTypes   []string `json:"resource_types"`
	ForceFull       bool     `json:"force_full"` // force full discovery. only matters if ResourceTypes is empty
	IntegrationInfo []struct {
		Integration        *string `json:"integration"`
		Type               *string `json:"type"`
		ID                 *string `json:"id"`
		IDName             *string `json:"id_name"`
		IntegrationTracker *string `json:"integration_tracker"`
	} `json:"integration_info"`
}

type RunDiscoveryResponse struct {
	JobId           uint            `json:"job_id"`
	ResourceType    string          `json:"resource_type"`
	Status          string          `json:"status"`
	FailureReason   string          `json:"failure_reason"`
	IntegrationInfo IntegrationInfo `json:"integration_info"`
}

type GetDescribeJobStatusResponse struct {
	JobId           uint            `json:"job_id"`
	IntegrationInfo IntegrationInfo `json:"integration_info"`
	JobStatus       string          `json:"job_status"`
	DiscoveryType   string          `json:"discovery_type"`
	ResourceType    string          `json:"resource_type"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

type GetComplianceJobStatusResponse struct {
	JobId           uint              `json:"job_id"`
	IntegrationInfo []IntegrationInfo `json:"integration_info"`
	JobStatus       string            `json:"job_status"`
	BenchmarkId     string            `json:"benchmark_id"`
	CreatedAt       time.Time         `json:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at"`
}
type GetAnalyticsJobStatusResponse struct {
	JobId     uint      `json:"job_id"`
	JobStatus string    `json:"job_status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type ListDescribeJobsRequest struct {
	IntegrationInfo []struct {
		Integration        *string `json:"integration"`
		Type               *string `json:"type"`
		ID                 *string `json:"id"`
		IDName             *string `json:"id_name"`
		IntegrationTracker *string `json:"integration_tracker"`
	} `json:"integration_info"`
	ResourceType  []string   `json:"resource_type"`
	DiscoveryType []string   `json:"discovery_type"`
	JobStatus     []string   `json:"job_status"`
	StartTime     time.Time  `json:"start_time"`
	EndTime       *time.Time `json:"end_time"`
	SortBy        *string    `json:"sort_by"`
	Cursor        *int64     `json:"cursor"`
	PerPage       *int64     `json:"per_page"`
}

type ListComplianceJobsRequest struct {
	IntegrationInfo []struct {
		Integration        *string `json:"integration"`
		Type               *string `json:"type"`
		ID                 *string `json:"id"`
		IDName             *string `json:"id_name"`
		IntegrationTracker *string `json:"integration_tracker"`
	} `json:"integration_info"`
	BenchmarkId []string   `json:"benchmark_id"`
	JobStatus   []string   `json:"job_status"`
	StartTime   time.Time  `json:"start_time"`
	EndTime     *time.Time `json:"end_time"`
	SortBy      *string    `json:"sort_by"`
	Cursor      *int64     `json:"cursor"`
	PerPage     *int64     `json:"per_page"`
}
