package healthcheck

import (
	"context"
	"errors"
	"fmt"
	"github.com/cloudflare/cloudflare-go"
	"time"
)

// Config represents the JSON input configuration
type Config struct {
	Token    string `json:"token"`
	MemberID string `json:"memberID"`
}

// HealthStatus represents the structure of the JSON output
type HealthStatus struct {
	MemberID string  `json:"member_id"`
	Healthy  bool    `json:"healthy"`
	Details  Details `json:"details"`
}

// Details contains all roles permissions of member
type Details struct {
	RolePermissions RolePermissions `json:"role_permissions"`
}

// RolePermissions contains name of each role and its permissions
type RolePermissions struct {
	Name        string                                      `json:"name"`
	Permissions map[string]cloudflare.AccountRolePermission `json:"permissions"`
}

// IsHealthy checks the member accesses
func IsHealthy(ctx context.Context, conn *cloudflare.API) error {
	// Get account associated with token
	account, _, err := conn.Accounts(ctx, cloudflare.PaginationOptions{})
	if err != nil {
		return err
	}

	// Get account roles
	roles, err := conn.AccountRoles(ctx, account[0].ID)

	for _, role := range roles {
		if role.Name == "Super Administrator - All Privileges" {
			if role.Permissions["access"].Read != true || role.Permissions["access"].Edit != true {
				return errors.New("user is not healthy due to missing permission")
			}
		}
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
	err = IsHealthy(ctx, conn)
	if err != nil {
		return false, err
	}

	return true, nil
}
