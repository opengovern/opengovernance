package healthcheck

import (
	"context"
	"fmt"
	"github.com/cloudflare/cloudflare-go"
	"time"
)

// Config represents the JSON input configuration
type Config struct {
	Token     string `json:"token"`
	AccountID string `json:"account_id"`
}

// IsHealthy checks the member accesses
func IsHealthy(ctx context.Context, conn *cloudflare.API, accountID string) error {
	// Get accounts associated with token
	_, _, err := conn.Account(ctx, accountID)
	if err != nil {
		return err
	}

	return nil
}

func CloudflareIntegrationHealthcheck(cfg Config) (bool, error) {
	token := cfg.Token
	if token == "" {
		return false, fmt.Errorf("no token provided")
	}

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Create cloudflare client
	conn, err := cloudflare.NewWithAPIToken(cfg.Token)
	if err != nil {
		return false, err
	}

	// Now process permissions for the admin user of account
	err = IsHealthy(ctx, conn, cfg.AccountID)
	if err != nil {
		return false, err
	}

	return true, nil
}
