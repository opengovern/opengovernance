package discovery

import (
	"context"
	"fmt"
	"github.com/cloudflare/cloudflare-go"
	"strings"
	"time"
)

// Config represents the JSON input configuration
type Config struct {
	Token    string `json:"token"`
	MemberID string `json:"member_id"`
}

// UserDetail defines the minimal information for user.
type UserDetail struct {
	ID     string `json:"id,omitempty"`
	Name   string `json:"name,omitempty"`
	Status string `json:"status,omitempty"`
}

// Discover retrieves member information
func Discover(ctx context.Context, conn *cloudflare.API, memberID string) (*cloudflare.AccountMember, error) {
	// Get account associated with token
	account, _, err := conn.Accounts(ctx, cloudflare.PaginationOptions{})
	if err != nil {
		return nil, err
	}

	// Get account member information
	member, err := conn.AccountMember(ctx, account[0].ID, memberID)
	if err != nil {
		return nil, err
	}

	return &member, nil
}

// accountMemberTitle creates member title according to firstname, lastname or email
func accountMemberTitle(accountMember cloudflare.AccountMember) string {
	if len(accountMember.User.FirstName) > 0 && len(accountMember.User.LastName) > 0 {
		return accountMember.User.FirstName + " " + accountMember.User.LastName
	}
	return strings.Split(accountMember.User.Email, "@")[0]
}

func CloudflareIntegrationDiscovery(cfg Config) (*UserDetail, error) {
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
	member, err := Discover(ctx, conn, cfg.MemberID)
	if err != nil {
		return nil, err
	}

	// Prepare the minimal organization information
	memberTitle := accountMemberTitle(*member)
	userDetail := UserDetail{
		ID:     member.ID,
		Name:   memberTitle,
		Status: member.Status,
	}

	return &userDetail, nil
}
