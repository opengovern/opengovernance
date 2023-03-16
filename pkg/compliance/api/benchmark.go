package api

import "gitlab.com/keibiengine/keibi-engine/pkg/source"

type Benchmark struct {
	ID          string
	Title       string
	Description string
	Connectors  []source.Type
	Enabled     bool
	Tags        map[string]string
	Policies    []Policy
}

type Policy struct {
	ID                    string
	Title                 string
	Description           string
	Tags                  map[string]string
	Provider              source.Type
	Category              string
	SubCategory           string
	Section               string
	Severity              string
	ManualVerification    string
	ManualRemedation      string
	CommandLineRemedation string
	QueryToRun            string
	KeibiManaged          bool
}
