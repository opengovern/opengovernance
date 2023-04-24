package api

import (
	"gitlab.com/keibiengine/keibi-engine/pkg/source"
)

type Connector struct {
	Name                source.Type                   `json:"name"`
	Label               string                        `json:"label"`
	ShortDescription    string                        `json:"shortDescription"`
	Description         string                        `json:"description"`
	Direction           source.ConnectorDirectionType `json:"direction"`
	Status              source.ConnectorStatus        `json:"status"`
	Logo                string                        `json:"logo"`
	AutoOnboardSupport  bool                          `json:"autoOnboardSupport"`
	AllowNewConnections bool                          `json:"allowNewConnections"`
	MaxConnectionLimit  int                           `json:"maxConnectionLimit"`
	Attributes          map[string]any                `json:"attributes"`
}

type ConnectorCount struct {
	Connector
	ConnectionCount int64 `json:"connection_count"`
}
