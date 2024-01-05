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

func GetResourceTypeFromTableName(tableName string, queryConnector source.Type) string {
	switch queryConnector {
	case source.CloudAWS:
		return awsSteampipe.ExtractResourceType(tableName)
	case source.CloudAzure:
		return azureSteampipe.ExtractResourceType(tableName)
	default:
		resourceType := awsSteampipe.ExtractResourceType(tableName)
		if resourceType == "" {
			resourceType = azureSteampipe.ExtractResourceType(tableName)
		}
		return resourceType
	}
}

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
		if tableName != "" {
			queryResourceType = GetResourceTypeFromTableName(tableName, w.ExecutionPlan.QueryConnector)
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

		var kaytuResourceId, connectionId, resourceID, resourceName, resourceLocation, reason string
		var status types.ConformanceStatus
		if v, ok := recordValue["kaytu_resource_id"].(string); ok {
			kaytuResourceId = v
		}
		if v, ok := recordValue["kaytu_account_id"].(string); ok {
			connectionId = v
		}
		if v, ok := recordValue["kaytu_table_name"].(string); ok && resourceType == "" {
			resourceType = GetResourceTypeFromTableName(v, w.ExecutionPlan.QueryConnector)
		}
		if v, ok := recordValue["resource"].(string); ok && v != "" && v != "null" {
			resourceID = v
		} else {
			continue
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
			status = types.ConformanceStatus(v)
		}

		severity := caller.ControlSeverity
		if severity == "" {
			severity = types.FindingSeverityNone
		}

		if (connectionId == "" || connectionId == "null") && w.ExecutionPlan.ConnectionID != nil {
			connectionId = *w.ExecutionPlan.ConnectionID
		}
		findings = append(findings, types.Finding{
			BenchmarkID:           caller.RootBenchmark,
			ControlID:             caller.ControlID,
			ConnectionID:          connectionId,
			EvaluatedAt:           w.CreatedAt.UnixMilli(),
			StateActive:           true,
			ConformanceStatus:     status,
			Severity:              severity,
			Evaluator:             w.ExecutionPlan.QueryEngine,
			Connector:             w.ExecutionPlan.QueryConnector,
			KaytuResourceID:       kaytuResourceId,
			ResourceID:            resourceID,
			ResourceName:          resourceName,
			ResourceLocation:      resourceLocation,
			ResourceType:          resourceType,
			Reason:                reason,
			ComplianceJobID:       w.ID,
			ParentComplianceJobID: w.ParentJobID,
			ParentBenchmarks:      caller.ParentBenchmarkIDs,
		})
	}
	return findings, nil
}
