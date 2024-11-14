package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"sync"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
)

// Config represents the JSON input configuration
type Config struct {
	ObjectID         string `json:"objectId,omitempty"`
	TenantID         string `json:"tenantId"`
	ClientID         string `json:"clientId"`
	ClientSecret     string `json:"clientSecret,omitempty"`
	CertPath         string `json:"certPath,omitempty"`
	CertContent      string `json:"certContent,omitempty"`
	CertPassword     string `json:"certPassword,omitempty"`
	SubscriptionID   string `json:"subscriptionId"`
	RoleDefinitionID string `json:"roleDefinitionId,omitempty"`
}

// TenantInfo represents the comprehensive information of a tenant (directory)
type TenantInfo struct {
	Name          string `json:"name"`
	TenantID      string `json:"tenantId"`
	PrimaryDomain string `json:"primaryDomain"`
	License       string `json:"license"`
}

func EntraidIntegrationDiscovery(config Config) ([]TenantInfo, error) {
	ctx := context.Background()
	// Authenticate and get credentials
	credential, err := authenticate(ctx, &config)
	if err != nil {
		return nil, fmt.Errorf("authentication failed: %v", err)
	}

	// Get the list of directories
	directories, err := getDirectories(ctx, credential, config)
	if err != nil {
		return nil, fmt.Errorf("failed to get directories: %v", err)
	}

	return directories, nil
}

// authenticate creates an azcore.TokenCredential based on the provided configuration.
func authenticate(ctx context.Context, config *Config) (azcore.TokenCredential, error) {
	var cred azcore.TokenCredential
	var err error

	if config.CertPath != "" || config.CertContent != "" {
		// Use certificate-based authentication

		var certData []byte
		if config.CertPath != "" {
			// Read the certificate file
			certData, err = ioutil.ReadFile(config.CertPath)
			if err != nil {
				return nil, fmt.Errorf("failed to read certificate file: %v", err)
			}
		} else {
			// Use certificate content provided directly
			certData = []byte(config.CertContent)
		}

		var password []byte
		if config.CertPassword != "" {
			password = []byte(config.CertPassword)
		}

		// Parse the certificate using azidentity.ParseCertificates
		certs, key, err := azidentity.ParseCertificates(certData, password)
		if err != nil {
			return nil, fmt.Errorf("failed to parse certificate: %v", err)
		}

		// Create the ClientCertificateCredential
		cred, err = azidentity.NewClientCertificateCredential(
			config.TenantID,
			config.ClientID,
			certs,
			key,
			nil, // Additional options can be set here if needed
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create certificate credential: %v", err)
		}
	} else if config.ClientSecret != "" {
		// Use client secret authentication
		cred, err = azidentity.NewClientSecretCredential(
			config.TenantID,
			config.ClientID,
			config.ClientSecret,
			nil, // Additional options can be set here if needed
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create client secret credential: %v", err)
		}
	} else {
		return nil, fmt.Errorf("no valid authentication method found. Provide clientSecret or certPath/certContent in the configuration.")
	}

	return cred, nil
}

// getDirectories retrieves the list of Entra ID directories the SPN has access to,
// including comprehensive information for each tenant.
func getDirectories(ctx context.Context, cred azcore.TokenCredential, config Config) ([]TenantInfo, error) {
	// Define the scope for Azure Resource Manager
	armScope := "https://management.azure.com/.default"

	// Get an access token for ARM
	armToken, err := cred.GetToken(ctx, policy.TokenRequestOptions{
		Scopes: []string{armScope},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get ARM token: %v", err)
	}

	// Create the HTTP GET request to list tenants
	req, err := http.NewRequestWithContext(ctx, "GET", "https://management.azure.com/tenants?api-version=2020-01-01", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create ARM request: %v", err)
	}

	// Set the Authorization header
	req.Header.Set("Authorization", "Bearer "+armToken.Token)
	req.Header.Set("Content-Type", "application/json")

	// Send the ARM request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send ARM request: %v", err)
	}
	defer resp.Body.Close()

	// Read the ARM response body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read ARM response body: %v", err)
	}

	// Check for HTTP errors in ARM response
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ARM request failed with status %s: %s", resp.Status, string(body))
	}

	// Parse the ARM JSON response
	var armResult struct {
		Value []struct {
			ID       string `json:"id"`
			TenantID string `json:"tenantId"`
		} `json:"value"`
	}
	err = json.Unmarshal(body, &armResult)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal ARM response: %v", err)
	}

	// Initialize the slice to hold tenant information
	var tenants []TenantInfo

	// Use a WaitGroup to handle concurrency
	var wg sync.WaitGroup
	var mu sync.Mutex // To protect the tenants slice

	// Iterate over each tenant to fetch additional details from Microsoft Graph
	for _, t := range armResult.Value {
		wg.Add(1)
		go func(tenantID string) {
			defer wg.Done()

			// Fetch additional details for each tenant
			tenantInfo, err := getTenantDetails(ctx, config, tenantID)
			if err != nil {
				log.Printf("Warning: Failed to get details for tenant %s: %v", tenantID, err)
				return
			}

			// Append the tenant info to the slice
			mu.Lock()
			tenants = append(tenants, tenantInfo)
			mu.Unlock()
		}(t.TenantID)
	}

	// Wait for all goroutines to finish
	wg.Wait()

	return tenants, nil
}

