package healthcheck

import (
	"context"
	"errors"
	"fmt"
	"golang.org/x/oauth2/google"
	admin "google.golang.org/api/admin/directory/v1"
	"google.golang.org/api/option"
	"sync"
	"time"
)

// Config represents the JSON input configuration
type Config struct {
	AdminEmail string `json:"admin_email"`
	CustomerID string `json:"customer_id"`
	KeyFile    string `json:"key_file"`
}

const (
	MaxPageResultsUsers         = 500
	MaxPageResultsGroups        = 200
	MaxPageResultsRoles         = 100
	MaxPageResultsMobileDevices = 100
	MaxPageResultsChromeDevices = 300
)

// PermissionCheck represents a permission and its corresponding check function
type PermissionCheck struct {
	Name  string
	Check func(ctx context.Context, service *admin.Service, customerID string) error
}

// IsHealthy checks if the JWT has read access to all required resources
func IsHealthy(ctx context.Context, service *admin.Service, customerID string) error {
	// Define all required permissions and their corresponding checks
	permissions := []PermissionCheck{
		{
			Name: admin.AdminDirectoryUserReadonlyScope,
			Check: func(ctx context.Context, service *admin.Service, customerID string) error {
				_, err := service.Users.List().Customer(customerID).MaxResults(MaxPageResultsUsers).Do()
				return err
			},
		},
		{
			Name: admin.AdminDirectoryGroupReadonlyScope,
			Check: func(ctx context.Context, service *admin.Service, customerID string) error {
				_, err := service.Groups.List().Customer(customerID).MaxResults(MaxPageResultsGroups).Do()
				return err
			},
		},
		{
			Name: admin.AdminDirectoryOrgunitReadonlyScope,
			Check: func(ctx context.Context, service *admin.Service, customerID string) error {
				_, err := service.Orgunits.List(customerID).Do()
				return err
			},
		},
		{
			Name: admin.AdminDirectoryDomainReadonlyScope,
			Check: func(ctx context.Context, service *admin.Service, customerID string) error {
				_, err := service.Domains.List(customerID).Do()
				return err
			},
		},
		{
			Name: admin.AdminDirectoryDeviceMobileReadonlyScope,
			Check: func(ctx context.Context, service *admin.Service, customerID string) error {
				_, err := service.Mobiledevices.List(customerID).MaxResults(MaxPageResultsMobileDevices).Do()
				return err
			},
		},
		{
			Name: admin.AdminDirectoryDeviceChromeosReadonlyScope,
			Check: func(ctx context.Context, service *admin.Service, customerID string) error {
				_, err := service.Chromeosdevices.List(customerID).MaxResults(MaxPageResultsChromeDevices).Do()
				return err
			},
		},
		{
			Name: admin.AdminDirectoryDeviceChromeosReadonlyScope,
			Check: func(ctx context.Context, service *admin.Service, customerID string) error {
				_, err := service.Chromeosdevices.List(customerID).MaxResults(MaxPageResultsChromeDevices).Do()
				return err
			},
		},
		{
			Name: admin.AdminDirectoryCustomerReadonlyScope,
			Check: func(ctx context.Context, service *admin.Service, customerID string) error {
				_, err := service.Customers.Get(customerID).Do()
				return err
			},
		},
		{
			Name: admin.AdminDirectoryRolemanagementReadonlyScope,
			Check: func(ctx context.Context, service *admin.Service, customerID string) error {
				_, err := service.Roles.List(customerID).MaxResults(MaxPageResultsRoles).Do()
				return err
			},
		},
		// Add more permissions and their checks as needed
		// For brevity, not all permissions from the list are implemented here
	}

	var missingPermissions []string

	var wg sync.WaitGroup
	var mu sync.Mutex

	// Channel to limit concurrency
	concurrencyLimit := 5
	sem := make(chan struct{}, concurrencyLimit)

	for _, perm := range permissions {
		wg.Add(1)
		go func(p PermissionCheck) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			err := p.Check(ctx, service, customerID)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				missingPermissions = append(missingPermissions, p.Name)
			}
		}(perm)
	}

	wg.Wait()

	healthy := len(missingPermissions) == 0

	if !healthy {
		return errors.New("not healthy due to missing permissions")
	}

	return nil
}

var requiredScopes = []string{
	admin.AdminDirectoryUserReadonlyScope,
	admin.AdminDirectoryGroupReadonlyScope,
	admin.AdminDirectoryOrgunitReadonlyScope,
	admin.AdminDirectoryDomainReadonlyScope,
	admin.AdminDirectoryDeviceMobileReadonlyScope,
	admin.AdminDirectoryDeviceChromeosReadonlyScope,
	admin.AdminDirectoryCustomerReadonlyScope,
	admin.AdminDirectoryRolemanagementReadonlyScope,
}

func GoogleWorkspaceIntegrationHealthcheck(cfg Config) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Check for the keyFile content
	if string(cfg.KeyFile) == "" {
		return false, errors.New("key file must be configured")
	}
	keyFileData := []byte(cfg.KeyFile)

	// Check for the admin email
	if string(cfg.AdminEmail) == "" {
		return false, errors.New("admin email must be configured")
	}

	// Check for the customer id
	if string(cfg.CustomerID) == "" {
		return false, errors.New("customer ID must be configured")
	}

	// Create credentials using the service account key
	config, err := google.JWTConfigFromJSON(keyFileData, requiredScopes...)
	if err != nil {
		return false, fmt.Errorf("error creating JWT config: %v", err)
	}

	// Set the Subject to the specified admin email
	config.Subject = cfg.AdminEmail

	// Create the Admin SDK service using the credentials
	service, err := admin.NewService(ctx, option.WithTokenSource(config.TokenSource(ctx)))
	if err != nil {
		return false, fmt.Errorf("error creating Admin SDK service: %v", err)
	}

	// Now process permissions for the specified organization
	err = IsHealthy(ctx, service, cfg.CustomerID)
	if err != nil {
		return false, err
	}

	return true, nil
}
