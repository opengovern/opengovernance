package models

import "time"

type PlatformConfiguration struct {
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
	Configured bool      `json:"configured"`
}
