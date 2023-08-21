package api

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/kaytu-io/kaytu-util/pkg/source"
)

type ConnectionLifecycleState string

const (
	ConnectionLifecycleStateOnboard    ConnectionLifecycleState = "ONBOARD"
	ConnectionLifecycleStateDisabled   ConnectionLifecycleState = "DISABLED"
	ConnectionLifecycleStateDiscovered ConnectionLifecycleState = "DISCOVERED"
	ConnectionLifecycleStateInProgress ConnectionLifecycleState = "IN_PROGRESS"
	ConnectionLifecycleStateArchived   ConnectionLifecycleState = "ARCHIVED"
)

func (c ConnectionLifecycleState) Validate() error {
	switch c {
	case ConnectionLifecycleStateInProgress, ConnectionLifecycleStateOnboard, ConnectionLifecycleStateDisabled:
		return nil
	default:
		return fmt.Errorf("invalid connection lifecycle state: %s", c)
	}
}

type ConnectionCountRequest struct {
	ConnectorsNames []string                  `json:"connectors" example:"Azure"`
	State           *ConnectionLifecycleState `json:"state" example:"enabled"`
}

type Connection struct {
	ID                   uuid.UUID                       `json:"id" example:"8e0f8e7a-1b1c-4e6f-b7e4-9c6af9d2b1c8"`
	ConnectionID         string                          `json:"providerConnectionID" example:"8e0f8e7a-1b1c-4e6f-b7e4-9c6af9d2b1c8"`
	ConnectionName       string                          `json:"providerConnectionName" example:"example-connection"`
	Email                string                          `json:"email" example:"johndoe@example.com"`
	Connector            source.Type                     `json:"connector" example:"Azure"`
	Description          string                          `json:"description" example:"This is an example connection"`
	OnboardDate          time.Time                       `json:"onboardDate" example:"2023-05-07T00:00:00Z"`
	AssetDiscoveryMethod source.AssetDiscoveryMethodType `json:"assetDiscoveryMethod" example:"scheduled"`

	CredentialID   string         `json:"credentialID" example:"7r6123ac-ca1c-434f-b1a3-91w2w9d277c8"`
	CredentialName *string        `json:"credentialName,omitempty"`
	CredentialType CredentialType `json:"credentialType" example:"manual"`
	Credential     Credential     `json:"credential,omitempty"`

	LifecycleState ConnectionLifecycleState `json:"lifecycleState" example:"enabled"`

	HealthState         source.HealthStatus `json:"healthState" example:"healthy"`
	LastHealthCheckTime time.Time           `json:"lastHealthCheckTime" example:"2023-05-07T00:00:00Z"`
	HealthReason        *string             `json:"healthReason,omitempty"`

	LastInventory        *time.Time `json:"lastInventory" example:"2023-05-07T00:00:00Z"`
	Cost                 *float64   `json:"cost" example:"1000.00" minimum:"0" maximum:"10000000"`
	DailyCostAtStartTime *float64   `json:"dailyCostAtStartTime" example:"1000.00" minimum:"0" maximum:"10000000"`
	DailyCostAtEndTime   *float64   `json:"dailyCostAtEndTime" example:"1000.00"  minimum:"0" maximum:"10000000"`
	ResourceCount        *int       `json:"resourceCount" example:"100" minimum:"0" maximum:"1000000"`
	OldResourceCount     *int       `json:"oldResourceCount" example:"100" minimum:"0" maximum:"1000000"`

	Metadata map[string]any `json:"metadata"`
}

func (c Connection) IsEnabled() bool {
	if c.LifecycleState == ConnectionLifecycleStateOnboard ||
		c.LifecycleState == ConnectionLifecycleStateInProgress {
		return true
	}
	return false
}

type ChangeConnectionLifecycleStateRequest struct {
	State ConnectionLifecycleState `json:"state"`
}

type ListConnectionSummaryResponse struct {
	ConnectionCount       int     `json:"connectionCount" example:"10" minimum:"0" maximum:"1000"`
	TotalCost             float64 `json:"totalCost" example:"1000.00" minimum:"0" maximum:"10000000"`
	TotalResourceCount    int     `json:"totalResourceCount" example:"100" minimum:"0" maximum:"1000000"`
	TotalOldResourceCount int     `json:"totalOldResourceCount" example:"100" minimum:"0" maximum:"1000000"`
	TotalUnhealthyCount   int     `json:"totalUnhealthyCount" example:"10" minimum:"0" maximum:"100"`

	TotalDisabledCount   int          `json:"totalDisabledCount" example:"10" minimum:"0" maximum:"100"`
	TotalDiscoveredCount int          `json:"totalDiscoveredCount" example:"10" minimum:"0" maximum:"100"`
	TotalOnboardedCount  int          `json:"totalOnboardedCount" example:"10" minimum:"0" maximum:"100"` // Also includes in-progress
	TotalArchivedCount   int          `json:"totalArchivedCount" example:"10" minimum:"0" maximum:"100"`
	Connections          []Connection `json:"connections"`
}

type ChangeConnectionRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Email       string `json:"email"`
}
