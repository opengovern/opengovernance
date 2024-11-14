package healthcheck

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/authorization/armauthorization/v2"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/subscription/armsubscription"
	"github.com/golang-jwt/jwt"
)

const (
	// Default role definition ID for the Reader role
	DefaultRoleDefinitionID = "/providers/Microsoft.Authorization/roleDefinitions/acdd72a7-3385-48ef-bd42-f606fba81ae7"
)

// Credential interface defines methods for Azure credentials
type Credential interface {
	GetTokenCredential() azcore.TokenCredential
	CheckCredential(ctx context.Context) bool
}

// ClientSecretCredential represents credentials using a client secret
type ClientSecretCredential struct {
	TenantID     string
	ClientID     string
	ClientSecret string
	credential   azcore.TokenCredential
}

// NewClientSecretCredential creates a new ClientSecretCredential
func NewClientSecretCredential(tenantID, clientID, clientSecret string) (*ClientSecretCredential, error) {
	cred, err := azidentity.NewClientSecretCredential(
		tenantID,
		clientID,
		clientSecret,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create ClientSecretCredential: %w", err)
	}
	return &ClientSecretCredential{
		TenantID:     tenantID,
		ClientID:     clientID,
		ClientSecret: clientSecret,
		credential:   cred,
	}, nil
}

// GetTokenCredential returns the azcore.TokenCredential
func (c *ClientSecretCredential) GetTokenCredential() azcore.TokenCredential {
	return c.credential
}

// CheckCredential checks if the credential is valid
func (c *ClientSecretCredential) CheckCredential(ctx context.Context) bool {
	return checkCredential(ctx, c.credential)
}

// ClientCertificateCredential represents credentials using a client certificate
type ClientCertificateCredential struct {
	TenantID     string
	ClientID     string
	CertPath     string
	CertContent  string // New Field
	CertPassword string
	credential   azcore.TokenCredential
}

// NewClientCertificateCredential creates a new ClientCertificateCredential
func NewClientCertificateCredential(tenantID, clientID, certPath, certContent, certPassword string) (*ClientCertificateCredential, error) {
	var certData []byte
	var err error

	if certContent != "" {
		certData = []byte(certContent)
	} else if certPath != "" {
		// Read the certificate file
		certData, err = os.ReadFile(certPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read certificate file: %w", err)
		}
	} else {
		return nil, fmt.Errorf("either certPath or certContent must be provided")
	}

	var password []byte
	if certPassword != "" {
		password = []byte(certPassword)
	}

	// Parse the certificate using azidentity.ParseCertificates
	certs, key, err := azidentity.ParseCertificates(certData, password)
	if err != nil {
		return nil, fmt.Errorf("failed to parse certificate: %w", err)
	}

	// Create the ClientCertificateCredential
	cred, err := azidentity.NewClientCertificateCredential(
		tenantID,
		clientID,
		certs,
		key,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create ClientCertificateCredential: %w", err)
	}
	return &ClientCertificateCredential{
		TenantID:     tenantID,
		ClientID:     clientID,
		CertPath:     certPath,
		CertContent:  certContent,
		CertPassword: certPassword,
		credential:   cred,
	}, nil
}

// GetTokenCredential returns the azcore.TokenCredential
func (c *ClientCertificateCredential) GetTokenCredential() azcore.TokenCredential {
	return c.credential
}

// CheckCredential checks if the credential is valid
func (c *ClientCertificateCredential) CheckCredential(ctx context.Context) bool {
	return checkCredential(ctx, c.credential)
}

// checkCredential verifies that the provided TokenCredential can authenticate
func checkCredential(ctx context.Context, cred azcore.TokenCredential) bool {
	// Attempt to get a token to verify credentials
	_, err := cred.GetToken(ctx, policy.TokenRequestOptions{
		Scopes: []string{"https://management.azure.com/.default"},
	})
	if err != nil {
		log.Printf("Credential check failed: %v", err)
		return false
	}
	return true
}

// getSPNObjectID retrieves the Object ID of the SPN by parsing the JWT token
func getSPNObjectID(ctx context.Context, cred azcore.TokenCredential) (string, error) {
	token, err := cred.GetToken(ctx, policy.TokenRequestOptions{
		Scopes: []string{"https://management.azure.com/.default"},
	})
	if err != nil {
		return "", fmt.Errorf("failed to get token: %w", err)
	}

	// Parse the token to extract the oid claim
	parser := jwt.Parser{}
	claims := jwt.MapClaims{}
	_, _, err = parser.ParseUnverified(token.Token, claims)
	if err != nil {
		return "", fmt.Errorf("failed to parse JWT token: %w", err)
	}

	oid, ok := claims["oid"].(string)
	if !ok {
		return "", fmt.Errorf("failed to get oid claim from token")
	}

	return oid, nil
}

