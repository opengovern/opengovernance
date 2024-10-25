package models

import (
	integration_type "github.com/opengovern/opengovernance/services/integration-v2/integration-type"
	"github.com/opengovern/opengovernance/services/integration-v2/models"
	"time"
)

type DiscoverIntegrationRequest struct {
	IntegrationType integration_type.IntegrationType `json:"integration_type"`
	CredentialType  string                           `json:"credential_type"`
	Credentials     map[string]any                   `json:"credentials"`
}

type DiscoverIntegrationResponse struct {
	CredentialID string        `json:"credential_id"`
	Integrations []Integration `json:"integrations"`
}

type AddIntegrationsRequest struct {
	IntegrationType integration_type.IntegrationType `json:"integration_type"`
	CredentialType  string                           `json:"credential_type"`
	IntegrationIDs  []string                         `json:"integration_ids"`
	CredentialID    string                           `json:"credential_id"`
}

type UpdateRequest struct {
	Credentials map[string]any `json:"credentials"`
}

type Integration struct {
	IntegrationTracker string                           `json:"integration_tracker"`
	IntegrationID      string                           `json:"integration_id"`
	IntegrationName    string                           `json:"integration_name"`
	Connector          string                           `json:"connector"`
	IntegrationType    integration_type.IntegrationType `json:"integration_type"`
	OnboardDate        time.Time                        `json:"onboard_date"`
	Metadata           map[string]string                `json:"metadata"`
	Annotations        map[string]string                `json:"annotations"`
	Labels             map[string]string                `json:"labels"`

	CredentialID string `json:"credential_id"`

	Lifecycle models.IntegrationLifecycle `json:"lifecycle"`
	Reason    string                      `json:"reason"`
	LastCheck *time.Time                  `json:"last_check,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type ListIntegrationsResponse struct {
	Integrations []Integration `json:"integrations"`
	TotalCount   int           `json:"total_count"`
}
