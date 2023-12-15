package api

type WorkspaceStatus string

func (ws WorkspaceStatus) String() string {
	return string(ws)
}

const (
	StatusReserving     WorkspaceStatus = "RESERVING"
	StatusReserved      WorkspaceStatus = "RESERVED"
	StatusBootstrapping WorkspaceStatus = "BOOTSTRAPPING"
	StatusProvisioned   WorkspaceStatus = "PROVISIONED"
	StatusDeleting      WorkspaceStatus = "DELETING"
	StatusDeleted       WorkspaceStatus = "DELETED"
)

type WorkspaceSize string

const (
	SizeXS WorkspaceSize = "xs"
	SizeSM WorkspaceSize = "sm"
	SizeMD WorkspaceSize = "md"
	SizeLG WorkspaceSize = "lg"
)
