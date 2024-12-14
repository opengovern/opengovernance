package healthcheck

import (
	"context"
)

type Config struct {
}

func IntegrationHealthcheck(ctx context.Context, config Config) (bool, error) {
	return true, nil
}