// getGUIDFromResourceID extracts the GUID from a resource ID
func getGUIDFromResourceID(resourceID string) string {
	parts := strings.Split(resourceID, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return resourceID
}

// hasRole checks if the SPN has the specified role assigned to the given subscription
func hasRole(ctx context.Context, authClientFactory *armauthorization.ClientFactory, subscriptionID, spnObjectID, roleDefinitionID string) (bool, error) {
	roleAssignmentsClient := authClientFactory.NewRoleAssignmentsClient()

	// Use the 'assignedTo' filter
	filter := fmt.Sprintf("assignedTo('%s')", spnObjectID)

	// Prepare the scope: subscriptions/{subscriptionID}
	scope := fmt.Sprintf("/subscriptions/%s", subscriptionID)

	// Create a pager to list role assignments
	pager := roleAssignmentsClient.NewListForScopePager(scope, &armauthorization.RoleAssignmentsClientListForScopeOptions{
		Filter: &filter,
	})

	roleDefinitionGUID := getGUIDFromResourceID(roleDefinitionID)

	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return false, fmt.Errorf("failed to get next page of role assignments: %w", err)
		}

		for _, roleAssignment := range page.Value {
			// Check if the role assignment matches the specified role definition ID
			if roleAssignment.Properties != nil && roleAssignment.Properties.RoleDefinitionID != nil {
				assignedRoleDefinitionID := *roleAssignment.Properties.RoleDefinitionID
				assignedRoleDefinitionGUID := getGUIDFromResourceID(assignedRoleDefinitionID)

				if strings.EqualFold(assignedRoleDefinitionGUID, roleDefinitionGUID) {
					return true, nil
				}
			}
		}
	}

	// If we reach here, no matching role assignment was found
	return false, nil
}

// RoleDetail contains details about the role definition
type RoleDetail struct {
	Description        *string                        `json:"description,omitempty"`
	Permissions        []*armauthorization.Permission `json:"permissions,omitempty"`
	RoleName           *string                        `json:"roleName,omitempty"`
	Type               *string                        `json:"type,omitempty"`
	RoleDefinitionId   *string                        `json:"roleDefinitionId,omitempty"`
	RoleDefinitionName *string                        `json:"roleDefinitionName,omitempty"`
}

// Output represents the final output structure
type Output struct {
	SubscriptionId    string       `json:"subscriptionId"`
	SubscriptionName  string       `json:"subscriptionName"`
	SubscriptionState string       `json:"subscriptionState"`
	RoleId            string       `json:"roleId"`
	Status            string       `json:"status"`
	RoleDetails       []RoleDetail `json:"roleDetails"`
}

// HealthChecker interface defines a method to check health status
type HealthChecker interface {
	IsHealthy(ctx context.Context, roleDefinitionID string) (bool, error)
}

// Subscription represents an Azure subscription
type Subscription struct {
	ID    string
	Name  string
	State string

	authClientFactory *armauthorization.ClientFactory
	spnObjectID       string
}

// IsHealthy checks if the subscription is healthy based on role assignment
func (s *Subscription) IsHealthy(ctx context.Context, roleDefinitionID string) (bool, error) {
	return hasRole(ctx, s.authClientFactory, s.ID, s.spnObjectID, roleDefinitionID)
}

// NewSubscription creates a new Subscription instance
func NewSubscription(ctx context.Context, subscriptionID string, credential azcore.TokenCredential, spnObjectID string) (*Subscription, error) {
	// Create an authorization client factory
	authClientFactory, err := armauthorization.NewClientFactory(subscriptionID, credential, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create authorization client factory: %w", err)
	}

	// Create a subscriptions client
	subscriptionsClient, err := armsubscription.NewSubscriptionsClient(credential, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create subscriptions client: %w", err)
	}

	// Get subscription details
	subscriptionResp, err := subscriptionsClient.Get(ctx, subscriptionID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get subscription details for subscription %s: %w", subscriptionID, err)
	}

	subscriptionName := ""
	if subscriptionResp.DisplayName != nil {
		subscriptionName = *subscriptionResp.DisplayName
	}

	subscriptionState := ""
	if subscriptionResp.State != nil {
		subscriptionState = string(*subscriptionResp.State)
	}

	return &Subscription{
		ID:                subscriptionID,
		Name:              subscriptionName,
		State:             subscriptionState,
		authClientFactory: authClientFactory,
		spnObjectID:       spnObjectID,
	}, nil
}

