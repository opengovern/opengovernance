package cost

import (
	"context"
	"encoding/json"
	"fmt"
	types2 "github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/kaytu-io/pennywise/pkg/cost"
	"github.com/kaytu-io/pennywise/pkg/schema"
	"github.com/opengovern/og-util/pkg/httpclient"
	kaytu_client "github.com/opengovern/opengovernance/pkg/steampipe-plugin-opengovernance/opengovernance-client"
	"github.com/opengovern/opengovernance/services/wastage/api/entity"
	"net/http"
	"time"
)

func (s *Service) GetRDSInstanceCost(ctx context.Context, region string, rdsInstance entity.AwsRds, metrics map[string][]types2.Datapoint) (float64, map[string]float64, error) {
	req := schema.Submission{
		ID:        "submission-1",
		CreatedAt: time.Now(),
		Resources: []schema.ResourceDef{},
	}

	valuesMap := map[string]any{}
	valuesMap["region"] = region
	valuesMap["instance_class"] = rdsInstance.InstanceType
	valuesMap["availability_zone"] = rdsInstance.AvailabilityZone
	valuesMap["engine"] = rdsInstance.Engine
	valuesMap["engine_version"] = rdsInstance.EngineVersion
	valuesMap["license_model"] = rdsInstance.LicenseModel
	if rdsInstance.ClusterType == entity.AwsRdsClusterTypeSingleInstance {
		valuesMap["multi_az"] = false
	} else {
		valuesMap["multi_az"] = true
	}
	valuesMap["cluster_type"] = rdsInstance.ClusterType

	if rdsInstance.StorageSize != nil {
		valuesMap["allocated_storage"] = *rdsInstance.StorageSize
	}
	valuesMap["backup_retention_period"] = rdsInstance.BackupRetentionPeriod
	if rdsInstance.StorageType != nil {
		valuesMap["storage_type"] = *rdsInstance.StorageType
	}
	if rdsInstance.StorageIops != nil {
		valuesMap["iops"] = *rdsInstance.StorageIops
	}
	valuesMap["performance_insights_enabled"] = rdsInstance.PerformanceInsightsEnabled
	valuesMap["performance_insights_retention_period"] = rdsInstance.PerformanceInsightsRetentionPeriod
	valuesMap["io_optimized"] = false // TODO: Check aws api rds response // Maybe needs some changes in pennywise logic

	valuesMap["pennywise_usage"] = map[string]any{
		//"monthly_io_requests":                              "",
		//"monthly_data_api_calls":                           "",
		//"additional_backup_storage_gb":                     "",
		//"monthly_additional_performance_insights_requests": "",
	}

	req.Resources = append(req.Resources, schema.ResourceDef{
		Address:      rdsInstance.HashedInstanceId,
		Type:         kaytu_client.ResourceTypeConversion("aws::rds::dbinstance"),
		Name:         "",
		RegionCode:   region,
		ProviderName: schema.AWSProvider,
		Values:       valuesMap,
	})

	reqBody, err := json.Marshal(req)
	if err != nil {
		return 0, nil, err
	}

	var response cost.State
	statusCode, err := httpclient.DoRequest(ctx, "GET", s.pennywiseBaseUrl+"/api/v1/cost/submission", nil, reqBody, &response)
	if err != nil {
		return 0, nil, err
	}

	if statusCode != http.StatusOK {
		return 0, nil, fmt.Errorf("failed to get pennywise cost, status code = %d", statusCode)
	}

	componentCost := make(map[string]float64)
	for _, component := range response.GetCostComponents() {
		if component.Cost().Decimal.InexactFloat64() == 0 {
			continue
		}
		componentCost[component.Name] = component.Cost().Decimal.InexactFloat64()
	}

	resourceCost, err := response.Cost()
	if err != nil {
		return 0, nil, err
	}

	return resourceCost.Decimal.InexactFloat64(), componentCost, nil
}

