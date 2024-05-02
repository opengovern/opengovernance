package recommendation

import (
	"fmt"
	types2 "github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/kaytu-io/kaytu-engine/services/wastage/api/entity"
	"github.com/kaytu-io/kaytu-engine/services/wastage/db/model"
	"github.com/kaytu-io/kaytu-engine/services/wastage/recommendation/preferences/aws_rds"
	"go.uber.org/zap"
	"strconv"
	"strings"
)

type awsRdsDbType struct {
	Engine  string
	Edition string
}

var dbTypeMap = map[string]awsRdsDbType{
	"aurora":            {"Aurora MySQL", ""},
	"aurora-mysql":      {"Aurora MySQL", ""},
	"aurora-postgresql": {"Aurora PostgreSQL", ""},
	"mariadb":           {"MariaDB", ""},
	"mysql":             {"MySQL", ""},
	"postgres":          {"PostgreSQL", ""},
	"oracle-se":         {"Oracle", "Standard"},
	"oracle-se1":        {"Oracle", "Standard One"},
	"oracle-se2":        {"Oracle", "Standard Two"},
	"oracle-se2-cdb":    {"Oracle", "Standard Two"},
	"oracle-ee":         {"Oracle", "Enterprise"},
	"oracle-ee-cdb":     {"Oracle", "Enterprise"},
	"sqlserver-se":      {"SQL Server", "Standard"},
	"sqlserver-ee":      {"SQL Server", "Enterprise"},
	"sqlserver-ex":      {"SQL Server", "Express"},
	"sqlserver-web":     {"SQL Server", "Web"},
}

