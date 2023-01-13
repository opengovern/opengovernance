package api

type SetConfigMetadataRequest struct {
	Key   string `json:"key"`
	Value any    `json:"value"`
}
