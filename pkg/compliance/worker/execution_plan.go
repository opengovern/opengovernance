package worker

import (
	"github.com/kaytu-io/kaytu-engine/pkg/auth/api"
	api2 "github.com/kaytu-io/kaytu-engine/pkg/compliance/api"
	"github.com/kaytu-io/kaytu-engine/pkg/internal/httpclient"
)

type ExecutionPlan struct {
	ParentBenchmarkIDs []string
	Policy             *api2.Policy
	Query              *api2.Query
}

func ListExecutionPlans(connectionID string, parentBenchmarkIDs []string, benchmarkID string, jc JobConfig) ([]ExecutionPlan, error) {
	var plans []ExecutionPlan

	hctx := &httpclient.Context{UserRole: api.InternalRole}
	benchmark, err := jc.complianceClient.GetBenchmark(hctx, benchmarkID)
	if err != nil {
		return nil, err
	}

	for _, childBenchmarkID := range benchmark.Children {
		executionPlans, err := ListExecutionPlans(connectionID, append(parentBenchmarkIDs, benchmarkID), childBenchmarkID, jc)
		if err != nil {
			return nil, err
		}

		plans = append(plans, executionPlans...)
	}

	for _, policyID := range benchmark.Policies {
		policy, err := jc.complianceClient.GetPolicy(hctx, policyID)
		if err != nil {
			return nil, err
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
			return nil, err
		}

		ep := ExecutionPlan{
			ParentBenchmarkIDs: append(parentBenchmarkIDs, benchmarkID),
			Policy:             policy,
			Query:              query,
		}

		plans = append(plans, ep)
	}

	var distinctPlans []ExecutionPlan
	planDuplicates := map[string]interface{}{}
	for _, plan := range plans {
		if _, ok := planDuplicates[plan.Query.ID]; ok {
			continue
		}
		planDuplicates[plan.Query.ID] = struct{}{}
		distinctPlans = append(distinctPlans, plan)
	}

	return distinctPlans, nil
}
