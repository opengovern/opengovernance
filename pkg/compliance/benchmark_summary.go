package compliance

import (
	"gitlab.com/keibiengine/keibi-engine/pkg/compliance/api"
	"gitlab.com/keibiengine/keibi-engine/pkg/summarizer/es"
	"gitlab.com/keibiengine/keibi-engine/pkg/types"
)

func BuildBenchmarkSummary(esb es.BenchmarkSummary, b Benchmark) api.BenchmarkSummary {
	bs := api.BenchmarkSummary{
		Title:                    b.Title,
		Description:              b.Description,
		ShortSummary:             types.ComplianceResultShortSummary{},
		Policies:                 nil,
		Resources:                nil,
		CompliancyTrend:          nil, //TODO-Saleh
		AssignedConnectionsCount: 0,
		TotalConnectionResources: 0,
		Tags:                     make(map[string]string),
		Enabled:                  b.Enabled,
	}
	for _, t := range b.Tags {
		bs.Tags[t.Key] = t.Value
	}
	for _, p := range b.Policies {
		ps := api.BenchmarkSummaryPolicySummary{
			Policy: types.FullPolicy{
				ID:    p.ID,
				Title: p.Title,
			},
			ShortSummary: types.ComplianceResultShortSummary{},
		}

		for _, pe := range esb.Policies {
			if pe.PolicyID == p.ID {
				for _, r := range pe.Resources {
					if r.Result.IsPassed() {
						ps.ShortSummary.Passed++
					} else {
						ps.ShortSummary.Failed++
					}
				}
			}
		}
		bs.Policies = append(bs.Policies, ps)
	}

	resourceMap := map[string]api.BenchmarkSummaryResourceSummary{}
	for _, pe := range esb.Policies {
		for _, r := range pe.Resources {
			v := api.BenchmarkSummaryResourceSummary{
				Resource: types.FullResource{
					ID:   r.ResourceID,
					Name: r.ResourceName,
				},
			}
			if ve, ok := resourceMap[r.ResourceID]; ok {
				v = ve
			}

			if r.Result.IsPassed() {
				bs.ShortSummary.Passed++
				v.ShortSummary.Passed++
			} else {
				bs.ShortSummary.Failed++
				v.ShortSummary.Failed++
			}
			resourceMap[r.ResourceID] = v
		}
	}

	for _, v := range resourceMap {
		bs.Resources = append(bs.Resources, v)
	}
	return bs
}
