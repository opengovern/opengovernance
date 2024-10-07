package compliance

import "github.com/kaytu-io/open-governance/services/migrator/job/migrations/shared"

type Benchmark struct {
	ID                string              `json:"ID" yaml:"ID"`
	Title             string              `json:"Title" yaml:"Title"`
	SectionCode       string              `json:"SectionCode" yaml:"SectionCode"`
	Description       string              `json:"Description" yaml:"Description"`
	Children          []string            `json:"Children" yaml:"Children"`
	Tags              map[string][]string `json:"Tags" yaml:"Tags"`
	AutoAssign        bool                `json:"AutoAssign" yaml:"AutoAssign"`
	Enabled           bool                `json:"Enabled" yaml:"Enabled"`
	TracksDriftEvents bool                `json:"TracksDriftEvents" yaml:"TracksDriftEvents"`
	Controls          []string            `json:"Controls" yaml:"Controls"`
}

type Control struct {
	ID                 string              `json:"ID" yaml:"ID"`
	Title              string              `json:"Title" yaml:"Title"`
	Connector          []string            `json:"Connector" yaml:"Connector"`
	Description        string              `json:"Description" yaml:"Description"`
	Query              *shared.Query       `json:"Query" yaml:"Query"`
	ManualVerification bool                `json:"ManualVerification" yaml:"ManualVerification"`
	Severity           string              `json:"Severity" yaml:"Severity"`
	Tags               map[string][]string `json:"Tags" yaml:"Tags"`
	Managed            bool                `json:"Managed" yaml:"Managed"`
}

type QueryView struct {
	ID    string `json:"id" yaml:"ID"`
	Query string `json:"query" yaml:"Query"`
}
