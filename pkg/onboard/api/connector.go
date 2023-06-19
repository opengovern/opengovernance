package api

import (
	"github.com/kaytu-io/kaytu-util/pkg/source"
)

type Connector struct {
	Name                source.Type                   `json:"name" example:"Azure"`
	Label               string                        `json:"label" example:"Azure"`
	ShortDescription    string                        `json:"shortDescription"`
	Description         string                        `json:"description"`
	Direction           source.ConnectorDirectionType `json:"direction"`
	Status              source.ConnectorStatus        `json:"status"`
	Logo                string                        `json:"logo"`
	AutoOnboardSupport  bool                          `json:"autoOnboardSupport"`
	AllowNewConnections bool                          `json:"allowNewConnections"`
	MaxConnectionLimit  int                           `json:"maxConnectionLimit"`
	Tags                map[string]any                `json:"tags"`
}

type ConnectorCount struct {
	Connector
	ConnectionCount int64 `json:"connection_count"`
}
