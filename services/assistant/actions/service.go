package actions

import "context"

type Service interface {
	RunActions(ctx context.Context)
}
