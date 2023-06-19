package api

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/kaytu-io/kaytu-util/pkg/source"
)

type ConnectionLifecycleState string

const (
	ConnectionLifecycleStatePending          ConnectionLifecycleState = "pending"
	ConnectionLifecycleStateInitialDiscovery ConnectionLifecycleState = "initial-discovery"
	ConnectionLifecycleStateEnabled          ConnectionLifecycleState = "enabled"
	ConnectionLifecycleStateDisabled         ConnectionLifecycleState = "disabled"
	ConnectionLifecycleStateDeleted          ConnectionLifecycleState = "deleted"
)

func (c ConnectionLifecycleState) Validate() error {
	switch c {
	case ConnectionLifecycleStateInitialDiscovery, ConnectionLifecycleStateEnabled, ConnectionLifecycleStateDisabled:
		return nil
	default:
		return fmt.Errorf("invalid connection lifecycle state: %s", c)
	}
}

type ConnectionCountRequest struct {
	ConnectorsNames []string                  `json:"connectors"`
	State           *ConnectionLifecycleState `json:"state"`
	Health          *source.HealthStatus      `json:"health"`
}

type Connection struct {
	ID                   uuid.UUID                       `json:"id" example:"8e0f8e7a-1b1c-4e6f-b7e4-9c6af9d2b1c8"`
	ConnectionID         string                          `json:"providerConnectionID" example:"8e0f8e7a-1b1c-4e6f-b7e4-9c6af9d2b1c8"`
	ConnectionName       string                          `json:"providerConnectionName" example:"example-connection"`
	Email                string                          `json:"email" example:"johndoe@example.com"`
	Connector            source.Type                     `json:"connector" example:"Azure"`
	Description          string                          `json:"description" example:"This is an example connection"`
	CredentialID         string                          `json:"credentialID" example:"7r6123ac-ca1c-434f-b1a3-91w2w9d277c8"`
	CredentialName       *string                         `json:"credentialName,omitempty"`
	OnboardDate          time.Time                       `json:"onboardDate" example:"2023-05-07T00:00:00Z"`
	LifecycleState       ConnectionLifecycleState        `json:"lifecycleState" example:"enabled"`
	AssetDiscoveryMethod source.AssetDiscoveryMethodType `json:"assetDiscoveryMethod" example:"scheduled"`
	HealthState          source.HealthStatus             `json:"healthState" example:"healthy"`
	LastHealthCheckTime  time.Time                       `json:"lastHealthCheckTime" example:"2023-05-07T00:00:00Z"`
	HealthReason         *string                         `json:"healthReason,omitempty"`

	LastInventory *time.Time `json:"lastInventory" example:"2023-05-07T00:00:00Z"`
	Cost          *float64   `json:"cost" example:"1000.00"`
	ResourceCount *int       `json:"resourceCount" example:"100"`

	Metadata map[string]any `json:"metadata"`
}

type ChangeConnectionLifecycleStateRequest struct {
	State ConnectionLifecycleState `json:"state"`
}

type ListConnectionSummaryResponse struct {
	ConnectionCount     int          `json:"connectionCount" example:"10"`
	TotalCost           float64      `json:"totalCost" example:"1000.00"`
	TotalResourceCount  int          `json:"TotalResourceCount" example:"100"`
	TotalUnhealthyCount int          `json:"totalUnhealthyCount" example:"10"`
	TotalDisabledCount  int          `json:"totalDisabledCount" example:"10"`
	Connections         []Connection `json:"connections"`
}
