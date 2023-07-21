package analytics

import "github.com/kaytu-io/kaytu-util/pkg/source"

type Metric struct {
	Connectors []source.Type       `json:"connectors"`
	Name       string              `json:"name"`
	Query      string              `json:"query"`
	Tags       map[string][]string `json:"tags"`
}
