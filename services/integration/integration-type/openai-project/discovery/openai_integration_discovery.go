package discovery

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
)

type OrganizationResponse struct {
	OrganizationID string `json:"organization"`
	Projects       []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"projects"`
}

func OpenAIIntegrationDiscovery(apiKey string) (*OrganizationResponse, error) {
	if apiKey == "" {
		return nil, errors.New("API key is required")
	}

	// Define the endpoint
	url := "https://api.openai.com/v1/organizations"

	// Create HTTP request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %v", err)
	}

	// Add headers
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

	// Parse the response
	var orgResponse OrganizationResponse
	err = json.NewDecoder(resp.Body).Decode(&orgResponse)
	if err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	return &orgResponse, nil
}
