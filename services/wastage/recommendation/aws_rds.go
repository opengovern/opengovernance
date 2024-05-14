package recommendation

import (
	"fmt"
	types2 "github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/kaytu-io/kaytu-engine/pkg/utils"
	"github.com/kaytu-io/kaytu-engine/services/wastage/api/entity"
	"github.com/kaytu-io/kaytu-engine/services/wastage/db/model"
	"github.com/kaytu-io/kaytu-engine/services/wastage/recommendation/preferences/aws_rds"
	"go.uber.org/zap"
	"math"
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

func awsRdsDbTypeToAPIDbType(engine, edition string) string {
	for k, v := range dbTypeMap {
		if strings.ToLower(v.Engine) == strings.ToLower(engine) && (v.Edition == "" || strings.ToLower(v.Edition) == strings.ToLower(edition)) {
			return k
		}
	}
	return ""
}

func (s *Service) AwsRdsRecommendation(
	region string,
	rdsInstance entity.AwsRds,
	metrics map[string][]types2.Datapoint,
	preferences map[string]*string,
	usageAverageType UsageAverageType,
) (*entity.AwsRdsRightsizingRecommendation, error) {
	usageCpuPercent := extractUsage(metrics["CPUUtilization"], usageAverageType)
	usageFreeMemoryBytes := extractUsage(metrics["FreeableMemory"], usageAverageType)
	usageFreeStorageBytes := extractUsage(metrics["FreeStorageSpace"], usageAverageType)
	usageVolumeBytesUsed := extractUsage(metrics["VolumeBytesUsed"], usageAverageType)
	usageNetworkThroughputBytes := extractUsage(sumMergeDatapoints(metrics["NetworkReceiveThroughput"], metrics["NetworkTransmitThroughput"]), usageAverageType)
	usageStorageIops := extractUsage(sumMergeDatapoints(metrics["ReadIOPS"], metrics["WriteIOPS"]), usageAverageType)
	usageStorageThroughputBytes := extractUsage(sumMergeDatapoints(metrics["ReadThroughput"], metrics["WriteThroughput"]), usageAverageType)
	usageStorageThroughputMB := entity.Usage{
		Avg: funcP(usageStorageThroughputBytes.Avg, usageStorageThroughputBytes.Avg, func(a, _ float64) float64 { return a / (1024 * 1024) }),
		Min: funcP(usageStorageThroughputBytes.Min, usageStorageThroughputBytes.Min, func(a, _ float64) float64 { return a / (1024 * 1024) }),
		Max: funcP(usageStorageThroughputBytes.Max, usageStorageThroughputBytes.Max, func(a, _ float64) float64 { return a / (1024 * 1024) }),
	}

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

	currentComputeCost, err := s.costSvc.GetRDSComputeCost(region, rdsInstance, metrics)
	if err != nil {
		s.logger.Error("failed to get rds compute cost", zap.Error(err))
		return nil, err
	}
	currentStorageCost, err := s.costSvc.GetRDSStorageCost(region, rdsInstance, metrics)
	if err != nil {
		s.logger.Error("failed to get rds storage cost", zap.Error(err))
		return nil, err
	}

	current := entity.RightsizingAwsRds{
		Region:            region,
		InstanceType:      rdsInstance.InstanceType,
		Engine:            rdsInstance.Engine,
		EngineVersion:     rdsInstance.EngineVersion,
		ClusterType:       rdsInstance.ClusterType,
		VCPU:              int64(currentInstanceRow.VCpu),
		MemoryGb:          int64(currentInstanceRow.MemoryGb),
		StorageType:       rdsInstance.StorageType,
		StorageSize:       rdsInstance.StorageSize,
		StorageIops:       rdsInstance.StorageIops,
		StorageThroughput: rdsInstance.StorageThroughput,

		Cost:        currentCost,
		ComputeCost: currentComputeCost,
		StorageCost: currentStorageCost,
	}
	if strings.Contains(strings.ToLower(rdsInstance.Engine), "aurora") {
		current.StorageSize = utils.GetPointer(int32(math.Ceil(*usageVolumeBytesUsed.Avg / (1024 * 1024 * 1024))))
		current.StorageIops = nil
		current.StorageThroughput = nil
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
	neededMemoryGb := float64(currentInstanceRow.MemoryGb) - (*usageFreeMemoryBytes.Avg / (1024 * 1024 * 1024))
	if v, ok := preferences["MemoryBreathingRoom"]; ok {
		vPercent, err := strconv.ParseInt(*v, 10, 64)
		if err != nil {
			s.logger.Error("invalid MemoryBreathingRoom value", zap.String("value", *v))
			return nil, fmt.Errorf("invalid MemoryBreathingRoom value: %s", *v)
		}
		neededMemoryGb = (1 + float64(vPercent)/100) * neededMemoryGb
	}
	neededNetworkThroughput := 0.0
	if usageNetworkThroughputBytes.Avg != nil {
		neededNetworkThroughput = *usageNetworkThroughputBytes.Avg
	}
	if v, ok := preferences["NetworkBreathingRoom"]; ok {
		vPercent, err := strconv.ParseInt(*v, 10, 64)
		if err != nil {
			s.logger.Error("invalid NetworkBreathingRoom value", zap.String("value", *v))
			return nil, fmt.Errorf("invalid NetworkBreathingRoom value: %s", *v)
		}
		neededNetworkThroughput = (1 + float64(vPercent)/100) * neededNetworkThroughput
	}

	neededStorageSize := int32(0)
	if rdsInstance.StorageSize != nil {
		neededStorageSizeFloat := float64(*rdsInstance.StorageSize) - (*usageFreeStorageBytes.Avg / (1024 * 1024 * 1024))
		if strings.Contains(strings.ToLower(rdsInstance.Engine), "aurora") {
			neededStorageSizeFloat = *usageVolumeBytesUsed.Avg / (1024 * 1024 * 1024)
		}
		if v, ok := preferences["StorageSizeBreathingRoom"]; ok {
			vPercent, err := strconv.ParseInt(*v, 10, 64)
			if err != nil {
				s.logger.Error("invalid StorageSizeBreathingRoom value", zap.String("value", *v))
				return nil, fmt.Errorf("invalid StorageBreathingRoom value: %s", *v)
			}
			neededStorageSizeFloat = (1 + float64(vPercent)/100) * neededStorageSizeFloat
		}
		neededStorageSize = int32(neededStorageSizeFloat)
	}
	neededStorageIops := int32(0)
	if usageStorageIops.Avg != nil {
		neededStorageIopsFloat := *usageStorageIops.Avg
		if v, ok := preferences["StorageIopsBreathingRoom"]; ok {
			vPercent, err := strconv.ParseInt(*v, 10, 64)
			if err != nil {
				s.logger.Error("invalid StorageIopsBreathingRoom value", zap.String("value", *v))
				return nil, fmt.Errorf("invalid StorageIopsBreathingRoom value: %s", *v)
			}
			neededStorageIopsFloat = (1 + float64(vPercent)/100) * neededStorageIopsFloat
		}
		neededStorageIops = int32(neededStorageIopsFloat)
	}
	neededStorageThroughputMB := 0.0
	if usageStorageThroughputMB.Avg != nil {
		neededStorageThroughputMB = *usageStorageThroughputMB.Avg
		if v, ok := preferences["StorageThroughputBreathingRoom"]; ok {
			vPercent, err := strconv.ParseInt(*v, 10, 64)
			if err != nil {
				s.logger.Error("invalid StorageThroughputBreathingRoom value", zap.String("value", *v))
				return nil, fmt.Errorf("invalid StorageThroughputBreathingRoom value: %s", *v)
			}
			neededStorageThroughputMB = (1 + float64(vPercent)/100) * neededStorageThroughputMB
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
		if _, ok := aws_rds.PreferenceInstanceDBKey[k]; !ok {
			continue
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
	if _, ok := instancePref["instance_type = ?"]; !ok {
		if value, ok := preferences["ExcludeBurstableInstances"]; ok && value != nil {
			if *value == "Yes" {
				instancePref["NOT(instance_type like ?)"] = "db.t%"
			}
		}
	}

	rightSizedInstanceRow, err := s.awsRDSDBInstanceRepo.GetCheapestByPref(instancePref)
	if err != nil {
		s.logger.Error("failed to get rds instance type", zap.Error(err))
		return nil, err
	}

	var resultEngine, resultEdition, resultClusterType string
	if rightSizedInstanceRow != nil {
		resultEngine = rightSizedInstanceRow.DatabaseEngine
		resultEdition = rightSizedInstanceRow.DatabaseEdition
		resultClusterType = rightSizedInstanceRow.DeploymentOption
	} else {
		resultEngine = awsRdsDbKind.Engine
		resultEdition = awsRdsDbKind.Edition
		resultClusterType = string(rdsInstance.ClusterType)
	}
	// Aurora instance types storage configs are very different from other RDS instance types
	isResultAurora := !((rightSizedInstanceRow != nil && !strings.Contains(strings.ToLower(rightSizedInstanceRow.InstanceType), "aurora")) || (rightSizedInstanceRow == nil && !strings.Contains(strings.ToLower(currentInstanceRow.InstanceType), "aurora")))

	var rightSizedStorageRow *model.RDSDBStorage

	var validTypes []model.RDSDBStorageVolumeType
	if v, ok := preferences["StorageType"]; ok {
		if v == nil {
			st := extractFromRdsInstance(rdsInstance, currentInstanceRow, region, "StorageType")
			volType := model.RDSDBStorageEBSTypeToVolumeType[st.(string)]
			validTypes = append(validTypes, volType)
		} else if *v != "" {
			validTypes = append(validTypes, model.RDSDBStorageVolumeType(*v))
		}
	}

	var resSize, resIops int32
	var resThroughputMB float64
	rightSizedStorageRow, resSize, resIops, resThroughputMB, err = s.awsRDSDBStorageRepo.GetCheapestBySpecs(region, resultEngine, resultEdition, resultClusterType, neededStorageSize, neededStorageIops, neededStorageThroughputMB, validTypes)
	if err != nil {
		s.logger.Error("failed to get rds storage type", zap.Error(err))
		return nil, err
	}
	neededStorageSize = resSize
	if !isResultAurora {
		neededStorageIops = resIops
		neededStorageThroughputMB = resThroughputMB
	}

	var recommended *entity.RightsizingAwsRds
	var newInstance entity.AwsRds
	if rightSizedInstanceRow != nil {
		newInstance = rdsInstance
		newInstance.InstanceType = rightSizedInstanceRow.InstanceType
		newInstance.ClusterType = entity.AwsRdsClusterType(rightSizedInstanceRow.DeploymentOption)
		for k, v := range dbTypeMap {
			if strings.ToLower(v.Engine) == strings.ToLower(rightSizedInstanceRow.DatabaseEngine) && (v.Edition == "" || strings.ToLower(v.Edition) == strings.ToLower(rightSizedInstanceRow.DatabaseEdition)) {
				newInstance.Engine = k
				break
			}
		}
		newInstance.LicenseModel = rightSizedInstanceRow.LicenseModel

		recommended = &entity.RightsizingAwsRds{
			Region:        rightSizedInstanceRow.RegionCode,
			InstanceType:  rightSizedInstanceRow.InstanceType,
			Engine:        awsRdsDbTypeToAPIDbType(rightSizedInstanceRow.DatabaseEngine, rightSizedInstanceRow.DatabaseEdition),
			EngineVersion: newInstance.EngineVersion,
			ClusterType:   newInstance.ClusterType,
			VCPU:          int64(rightSizedInstanceRow.VCpu),
			MemoryGb:      int64(rightSizedInstanceRow.MemoryGb),
			Cost:          0,
			ComputeCost:   0,
			StorageCost:   0,
		}
		if rightSizedStorageRow == nil {
			recommended.StorageType = newInstance.StorageType
			recommended.StorageSize = newInstance.StorageSize
			recommended.StorageIops = newInstance.StorageIops
			recommended.StorageThroughput = newInstance.StorageThroughput
		}
	}
	if rightSizedStorageRow != nil {
		if recommended == nil {
			recommended = &entity.RightsizingAwsRds{
				Region:        region,
				InstanceType:  currentInstanceRow.InstanceType,
				Engine:        currentInstanceRow.DatabaseEngine,
				EngineVersion: rdsInstance.EngineVersion,
				ClusterType:   rdsInstance.ClusterType,
				VCPU:          int64(currentInstanceRow.VCpu),
				MemoryGb:      int64(currentInstanceRow.MemoryGb),
				Cost:          currentCost,
				ComputeCost:   currentComputeCost,
				StorageCost:   currentStorageCost,
			}
		}
		ebsType := model.RDSDBStorageVolumeTypeToEBSType[rightSizedStorageRow.VolumeType]
		recommended.StorageType = &ebsType
		newInstance.StorageType = &ebsType

		recommended.StorageSize = &neededStorageSize
		newInstance.StorageSize = &neededStorageSize

		if ebsType == "io1" || ebsType == "io2" || ebsType == "gp3" {
			recommended.StorageIops = &neededStorageIops
			newInstance.StorageIops = &neededStorageIops
		} else {
			recommended.StorageIops = nil
			newInstance.StorageIops = nil
		}
		if ebsType == "gp3" {
			recommended.StorageThroughput = &neededStorageThroughputMB
			newInstance.StorageThroughput = &neededStorageThroughputMB
		} else {
			recommended.StorageThroughput = nil
			newInstance.StorageThroughput = nil
		}
	}

	if recommended != nil {
		recommendedCost, err := s.costSvc.GetRDSInstanceCost(region, newInstance, metrics)
		if err != nil {
			s.logger.Error("failed to get rds instance cost", zap.Error(err))
			return nil, err
		}
		recommended.Cost = recommendedCost

		recommendedComputeCost, err := s.costSvc.GetRDSComputeCost(region, newInstance, metrics)
		if err != nil {
			s.logger.Error("failed to get rds instance cost", zap.Error(err))
			return nil, err
		}
		recommendedStorageCost, err := s.costSvc.GetRDSStorageCost(region, newInstance, metrics)
		if err != nil {
			s.logger.Error("failed to get rds instance cost", zap.Error(err))
			return nil, err
		}
		recommended.ComputeCost = recommendedComputeCost
		recommended.StorageCost = recommendedStorageCost
	}

	recommendation := entity.AwsRdsRightsizingRecommendation{
		Current:     current,
		Recommended: recommended,

		VCPU:                   usageCpuPercent,
		StorageIops:            usageStorageIops,
		FreeMemoryBytes:        usageFreeMemoryBytes,
		NetworkThroughputBytes: usageNetworkThroughputBytes,
		FreeStorageBytes:       usageFreeStorageBytes,
		StorageThroughput:      usageStorageThroughputBytes,

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
	case "InstanceFamily":
		return i.InstanceFamily
	case "LicenseModel":
		return i.LicenseModel
	}
	return ""
}
