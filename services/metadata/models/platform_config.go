package models

import (
	"github.com/google/uuid"
	"time"
)

type PlatformConfiguration struct {
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
	InstallID  uuid.UUID `json:"install_id"`
	Configured bool      `json:"configured"`
}
