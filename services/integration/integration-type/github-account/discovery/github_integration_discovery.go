package discovery

import (
	"context"
	"github.com/opengovern/opengovernance/services/integration/integration-type/github-account/healthcheck"
	"strconv"
)

// Config represents the JSON input configuration
type Config struct {
	Token          string `json:"token"`
	BaseURL        string `json:"base_url"`
	AppId          string `json:"app_id"`
	InstallationId string `json:"installation_id"`
	PrivateKeyPath string `json:"private_key_path"`
}

type Account struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Type      string `json:"type"`
	IsHealthy bool   `json:"isHealthy"`
}

func GithubIntegrationDiscovery(config Config) ([]Account, error) {
	isHealthy, client, err := healthcheck.GithubIntegrationHealthcheck(healthcheck.Config{
		Token:          config.Token,
		BaseURL:        config.BaseURL,
		AppId:          config.AppId,
		InstallationId: config.InstallationId,
		PrivateKeyPath: config.PrivateKeyPath,
	})
	account, _, err := client.Users.Get(context.Background(), "")
	if err != nil {
		return nil, err
	}
	return []Account{{
		ID:        strconv.Itoa(int(*account.ID)),
		Name:      *account.Login,
		Type:      *account.Type,
		IsHealthy: isHealthy,
	}}, nil
}
