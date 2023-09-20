package api

type WorkspaceStatus string

func (ws WorkspaceStatus) String() string {
	return string(ws)
}

const (
	StatusProvisioned        WorkspaceStatus = "PROVISIONED"
	StatusProvisioning       WorkspaceStatus = "PROVISIONING"
	StatusProvisioningFailed WorkspaceStatus = "PROVISIONING_FAILED"
	StatusDeleting           WorkspaceStatus = "DELETING"
	StatusDeleted            WorkspaceStatus = "DELETED"
	StatusSuspending         WorkspaceStatus = "SUSPENDING"
	StatusSuspended          WorkspaceStatus = "SUSPENDED"
)

type WorkspaceSize string

const (
	SizeXS WorkspaceSize = "xs"
	SizeSM WorkspaceSize = "sm"
	SizeMD WorkspaceSize = "md"
	SizeLG WorkspaceSize = "lg"
)
