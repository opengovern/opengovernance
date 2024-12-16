package discovery

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

// Workplace defines the information for doppler workplace.
type Workplace struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	BillingEmail  string `json:"billing_email"`
	SecurityEmail string `json:"security_email"`
}

type Response struct {
	Workplace Workplace `json:"workplace"`
}

// Discover retrieves Render customer info
func Discover(token string) (*Workplace, error) {
	var response Response

	url := "https://api.doppler.com/v3/workplace"

	client := http.DefaultClient

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request execution failed: %w", err)
	}
	defer resp.Body.Close()

	if err = json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &response.Workplace, nil
}

func DopplerIntegrationDiscovery(cfg Config) (*Workplace, error) {
	// Check for the token
	if cfg.Token == "" {
		return nil, errors.New("token must be configured")
	}

	return Discover(cfg.Token)
}