// getTenantDetails fetches comprehensive details for a given tenant using Microsoft Graph API.
func getTenantDetails(ctx context.Context, config Config, tenantID string) (TenantInfo, error) {
	var tenantInfo TenantInfo
	tenantInfo.TenantID = tenantID

	// Define the scope for Microsoft Graph
	graphScope := "https://graph.microsoft.com/.default"

	// Create a new credential specific for the tenant
	var cred azcore.TokenCredential
	var err error

	if config.ClientSecret != "" {
		cred, err = azidentity.NewClientSecretCredential(
			tenantID,
			config.ClientID,
			config.ClientSecret,
			nil,
		)
		if err != nil {
			return tenantInfo, fmt.Errorf("failed to create client secret credential for tenant %s: %v", tenantID, err)
		}
	} else if config.CertPath != "" || config.CertContent != "" {
		// Use certificate-based authentication
		var certData []byte
		if config.CertPath != "" {
			// Read the certificate file
			certData, err = ioutil.ReadFile(config.CertPath)
			if err != nil {
				return tenantInfo, fmt.Errorf("failed to read certificate file: %v", err)
			}
		} else {
			// Use certificate content provided directly
			certData = []byte(config.CertContent)
		}

		var password []byte
		if config.CertPassword != "" {
			password = []byte(config.CertPassword)
		}

		// Parse the certificate using azidentity.ParseCertificates
		certs, key, err := azidentity.ParseCertificates(certData, password)
		if err != nil {
			return tenantInfo, fmt.Errorf("failed to parse certificate: %v", err)
		}

		// Create the ClientCertificateCredential
		cred, err = azidentity.NewClientCertificateCredential(
			tenantID,
			config.ClientID,
			certs,
			key,
			nil, // Additional options can be set here if needed
		)
		if err != nil {
			return tenantInfo, fmt.Errorf("failed to create certificate credential for tenant %s: %v", tenantID, err)
		}
	} else {
		return tenantInfo, fmt.Errorf("no valid authentication method found for tenant %s", tenantID)
	}

	// Get an access token for Microsoft Graph
	graphToken, err := cred.GetToken(ctx, policy.TokenRequestOptions{
		Scopes: []string{graphScope},
	})
	if err != nil {
		return tenantInfo, fmt.Errorf("failed to get Graph token for tenant %s: %v", tenantID, err)
	}

	// Create a new HTTP client for Microsoft Graph
	client := &http.Client{}

	// 1. Fetch Organization Details (Name)
	orgReq, err := http.NewRequestWithContext(ctx, "GET", "https://graph.microsoft.com/v1.0/organization", nil)
	if err != nil {
		return tenantInfo, fmt.Errorf("failed to create Graph organization request: %v", err)
	}
	orgReq.Header.Set("Authorization", "Bearer "+graphToken.Token)
	orgReq.Header.Set("Content-Type", "application/json")

	orgResp, err := client.Do(orgReq)
	if err != nil {
		return tenantInfo, fmt.Errorf("failed to send Graph organization request: %v", err)
	}
	defer orgResp.Body.Close()

	if orgResp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(orgResp.Body)
		return tenantInfo, fmt.Errorf("Graph organization request failed with status %s: %s", orgResp.Status, string(body))
	}

	orgBody := readBody(orgResp)
	var orgResult struct {
		Value []struct {
			DisplayName string `json:"displayName"`
		} `json:"value"`
	}
	err = json.Unmarshal([]byte(orgBody), &orgResult)
	if err != nil {
		return tenantInfo, fmt.Errorf("failed to unmarshal Graph organization response: %v", err)
	}

	if len(orgResult.Value) > 0 {
		tenantInfo.Name = orgResult.Value[0].DisplayName
	} else {
		tenantInfo.Name = "N/A"
	}

	// 2. Fetch Primary Domain
	primaryDomain, err := getPrimaryDomain(ctx, client, graphToken.Token)
	if err != nil {
		log.Printf("Warning: Failed to get primary domain for tenant %s: %v", tenantID, err)
		tenantInfo.PrimaryDomain = "N/A"
	} else {
		tenantInfo.PrimaryDomain = primaryDomain
	}

	// 3. Fetch License Information
	license, err := getLicenseInfo(ctx, client, graphToken.Token)
	if err != nil {
		log.Printf("Warning: Failed to get license info for tenant %s: %v", tenantID, err)
		tenantInfo.License = "N/A"
	} else {
		tenantInfo.License = license
	}

	return tenantInfo, nil
}

