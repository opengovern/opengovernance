package api

type TransactionID string

const (
	Transaction_CreateWorkspaceKeyId      TransactionID = "CreateWorkspaceKeyId"
	Transaction_CreateServiceAccountRoles TransactionID = "CreateServiceAccountRoles"
	Transaction_CreateRoleBinding         TransactionID = "CreateRoleBinding"
	Transaction_CreateMasterCredential    TransactionID = "CreateMasterCredential"
	Transaction_CreateHelmRelease         TransactionID = "CreateHelmRelease"
	Transaction_EnsureCredentialOnboarded TransactionID = "EnsureCredentialOnboarded"
	Transaction_EnsureDiscoveryFinished   TransactionID = "EnsureDiscoveryFinished"
	Transaction_EnsureJobsRunning         TransactionID = "EnsureJobsRunning"
	Transaction_EnsureJobsFinished        TransactionID = "EnsureJobsFinished"
	Transaction_EnsureCredentialExists    TransactionID = "EnsureCredentialExists"
)
