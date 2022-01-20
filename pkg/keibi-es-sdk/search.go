package keibi

type PointInTime struct {
	ID        string `json:"id"`
	KeepAlive string `json:"keep_alive"`
}

type SearchRequest struct {
	Size        *int64                   `json:"size,omitempty"`
	Query       interface{}              `json:"query,omitempty"`
	PIT         *PointInTime             `json:"pit,omitempty"`
	Sort        []map[string]interface{} `json:"sort,omitempty"`
	SearchAfter []interface{}            `json:"search_after,omitempty"`
}

type Relation string

const (
	EqRelation  Relation = "eq"
	GteRelation Relation = "gte"
)

type SearchTotal struct {
	Value    int64  `json:"value"`
	Relation string `json:"relation"`
}
