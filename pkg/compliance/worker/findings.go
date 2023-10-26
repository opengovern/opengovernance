package worker

import (
	"fmt"
	"github.com/kaytu-io/kaytu-engine/pkg/compliance/es"
	"github.com/kaytu-io/kaytu-engine/pkg/types"
	"github.com/kaytu-io/kaytu-util/pkg/source"
	"github.com/kaytu-io/kaytu-util/pkg/steampipe"
)

func (j *Job) FilterFindings(plan ExecutionPlan, findings []types.Finding, jc JobConfig) ([]types.Finding, error) {
	// get all active findings from ES page by page
	// go through the ones extracted and remove duplicates
	// if a finding fetched from es is not duplicated disable it
	from := 0
	const esFetchSize = 1000
	for {
		resp, err := es.GetActiveFindings(jc.esClient, plan.Policy.ID, from, esFetchSize)
		if err != nil {
			return nil, err
		}
		fmt.Println("+++++++++ active old findings:", len(resp.Hits.Hits))
		from += esFetchSize

		if len(resp.Hits.Hits) == 0 {
			break
		}

		for _, hit := range resp.Hits.Hits {
			dup := false

			for idx, finding := range findings {
				if finding.ResourceID == hit.Source.ResourceID &&
					finding.PolicyID == hit.Source.PolicyID &&
					finding.ConnectionID == hit.Source.ConnectionID &&
					finding.Result == hit.Source.Result {
					dup = true
					fmt.Println("+++++++++ removing dup:", finding.ID, hit.Source.ID)
					findings = append(findings[:idx], findings[idx+1:]...)
					break
				}
			}

			if !dup {
				f := hit.Source
				f.StateActive = false
				fmt.Println("+++++++++ making this disabled:", f.ID)
				findings = append(findings, f)
			}
		}
	}
	return findings, nil
}

func (j *Job) ExtractFindings(plan ExecutionPlan, connectionID string, resourceCollection *string, res *steampipe.Result, jc JobConfig) ([]types.Finding, error) {
	var findings []types.Finding
	resourceType := ""
	for _, record := range res.Data {
		if len(record) != len(res.Headers) {
			return nil, fmt.Errorf("invalid record length, record=%d headers=%d", len(record), len(res.Headers))
		}
		recordValue := make(map[string]any)
		for idx, header := range res.Headers {
			value := record[idx]
			recordValue[header] = value
		}

		var resourceID, resourceName, resourceLocation, reason string
		var status types.ComplianceResult
		if v, ok := recordValue["resource"].(string); ok {
			resourceID = v
			if resourceType == "" {
				lookupResource, err := es.FetchLookupsByResourceIDWildcard(jc.esClient, resourceID)
				if err != nil {
					return nil, err
				}
				if len(lookupResource.Hits.Hits) > 0 {
					resourceType = lookupResource.Hits.Hits[0].Source.ResourceType
				}
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
		})
	}
	return findings, nil
}
