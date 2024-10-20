package models

import (
	"time"
)

type DiscoverIntegrationRequest struct {
	IntegrationType string         `json:"integration_type"`
	CredentialType  string         `json:"credential_type"`
	Credentials     map[string]any `json:"credentials"`
}

type DiscoverIntegrationResponse struct {
	CredentialID string        `json:"credential_id"`
	Integrations []Integration `json:"integrations"`
}

type AddIntegrationsRequest struct {
	IntegrationType string   `json:"integration_type"`
	CredentialType  string   `json:"credential_type"`
	IntegrationIDs  []string `json:"integration_ids"`
	CredentialID    string   `json:"credential_id"`
}

type UpdateRequest struct {
	Credentials map[string]any `json:"credentials"`
}

type CredentialItem struct {
	ID             string `json:"id"`
	CredentialType string `json:"credential_type"`
}

type Integration struct {
	IntegrationTracker string            `json:"integration_tracker"`
	IntegrationID      string            `json:"integration_id"`
	IntegrationName    string            `json:"integration_name"`
	Connector          string            `json:"connector"`
	IntegrationType    string            `json:"integration_type"`
	OnboardDate        time.Time         `json:"onboard_date"`
	Metadata           map[string]string `json:"metadata"`
	Annotations        map[string]string `json:"annotations"`
	Labels             map[string]string `json:"labels"`

	Lifecycle string
	Reason    string
	LastCheck *time.Time

	CreatedAt time.Time
	UpdatedAt time.Time
}

type ListResponse struct {
	Integrations []Integration `json:"integrations"`
	TotalCount   int           `json:"total_count"`
}
