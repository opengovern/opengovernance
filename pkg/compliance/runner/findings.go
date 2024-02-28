package runner

import (
	"fmt"
	awsSteampipe "github.com/kaytu-io/kaytu-aws-describer/pkg/steampipe"
	azureSteampipe "github.com/kaytu-io/kaytu-azure-describer/pkg/steampipe"
	"github.com/kaytu-io/kaytu-engine/pkg/compliance/api"
	"github.com/kaytu-io/kaytu-engine/pkg/types"
	"github.com/kaytu-io/kaytu-engine/pkg/utils"
	"github.com/kaytu-io/kaytu-util/pkg/source"
	"github.com/kaytu-io/kaytu-util/pkg/steampipe"
	"go.uber.org/zap"
	"reflect"
	"strconv"
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

func (w *Job) ExtractFindings(_ *zap.Logger, benchmarkCache map[string]api.Benchmark, caller Caller, res *steampipe.Result, query api.Query) ([]types.Finding, error) {
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
			queryResourceType = GetResourceTypeFromTableName(tableName, w.ExecutionPlan.Query.Connector)
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
		var costOptimization *float64
		var status types.ConformanceStatus
		if v, ok := recordValue["kaytu_resource_id"].(string); ok {
			kaytuResourceId = v
		}
		if v, ok := recordValue["kaytu_account_id"].(string); ok {
			connectionId = v
		}
		if v, ok := recordValue["kaytu_table_name"].(string); ok && resourceType == "" {
			resourceType = GetResourceTypeFromTableName(v, w.ExecutionPlan.Query.Connector)
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
		if v, ok := recordValue["cost_optimization"]; ok {
			// cast to proper types
			reflectValue := reflect.ValueOf(v)
			switch reflectValue.Kind() {
			case reflect.Float32:
				costOptimization = utils.GetPointer(float64(v.(float32)))
			case reflect.Float64:
				costOptimization = utils.GetPointer(v.(float64))
			case reflect.String:
				c, err := strconv.ParseFloat(v.(string), 64)
				if err == nil {
					costOptimization = &c
				} else {
					fmt.Printf("error parsing cost_optimization: %s\n", err)
					costOptimization = utils.GetPointer(0.0)
				}
			case reflect.Int:
				costOptimization = utils.GetPointer(float64(v.(int)))
			case reflect.Int8:
				costOptimization = utils.GetPointer(float64(v.(int8)))
			case reflect.Int16:
				costOptimization = utils.GetPointer(float64(v.(int16)))
			case reflect.Int32:
				costOptimization = utils.GetPointer(float64(v.(int32)))
			case reflect.Int64:
				costOptimization = utils.GetPointer(float64(v.(int64)))
			case reflect.Uint:
				costOptimization = utils.GetPointer(float64(v.(uint)))
			case reflect.Uint8:
				costOptimization = utils.GetPointer(float64(v.(uint8)))
			case reflect.Uint16:
				costOptimization = utils.GetPointer(float64(v.(uint16)))
			case reflect.Uint32:
				costOptimization = utils.GetPointer(float64(v.(uint32)))
			case reflect.Uint64:
				costOptimization = utils.GetPointer(float64(v.(uint64)))
			default:
				fmt.Printf("error parsing cost_optimization: unknown type %s\n", reflectValue.Kind())
			}
		}
		severity := caller.ControlSeverity
		if severity == "" {
			severity = types.FindingSeverityNone
		}

		if (connectionId == "" || connectionId == "null") && w.ExecutionPlan.ConnectionID != nil {
			connectionId = *w.ExecutionPlan.ConnectionID
		}

		benchmarkReferences := make([]string, 0, len(caller.ParentBenchmarkIDs))
		for _, parentBenchmarkID := range caller.ParentBenchmarkIDs {
			benchmarkReferences = append(benchmarkReferences, benchmarkCache[parentBenchmarkID].ReferenceCode)
		}

		findings = append(findings, types.Finding{
			BenchmarkID:               caller.RootBenchmark,
			ControlID:                 caller.ControlID,
			ConnectionID:              connectionId,
			EvaluatedAt:               w.CreatedAt.UnixMilli(),
			StateActive:               true,
			ConformanceStatus:         status,
			Severity:                  severity,
			Evaluator:                 w.ExecutionPlan.Query.Engine,
			Connector:                 w.ExecutionPlan.Query.Connector,
			KaytuResourceID:           kaytuResourceId,
			ResourceID:                resourceID,
			ResourceName:              resourceName,
			ResourceLocation:          resourceLocation,
			ResourceType:              resourceType,
			Reason:                    reason,
			CostOptimization:          costOptimization,
			ComplianceJobID:           w.ID,
			ParentComplianceJobID:     w.ParentJobID,
			ParentBenchmarkReferences: benchmarkReferences,
			ParentBenchmarks:          caller.ParentBenchmarkIDs,
			LastTransition:            w.CreatedAt.UnixMilli(),
		})
	}
	return findings, nil
}
