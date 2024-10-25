package api


// CreateConnectorRequest represents the expected payload for creating or updating a connector.
type CreateConnectorRequest struct {

	ConnectorType    string `json:"connector_type" validate:"required,oneof=oidc"`                                  // 'oidc' is supported for now
	ConnectorSubType string `json:"connector_sub_type" validate:"omitempty,oneof=general google-workspace entraid"` // Optional sub-type
	Issuer           string `json:"issuer,omitempty" validate:"omitempty,url"`
	TenantID         string `json:"tenant_id,omitempty" validate:"omitempty,uuid"`
	ClientID         string `json:"client_id" validate:"required"`
	ClientSecret     string `json:"client_secret" validate:"required"`
	ID               string `json:"id,omitempty"`   // Optional
	Name             string `json:"name,omitempty"` // Optional
}
type UpdateConnectorRequest struct {

	ConnectorType    string `json:"connector_type" validate:"required,oneof=oidc"`                                  // 'oidc' is supported for now
	ConnectorSubType string `json:"connector_sub_type" validate:"omitempty,oneof=general google-workspace entraid"` // Optional sub-type
	Issuer           string `json:"issuer,omitempty" validate:"omitempty,url"`
	TenantID         string `json:"tenant_id,omitempty" validate:"omitempty,uuid"`
	ClientID         string `json:"client_id" validate:"required"`
	ClientSecret     string `json:"client_secret" validate:"required"`
	ID               string `json:"id,omitempty"`   // Optional
	Name             string `json:"name,omitempty"` // Optional
	IsActive		 bool `json:"is_active"` 

}

type OIDCConfig struct {
	Issuer       string `json:"issuer,omitempty"`
	TenantID     string `json:"tenantID,omitempty"` // Added TenantID for entraid sub-type
	ClientID     string `json:"clientID"`
	ClientSecret string `json:"clientSecret"`
}

type GetConnectorsResponse struct {
		ID       uint `json:"id"`
		ConnectorID string `json:"connector_id"`
		Type     string `json:"type"`
		SubType  string `json:"sub_type"`
		Name     string `json:"name"`
		Issuer   string `json:"issuer,omitempty"`
		ClientID string `json:"client_id,omitempty"`
		TenantID string `json:"tenant_id,omitempty"`
		IsActive bool   `json:"is_active"`
		UserCount uint `json:"user_count"`
		CreatedAt any `json:"created_at"`
	}


type GetSupportedConnectorTypeResponse struct {
		ConnectorType string   `json:"connector_type"`
		SubTypes      []string `json:"sub_types"`
	}
