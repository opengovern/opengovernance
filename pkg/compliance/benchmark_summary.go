package compliance

import (
	"gitlab.com/keibiengine/keibi-engine/pkg/compliance/api"
	"gitlab.com/keibiengine/keibi-engine/pkg/compliance/db"
	"gitlab.com/keibiengine/keibi-engine/pkg/keibi-es-sdk"
	"gitlab.com/keibiengine/keibi-engine/pkg/summarizer/es"
	"gitlab.com/keibiengine/keibi-engine/pkg/summarizer/query"
	"gitlab.com/keibiengine/keibi-engine/pkg/types"
)

func BuildBenchmarkSummary(esb es.BenchmarkSummary, b db.Benchmark) api.BenchmarkSummary {
	bs := api.BenchmarkSummary{
		ID:                       b.ID,
		Title:                    b.Title,
		Description:              b.Description,
		Result:                   map[types.ComplianceResult]int{},
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
					bs.Result[r.Result]++
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

type ShortSummary struct {
	PassedResourceIDs []string
	FailedResourceIDs []string
	ConnectionIDs     []string
	Result            types.ComplianceResultSummary
}

func GetShortSummary(client keibi.Client, db db.Database, benchmark db.Benchmark) (ShortSummary, error) {
	resp := ShortSummary{}
	for _, child := range benchmark.Children {
		childBenchmark, err := db.GetBenchmark(child.ID)
		if err != nil {
			return resp, err
		}

		s, err := GetShortSummary(client, db, *childBenchmark)
		if err != nil {
			return resp, err
		}

		resp.Result.OkCount += s.Result.OkCount
		resp.Result.AlarmCount += s.Result.AlarmCount
		resp.Result.InfoCount += s.Result.InfoCount
		resp.Result.SkipCount += s.Result.SkipCount
		resp.Result.ErrorCount += s.Result.ErrorCount
		resp.FailedResourceIDs = append(resp.FailedResourceIDs, s.FailedResourceIDs...)
		resp.PassedResourceIDs = append(resp.PassedResourceIDs, s.PassedResourceIDs...)
		resp.ConnectionIDs = append(resp.ConnectionIDs, s.ConnectionIDs...)
	}

	res, err := query.ListBenchmarkSummaries(client, &benchmark.ID)
	if err != nil {
		return resp, err
	}

	for _, summ := range res {
		for _, policy := range summ.Policies {
			for _, resource := range policy.Resources {
				switch resource.Result {
				case types.ComplianceResultOK:
					resp.Result.OkCount++
				case types.ComplianceResultALARM:
					resp.Result.AlarmCount++
				case types.ComplianceResultINFO:
					resp.Result.InfoCount++
				case types.ComplianceResultSKIP:
					resp.Result.SkipCount++
				case types.ComplianceResultERROR:
					resp.Result.ErrorCount++
				}

				if resource.Result.IsPassed() {
					resp.PassedResourceIDs = append(resp.PassedResourceIDs, resource.ResourceID)
				} else {
					resp.FailedResourceIDs = append(resp.FailedResourceIDs, resource.ResourceID)
				}
				resp.ConnectionIDs = append(resp.ConnectionIDs, resource.SourceID)
			}
		}
	}

	resp.PassedResourceIDs = UniqueArray(resp.PassedResourceIDs, func(t, t2 string) bool {
		return t == t2
	})
	resp.FailedResourceIDs = UniqueArray(resp.FailedResourceIDs, func(t, t2 string) bool {
		return t == t2
	})
	resp.ConnectionIDs = UniqueArray(resp.ConnectionIDs, func(t, t2 string) bool {
		return t == t2
	})

	var successfuls []string
	for _, passed := range resp.PassedResourceIDs {
		failedExists := false
		for _, failed := range resp.FailedResourceIDs {
			if passed == failed {
				failedExists = true
			}
		}

		if !failedExists {
			successfuls = append(successfuls, passed)
		}
	}
	resp.PassedResourceIDs = successfuls
	return resp, nil
}

func UniqueArray[T any](input []T, equals func(T, T) bool) []T {
	var out []T
	for _, i := range input {
		exists := false
		for _, o := range out {
			if equals(i, o) {
				exists = true
			}
		}

		if !exists {
			out = append(out, i)
		}
	}
	return out
}
