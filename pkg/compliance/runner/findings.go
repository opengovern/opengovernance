package runner

import (
	"fmt"
	awsSteampipe "github.com/kaytu-io/kaytu-aws-describer/pkg/steampipe"
	azureSteampipe "github.com/kaytu-io/kaytu-azure-describer/pkg/steampipe"
	"github.com/kaytu-io/kaytu-engine/pkg/compliance/api"
	"github.com/kaytu-io/kaytu-engine/pkg/types"
	"github.com/kaytu-io/kaytu-util/pkg/source"
	"github.com/kaytu-io/kaytu-util/pkg/steampipe"
	"go.uber.org/zap"
)

func (j *Job) ExtractFindings(_ *zap.Logger, caller Caller, res *steampipe.Result, query api.Query) ([]types.Finding, error) {
	var findings []types.Finding

	queryResourceType := ""
	if len(query.ListOfTables) == 1 {
		switch query.Connector {
		case source.CloudAWS.String():
			queryResourceType = awsSteampipe.ExtractResourceType(query.ListOfTables[0])
		case source.CloudAzure.String():
			queryResourceType = azureSteampipe.ExtractResourceType(query.ListOfTables[0])
		default:
			if queryResourceType == "" {
				queryResourceType = awsSteampipe.ExtractResourceType(query.ListOfTables[0])
			}
			if queryResourceType == "" {
				queryResourceType = azureSteampipe.ExtractResourceType(query.ListOfTables[0])
			}
		}
	}

	for _, record := range res.Data {
		if len(record) != len(res.Headers) {
			return nil, fmt.Errorf("invalid record length, record=%d headers=%d", len(record), len(res.Headers))
		}
		recordValue := make(map[string]any)
		for idx, header := range res.Headers {
			value := record[idx]
			recordValue[header] = value
		}
		resourceType := queryResourceType

		var resourceID, resourceName, resourceLocation, reason string
		var status types.ComplianceResult
		if v, ok := recordValue["resource"].(string); ok {
			resourceID = v
			//
			//lookupResource, err := es.FetchLookupsByResourceIDWildcard(jc.esClient, resourceID)
			//if err != nil {
			//	return nil, err
			//}
			//if len(lookupResource.Hits.Hits) > 0 {
			//	resourceType = lookupResource.Hits.Hits[0].Source.ResourceType
			//}
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
			severity = caller.PolicySeverity
			if severity == "" {
				severity = types.FindingSeverityNone
			}
		} else if status == types.ComplianceResultOK {
			severity = types.FindingSeverityPassed
		}

		connectionID := "all"
		if j.ExecutionPlan.ConnectionID != nil {
			connectionID = *j.ExecutionPlan.ConnectionID
		}
		findings = append(findings, types.Finding{
			BenchmarkID:        caller.RootBenchmark,
			PolicyID:           caller.PolicyID,
			ConnectionID:       connectionID,
			EvaluatedAt:        j.CreatedAt.UnixMilli(),
			StateActive:        true,
			Result:             status,
			Severity:           severity,
			Evaluator:          j.ExecutionPlan.QueryEngine,
			Connector:          j.ExecutionPlan.QueryConnector,
			ResourceID:         resourceID,
			ResourceName:       resourceName,
			ResourceLocation:   resourceLocation,
			ResourceType:       resourceType,
			Reason:             reason,
			ComplianceJobID:    j.ID,
			ResourceCollection: j.ExecutionPlan.ResourceCollectionID,
			ParentBenchmarks:   caller.ParentBenchmarkIDs,
		})
	}
	return findings, nil
}
