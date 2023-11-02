package worker

import (
	"github.com/kaytu-io/kaytu-engine/pkg/auth/api"
	api2 "github.com/kaytu-io/kaytu-engine/pkg/compliance/api"
	"github.com/kaytu-io/kaytu-engine/pkg/internal/httpclient"
)

type Plan struct {
	ParentBenchmarkIDs []string
	Query              *api2.Query
	Policy             *api2.Policy
}

type ExecutionPlan struct {
	Plans map[string][]Plan
}

func ListExecutionPlans(connectionID string, parentBenchmarkIDs []string, benchmarkID string, jc JobConfig) (ExecutionPlan, error) {
	executionPlan := ExecutionPlan{
		Plans: map[string][]Plan{},
	}

	hctx := &httpclient.Context{UserRole: api.InternalRole}
	benchmark, err := jc.complianceClient.GetBenchmark(hctx, benchmarkID)
	if err != nil {
		return executionPlan, err
	}

	for _, childBenchmarkID := range benchmark.Children {
		childExecutionPlan, err := ListExecutionPlans(connectionID, append(parentBenchmarkIDs, benchmarkID), childBenchmarkID, jc)
		if err != nil {
			return executionPlan, err
		}

		for k, v := range childExecutionPlan.Plans {
			o, ok := executionPlan.Plans[k]
			if ok {
				o = append(o, v...)
			} else {
				o = v
			}
			executionPlan.Plans[k] = o
		}
	}

	for _, policyID := range benchmark.Policies {
		policy, err := jc.complianceClient.GetPolicy(hctx, policyID)
		if err != nil {
			return executionPlan, err
		}

		if policy.ManualVerification {
			continue
		}

		if !policy.Enabled {
			continue
		}

		if policy.QueryID == nil {
			continue
		}

		query, err := jc.complianceClient.GetQuery(hctx, *policy.QueryID)
		if err != nil {
			return executionPlan, err
		}

		ep := Plan{
			ParentBenchmarkIDs: append(parentBenchmarkIDs, benchmarkID),
			Policy:             policy,
			Query:              query,
		}

		o, ok := executionPlan.Plans[ep.Query.ID]
		if ok {
			o = append(o, ep)
		} else {
			o = []Plan{ep}
		}
		executionPlan.Plans[ep.Query.ID] = o
	}

	return executionPlan, nil
}
