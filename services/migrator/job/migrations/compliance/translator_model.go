package compliance

import "github.com/kaytu-io/kaytu-engine/services/migrator/job/migrations/shared"

type Benchmark struct {
	ID            string              `json:"ID" yaml:"ID"`
	Title         string              `json:"Title" yaml:"Title"`
	ReferenceCode string              `json:"ReferenceCode" yaml:"ReferenceCode"`
	Connector     string              `json:"Connector" yaml:"Connector"`
	Description   string              `json:"Description" yaml:"Description"`
	Children      []string            `json:"Children" yaml:"Children"`
	Tags          map[string][]string `json:"Tags" yaml:"Tags"`
	Managed       bool                `json:"Managed" yaml:"Managed"`
	Enabled       bool                `json:"Enabled" json:"Enabled"`
	AutoAssign    bool                `json:"AutoAssign" json:"AutoAssign"`
	Baseline      bool                `json:"Baseline" yaml:"Baseline"`
	Controls      []string            `json:"Controls" yaml:"Controls"`
}

type Control struct {
	ID                 string              `json:"ID" yaml:"ID"`
	Title              string              `json:"Title" yaml:"Title"`
	Description        string              `json:"Description" yaml:"Description"`
	Query              *shared.Query       `json:"Query" yaml:"Query"`
	ManualVerification bool                `json:"ManualVerification" yaml:"ManualVerification"`
	Severity           string              `json:"Severity" json:"Severity"`
	Tags               map[string][]string `json:"Tags" json:"Tags"`
	Managed            bool                `json:"Managed" yaml:"Managed"`
}
