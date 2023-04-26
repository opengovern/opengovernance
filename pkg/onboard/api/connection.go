package api

import (
	"gitlab.com/keibiengine/keibi-engine/pkg/source"
)

type ConnectionState string

const (
	ConnectionState_ENABLED  ConnectionState = "ENABLED"
	ConnectionState_DISABLED ConnectionState = "DISABLED"
)

type ConnectionLifecycleState string

const (
	ConnectionLifecycleStatePending          ConnectionLifecycleState = "pending"
	ConnectionLifecycleStateInitialDiscovery ConnectionLifecycleState = "initial-discovery"
	ConnectionLifecycleStateEnabled          ConnectionLifecycleState = "enabled"
	ConnectionLifecycleStateDisabled         ConnectionLifecycleState = "disabled"
	ConnectionLifecycleStateDeleted          ConnectionLifecycleState = "deleted"
)

type ConnectionCountRequest struct {
	ConnectorsNames []string             `json:"connectors"`
	State           *ConnectionState     `json:"state"`
	Health          *source.HealthStatus `json:"health"`
}
