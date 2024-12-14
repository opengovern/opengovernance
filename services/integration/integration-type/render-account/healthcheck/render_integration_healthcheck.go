package healthcheck

import (
	"context"
	"errors"
	"fmt"
	admin "google.golang.org/api/admin/directory/v1"
	"net/http"
)

// Config represents the JSON input configuration
type Config struct {
	APIKey string `json:"api_key"`
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
func IsHealthy(apiKey string) error {
	url := "https://api.render.com/v1/users"

	client := http.DefaultClient

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("request execution failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to retrieve user information correctly. status code: %v", resp.StatusCode)
	}

	return nil
}

func RenderIntegrationHealthcheck(cfg Config) (bool, error) {
	// Check for the api key
	if cfg.APIKey == "" {
		return false, errors.New("api key must be configured")
	}

	err := IsHealthy(cfg.APIKey)
	if err != nil {
		return false, err
	}

	return true, nil
}
