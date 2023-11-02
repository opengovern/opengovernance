package worker

import (
	"context"
	"fmt"
	"github.com/kaytu-io/kaytu-engine/pkg/compliance/es"
	"github.com/kaytu-io/kaytu-engine/pkg/types"
	"github.com/kaytu-io/kaytu-util/pkg/source"
	"github.com/kaytu-io/kaytu-util/pkg/steampipe"
)

func (j *Job) FilterFindings(plan Plan, findings []types.Finding, jc JobConfig) ([]types.Finding, error) {
	// get all active findings from ES page by page
	// go through the ones extracted and remove duplicates
	// if a finding fetched from es is not duplicated disable it

	ctx := context.Background()
	resp, err := es.ListActiveFindings(jc.esClient, plan.Policy.ID)
	if err != nil {
		return nil, err
	}

	for resp.HasNext() {
		page, err := resp.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		fmt.Println("+++++++++ active old findings:", len(page), plan.Policy.ID)

		for _, hit := range page {
			dup := false

			for idx, finding := range findings {
				if finding.ResourceID == hit.ResourceID &&
					finding.PolicyID == hit.PolicyID &&
					finding.ConnectionID == hit.ConnectionID &&
					finding.Result == hit.Result {
					dup = true
					fmt.Println("+++++++++ removing dup:", finding.ID, hit.ID)
					findings = append(findings[:idx], findings[idx+1:]...)
					break
				}
			}

			if !dup {
				f := hit
				f.StateActive = false
				fmt.Println("+++++++++ making this disabled:", f.ID)
				findings = append(findings, f)
			}
		}
	}

	return findings, nil
}

func (j *Job) ExtractFindings(plan Plan, connectionID string, resourceCollection *string, res *steampipe.Result, jc JobConfig) ([]types.Finding, error) {
	var findings []types.Finding

	for _, record := range res.Data {
		if len(record) != len(res.Headers) {
			return nil, fmt.Errorf("invalid record length, record=%d headers=%d", len(record), len(res.Headers))
		}
		recordValue := make(map[string]any)
		for idx, header := range res.Headers {
			value := record[idx]
			recordValue[header] = value
		}
		resourceType := ""

		var resourceID, resourceName, resourceLocation, reason string
		var status types.ComplianceResult
		if v, ok := recordValue["resource"].(string); ok {
			resourceID = v

			lookupResource, err := es.FetchLookupsByResourceIDWildcard(jc.esClient, resourceID)
			if err != nil {
				return nil, err
			}
			if len(lookupResource.Hits.Hits) > 0 {
				resourceType = lookupResource.Hits.Hits[0].Source.ResourceType
			}
		}
		if v, ok := recordValue["name"].(string); ok {
			resourceName = v
		}
		if v, ok := recordValue["location"].(string); ok {
			resourceLocation = v
		}
		if v, ok := recordValue["reason"].(string); ok {
			reason = v
		}
		if v, ok := recordValue["status"].(string); ok {
			status = types.ComplianceResult(v)
		}

		severity := types.FindingSeverityNone
		if status == types.ComplianceResultALARM {
			severity = plan.Policy.Severity
			if severity == "" {
				severity = types.FindingSeverityNone
			}
		} else if status == types.ComplianceResultOK {
			severity = types.FindingSeverityPassed
		}

		findings = append(findings, types.Finding{
			ID:                 fmt.Sprintf("%s-%s", resourceID, plan.Policy.ID),
			BenchmarkID:        j.BenchmarkID,
			PolicyID:           plan.Policy.ID,
			ConnectionID:       connectionID,
			EvaluatedAt:        j.CreatedAt.UnixMilli(),
			StateActive:        true,
			Result:             status,
			Severity:           severity,
			Evaluator:          plan.Query.Engine,
			Connector:          source.Type(plan.Query.Connector),
			ResourceID:         resourceID,
			ResourceName:       resourceName,
			ResourceLocation:   resourceLocation,
			ResourceType:       resourceType,
			Reason:             reason,
			ComplianceJobID:    j.ID,
			ResourceCollection: resourceCollection,
			ParentBenchmarks:   plan.ParentBenchmarkIDs,
		})
	}
	return findings, nil
}
