package discovery

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

type LinodeAccount struct {
	ID             int    `json:"id"`
	Email          string `json:"email"`
	Username       string `json:"username"`
	Company        string `json:"company"`
	TaxID          string `json:"tax_id"`
	CreditBalance  string `json:"credit_balance"`
	Restricted     bool   `json:"restricted"`
	BillingAddress struct {
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
		Company   string `json:"company"`
		Address1  string `json:"address_1"`
		Address2  string `json:"address_2"`
		City      string `json:"city"`
		State     string `json:"state"`
		Country   string `json:"country"`
		Zip       string `json:"zip"`
	} `json:"billing_address"`
}

// LinodeIntegrationDiscovery fetches Linode account details using the provided token.
func LinodeIntegrationDiscovery(token string) (*LinodeAccount, error) {
	const linodeAPIURL = "https://api.linode.com/v4/account"

	client := &http.Client{}
	req, err := http.NewRequest("GET", linodeAPIURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set the Authorization header with the token.
	req.Header.Set("Authorization", "Bearer "+token)

	// Perform the request.
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	// Check if the response status is not OK.
	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to fetch account: %s, status: %d", string(body), resp.StatusCode)
	}

	// Decode the JSON response.
	var account LinodeAccount
	if err := json.NewDecoder(resp.Body).Decode(&account); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &account, nil
}
