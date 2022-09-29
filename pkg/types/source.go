package types

import "github.com/google/uuid"

type Connection struct {
	ID uuid.UUID
}

type FullConnection struct {
	ID           uuid.UUID
	ProviderID   string
	ProviderName string
}