// authenticate handles credential creation and validation
func authenticate(ctx context.Context, config *Config) (Credential, error) {
	var credential Credential
	var err error

	if config.CertContent != "" || config.CertPath != "" {
		// Use certificate-based authentication
		credential, err = NewClientCertificateCredential(config.TenantID, config.ClientID, config.CertPath, config.CertContent, config.CertPassword)
		if err != nil {
			return nil, fmt.Errorf("failed to create certificate credential: %w", err)
		}
	} else if config.ClientSecret != "" {
		// Use client secret authentication
		credential, err = NewClientSecretCredential(config.TenantID, config.ClientID, config.ClientSecret)
		if err != nil {
			return nil, fmt.Errorf("failed to create client secret credential: %w", err)
		}
	} else {
		return nil, fmt.Errorf("no valid authentication method found. Provide clientSecret or certPath/certContent in the JSON input.")
	}

	// Check if the credentials are valid
	if !credential.CheckCredential(ctx) {
		return nil, fmt.Errorf("credentials are invalid or cannot authenticate")
	}

	return credential, nil
}

// getRoleDefinition retrieves the role definition
func getRoleDefinition(ctx context.Context, authClientFactory *armauthorization.ClientFactory, roleDefinitionID string) (*armauthorization.RoleDefinition, error) {
	roleDefinitionsClient := authClientFactory.NewRoleDefinitionsClient()

	roleDefinitionResp, err := roleDefinitionsClient.GetByID(ctx, roleDefinitionID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get role definition: %w", err)
	}

	return &roleDefinitionResp.RoleDefinition, nil
}

// buildOutput constructs the output data
func buildOutput(subscription *Subscription, roleDefinitionID string, status string, roleDefinition *armauthorization.RoleDefinition) Output {
	return Output{
		SubscriptionId:    subscription.ID,
		SubscriptionName:  subscription.Name,
		SubscriptionState: subscription.State,
		RoleId:            roleDefinitionID,
		Status:            status,
		RoleDetails: []RoleDetail{
			{
				Description:        roleDefinition.Properties.Description,
				Permissions:        roleDefinition.Properties.Permissions,
				RoleName:           roleDefinition.Properties.RoleName,
				Type:               roleDefinition.Type,
				RoleDefinitionId:   roleDefinition.ID,
				RoleDefinitionName: roleDefinition.Name,
			},
		},
	}
}

// Config represents the JSON input configuration
type Config struct {
	ObjectID         string `json:"object_id,omitempty"`
	TenantID         string `json:"tenant_id"`
	ClientID         string `json:"client_id"`
	ClientSecret     string `json:"client_secret,omitempty"`
	CertPath         string `json:"cert_path,omitempty"`
	CertContent      string `json:"cert_content,omitempty"` // New Field
	CertPassword     string `json:"cert_password,omitempty"`
	SubscriptionID   string `json:"subscription_id"`
	RoleDefinitionID string `json:"role_definition_id,omitempty"`
}

func AzureIntegrationHealthcheck(config Config) (bool, error) {
	ctx := context.Background()

	// Validate required fields
	if config.TenantID == "" || config.ClientID == "" || config.SubscriptionID == "" {
		return false, fmt.Errorf("tenantId, clientId, and subscriptionId are required in the configuration.")
	}

	// Authenticate and get credentials
	credential, err := authenticate(ctx, &config)
	if err != nil {
		return false, fmt.Errorf("authentication failed: %v", err)
	}

	// Get the SPN's Object ID if not provided
	spnObjectID := config.ObjectID
	if spnObjectID == "" {
		spnObjectID, err = getSPNObjectID(ctx, credential.GetTokenCredential())
		if err != nil {
			return false, fmt.Errorf("failed to get SPN Object ID: %v", err)
		}
	}

	// Create a Subscription instance
	subscription, err := NewSubscription(ctx, config.SubscriptionID, credential.GetTokenCredential(), spnObjectID)
	if err != nil {
		return false, fmt.Errorf("failed to create subscription instance: %v", err)
	}

	// Use provided RoleDefinitionID or default
	roleDefinitionID := config.RoleDefinitionID
	if roleDefinitionID == "" {
		roleDefinitionID = DefaultRoleDefinitionID // Use the default role ID
	}

	// Check if the subscription is healthy
	isHealthy, err := subscription.IsHealthy(ctx, roleDefinitionID)
	if err != nil {
		log.Fatalf("Failed to check subscription health: %v", err)
	}

	return isHealthy, nil
}
