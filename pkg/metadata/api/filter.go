package api

type SetConfigFilter struct {
	Name     string            `json:"name"`
	KeyValue map[string]string `json:"keyValue"`
}
