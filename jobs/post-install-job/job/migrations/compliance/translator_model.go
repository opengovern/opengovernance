package compliance

import (
	"github.com/opengovern/opencomply/jobs/post-install-job/job/migrations/shared"
)

type Benchmark struct {
	ID                string              `json:"ID" yaml:"ID"`
	Title             string              `json:"Title" yaml:"Title"`
	SectionCode       string              `json:"SectionCode" yaml:"SectionCode"`
	Description       string              `json:"Description" yaml:"Description"`
	Children          []string            `json:"Children" yaml:"Children"`
	Tags              map[string][]string `json:"Tags" yaml:"Tags"`
	AutoAssign        *bool               `json:"AutoAssign" yaml:"AutoAssign"`
	Enabled           bool                `json:"Enabled" yaml:"Enabled"`
	TracksDriftEvents bool                `json:"TracksDriftEvents" yaml:"TracksDriftEvents"`
	Controls          []string            `json:"Controls" yaml:"Controls"`
}

type Control struct {
	ID                 string              `json:"ID" yaml:"ID"`
	Title              string              `json:"Title" yaml:"Title"`
	IntegrationType    []string            `json:"IntegrationType" yaml:"IntegrationType"`
	Description        string              `json:"Description" yaml:"Description"`
	Query              *shared.Query       `json:"Query" yaml:"Query"`
	ManualVerification bool                `json:"ManualVerification" yaml:"ManualVerification"`
	Severity           string              `json:"Severity" yaml:"Severity"`
	Tags               map[string][]string `json:"Tags" yaml:"Tags"`
	Managed            bool                `json:"Managed" yaml:"Managed"`
}

type QueryView struct {
	ID          string        `json:"id" yaml:"ID"`
	Title       string        `json:"title" yaml:"Title"`
	Description string        `json:"description" yaml:"Description"`
	Query       *shared.Query `json:"query" yaml:"Query"`

	Dependencies []string `json:"dependencies" yaml:"Dependencies"`
}
