package api

type TransactionID string

const (
	Transaction_CreateWorkspaceKeyId       TransactionID = "CreateWorkspaceKeyId"
	Transaction_CreateRoleBinding          TransactionID = "CreateRoleBinding"
	Transaction_EnsureWorkspacePodsRunning TransactionID = "CreateHelmRelease"
	Transaction_EnsureDiscoveryFinished    TransactionID = "EnsureDiscoveryFinished"
	Transaction_EnsureJobsRunning          TransactionID = "EnsureJobsRunning"
	Transaction_EnsureJobsFinished         TransactionID = "EnsureJobsFinished"
)
