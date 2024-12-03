package discovery

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

)

type ConnectorResponse struct {
	Connectors []Connector `json:"connectors"`
	TotalCount float64     `json:"total_count"`
}




type Connector struct {
	ID                string    `json:"id"`
	Name              string    `json:"name"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
	OrganizationID    string    `json:"organization_id"`
	Description       string    `json:"description"`
	URL               string    `json:"url"`
	Excludes          []string  `json:"excludes"`
	AuthType          string    `json:"auth_type"`
	Oauth             Oauth     `json:"oauth"`
	AuthStatus        string    `json:"auth_status"`
	Active            bool      `json:"active"`
	ContinueOnFailure bool      `json:"continue_on_failure"`
}

type Oauth struct {
	AuthorizeURL string `json:"authorize_url"`
	TokenURL     string `json:"token_url"`
}



func CohereAIIntegrationDiscovery(apiKey string) ([]Connector, error) {
	if apiKey == "" {
		return nil, errors.New("API key is required")
	}

	// Endpoint to test access
	url := "https://api.cohere.com/v1/connectors"

	// Create HTTP request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %v", err)
	}

	// Add Authorization header
	req.Header.Add("Authorization", "Bearer "+apiKey)

	// Execute the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making request: %v", err)
	}
	defer resp.Body.Close()

	// Check for non-200 status codes
	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error: %s, status code: %d", string(body), resp.StatusCode)
	}

	// Parse the response to ensure it contains models data
	var modelsResponse ConnectorResponse
	err = json.NewDecoder(resp.Body).Decode(&modelsResponse)
	if err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	// Validate that the token provides access to at least one model
	if len(modelsResponse.Connectors) == 0 {
		return nil, nil // Token valid but no accessible models
	}

	return modelsResponse.Connectors, nil // Token valid and has access
}
