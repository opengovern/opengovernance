package api

type ConnectionGroup struct {
	Name          string       `json:"name" example:"UltraSightApplication"`
	Query         string       `json:"query" example:"SELECT kaytu_id FROM kaytu_connections WHERE tags->'application' IS NOT NULL AND tags->'application' @> '\"UltraSight\"'"`
	ConnectionIds []string     `json:"connectionIds,omitempty" example:"[\"1e8ac3bf-c268-4a87-9374-ce04cc40a596\"]"`
	Connections   []Connection `json:"connections,omitempty"`
}
