package healthcheck

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

// Config represents the JSON input configuration
type Config struct {
	Token string `json:"token"`
}

type Response struct {
	Success bool `json:"success"`
}

// IsHealthy checks if the JWT has read access to all required resources
func IsHealthy(token string) error {
	var response Response

	url := "https://api.doppler.com/v3/workplace"

	client := http.DefaultClient

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("request execution failed: %w", err)
	}
	defer resp.Body.Close()

	if err = json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	if !response.Success {
		return fmt.Errorf("failed to fetch workplace")
	}

	return nil
}

func DopplerIntegrationHealthcheck(cfg Config) (bool, error) {
	// Check for the token
	if cfg.Token == "" {
		return false, errors.New("api key must be configured")
	}

	err := IsHealthy(cfg.Token)
	if err != nil {
		return false, err
	}

	return true, nil
}
