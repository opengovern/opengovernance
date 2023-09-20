package api

type ApiAction struct {
	ID      uint              `json:"id"`
	Method  string            `json:"method"`
	Url     string            `json:"url"`
	Headers map[string]string `json:"headers"`
	Body    string            `json:"body"`
}

type UpdateActionRequest struct {
	ID      uint              `json:"id"`
	Method  *string           `json:"method"`
	Url     *string           `json:"url"`
	Headers map[string]string `json:"headers"`
	Body    *string           `json:"body"`
}
