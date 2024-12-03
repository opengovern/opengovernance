package discovery

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"golang.org/x/oauth2/google"
	admin "google.golang.org/api/admin/directory/v1"
	"google.golang.org/api/option"
	"time"
)

// Config represents the JSON input configuration
type Config struct {
	AdminEmail string          `json:"admin_email"`
	CustomerID string          `json:"customer_id"`
	KeyFile    json.RawMessage `json:"key_file"`
}

// CustomerDetail defines the minimal information for customer.
type CustomerDetail struct {
	ID     string `json:"id,omitempty"`
	Domain string `json:"domain,omitempty"`
}

// Discover retrieves Google Workspace customer info
func Discover(ctx context.Context, service *admin.Service, customerID string) (*admin.Customer, error) {
	var customer *admin.Customer
	var err error

	customer, err = service.Customers.Get(customerID).Do()
	if err != nil {
		return nil, err
	}

	return customer, nil
}

var scopes = []string{
	admin.AdminDirectoryUserReadonlyScope,
	admin.AdminDirectoryGroupReadonlyScope,
	admin.AdminDirectoryOrgunitReadonlyScope,
	admin.AdminDirectoryDomainReadonlyScope,
	admin.AdminDirectoryDeviceMobileReadonlyScope,
	admin.AdminDirectoryDeviceChromeosReadonlyScope,
	admin.AdminDirectoryCustomerReadonlyScope,
	admin.AdminDirectoryRolemanagementReadonlyScope,
}

func GoogleWorkspaceIntegrationDiscovery(cfg Config) (*CustomerDetail, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Check for the keyFile content
	if string(cfg.KeyFile) == "" {
		return nil, errors.New("key file must be configured")
	}

	// Check for the admin email
	if string(cfg.AdminEmail) == "" {
		return nil, errors.New("admin email must be configured")
	}

	// Check for the customer id
	if string(cfg.CustomerID) == "" {
		return nil, errors.New("customer ID must be configured")
	}

	// Create credentials using the service account key
	config, err := google.JWTConfigFromJSON(cfg.KeyFile, scopes...)
	if err != nil {
		return nil, fmt.Errorf("error creating JWT config: %v", err)
	}

	// Set the Subject to the specified admin email
	config.Subject = cfg.AdminEmail

	// Create the Admin SDK service using the credentials
	service, err := admin.NewService(ctx, option.WithTokenSource(config.TokenSource(ctx)))
	if err != nil {
		return nil, fmt.Errorf("error creating Admin SDK service: %v", err)
	}

	// Get the customer information
	customer, err := Discover(ctx, service, cfg.CustomerID)
	if err != nil {
		return nil, err
	}

	// Prepare the minimal customer information
	var customerDetail CustomerDetail
	customerDetail = CustomerDetail{
		ID:     customer.Id,
		Domain: customer.CustomerDomain,
	}

	return &customerDetail, nil
}
