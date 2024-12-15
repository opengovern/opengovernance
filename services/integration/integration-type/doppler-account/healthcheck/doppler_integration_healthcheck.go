package healthcheck

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"

)

type ConnectorResponse struct {
	Connectors []Connector `json:"connectors"`
	TotalCount float64     `json:"total_count"`
}


type Connector struct {
	ID                string    `json:"id"`
	
}



func DopplerIntegrationHealthcheck(apiKey string) (bool, error) {
	if apiKey == "" {
		return false, errors.New("API key is required")
	}

	// Endpoint to test access
	url := "https://api.cohere.com/v1/connectors"

	// Create HTTP request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return false, fmt.Errorf("error creating request: %v", err)
	}

	// Add Authorization header
	req.Header.Add("Authorization", "Bearer "+apiKey)

	// Execute the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return false, fmt.Errorf("error making request: %v", err)
	}
	defer resp.Body.Close()

	// Check for non-200 status codes
	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return false, fmt.Errorf("API error: %s, status code: %d", string(body), resp.StatusCode)
	}

	// Parse the response to ensure it contains models data
	var modelsResponse ConnectorResponse
	err = json.NewDecoder(resp.Body).Decode(&modelsResponse)
	if err != nil {
		return false, fmt.Errorf("error parsing response: %v", err)
	}

	// Validate that the token provides access to at least one model
	if len(modelsResponse.Connectors) == 0 {
		return false, nil // Token valid but no accessible models
	}

	return true, nil // Token valid and has access
}
