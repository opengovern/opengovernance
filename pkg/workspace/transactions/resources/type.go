package resources

import (
	"context"
	"errors"
)

var ErrResourceNeedsTime = errors.New("resource creation/deletion needs time")

type Resource interface {
	CreateIdempotent(ctx context.Context) error
	DeleteIdempotent(ctx context.Context) error
}
