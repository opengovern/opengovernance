package api

import (
	"encoding/json"
	"fmt"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/subscription/armsubscription"
	kaytuAws "github.com/opengovern/og-aws-describer/aws"
	kaytuAzure "github.com/opengovern/og-azure-describer/azure"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/opengovern/og-util/pkg/source"
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
	AssetDiscovery      *bool               `json:"assetDiscovery,omitempty"`
	SpendDiscovery      *bool               `json:"spendDiscovery,omitempty"`

	LastInventory        *time.Time `json:"lastInventory" example:"2023-05-07T00:00:00Z"`
	Cost                 *float64   `json:"cost" example:"1000.00" minimum:"0" maximum:"10000000"`
	DailyCostAtStartTime *float64   `json:"dailyCostAtStartTime" example:"1000.00" minimum:"0" maximum:"10000000"`
	DailyCostAtEndTime   *float64   `json:"dailyCostAtEndTime" example:"1000.00"  minimum:"0" maximum:"10000000"`
	ResourceCount        *int       `json:"resourceCount" example:"100" minimum:"0" maximum:"1000000"`
	OldResourceCount     *int       `json:"oldResourceCount" example:"100" minimum:"0" maximum:"1000000"`

	Metadata           map[string]any `json:"metadata"`
	DescribeJobRunning bool

	supportedResourceTypes map[string]bool
}

func (c *Connection) TenantID() string {
	var TenantID string
	if tenantId, ok := c.Metadata["tenant_id"]; ok {
		if tenantIdStr, ok := tenantId.(string); ok {
			TenantID = tenantIdStr
		}
	}
	return TenantID
}

func GetAWSSupportedResourceTypeMap() map[string]bool {
	supportedMap := make(map[string]bool)
	rts := kaytuAws.GetResourceTypesMap()
	for rt := range rts {
		supportedMap[strings.ToLower(rt)] = true
	}
	return supportedMap
}

func GetAzureSupportedResourceTypeMap() map[string]bool {
	supportedMap := make(map[string]bool)
	rts := kaytuAzure.GetResourceTypesMap()
	for rt := range rts {
		supportedMap[strings.ToLower(rt)] = true
	}
	return supportedMap
}

func (c *Connection) GetSupportedResourceTypeMap() map[string]bool {
	if c.supportedResourceTypes != nil {
		return c.supportedResourceTypes
	} else {
		c.supportedResourceTypes = make(map[string]bool)
	}
	switch c.Connector {
	case source.CloudAWS:
		rts := kaytuAws.GetResourceTypesMap()
		for rt := range rts {
			c.supportedResourceTypes[strings.ToLower(rt)] = true
		}
		return c.supportedResourceTypes
	case source.CloudAzure:
		rts := kaytuAzure.GetResourceTypesMap()
		jsonC, _ := json.Marshal(c)
		// Remove cost resources if quota is not supported so we don't describe em
		if subscriptionModel, ok := c.Metadata["subscription_model"]; ok {
			jsonSubModel, _ := json.Marshal(subscriptionModel)
			var subscriptionModelObj armsubscription.Subscription
			err := json.Unmarshal(jsonSubModel, &subscriptionModelObj)
			if err == nil {
				if subscriptionModelObj.SubscriptionPolicies != nil && subscriptionModelObj.SubscriptionPolicies.QuotaID != nil {
					quotaId := *subscriptionModelObj.SubscriptionPolicies.QuotaID
					unsupportedQuotas := kaytuAzure.GetUnsupportedCostQuotaIds()
					for _, unsupportedQuota := range unsupportedQuotas {
						if strings.ToLower(quotaId) == strings.ToLower(unsupportedQuota) {
							delete(rts, "Microsoft.CostManagement/CostBySubscription")
							delete(rts, "Microsoft.CostManagement/CostByResourceType")
						}
					}
				} else {
					fmt.Printf("subscription model obj quota id not found for connection: %v\n", string(jsonC))
				}
			} else {
				fmt.Printf("subscription model obj not found for connection: %v\n", string(jsonC))
			}
		} else {
			fmt.Printf("subscription model not found for connection: %v\n", string(jsonC))
		}

		for rt := range rts {
			c.supportedResourceTypes[strings.ToLower(rt)] = true
		}

		return c.supportedResourceTypes
	}

	return nil
}

func (c Connection) IsEnabled() bool {
	if c.LifecycleState == ConnectionLifecycleStateOnboard ||
		c.LifecycleState == ConnectionLifecycleStateInProgress {
		return true
	}
	return false
}

func (c Connection) IsDiscovered() bool {
	return c.LifecycleState == ConnectionLifecycleStateDiscovered
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
