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

func (w *Job) ExtractFindings(_ *zap.Logger, caller Caller, res *steampipe.Result, query api.Query) ([]types.Finding, error) {
	var findings []types.Finding

	queryResourceType := ""
	if query.PrimaryTable != nil || len(query.ListOfTables) == 1 {
		tableName := ""
		if query.PrimaryTable != nil {
			tableName = *query.PrimaryTable
		} else {
			tableName = query.ListOfTables[0]
		}

		switch query.Connector {
		case source.CloudAWS.String():
			queryResourceType = awsSteampipe.ExtractResourceType(tableName)
		case source.CloudAzure.String():
			queryResourceType = azureSteampipe.ExtractResourceType(tableName)
		default:
			if queryResourceType == "" {
				queryResourceType = awsSteampipe.ExtractResourceType(tableName)
			}
			if queryResourceType == "" {
				queryResourceType = azureSteampipe.ExtractResourceType(tableName)
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
			severity = caller.ControlSeverity
			if severity == "" {
				severity = types.FindingSeverityNone
			}
		} else if status == types.ComplianceResultOK {
			severity = types.FindingSeverityPassed
		}

		connectionID := "all"
		if w.ExecutionPlan.ConnectionID != nil {
			connectionID = *w.ExecutionPlan.ConnectionID
		}
		findings = append(findings, types.Finding{
			BenchmarkID:           caller.RootBenchmark,
			ControlID:             caller.ControlID,
			ConnectionID:          connectionID,
			EvaluatedAt:           w.CreatedAt.UnixMilli(),
			StateActive:           true,
			Result:                status,
			Severity:              severity,
			Evaluator:             w.ExecutionPlan.QueryEngine,
			Connector:             w.ExecutionPlan.QueryConnector,
			ResourceID:            resourceID,
			ResourceName:          resourceName,
			ResourceLocation:      resourceLocation,
			ResourceType:          resourceType,
			Reason:                reason,
			ComplianceJobID:       w.ID,
			ParentComplianceJobID: w.ParentJobID,
			ResourceCollection:    w.ExecutionPlan.ResourceCollectionID,
			ParentBenchmarks:      caller.ParentBenchmarkIDs,
		})
	}
	return findings, nil
}
