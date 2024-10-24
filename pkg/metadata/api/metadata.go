package api

import (
	authApi "github.com/opengovern/opengovernance/pkg/auth/api"
	"github.com/opengovern/opengovernance/pkg/onboard/api"
	"github.com/opengovern/opengovernance/services/migrator/db/model"
	"time"
)

type SetConfigMetadataRequest struct {
	Key   string `json:"key"`
	Value any    `json:"value"`
}

type DexConnectorInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

type About struct {
	DexConnectors         []DexConnectorInfo          `json:"dex_connectors"`
	AppVersion            string                      `json:"app_version"`
	WorkspaceCreationTime time.Time                   `json:"workspace_creation_time"`
	Users                 []authApi.GetUsersResponse  `json:"users"`
	PrimaryDomainURL      string                      `json:"primary_domain_url"`
	APIKeys               []authApi.APIKeyResponse    `json:"api_keys"`
	Integrations          map[string][]api.Connection `json:"integrations"`
	SampleData            bool                        `json:"sample_data"`
	TotalSpendGoverned    float64                     `json:"total_spend_governed"`
}

type GetMigrationStatusResponse struct {
	Status     string                   `json:"status"`
	JobsStatus map[string]model.JobInfo `json:"jobs_status"`
	Summary    struct {
		TotalJobs          int     `json:"total_jobs"`
		CompletedJobs      int     `json:"completed_jobs"`
		ProgressPercentage float64 `json:"progress_percentage"`
	}
}

type GetSampleSyncStatusResponse struct {
	Status   string  `json:"status"`
	Progress float64 `json:"progress"`
}
