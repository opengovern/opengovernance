package api

type ConnectionGroup struct {
	Name        string       `json:"name" example:"UltraSightApplication"`
	Query       string       `json:"query" example:"SELECT id FROM kaytu_connections WHERE tags->'application' IS NOT NULL AND tags->'application' @> '\"UltraSight\"'"`
	Connections []Connection `json:"connections,omitempty"`
}
