package runner

import (
	"fmt"
	"github.com/labstack/echo/v4"
	"github.com/opengovern/og-util/pkg/integration"
	"github.com/opengovern/og-util/pkg/steampipe"
	"github.com/opengovern/opengovernance/pkg/compliance/api"
	"github.com/opengovern/opengovernance/pkg/types"
	"github.com/opengovern/opengovernance/pkg/utils"
	integration_type "github.com/opengovern/opengovernance/services/integration-v2/integration-type"
	"go.uber.org/zap"
	"net/http"
	"reflect"
	"strconv"
)

func GetResourceTypeFromTableName(tableName string, queryIntegrationType []integration.Type) (string, integration.Type, error) {
	var integrationType integration.Type
	if len(queryIntegrationType) == 1 {
		integrationType = queryIntegrationType[0]
	} else {
		integrationType = ""
	}
	integrationTypeObj, ok := integration_type.IntegrationTypes[integrationType]
	if !ok {
		return "", "", echo.NewHTTPError(http.StatusInternalServerError, "unknown integration type")
	}
	integration, err := integrationTypeObj()
	if err != nil {
		return "", "", echo.NewHTTPError(http.StatusInternalServerError, "failed to get integration type")
	}
	return integration.GetResourceTypeFromTableName(tableName), integrationType, nil
}

func (w *Job) ExtractFindings(_ *zap.Logger, benchmarkCache map[string]api.Benchmark, caller Caller, res *steampipe.Result, query api.Query) ([]types.Finding, error) {
	var findings []types.Finding
	var integrationType integration.Type
	var err error
	queryResourceType := ""
	if query.PrimaryTable != nil || len(query.ListOfTables) == 1 {
		tableName := ""
		if query.PrimaryTable != nil {
			tableName = *query.PrimaryTable
		} else {
			tableName = query.ListOfTables[0]
		}
		if tableName != "" {
			queryResourceType, integrationType, err = GetResourceTypeFromTableName(tableName, w.ExecutionPlan.Query.IntegrationType)
			if err != nil {
				return nil, err
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

		var opengovernanceResourceId, connectionId, resourceID, resourceName, resourceLocation, reason string
		var costOptimization *float64
		var status types.ConformanceStatus
		if v, ok := recordValue["og_resource_id"].(string); ok {
			opengovernanceResourceId = v
		}
		if v, ok := recordValue["og_account_id"].(string); ok {
			connectionId = v
		}
		if v, ok := recordValue["og_table_name"].(string); ok && resourceType == "" {
			resourceType, integrationType, err = GetResourceTypeFromTableName(v, w.ExecutionPlan.Query.IntegrationType)
			if err != nil {
				return nil, err
			}
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

		benchmarkReferences := make([]string, 0, len([]string{caller.RootBenchmark}))
		for _, parentBenchmarkID := range []string{caller.RootBenchmark} {
			benchmarkReferences = append(benchmarkReferences, benchmarkCache[parentBenchmarkID].ReferenceCode)
		}

		if status != types.ConformanceStatusOK && status != types.ConformanceStatusALARM {
			continue
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
			IntegrationType:           integrationType,
			OpenGovernanceResourceID:  opengovernanceResourceId,
			ResourceID:                resourceID,
			ResourceName:              resourceName,
			ResourceLocation:          resourceLocation,
			ResourceType:              resourceType,
			Reason:                    reason,
			CostOptimization:          costOptimization,
			ComplianceJobID:           w.ID,
			ParentComplianceJobID:     w.ParentJobID,
			ParentBenchmarkReferences: benchmarkReferences,
			ParentBenchmarks:          []string{caller.RootBenchmark},
			LastTransition:            w.CreatedAt.UnixMilli(),
		})
	}
	return findings, nil
}
