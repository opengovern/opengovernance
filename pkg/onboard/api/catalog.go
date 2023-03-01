package api

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
	Logo            string         `json:"logo"`
	Name            string         `json:"name"`
	Category        string         `json:"category"`
	Description     string         `json:"description"`
	ConnectionCount int            `json:"connectionCount"`
	State           ConnectorState `json:"state"`
}
