package entity

import (
	"github.com/kaytu-io/kaytu-util/pkg/source"
)

type Connector struct {
	Name                source.Type                   `json:"name" example:"Azure"`
	Label               string                        `json:"label" example:"Azure"`
	ShortDescription    string                        `json:"shortDescription" example:"This is a short Description for this connector"`
	Description         string                        `json:"description" example:"This is a long volume of words for just showing the case of the description for the demo and checking value purposes only and has no meaning whatsoever"`
	Direction           source.ConnectorDirectionType `json:"direction"`
	Status              source.ConnectorStatus        `json:"status" example:"enabled"`
	Logo                string                        `json:"logo" example:"https://kaytu.io/logo.png"`
	AutoOnboardSupport  bool                          `json:"autoOnboardSupport" example:"false"`
	AllowNewConnections bool                          `json:"allowNewConnections" example:"true"`
	MaxConnectionLimit  int                           `json:"maxConnectionLimit" example:"10000" minimum:"0"`
	Tags                map[string]any                `json:"tags"`
}

type ConnectorCount struct {
	Connector
	ConnectionCount int64 `json:"connection_count" example:"1024" minimum:"0"`
}

type ConnectorState = string

const (
	ConnectorState_Active   ConnectorState = "ACTIVE"
	ConnectorState_NotSetup ConnectorState = "NOT_SETUP"
)

type CatalogMetrics struct {
	TotalConnections      int `json:"totalConnections" example:"20" minimum:"0"`
	ConnectionsEnabled    int `json:"connectionsEnabled" example:"20" minimum:"0"`
	HealthyConnections    int `json:"healthyConnections" example:"15" minimum:"0"`
	UnhealthyConnections  int `json:"unhealthyConnections" example:"5" minimum:"0"`
	InProgressConnections int `json:"inProgressConnections" example:"5" minimum:"0"`
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
