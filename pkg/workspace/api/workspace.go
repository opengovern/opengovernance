package api

import (
	"github.com/kaytu-io/kaytu-util/pkg/source"
	authApi "github.com/kaytu-io/open-governance/pkg/auth/api"
	"github.com/kaytu-io/open-governance/pkg/onboard/api"
	apiv2 "github.com/kaytu-io/open-governance/pkg/onboard/api/v2"
	"github.com/kaytu-io/open-governance/services/migrator/db/model"
	"time"
)

type CreateWorkspaceRequest struct {
	Name           string `json:"name"`
	Tier           string `json:"tier"`
	OrganizationID int    `json:"organization_id"`
}

type CreateWorkspaceResponse struct {
	ID string `json:"id"`
}

type AddCredentialRequest struct {
	SingleConnection bool                         `json:"singleConnection"`
	AWSConfig        *apiv2.AWSCredentialV2Config `json:"awsConfig"`
	AzureConfig      *api.AzureCredentialConfig   `json:"azureConfig"`
	ConnectorType    source.Type                  `json:"connectorType"`
}

type BootstrapStatus string

const (
	BootstrapStatus_OnboardConnection    BootstrapStatus = "OnboardConnection"
	BootstrapStatus_CreatingWorkspace    BootstrapStatus = "CreatingWorkspace"
	BootstrapStatus_WaitingForDiscovery  BootstrapStatus = "WaitingForDiscovery"
	BootstrapStatus_WaitingForAnalytics  BootstrapStatus = "WaitingForAnalytics"
	BootstrapStatus_WaitingForCompliance BootstrapStatus = "WaitingForCompliance"
	BootstrapStatus_Finished             BootstrapStatus = "Finished"
)

type BootstrapProgress struct {
	Done  int64 `json:"done"`
	Total int64 `json:"total"`
}

type BootstrapStatusResponse struct {
	MinRequiredConnections  int64                 `json:"minRequiredConnections"`
	MaxConnections          int64                 `json:"maxConnections"`
	ConnectionCount         map[source.Type]int64 `json:"connection_count"`
	WorkspaceCreationStatus BootstrapProgress     `json:"workspaceCreationStatus"`
	DiscoveryStatus         BootstrapProgress     `json:"discoveryStatus"`
	AnalyticsStatus         BootstrapProgress     `json:"analyticsStatus"`
	ComplianceStatus        BootstrapProgress     `json:"complianceStatus"`
}

type ChangeWorkspaceOwnershipRequest struct {
	NewOwnerUserID string `json:"newOwnerUserID"`
}

type ChangeWorkspaceNameRequest struct {
	NewName string `json:"newName"`
}

type ChangeWorkspaceTierRequest struct {
	NewTier Tier `json:"newName"`
}

type ChangeWorkspaceOrganizationRequest struct {
	NewOrgID uint `json:"newOrgID"`
}

type Workspace struct {
	ID                       string        `json:"id" example:"ws-698542025141040315"`
	Name                     string        `json:"name" example:"kaytu"`
	OwnerId                  *string       `json:"ownerId" example:"google-oauth2|204590896945502695694"`
	Status                   StateID       `json:"status" example:"PROVISIONED"`
	Tier                     Tier          `json:"tier" example:"ENTERPRISE"`
	Organization             *Organization `json:"organization,omitempty"`
	Size                     WorkspaceSize `json:"size" example:"sm"`
	CreatedAt                time.Time     `json:"createdAt" example:"2023-05-17T14:39:02.707659Z"`
	IsCreated                bool          `json:"is_created"`
	IsBootstrapInputFinished bool          `json:"is_bootstrap_input_finished"`
}

type WorkspaceResponse struct {
	Workspace
	Version string `json:"version" example:"v0.45.4"`
}

type Organization struct {
	ID           uint   `json:"id"`
	CompanyName  string `json:"companyName"`
	Url          string `json:"url"`
	Address      string `json:"address"`
	City         string `json:"city"`
	State        string `json:"state"`
	Country      string `json:"country"`
	ContactPhone string `json:"contactPhone"`
	ContactEmail string `json:"contactEmail"`
	ContactName  string `json:"contactName"`
}

type WorkspaceLimits struct {
	MaxUsers       int64 `json:"maxUsers"`
	MaxConnections int64 `json:"maxConnections"`
	MaxResources   int64 `json:"maxResources"`
}

type WorkspaceLimitsUsage struct {
	ID   string `json:"id" example:"ws-698542025141040315"`
	Name string `json:"name" example:"kaytu"`

	CurrentUsers       int64 `json:"currentUsers" example:"10"`
	CurrentConnections int64 `json:"currentConnections" example:"100"`
	CurrentResources   int64 `json:"currentResources" example:"10000"`

	MaxUsers       int64 `json:"maxUsers" example:"100"`
	MaxConnections int64 `json:"maxConnections" example:"1000"`
	MaxResources   int64 `json:"maxResources" example:"100000"`
}

type DexConnectorInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

type About struct {
	DexConnectors         []DexConnectorInfo             `json:"dex_connectors"`
	AppVersion            string                         `json:"app_version"`
	WorkspaceCreationTime time.Time                      `json:"workspace_creation_time"`
	Users                 []authApi.WorkspaceRoleBinding `json:"users"`
	PrimaryDomainURL      string                         `json:"primary_domain_url"`
	APIKeys               []authApi.WorkspaceApiKey      `json:"api_keys"`
	Integrations          map[string][]api.Connection    `json:"integrations"`
	SampleData            bool                           `json:"sample_data"`
	TotalSpendGoverned    float64                        `json:"total_spend_governed"`
}

type GetMigrationStatusResponse struct {
	Status     string                      `json:"status"`
	JobsStatus map[string]model.JobsStatus `json:"jobs_status"`
	Summary    struct {
		TotalJobs     int     `json:"total_jobs"`
		CompletedJobs int     `json:"completed_jobs"`
		Progress      float64 `json:"progress"`
	}
}

type GetSampleSyncStatusResponse struct {
	Status   string  `json:"status"`
	Progress float64 `json:"progress"`
}
