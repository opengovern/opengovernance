package api

import "gitlab.com/keibiengine/keibi-engine/pkg/source"

type ConnectorState = string

const (
	ConnectorState_Active   ConnectorState = "ACTIVE"
	ConnectorState_NotSetup ConnectorState = "NOT_SETUP"
)

type CatalogMetrics struct {
	ConnectionsEnabled   int   `json:"connectionsEnabled"`
	HealthyConnections   int   `json:"healthyConnections"`
	UnhealthyConnections int   `json:"unhealthyConnections"`
	ResourcesDiscovered  int64 `json:"resourcesDiscovered"`
}

type CatalogConnector struct {
	ID                  int            `json:"ID"`
	Logo                string         `json:"logo"`
	DisplayName         string         `json:"displayName"`
	Name                string         `json:"name"`
	Category            string         `json:"category"`
	Description         string         `json:"description"`
	ConnectionCount     int64          `json:"connectionCount"`
	State               ConnectorState `json:"state"`
	SourceType          source.Type    `json:"sourceType"`
	AllowNewConnections bool           `json:"allowNewConnections"`
	MaxConnectionsLimit int            `json:"maxConnectionsLimit"`
	ConnectionFederator string         `json:"connectionFederator"`
}