// getPrimaryDomain fetches the primary (default and verified) domain for the tenant.
func getPrimaryDomain(ctx context.Context, client *http.Client, token string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://graph.microsoft.com/v1.0/domains", nil)
	if err != nil {
		return "", fmt.Errorf("failed to create Graph domains request: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send Graph domains request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return "", fmt.Errorf("Graph domains request failed with status %s: %s", resp.Status, string(body))
	}

	domainsBody := readBody(resp)
	var domainsResult struct {
		Value []struct {
			ID             string `json:"id"` // Domain name
			IsDefault      bool   `json:"isDefault"`
			IsVerified     bool   `json:"isVerified"`
			IsInitial      bool   `json:"isInitial"`
			IsAdminManaged bool   `json:"isAdminManaged"`
			IsRoot         bool   `json:"isRoot"`
			// Other fields can be added if needed
		} `json:"value"`
	}
	err = json.Unmarshal([]byte(domainsBody), &domainsResult)
	if err != nil {
		return "", fmt.Errorf("failed to unmarshal Graph domains response: %v", err)
	}

	for _, domain := range domainsResult.Value {
		if domain.IsDefault && domain.IsVerified {
			return domain.ID, nil // Use domain.ID instead of domain.Name
		}
	}

	return "N/A", nil
}

// getLicenseInfo fetches the license information (e.g., Microsoft Entra ID P2) for the tenant.
func getLicenseInfo(ctx context.Context, client *http.Client, token string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://graph.microsoft.com/v1.0/subscribedSkus", nil)
	if err != nil {
		return "", fmt.Errorf("failed to create Graph subscribedSkus request: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send Graph subscribedSkus request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return "", fmt.Errorf("Graph subscribedSkus request failed with status %s: %s", resp.Status, string(body))
	}

	skusBody := readBody(resp)
	var skusResult struct {
		Value []struct {
			SkuPartNumber string `json:"skuPartNumber"`
		} `json:"value"`
	}
	err = json.Unmarshal([]byte(skusBody), &skusResult)
	if err != nil {
		return "", fmt.Errorf("failed to unmarshal Graph subscribedSkus response: %v", err)
	}

	// Collect the SKU part numbers
	var licenses []string
	for _, sku := range skusResult.Value {
		licenses = append(licenses, sku.SkuPartNumber)
	}

	if len(licenses) == 0 {
		return "N/A", nil
	}

	// Join all licenses into a comma-separated string
	return joinStrings(licenses, ", "), nil
}

// readBody reads the response body and returns it as a string.
// It handles errors internally and returns an empty string on failure.
func readBody(resp *http.Response) string {
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Warning: Failed to read response body: %v", err)
		return ""
	}
	return string(body)
}

// joinStrings joins a slice of strings with the specified separator.
func joinStrings(items []string, sep string) string {
	result := ""
	for i, item := range items {
		if i > 0 {
			result += sep
		}
		result += item
	}
	return result
}
