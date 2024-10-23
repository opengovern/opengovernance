package api

import(
	"github.com/opengovern/opengovernance/pkg/authV2/utils"
	dexapi "github.com/dexidp/dex/api/v2"
	
)
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

type OIDCConfig struct {
	Issuer       string `json:"issuer,omitempty"`
	TenantID     string `json:"tenantID,omitempty"` // Added TenantID for entraid sub-type
	ClientID     string `json:"clientID"`
	ClientSecret string `json:"clientSecret"`
}

type ConnectorCreator func( params utils.CreateConnectorRequest) (*dexapi.CreateConnectorReq, error)

var connectorCreators = map[string]ConnectorCreator{
	"oidc": utils.CreateOIDCConnector,
	// Future connector types can be added here, e.g., "saml": (*DexClient).CreateSAMLConnector
}
var SupportedConnectors = map[string][]string{
	"oidc": {"general", "google-workspace", "entraid"},
	// Add more connector types and their sub-types here as needed.
}