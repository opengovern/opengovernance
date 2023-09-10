package api

type Headers struct {
	Authorizations string
}

type ApiAction struct {
	ID      uint    `json:"id"`
	Method  string  `json:"method"`
	Url     string  `json:"url"`
	Headers Headers `json:"headers"`
	Body    string  `json:"body"`
}
