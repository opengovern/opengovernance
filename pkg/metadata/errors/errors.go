package errors

import "errors"

var (
	ErrMetadataKeyNotFound            = errors.New("metadata key not found")
	ErrMetadataValueTypeMismatch      = errors.New("metadata value type mismatch")
	ErrConfigMetadataTypeNotSupported = errors.New("config metadata type not supported")
)
