package api

type CreateActionReq struct {
	Method  string            `json:"method"`
	Url     string            `json:"url"`
	Headers map[string]string `json:"headers"`
	Body    string            `json:"body"`
}

type Action struct {
	Id      uint              `json:"id"`
	Method  string            `json:"method"`
	Url     string            `json:"url"`
	Headers map[string]string `json:"headers"`
	Body    string            `json:"body"`
}

type UpdateActionRequest struct {
	Id      uint              `json:"id"`
	Method  *string           `json:"method"`
	Url     *string           `json:"url"`
	Headers map[string]string `json:"headers"`
	Body    *string           `json:"body"`
}
