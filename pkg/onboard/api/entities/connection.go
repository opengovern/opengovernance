package entities

import (
	"encoding/json"
	"github.com/kaytu-io/kaytu-engine/pkg/onboard/api"
	"github.com/kaytu-io/kaytu-engine/services/integration/model"
)

func NewConnection(s model.Connection) api.Connection {
	metadata := make(map[string]any)
	if len(s.Metadata) > 0 {
		_ = json.Unmarshal(s.Metadata, &metadata)
	}

	conn := api.Connection{
		ID:                   s.ID,
		ConnectionID:         s.SourceId,
		ConnectionName:       s.Name,
		Email:                s.Email,
		Connector:            s.Type,
		Description:          s.Description,
		CredentialID:         s.CredentialID.String(),
		CredentialName:       s.Credential.Name,
		CredentialType:       NewCredentialType(s.Credential.CredentialType),
		OnboardDate:          s.CreatedAt,
		HealthState:          s.HealthState,
		LifecycleState:       api.ConnectionLifecycleState(s.LifecycleState),
		AssetDiscoveryMethod: s.AssetDiscoveryMethod,
		LastHealthCheckTime:  s.LastHealthCheckTime,
		HealthReason:         s.HealthReason,
		Metadata:             metadata,
		AssetDiscovery:       s.AssetDiscovery,
		SpendDiscovery:       s.SpendDiscovery,

		ResourceCount: nil,
		Cost:          nil,
		LastInventory: nil,
	}
	return conn
}
