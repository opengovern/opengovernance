package api

import (
	"time"

	"github.com/kaytu-io/kaytu-util/pkg/source"
	"gitlab.com/keibiengine/keibi-engine/pkg/types"
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
	Connectors  []source.Type     `json:"connectors"`
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
	Connector          source.Type       `json:"connector"`
	Enabled            bool              `json:"enabled"`
	DocumentURI        string            `json:"documentURI"`
	QueryID            *string           `json:"queryID"`
	Severity           types.Severity    `json:"severity"`
	ManualVerification bool              `json:"manualVerification"`
	Managed            bool              `json:"managed"`
	CreatedAt          time.Time         `json:"createdAt"`
	UpdatedAt          time.Time         `json:"updatedAt"`
}

type Query struct {
	ID             string    `json:"id"`
	QueryToExecute string    `json:"queryToExecute"`
	Connector      string    `json:"connector"`
	ListOfTables   string    `json:"listOfTables"`
	Engine         string    `json:"engine"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}
