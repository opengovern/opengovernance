package api

import (
	"time"
)

type Benchmark struct {
	ID          string            `json:"id"`
	Title       string            `json:"title"`
	Description string            `json:"description"`
	LogoURI     string            `json:"logoURI"`
	Category    string            `json:"category"`
	DocumentURI string            `json:"documentURI"`
	Enabled     bool              `json:"enabled"`
	Managed     bool              `json:"managed"`
	AutoAssign  bool              `json:"autoAssign"`
	Baseline    bool              `json:"baseline"`
	Tags        map[string]string `json:"tags"`
	Children    []string          `json:"children"`
	Policies    []string          `json:"policies"`
	CreatedAt   time.Time         `json:"createdAt"`
	UpdatedAt   time.Time         `json:"updatedAt"`
}

type Policy struct {
	ID                 string            `json:"id"`
	Title              string            `json:"title"`
	Description        string            `json:"description"`
	Tags               map[string]string `json:"tags"`
	DocumentURI        string            `json:"documentURI"`
	QueryID            *string           `json:"queryID"`
	Severity           string            `json:"severity"`
	ManualVerification bool              `json:"manualVerification"`
	Managed            bool              `json:"managed"`
	CreatedAt          time.Time         `json:"createdAt"`
	UpdatedAt          time.Time         `json:"updatedAt"`
}
