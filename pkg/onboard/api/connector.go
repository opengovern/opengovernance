package api

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
