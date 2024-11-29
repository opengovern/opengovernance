package healthcheck

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// ProfileResponse represents the structure of the response from the /profile endpoint.
type ProfileResponse struct {
	Restricted bool `json:"restricted"`
}

// LinodeIntegrationHealthcheck checks if the given Linode API token is valid and determines if it is restricted.
func LinodeIntegrationHealthcheck(token string) (bool, error) {
	const linodeProfileURL = "https://api.linode.com/v4/profile" // Linode profile endpoint.

	client := &http.Client{}
	req, err := http.NewRequest("GET", linodeProfileURL, nil)
	if err != nil {
		return false, fmt.Errorf("failed to create request: %w", err)
	}

	// Set Authorization header with the token.
	req.Header.Set("Authorization", "Bearer "+token)

	// Execute the request.
	resp, err := client.Do(req)
	if err != nil {
		return false, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	// Check if the response indicates the token is valid.
	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("invalid token or insufficient permissions, status code: %d", resp.StatusCode)
	}

	// Parse the response to check the `Restricted` field.
	var profile ProfileResponse
	if err := json.NewDecoder(resp.Body).Decode(&profile); err != nil {
		return false, fmt.Errorf("failed to parse response: %w", err)
	}

	// If the token is valid but restricted, log the restriction (optional).
	if profile.Restricted {
		return false, nil
	}

	// Return true since the token is valid, even if restricted.
	return true, nil
}
