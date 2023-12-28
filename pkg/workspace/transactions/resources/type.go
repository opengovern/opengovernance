package resources

import "errors"

var ErrResourceNeedsTime = errors.New("resource creation/deletion needs time")

type Resource interface {
	CreateIdempotent() error
	DeleteIdempotent() error
}
