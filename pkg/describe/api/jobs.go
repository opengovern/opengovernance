package api

import (
	"github.com/kaytu-io/kaytu-util/pkg/source"
	"github.com/kaytu-io/open-governance/pkg/analytics/api"
	queryrunner "github.com/kaytu-io/open-governance/pkg/inventory/query-runner"
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
	JobSort_ByCreatedAt    = "created_at"
	JobSort_ByUpdatedAt    = "updated_at"
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
	BenchmarkId []string   `json:"benchmark_id"`
	JobStatus   []string   `json:"job_status"`
	StartTime   time.Time  `json:"start_time"`
	EndTime     *time.Time `json:"end_time"`
	SortBy      *string    `json:"sort_by"`
	Cursor      *int64     `json:"cursor"`
	PerPage     *int64     `json:"per_page"`
}

type GetComplianceJobsHistoryResponse struct {
	JobId           uint                `json:"job_id"`
	BenchmarkId     string              `json:"benchmark_id"`
	JobStatus       ComplianceJobStatus `json:"job_status"`
	DateTime        time.Time           `json:"date_time"`
	IntegrationInfo []IntegrationInfo   `json:"integration_info"`
}

type GetAnalyticsJobsHistoryRequest struct {
	SortBy  *string `json:"sort_by"`
	Cursor  *int64  `json:"cursor"`
	PerPage *int64  `json:"per_page"`
}

type GetAnalyticsJobsHistoryResponse struct {
	JobId     uint          `json:"job_id"`
	Type      string        `json:"type"`
	JobStatus api.JobStatus `json:"job_status"`
	DateTime  time.Time     `json:"date_time"`
}

type RunBenchmarkByIdRequest struct {
	IntegrationInfo []struct {
		Integration        *string `json:"integration"`
		Type               *string `json:"type"`
		ID                 *string `json:"id"`
		IDName             *string `json:"id_name"`
		IntegrationTracker *string `json:"integration_tracker"`
	} `json:"integration_info"`
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

type GetAsyncQueryRunJobStatusResponse struct {
	JobId          uint                          `json:"job_id"`
	QueryId        string                        `json:"query_id"`
	CreatedAt      time.Time                     `json:"created_at"`
	UpdatedAt      time.Time                     `json:"updated_at"`
	CreatedBy      string                        `json:"created_by"`
	JobStatus      queryrunner.QueryRunnerStatus `json:"job_status"`
	FailureMessage string                        `json:"failure_message"`
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

type GetDescribeJobsHistoryByIntegrationRequest struct {
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

type GetComplianceJobsHistoryByIntegrationRequest struct {
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

type CancelJobRequest struct {
	JobType         string   `json:"job_type"`
	Selector        string   `json:"selector"`
	JobId           []string `json:"job_id"`
	IntegrationInfo []struct {
		Integration        *string `json:"integration"`
		Type               *string `json:"type"`
		ID                 *string `json:"id"`
		IDName             *string `json:"id_name"`
		IntegrationTracker *string `json:"integration_tracker"`
	} `json:"integration_info"`
	Status []string `json:"status"`
}

type CancelJobResponse struct {
	JobId    string `json:"job_id"`
	JobType  string `json:"job_type"`
	Canceled bool   `json:"canceled"`
	Reason   string `json:"reason"`
}

type ListJobsByTypeRequest struct {
	JobType         string   `json:"job_type"`
	Selector        string   `json:"selector"`
	JobId           []string `json:"job_id"`
	IntegrationInfo []struct {
		Integration        *string `json:"integration"`
		Type               *string `json:"type"`
		ID                 *string `json:"id"`
		IDName             *string `json:"id_name"`
		IntegrationTracker *string `json:"integration_tracker"`
	} `json:"integration_info"`
	Status    []string     `json:"status"`
	Benchmark []string     `json:"benchmark"`
	SortBy    JobSort      `json:"sort_by"`
	SortOrder JobSortOrder `json:"sort_order"`
	UpdatedAt struct {
		From *int64 `json:"from"`
		To   *int64 `json:"to"`
	} `json:"updated_at"`
	CreatedAt struct {
		From *int64 `json:"from"`
		To   *int64 `json:"to"`
	} `json:"created_at"`
	Cursor  *int64 `json:"cursor"`
	PerPage *int64 `json:"per_page"`
}

type ListJobsByTypeItem struct {
	JobId     string    `json:"job_id"`
	JobType   string    `json:"job_type"`
	JobStatus string    `json:"job_status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type ListJobsByTypeResponse struct {
	Items      []ListJobsByTypeItem `json:"items"`
	TotalCount int                  `json:"total_count"`
}

type RunQueryResponse struct {
	ID        uint                          `json:"id"`
	CreatedAt time.Time                     `json:"created_at"`
	QueryId   string                        `json:"query_id"`
	CreatedBy string                        `json:"created_by"`
	Status    queryrunner.QueryRunnerStatus `json:"status"`
}

type GetIntegrationDiscoveryProgressRequest struct {
	IntegrationInfo []struct {
		Integration        *string `json:"integration"`
		Type               *string `json:"type"`
		ID                 *string `json:"id"`
		IDName             *string `json:"id_name"`
		IntegrationTracker *string `json:"integration_tracker"`
	} `json:"integration_info"`
	TriggerID string `json:"trigger_id"`
}

type DiscoveryProgressStatus struct {
	CreatedCount             int64 `json:"created_count"`
	QueuedCount              int64 `json:"queued_count"`
	InProgressCount          int64 `json:"in_progress_count"`
	OldResourceDeletionCount int64 `json:"old_resource_deletion"`
	TimeoutCount             int64 `json:"timeout_count"`
	FailedCount              int64 `json:"failed_count"`
	SucceededCount           int64 `json:"succeeded_count"`
	RemovingResourcesCount   int64 `json:"removing_resources_count"`
	CanceledCount            int64 `json:"canceled_count"`
}

type IntegrationDiscoveryProgressStatus struct {
	Integration    IntegrationInfo          `json:"integration"`
	ProgressStatus *DiscoveryProgressStatus `json:"progress_status"`
}

type GetIntegrationDiscoveryProgressResponse struct {
	IntegrationProgress []IntegrationDiscoveryProgressStatus `json:"integration_progress"`
	TriggerIdProgress   *DiscoveryProgressStatus             `json:"trigger_id_progress"`
}
