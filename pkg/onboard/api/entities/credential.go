package entities

import (
	"encoding/json"
	"github.com/opengovern/opengovernance/pkg/onboard/api"
	"github.com/opengovern/opengovernance/services/integration/model"
)

func NewCredential(credential model.Credential) api.Credential {
	metadata := make(map[string]any)
	if string(credential.Metadata) == "" {
		credential.Metadata = []byte("{}")
	}
	_ = json.Unmarshal(credential.Metadata, &metadata)
	apiCredential := api.Credential{
		ID:                  credential.ID.String(),
		Name:                credential.Name,
		ConnectorType:       credential.ConnectorType,
		CredentialType:      NewCredentialType(credential.CredentialType),
		Enabled:             credential.Enabled,
		AutoOnboardEnabled:  credential.AutoOnboardEnabled,
		OnboardDate:         credential.CreatedAt,
		LastHealthCheckTime: credential.LastHealthCheckTime,
		HealthStatus:        credential.HealthStatus,
		HealthReason:        credential.HealthReason,
		Metadata:            metadata,
		Version:             credential.Version,
		SpendDiscovery:      credential.SpendDiscovery,

		Config: "",

		Connections:           nil,
		TotalConnections:      nil,
		OnboardConnections:    nil,
		UnhealthyConnections:  nil,
		DiscoveredConnections: nil,
	}

	return apiCredential
}

func NewCredentialType(c model.CredentialType) api.CredentialType {
	return api.CredentialType(c)
}
