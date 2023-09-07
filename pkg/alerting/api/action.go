package api

import "encoding/json"

type RequestAction struct {
	ID      uint            `json:"id"`
	Method  string          `json:"method"`
	Url     string          `json:"url"`
	Headers json.RawMessage `json:"headers"`
	Body    string          `json:"body"`
}

type ResponseAction struct {
	ID      uint            `json:"id"`
	Method  string          `json:"method"`
	Url     string          `json:"url"`
	Headers json.RawMessage `json:"headers"`
	Body    string          `json:"body"`
}
