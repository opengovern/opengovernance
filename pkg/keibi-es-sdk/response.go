package keibi

import "fmt"

type ErrorResponse struct {
	Info ErrorInfo `json:"error,omitempty"`
}

func (e ErrorResponse) Error() string {
	return fmt.Sprintf("%s: %s", e.Info.Type, e.Info.Reason)
}

type ErrorInfo struct {
	RootCause []ErrorInfo `json:"root_cause"`
	Type      string      `json:"type"`
	Reason    string      `json:"reason"`
	Phase     string      `json:"phase"`
}

type IndexResponse struct {
	Index   string `json:"_index"`
	ID      string `json:"_id"`
	Version int    `json:"_version"`
	Result  string
}

type PointInTimeResponse struct {
	ID string `json:"id"`
}
