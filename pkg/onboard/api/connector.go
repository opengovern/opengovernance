package api

import (
	"time"

	"gitlab.com/keibiengine/keibi-engine/pkg/source"
)

type Connector struct {
	Code             source.Type                   `json:"code"`
	Name             string                        `json:"name"`
	Description      string                        `json:"description"`
	Direction        source.ConnectorDirectionType `json:"direction"`
	Status           source.ConnectorStatus        `json:"status"`
	Category         string                        `json:"category"`
	StartSupportDate time.Time                     `json:"startSupportDate"`
}

type ConnectorCount struct {
	Connector
	ConnectionCount int64 `json:"connection_count"`
}
