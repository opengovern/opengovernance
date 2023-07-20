package api

type WorkspaceStatus string

func (ws WorkspaceStatus) String() string {
	return string(ws)
}

const (
	StatusProvisioning       WorkspaceStatus = "PROVISIONING"
	StatusProvisioned        WorkspaceStatus = "PROVISIONED"
	StatusProvisioningFailed WorkspaceStatus = "PROVISIONING_FAILED"
	StatusDeleting           WorkspaceStatus = "DELETING"
	StatusDeleted            WorkspaceStatus = "DELETED"
	StatusSuspending         WorkspaceStatus = "SUSPENDING"
	StatusSuspended          WorkspaceStatus = "SUSPENDED"
)
