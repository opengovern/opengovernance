package discovery

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

// Config represents the JSON input configuration
type Config struct {
	APIKey string `json:"api_key"`
}

// User defines the information for user.
type User struct {
	Email string `json:"email"`
	Name  string `json:"name"`
}

// Discover retrieves Render customer info
func Discover(apiKey string) (*User, error) {
	var user User

	url := "https://api.render.com/v1/users"

	client := http.DefaultClient

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request execution failed: %w", err)
	}
	defer resp.Body.Close()

	if err = json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &user, nil
}

func RenderIntegrationDiscovery(cfg Config) (*User, error) {
	// Check for the api key
	if cfg.APIKey == "" {
		return nil, errors.New("api key must be configured")
	}

	return Discover(cfg.APIKey)
}