func (s *Service) AwsRdsRecommendation(
	region string,
	rdsInstance entity.AwsRds,
	metrics map[string][]types2.Datapoint,
	preferences map[string]*string,
) (*entity.AwsRdsRightsizingRecommendation, error) {
	usageCpuPercent := extractUsage(metrics["CPUUtilization"])
	usageFreeMemoryBytes := extractUsage(metrics["FreeableMemory"])
	usageFreeStorageBytes := extractUsage(metrics["FreeStorageSpace"])
	usageNetworkThroughputBytes := extractUsage(sumMergeDatapoints(metrics["NetworkReceiveThroughput"], metrics["NetworkTransmitThroughput"]))
	usageStorageIops := extractUsage(sumMergeDatapoints(metrics["ReadIOPS"], metrics["WriteIOPS"]))
	usageStorageThroughputBytes := extractUsage(sumMergeDatapoints(metrics["ReadThroughput"], metrics["WriteThroughput"]))

	awsRdsDbKind, ok := dbTypeMap[strings.ToLower(rdsInstance.Engine)]
	if !ok {
		s.logger.Warn("rds engine not found", zap.String("engine", rdsInstance.Engine))
		awsRdsDbKind = awsRdsDbType{strings.ToLower(rdsInstance.Engine), ""}
	}

	currentInstanceTypeList, err := s.awsRDSDBInstanceRepo.ListByInstanceType(region, rdsInstance.InstanceType, awsRdsDbKind.Engine, awsRdsDbKind.Edition, string(rdsInstance.ClusterType))
	if err != nil {
		return nil, err
	}
	if len(currentInstanceTypeList) == 0 {
		s.logger.Error("rds instance type not found", zap.String("instance_type", rdsInstance.InstanceType))
		return nil, fmt.Errorf("rds instance type not found: %s", rdsInstance.InstanceType)
	}
	currentInstanceRow := currentInstanceTypeList[0]

	currentCost, err := s.costSvc.GetRDSInstanceCost(region, rdsInstance, metrics)
	if err != nil {
		s.logger.Error("failed to get rds instance cost", zap.Error(err))
		return nil, err
	}

	current := entity.RightsizingAwsRds{
		Region:            region,
		InstanceType:      rdsInstance.InstanceType,
		Engine:            rdsInstance.Engine,
		EngineVersion:     rdsInstance.EngineVersion,
		ClusterType:       rdsInstance.ClusterType,
		VCPU:              currentInstanceRow.VCpu,
		MemoryGb:          currentInstanceRow.MemoryGb,
		StorageType:       rdsInstance.StorageType,
		StorageSize:       rdsInstance.StorageSize,
		StorageIops:       rdsInstance.StorageIops,
		StorageThroughput: rdsInstance.StorageThroughput,
		Cost:              currentCost,
	}

	neededVCPU := (*usageCpuPercent.Avg / 100) * float64(currentInstanceRow.VCpu)
	if v, ok := preferences["CpuBreathingRoom"]; ok {
		vPercent, err := strconv.ParseInt(*v, 10, 64)
		if err != nil {
			s.logger.Error("invalid CpuBreathingRoom value", zap.String("value", *v))
			return nil, fmt.Errorf("invalid CpuBreathingRoom value: %s", *v)
		}
		neededVCPU = (1 + float64(vPercent)/100) * neededVCPU
	}
	neededMemoryGb := float64(currentInstanceRow.MemoryGb) - (*usageFreeMemoryBytes.Avg / 1e9)
	if v, ok := preferences["MemoryBreathingRoom"]; ok {
		vPercent, err := strconv.ParseInt(*v, 10, 64)
		if err != nil {
			s.logger.Error("invalid MemoryBreathingRoom value", zap.String("value", *v))
			return nil, fmt.Errorf("invalid MemoryBreathingRoom value: %s", *v)
		}
		neededMemoryGb = (1 + float64(vPercent)/100) * neededMemoryGb
	}
	neededNetworkThroughput := 0.0
	if currentInstanceRow.NetworkThroughput != nil {
		neededNetworkThroughput = *currentInstanceRow.NetworkThroughput - *usageNetworkThroughputBytes.Avg
	}
	if v, ok := preferences["NetworkBreathingRoom"]; ok {
		vPercent, err := strconv.ParseInt(*v, 10, 64)
		if err != nil {
			s.logger.Error("invalid NetworkBreathingRoom value", zap.String("value", *v))
			return nil, fmt.Errorf("invalid NetworkBreathingRoom value: %s", *v)
		}
		neededNetworkThroughput = (1 + float64(vPercent)/100) * neededNetworkThroughput
	}

	neededStorageSize := 0.0
	if rdsInstance.StorageSize != nil {
		neededStorageSize = float64(*rdsInstance.StorageSize) - (*usageFreeStorageBytes.Avg / 1e9)
		if v, ok := preferences["StorageSizeBreathingRoom"]; ok {
			vPercent, err := strconv.ParseInt(*v, 10, 64)
			if err != nil {
				s.logger.Error("invalid StorageSizeBreathingRoom value", zap.String("value", *v))
				return nil, fmt.Errorf("invalid StorageBreathingRoom value: %s", *v)
			}
			neededStorageSize = (1 + float64(vPercent)/100) * neededStorageSize
		}
	}
	neededStorageIops := 0.0
	if rdsInstance.StorageIops != nil {
		neededStorageIops = float64(*rdsInstance.StorageIops) - *usageStorageIops.Avg
		if v, ok := preferences["StorageIopsBreathingRoom"]; ok {
			vPercent, err := strconv.ParseInt(*v, 10, 64)
			if err != nil {
				s.logger.Error("invalid StorageIopsBreathingRoom value", zap.String("value", *v))
				return nil, fmt.Errorf("invalid StorageIopsBreathingRoom value: %s", *v)
			}
			neededStorageIops = (1 + float64(vPercent)/100) * neededStorageIops
		}
	}
	neededStorageThroughput := 0.0
	if rdsInstance.StorageThroughput != nil {
		neededStorageThroughput = float64(*rdsInstance.StorageThroughput) - *usageStorageThroughputBytes.Avg
		if v, ok := preferences["StorageThroughputBreathingRoom"]; ok {
			vPercent, err := strconv.ParseInt(*v, 10, 64)
			if err != nil {
				s.logger.Error("invalid StorageThroughputBreathingRoom value", zap.String("value", *v))
				return nil, fmt.Errorf("invalid StorageThroughputBreathingRoom value: %s", *v)
			}
			neededStorageThroughput = (1 + float64(vPercent)/100) * neededStorageThroughput
		}
	}

	instancePref := map[string]any{}
	for k, v := range preferences {
		var vl any
		if v == nil {
			vl = extractFromRdsInstance(rdsInstance, currentInstanceRow, region, k)
		} else {
			vl = *v
		}
		if aws_rds.PreferenceInstanceDBKey[k] == "" {
			continue
		}

		cond := "="
		if sc, ok := aws_rds.PreferenceInstanceSpecialCond[k]; ok {
			cond = sc
		}
		instancePref[fmt.Sprintf("%s %s ?", aws_rds.PreferenceInstanceDBKey[k], cond)] = vl
	}
	if _, ok := preferences["vCPU"]; !ok {
		instancePref["v_cpu >= ?"] = neededVCPU
	}
	if _, ok := preferences["MemoryGB"]; !ok {
		instancePref["memory_gb >= ?"] = neededMemoryGb
	}
	if _, ok := preferences["NetworkThroughput"]; !ok {
		instancePref["network_throughput IS NULL OR network_throughput >= ?"] = neededNetworkThroughput
	}
	if v, ok := instancePref["database_engine = ?"]; ok {
		kind := dbTypeMap[strings.ToLower(v.(string))]
		instancePref["database_engine = ?"] = kind.Engine
		if kind.Edition != "" {
			instancePref["database_edition = ?"] = kind.Edition
		}
	}

	rightSizedInstanceRow, err := s.awsRDSDBInstanceRepo.GetCheapestByPref(instancePref)
	if err != nil {
		s.logger.Error("failed to get rds instance type", zap.Error(err))
		return nil, err
	}

	// Aurora instance types storage configs are very different from other RDS instance types
	if (rightSizedInstanceRow != nil && !strings.Contains(strings.ToLower(rightSizedInstanceRow.InstanceType), "aurora")) ||
		(rightSizedInstanceRow == nil && !strings.Contains(strings.ToLower(currentInstanceRow.InstanceType), "aurora")) {
		storagePref := map[string]any{}
		for k, v := range preferences {
			var vl any
			if v == nil {
				vl = extractFromRdsInstance(rdsInstance, currentInstanceRow, region, k)
			} else {
				vl = *v
			}
			if aws_rds.PreferenceStorageDBKey[k] == "" {
				continue
			}

			cond := "="
			if sc, ok := aws_rds.PreferenceStorageSpecialCond[k]; ok {
				cond = sc
			}
			storagePref[fmt.Sprintf("%s %s ?", aws_rds.PreferenceInstanceDBKey[k], cond)] = vl
		}
	} else {
		// TODO handle aurora, suggest normal or io optimized storage
	}

	var recommended *entity.RightsizingAwsRds
	if rightSizedInstanceRow != nil {
		newInstance := rdsInstance
		newInstance.InstanceType = rightSizedInstanceRow.InstanceType
		newInstance.ClusterType = entity.AwsRdsClusterType(rightSizedInstanceRow.DeploymentOption)
		for k, v := range dbTypeMap {
			if strings.ToLower(v.Engine) == strings.ToLower(rightSizedInstanceRow.DatabaseEngine) && (v.Edition == "" || strings.ToLower(v.Edition) == strings.ToLower(rightSizedInstanceRow.DatabaseEdition)) {
				newInstance.Engine = k
				break
			}
		}
		newInstance.LicenseModel = rightSizedInstanceRow.LicenseModel

		recommendedCost, err := s.costSvc.GetRDSInstanceCost(region, newInstance, metrics)
		if err != nil {
			s.logger.Error("failed to get rds instance cost", zap.Error(err))
			return nil, err
		}

		recommended = &entity.RightsizingAwsRds{
			Region:            rightSizedInstanceRow.RegionCode,
			InstanceType:      rightSizedInstanceRow.InstanceType,
			Engine:            rightSizedInstanceRow.DatabaseEngine,
			EngineVersion:     newInstance.EngineVersion,
			ClusterType:       newInstance.ClusterType,
			VCPU:              rightSizedInstanceRow.VCpu,
			MemoryGb:          rightSizedInstanceRow.MemoryGb,
			StorageType:       newInstance.StorageType,
			StorageSize:       newInstance.StorageSize,
			StorageIops:       newInstance.StorageIops,
			StorageThroughput: newInstance.StorageThroughput,
			Cost:              recommendedCost,
		}
	}

	recommendation := entity.AwsRdsRightsizingRecommendation{
		Current:     current,
		Recommended: recommended,

		Description: "",
	}

	return &recommendation, nil
}

func extractFromRdsInstance(instance entity.AwsRds, i model.RDSDBInstance, region string, k string) any {
	switch k {
	case "Region":
		return region
	case "vCPU":
		return i.VCpu
	case "MemoryGB":
		return i.MemoryGb
	case "InstanceType":
		return instance.InstanceType
	case "Engine":
		return instance.Engine
	case "ClusterType":
		return instance.ClusterType
	case "StorageType":
		return instance.StorageType
	case "StorageSize":
		return instance.StorageSize
	case "StorageIops":
		return instance.StorageIops
	case "StorageThroughput":
		return instance.StorageThroughput
	}
	return ""
}
