package discovery

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/google/go-github/v66/github"
	"golang.org/x/oauth2"
)

// Config represents the JSON input configuration
type Config struct {
	Token string `json:"token"`
}

// Output defines the structure of the JSON response.
type Output struct {
	Organizations []OrgDetail `json:"organizations,omitempty"`
	Error         string      `json:"error,omitempty"`
}

// OrgDetail defines the minimal information for each organization.
type OrgDetail struct {
	Login string `json:"login,omitempty"`
	ID    int64  `json:"id,omitempty"`
	Name  string `json:"name,omitempty"`
}

// Discover retrieves all active GitHub organizations accessible by the token.
// An organization is considered active if it has at least one non-archived repository.
func Discover(ctx context.Context, client *github.Client) ([]*github.Organization, error) {
	var allOrgs []*github.Organization
	opts := &github.ListOptions{PerPage: 100}

	// Step 1: List all organizations accessible by the token.
	for {
		orgs, resp, err := client.Organizations.List(ctx, "", opts)
		if err != nil {
			// Check if the error is due to authentication failure
			if resp != nil && resp.StatusCode == 401 {
				return nil, errors.New("authentication failed: invalid or insufficient token")
			}
			return nil, err
		}

		allOrgs = append(allOrgs, orgs...)
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	// If no organizations are found, return an error.
	if len(allOrgs) == 0 {
		return nil, errors.New("no organizations found. user does not have access to any organizations")
	}

	// Step 2: Filter organizations to include only active ones.
	activeOrgs := []*github.Organization{}
	for _, org := range allOrgs {
		if org.Login == nil {
			continue // Skip organizations without a login name.
		}

		// List repositories for the organization.
		repoOpts := &github.RepositoryListByOrgOptions{
			ListOptions: github.ListOptions{PerPage: 100},
		}

		hasActiveRepo := false

		for {
			repos, resp, err := client.Repositories.ListByOrg(ctx, *org.Login, repoOpts)
			if err != nil {
				// If there's an error fetching repositories, skip this organization.
				log.Printf("Error fetching repositories for organization '%s': %v", *org.Login, err)
				break
			}

			for _, repo := range repos {
				if repo.Archived != nil && !*repo.Archived {
					hasActiveRepo = true
					break
				}
			}

			if hasActiveRepo || resp.NextPage == 0 {
				break
			}
			repoOpts.Page = resp.NextPage
		}

		if hasActiveRepo {
			activeOrgs = append(activeOrgs, org)
		}
	}

	// If no active organizations are found, return an error.
	if len(activeOrgs) == 0 {
		return nil, errors.New("no active organizations found. user does not have access to any active organizations")
	}

	return activeOrgs, nil
}

func GithubIntegrationDiscovery(cfg Config) ([]OrgDetail, error) {
	token := cfg.Token
	if token == "" {
		return nil, fmt.Errorf("no token provided")
	}

	// Create a context with timeout to avoid hanging indefinitely
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create an OAuth2 token source
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)

	// Create an OAuth2 client
	tc := oauth2.NewClient(ctx, ts)

	// Create a new GitHub client
	client := github.NewClient(tc)

	// Get the list of active organizations using Discover
	orgs, err := Discover(ctx, client)
	if err != nil {
		output := Output{
			Error: err.Error(),
		}
		jsonOutput, _ := json.Marshal(output)
		fmt.Println(string(jsonOutput))
		os.Exit(1)
	}

	// Prepare the minimal organization information
	var detailedOrgs []OrgDetail
	for _, org := range orgs {
		detail := OrgDetail{
			Login: safeString(org.Login),
			ID:    safeInt64(org.ID),
			Name:  safeString(org.Name),
		}
		detailedOrgs = append(detailedOrgs, detail)
	}

	// If user doesn't have access to any active organizations, error out
	if len(detailedOrgs) == 0 {
		output := Output{
			Error: "No active organizations found. User does not have access to any active organizations.",
		}
		jsonOutput, _ := json.Marshal(output)
		fmt.Println(string(jsonOutput))
		os.Exit(1)
	}

	return detailedOrgs, nil
}

// Helper functions to safely dereference pointers
func safeString(s *string) string {
	if s != nil {
		return *s
	}
	return ""
}

func safeInt64(i *int64) int64 {
	if i != nil {
		return *i
	}
	return 0
}
