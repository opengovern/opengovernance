package compliance

import (
	"sort"

	"gitlab.com/keibiengine/keibi-engine/pkg/compliance/api"
	"gitlab.com/keibiengine/keibi-engine/pkg/compliance/db"
	"gitlab.com/keibiengine/keibi-engine/pkg/keibi-es-sdk"
	"gitlab.com/keibiengine/keibi-engine/pkg/summarizer/query"
	"gitlab.com/keibiengine/keibi-engine/pkg/types"
)

type ShortSummary struct {
	PassedResourceIDs []string
	FailedResourceIDs []string
	ConnectionIDs     []string
	Result            types.ComplianceResultSummary
	Checks            types.SeverityResult
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

		resp.Checks.PassedCount += s.Checks.PassedCount
		resp.Checks.UnknownCount += s.Checks.UnknownCount
		resp.Checks.CriticalCount += s.Checks.CriticalCount
		resp.Checks.HighCount += s.Checks.HighCount
		resp.Checks.MediumCount += s.Checks.MediumCount
		resp.Checks.LowCount += s.Checks.LowCount

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
			p, err := db.GetPolicy(policy.PolicyID)
			if err != nil {
				return resp, err
			}

			for _, resource := range policy.Resources {
				switch resource.Result {
				case types.ComplianceResultOK:
					resp.Checks.PassedCount++
					resp.Result.OkCount++
				case types.ComplianceResultALARM:
					resp.Checks.IncreaseBySeverity(p.Severity)
					resp.Result.AlarmCount++
				case types.ComplianceResultINFO:
					resp.Checks.UnknownCount++
					resp.Result.InfoCount++
				case types.ComplianceResultSKIP:
					resp.Checks.UnknownCount++
					resp.Result.SkipCount++
				case types.ComplianceResultERROR:
					resp.Checks.IncreaseBySeverity(p.Severity)
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

func GetBenchmarkTree(db db.Database, client keibi.Client, b db.Benchmark, status []types.PolicyStatus) (api.BenchmarkTree, error) {
	tree := api.BenchmarkTree{
		ID:       b.ID,
		Title:    b.Title,
		Children: nil,
		Policies: nil,
	}
	for _, child := range b.Children {
		childObj, err := db.GetBenchmark(child.ID)
		if err != nil {
			return tree, err
		}

		childTree, err := GetBenchmarkTree(db, client, *childObj, status)
		if err != nil {
			return tree, err
		}

		tree.Children = append(tree.Children, childTree)
	}

	res, err := query.ListBenchmarkSummaries(client, &b.ID)
	if err != nil {
		return tree, err
	}

	for _, policy := range b.Policies {
		pt := api.PolicyTree{
			ID:          policy.ID,
			Title:       policy.Title,
			Severity:    policy.Severity,
			Status:      types.PolicyStatusPASSED,
			LastChecked: 0,
		}

		for _, bs := range res {
			for _, ps := range bs.Policies {
				if ps.PolicyID == policy.ID {
					pt.LastChecked = bs.EvaluatedAt
					for _, resource := range ps.Resources {
						switch resource.Result {
						case types.ComplianceResultOK:
						case types.ComplianceResultALARM:
							pt.Status = types.PolicyStatusFAILED
						case types.ComplianceResultERROR:
							pt.Status = types.PolicyStatusFAILED
						case types.ComplianceResultINFO:
							if pt.Status == types.PolicyStatusPASSED {
								pt.Status = types.PolicyStatusUNKNOWN
							}
						case types.ComplianceResultSKIP:
							if pt.Status == types.PolicyStatusPASSED {
								pt.Status = types.PolicyStatusUNKNOWN
							}
						}
					}
				}
			}
		}
		if len(status) > 0 {
			contains := false
			for _, s := range status {
				if s == pt.Status {
					contains = true
				}
			}

			if !contains {
				continue
			}
		}
		tree.Policies = append(tree.Policies, pt)
	}

	return tree, nil
}

func (h *HttpHandler) BuildBenchmarkResultTrend(b db.Benchmark, startDate, endDate int64) ([]api.ResultDatapoint, error) {
	trendPoints := map[int64]types.SeverityResult{}

	for _, child := range b.Children {
		childObj, err := h.db.GetBenchmark(child.ID)
		if err != nil {
			return nil, err
		}

		childTrend, err := h.BuildBenchmarkResultTrend(*childObj, startDate, endDate)
		if err != nil {
			return nil, err
		}

		for _, t := range childTrend {
			v := trendPoints[t.Time]
			v.PassedCount += t.Result.PassedCount
			v.UnknownCount += t.Result.UnknownCount
			v.CriticalCount += t.Result.CriticalCount
			v.HighCount += t.Result.HighCount
			v.MediumCount += t.Result.MediumCount
			v.LowCount += t.Result.LowCount
			trendPoints[t.Time] = v
		}
	}

	res, err := query.FetchBenchmarkSummaryHistory(h.client, &b.ID, startDate, endDate)
	if err != nil {
		return nil, err
	}

	for _, bs := range res {
		for _, ps := range bs.Policies {
			p, err := h.db.GetPolicy(ps.PolicyID)
			if err != nil {
				return nil, err
			}

			for _, resource := range ps.Resources {
				v := trendPoints[bs.EvaluatedAt]

				switch resource.Result {
				case types.ComplianceResultOK:
					v.PassedCount++
				case types.ComplianceResultALARM:
					v.IncreaseBySeverity(p.Severity)
				case types.ComplianceResultINFO:
					v.UnknownCount++
				case types.ComplianceResultSKIP:
					v.UnknownCount++
				case types.ComplianceResultERROR:
					v.IncreaseBySeverity(p.Severity)
				}

				trendPoints[bs.EvaluatedAt] = v
			}
		}
	}

	var datapoints []api.ResultDatapoint
	for time, result := range trendPoints {
		datapoints = append(datapoints, api.ResultDatapoint{
			Time:   time,
			Result: result,
		})
	}
	sort.Slice(datapoints, func(i, j int) bool {
		return datapoints[i].Time < datapoints[j].Time
	})
	return datapoints, nil
}

func (h *HttpHandler) GetBenchmarkTreeIDs(rootID string) ([]string, error) {
	ids := []string{rootID}

	root, err := h.db.GetBenchmark(rootID)
	if err != nil {
		return nil, err
	}

	for _, child := range root.Children {
		cids, err := h.GetBenchmarkTreeIDs(child.ID)
		if err != nil {
			return nil, err
		}

		ids = append(ids, cids...)
	}
	return ids, nil
}
