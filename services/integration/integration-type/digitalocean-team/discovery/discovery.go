package discovery

import (
	"context"
	"errors"
	"github.com/digitalocean/godo"
)

type Config struct {
	AuthToken string
}

type Team struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func DigitalOceanTeamDiscovery(ctx context.Context, config Config) (*Team, error) {
	client := godo.NewFromToken(config.AuthToken)

	account, resp, err := client.Account.Get(ctx)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		return nil, err
	}
	if account.Team == nil {
		return nil, errors.New("team not found")
	}

	return &Team{
		ID:   account.Team.UUID,
		Name: account.Team.Name,
	}, nil
}
