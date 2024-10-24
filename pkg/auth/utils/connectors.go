package utils

import (
	"encoding/json"
	"fmt"
	"strings"

	dexapi "github.com/dexidp/dex/api/v2"
)
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
type ConnectorCreator func( params CreateConnectorRequest) (*dexapi.CreateConnectorReq, error)

var  connectorCreators = map[string]ConnectorCreator{
	"oidc": CreateOIDCConnector,
	// Future connector types can be added here, e.g., "saml": (*DexClient).CreateSAMLConnector
}
var SupportedConnectors = map[string][]string{
	"oidc": {"general", "google-workspace", "entraid"},
	// Add more connector types and their sub-types here as needed.
}

func  CreateOIDCConnector(params CreateConnectorRequest) (*dexapi.CreateConnectorReq, error) {


	var oidcConfig OIDCConfig
	var connectorID, connectorName string
	connectorID = params.ID
	connectorName = params.Name
	switch params.ConnectorSubType {
	case "general":
		// Required: issuer, clientID, clientSecret
		oidcConfig = OIDCConfig{
			Issuer:       params.Issuer,
			ClientID:     params.ClientID,
			ClientSecret: params.ClientSecret,
		}
		

		if connectorID == "" {
			connectorID = "default-oidc"
		}
		if connectorName == "" {
			connectorName = "OIDC SSO"
		}

	case "entraid":
		// Required: tenantID, clientID, clientSecret
		oidcConfig = OIDCConfig{
			TenantID:     params.TenantID,
			ClientID:     params.ClientID,
			ClientSecret: params.ClientSecret,
		}
		

		if connectorID == "" {
			connectorID = "entraid-oidc"
		}
		if connectorName == "" {
			connectorName = "Microsoft AzureAD SSO"
		}

	case "google-workspace":
		// Required: clientID, clientSecret
		oidcConfig = OIDCConfig{
		ClientID:     params.ClientID,
			ClientSecret: params.ClientSecret,
		}
	

		if connectorID == "" {
			connectorID = "google-workspace-oidc"
		}
		if connectorName == "" {
			connectorName = "Google Workspace SSO"
		}

	default:
		return nil, fmt.Errorf("unsupported connector_sub_type: %s", params.ConnectorSubType)
	}

	// Serialize the OIDCConfig to JSON.
	configBytes, err := json.Marshal(oidcConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal OIDC config: %w", err)
	}

	// Construct the Connector message.
	connector := &dexapi.Connector{
		Id:     connectorID,
		Type:   "oidc",
		Name:   connectorName,
		Config: configBytes,
	}

	// Create the CreateConnectorReq message.
	req := &dexapi.CreateConnectorReq{
		Connector: connector,
	}

	

	// Execute the CreateConnector RPC.
	

	return req, nil
}
func IsSupportedSubType(connectorType, subType string) bool {
	subTypes, exists := SupportedConnectors[connectorType]
	if !exists {
		return false
	}
	for _, st := range subTypes {
		if strings.ToLower(st) == subType {
			return true
		}
	}
	return false
}

func GetConnectorCreator(connectorType string) ConnectorCreator {
	return connectorCreators[connectorType]
}
func GetSupportedConnectors(connectorType string) []string {
	return SupportedConnectors[connectorType]
}