func (s *Service) GetRDSStorageCost(ctx context.Context, region string, rdsInstance entity.AwsRds, metrics map[string][]types2.Datapoint) (float64, map[string]float64, error) {
	req := schema.Submission{
		ID:        "submission-1",
		CreatedAt: time.Now(),
		Resources: []schema.ResourceDef{},
	}

	valuesMap := map[string]any{}
	valuesMap["region"] = region
	valuesMap["instance_class"] = rdsInstance.InstanceType
	valuesMap["availability_zone"] = rdsInstance.AvailabilityZone
	valuesMap["engine"] = rdsInstance.Engine
	valuesMap["engine_version"] = rdsInstance.EngineVersion
	valuesMap["license_model"] = rdsInstance.LicenseModel
	if rdsInstance.ClusterType == entity.AwsRdsClusterTypeSingleInstance {
		valuesMap["multi_az"] = false
	} else {
		valuesMap["multi_az"] = true
	}
	valuesMap["cluster_type"] = rdsInstance.ClusterType

	if rdsInstance.StorageSize != nil {
		valuesMap["allocated_storage"] = *rdsInstance.StorageSize
	}
	valuesMap["backup_retention_period"] = rdsInstance.BackupRetentionPeriod
	if rdsInstance.StorageType != nil {
		valuesMap["storage_type"] = *rdsInstance.StorageType
	}
	if rdsInstance.StorageIops != nil {
		valuesMap["iops"] = *rdsInstance.StorageIops
	}
	if rdsInstance.StorageThroughput != nil {
		valuesMap["throughput"] = *rdsInstance.StorageThroughput
	}
	valuesMap["performance_insights_enabled"] = rdsInstance.PerformanceInsightsEnabled
	valuesMap["performance_insights_retention_period"] = rdsInstance.PerformanceInsightsRetentionPeriod
	valuesMap["io_optimized"] = false // TODO: Check aws api rds response // Maybe needs some changes in pennywise logic

	valuesMap["pennywise_usage"] = map[string]any{
		//"monthly_io_requests":                              "",
		//"additional_backup_storage_gb":                     "",
	}

	req.Resources = append(req.Resources, schema.ResourceDef{
		Address:      rdsInstance.HashedInstanceId,
		Type:         "aws_db_storage",
		Name:         "",
		RegionCode:   region,
		ProviderName: schema.AWSProvider,
		Values:       valuesMap,
	})

	reqBody, err := json.Marshal(req)
	if err != nil {
		return 0, nil, err
	}

	var response cost.State
	statusCode, err := httpclient.DoRequest(ctx, "GET", s.pennywiseBaseUrl+"/api/v1/cost/submission", nil, reqBody, &response)
	if err != nil {
		return 0, nil, err
	}

	if statusCode != http.StatusOK {
		return 0, nil, fmt.Errorf("failed to get pennywise cost, status code = %d", statusCode)
	}

	componentCost := make(map[string]float64)
	for _, component := range response.GetCostComponents() {
		if component.Cost().Decimal.InexactFloat64() == 0 {
			continue
		}
		componentCost[component.Name] = component.Cost().Decimal.InexactFloat64()
	}

	resourceCost, err := response.Cost()
	if err != nil {
		return 0, nil, err
	}

	return resourceCost.Decimal.InexactFloat64(), componentCost, nil
}

func (s *Service) GetRDSComputeCost(ctx context.Context, region string, rdsInstance entity.AwsRds, metrics map[string][]types2.Datapoint) (float64, map[string]float64, error) {
	req := schema.Submission{
		ID:        "submission-1",
		CreatedAt: time.Now(),
		Resources: []schema.ResourceDef{},
	}

	valuesMap := map[string]any{}
	valuesMap["region"] = region
	valuesMap["instance_class"] = rdsInstance.InstanceType
	valuesMap["availability_zone"] = rdsInstance.AvailabilityZone
	valuesMap["engine"] = rdsInstance.Engine
	valuesMap["engine_version"] = rdsInstance.EngineVersion
	valuesMap["license_model"] = rdsInstance.LicenseModel
	if rdsInstance.ClusterType == entity.AwsRdsClusterTypeSingleInstance {
		valuesMap["multi_az"] = false
	} else {
		valuesMap["multi_az"] = true
	}
	valuesMap["cluster_type"] = rdsInstance.ClusterType

	if rdsInstance.StorageSize != nil {
		valuesMap["allocated_storage"] = *rdsInstance.StorageSize
	}
	valuesMap["backup_retention_period"] = rdsInstance.BackupRetentionPeriod
	if rdsInstance.StorageType != nil {
		valuesMap["storage_type"] = *rdsInstance.StorageType
	}
	if rdsInstance.StorageIops != nil {
		valuesMap["iops"] = *rdsInstance.StorageIops
	}
	valuesMap["performance_insights_enabled"] = rdsInstance.PerformanceInsightsEnabled
	valuesMap["performance_insights_retention_period"] = rdsInstance.PerformanceInsightsRetentionPeriod
	valuesMap["io_optimized"] = false // TODO: Check aws api rds response // Maybe needs some changes in pennywise logic

	valuesMap["pennywise_usage"] = map[string]any{
		//"monthly_io_requests":                              "",
		//"monthly_data_api_calls":                           "",
		//"additional_backup_storage_gb":                     "",
		//"monthly_additional_performance_insights_requests": "",
	}

	req.Resources = append(req.Resources, schema.ResourceDef{
		Address:      rdsInstance.HashedInstanceId,
		Type:         "aws_db_compute_instance",
		Name:         "",
		RegionCode:   region,
		ProviderName: schema.AWSProvider,
		Values:       valuesMap,
	})

	reqBody, err := json.Marshal(req)
	if err != nil {
		return 0, nil, err
	}

	var response cost.State
	statusCode, err := httpclient.DoRequest(ctx, "GET", s.pennywiseBaseUrl+"/api/v1/cost/submission", nil, reqBody, &response)
	if err != nil {
		return 0, nil, err
	}

	if statusCode != http.StatusOK {
		return 0, nil, fmt.Errorf("failed to get pennywise cost, status code = %d", statusCode)
	}

	componentCost := make(map[string]float64)
	for _, component := range response.GetCostComponents() {
		if component.Cost().Decimal.InexactFloat64() == 0 {
			continue
		}
		componentCost[component.Name] = component.Cost().Decimal.InexactFloat64()
	}

	resourceCost, err := response.Cost()
	if err != nil {
		return 0, nil, err
	}

	return resourceCost.Decimal.InexactFloat64(), componentCost, nil
}
