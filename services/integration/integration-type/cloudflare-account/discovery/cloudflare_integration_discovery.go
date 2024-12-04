package discovery

import (
	"context"
	"fmt"
	"github.com/cloudflare/cloudflare-go"
	"time"
)

// Config represents the JSON input configuration
type Config struct {
	Token    string `json:"token"`
	MemberID string `json:"member_id"`
}

// AccountDetail defines the minimal information for account.
type AccountDetail struct {
	ID   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
	Type string `json:"type,omitempty"`
}

// Discover retrieves member information
func Discover(ctx context.Context, conn *cloudflare.API, memberID string) (*cloudflare.Account, error) {
	// Get account associated with token
	account, _, err := conn.Accounts(ctx, cloudflare.PaginationOptions{})
	if err != nil {
		return nil, err
	}

	return &account[0], nil
}

func CloudflareIntegrationDiscovery(cfg Config) (*AccountDetail, error) {
	token := cfg.Token
	if token == "" {
		return nil, fmt.Errorf("no token provided")
	}

	// Create a context with timeout to avoid hanging indefinitely
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create cloudflare client
	conn, err := cloudflare.NewWithAPIToken(cfg.Token)
	if err != nil {
		return nil, err
	}

	// Get the member Discover
	account, err := Discover(ctx, conn, cfg.MemberID)
	if err != nil {
		return nil, err
	}

	// Prepare the minimal organization information
	accountDetail := AccountDetail{
		ID:   account.ID,
		Name: account.Name,
		Type: account.Type,
	}

	return &accountDetail, nil
}
