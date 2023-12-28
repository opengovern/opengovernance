package api

type TransactionID string

const (
	Transaction_CreateServiceAccountRoles TransactionID = "CreateServiceAccountRoles"
	Transaction_CreateOpenSearch          TransactionID = "CreateOpenSearch"
	Transaction_CreateIngestionPipeline   TransactionID = "CreateIngestionPipeline"
	Transaction_StopIngestionPipeline     TransactionID = "StopIngestionPipeline"
	Transaction_CreateInsightBucket       TransactionID = "CreateInsightBucket"
	Transaction_CreateRoleBinding         TransactionID = "CreateRoleBinding"
	Transaction_CreateMasterCredential    TransactionID = "CreateMasterCredential"
	Transaction_CreateHelmRelease         TransactionID = "CreateHelmRelease"
	Transaction_EnsureCredentialOnboarded TransactionID = "EnsureCredentialOnboarded"
	Transaction_EnsureDiscoveryFinished   TransactionID = "EnsureDiscoveryFinished"
	Transaction_EnsureJobsRunning         TransactionID = "EnsureJobsRunning"
	Transaction_EnsureJobsFinished        TransactionID = "EnsureJobsFinished"
	Transaction_EnsureCredentialExists    TransactionID = "EnsureCredentialExists"
)
