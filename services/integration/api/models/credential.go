package models

import "time"

type Credential struct {
	ID        string            `json:"id"`
	Secret    string            `json:"secret"`
	Metadata  map[string]string `json:"metadata"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
}

type ListCredentialsResponse struct {
	Credentials []Credential `json:"credentials"`
	TotalCount  int          `json:"total_count"`
}
