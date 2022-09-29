package compliance

import (
	"gitlab.com/keibiengine/keibi-engine/pkg/compliance/api"
	"gitlab.com/keibiengine/keibi-engine/pkg/summarizer/es"
	"gitlab.com/keibiengine/keibi-engine/pkg/types"
)

func BuildBenchmarkSummary(es es.BenchmarkSummary, b Benchmark) api.BenchmarkSummary {
	bs := api.BenchmarkSummary{
		Title:       b.Title,
		Description: b.Description,
		ShortSummary: types.ComplianceResultShortSummary{
			Passed: 0, //TODO
			Failed: 0, //TODO
		},
		Policies:                 nil,
		Resources:                nil,
		CompliancyTrend:          nil, //TODO
		AssignedConnectionsCount: 0,   //TODO
		TotalConnectionResources: 0,   //TODO
		Tags:                     make(map[string]string),
		Enabled:                  b.Enabled,
	}
	for _, t := range b.Tags {
		bs.Tags[t.Key] = t.Value
	}
	for _, p := range b.Policies {
		bs.Policies = append(bs.Policies, api.BenchmarkSummaryPolicySummary{
			Policy: types.FullPolicy{
				ID:    p.ID,
				Title: p.Title,
			},
			ShortSummary: types.ComplianceResultShortSummary{
				Passed: 0,
				Failed: 0,
			},
		})
	}

}
