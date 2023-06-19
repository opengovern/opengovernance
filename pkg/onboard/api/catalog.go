package api

import "github.com/kaytu-io/kaytu-util/pkg/source"

type ConnectorState = string

const (
	ConnectorState_Active   ConnectorState = "ACTIVE"
	ConnectorState_NotSetup ConnectorState = "NOT_SETUP"
)

type CatalogMetrics struct {
	TotalConnections     int `json:"totalConnections" example:"20"`
	ConnectionsEnabled   int `json:"connectionsEnabled" example:"20"`
	HealthyConnections   int `json:"healthyConnections" example:"15"`
	UnhealthyConnections int `json:"unhealthyConnections" example:"5"`
}

type CatalogConnector struct {
	ID                  int            `json:"ID"`
	Logo                string         `json:"logo"`
	DisplayName         string         `json:"displayName"`
	Name                string         `json:"name"`
	Category            string         `json:"category"`
	Description         string         `json:"description"`
	ConnectionCount     int64          `json:"connectionCount" example:"1"`
	State               ConnectorState `json:"state" enums:"ACTIVE,NOT_SETUP" example:"ACTIVE"` // ACTIVE, NOT_SETUP
	SourceType          source.Type    `json:"sourceType" example:"Azure"`                      // Cloud provider
	AllowNewConnections bool           `json:"allowNewConnections" example:"true"`
	MaxConnectionsLimit int            `json:"maxConnectionsLimit" example:"10"`
	ConnectionFederator string         `json:"connectionFederator"`
}
