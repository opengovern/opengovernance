package recommendation

import (
	"context"
	"errors"
	"fmt"
	types2 "github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/kaytu-io/kaytu-engine/pkg/utils"
	"github.com/kaytu-io/kaytu-engine/services/wastage/api/entity"
	"github.com/kaytu-io/kaytu-engine/services/wastage/db/model"
	"github.com/kaytu-io/kaytu-engine/services/wastage/recommendation/preferences/aws_rds"
	"github.com/labstack/echo/v4"
	"github.com/sashabaranov/go-openai"
	"go.uber.org/zap"
	"math"
	"net/http"
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

func calculateHeadroom(needed float64, percent int64) float64 {
	return needed / (1.0 - (float64(percent) / 100.0))
}

func pCalculateHeadroom(needed *float64, percent int64) float64 {
	if needed == nil {
		return 0.0
	}
	return *needed / (1.0 - (float64(percent) / 100.0))
}

func (s *Service) AwsRdsRecommendation(
	ctx context.Context,
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
		return nil, echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("rds instance type %s with %s engine not found", rdsInstance.InstanceType, rdsInstance.Engine))
	}
	currentInstanceRow := currentInstanceTypeList[0]

	if strings.Contains(strings.ToLower(rdsInstance.Engine), "aurora") {
		rdsInstance.StorageSize = utils.GetPointer(int32(math.Ceil(getValueOrZero(usageVolumeBytesUsed.Avg) / (1024 * 1024 * 1024))))
		if usageVolumeBytesUsed.Last.Maximum != nil {
			rdsInstance.StorageSize = utils.GetPointer(int32(math.Ceil(getValueOrZero(usageVolumeBytesUsed.Last.Maximum) / (1024 * 1024 * 1024))))
		}
		rdsInstance.StorageIops = nil
		rdsInstance.StorageThroughput = nil
	}

	currentComputeCost, err := s.costSvc.GetRDSComputeCost(ctx, region, rdsInstance, metrics)
	if err != nil {
		s.logger.Error("failed to get rds compute cost", zap.Error(err))
		return nil, err
	}
	currentStorageCost, err := s.costSvc.GetRDSStorageCost(ctx, region, rdsInstance, metrics)
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
		Architecture:      currentInstanceRow.ProcessorArchitecture,
		Processor:         currentInstanceRow.PhysicalProcessor,
		VCPU:              int64(currentInstanceRow.VCpu),
		MemoryGb:          int64(currentInstanceRow.MemoryGb),
		StorageType:       rdsInstance.StorageType,
		StorageSize:       rdsInstance.StorageSize,
		StorageIops:       rdsInstance.StorageIops,
		StorageThroughput: rdsInstance.StorageThroughput,

		Cost:        currentComputeCost + currentStorageCost,
		ComputeCost: currentComputeCost,
		StorageCost: currentStorageCost,
	}

	neededVCPU := (getValueOrZero(usageCpuPercent.Avg) / 100.0) * currentInstanceRow.VCpu
	if v, ok := preferences["CpuBreathingRoom"]; ok && v != nil {
		vPercent, err := strconv.ParseInt(*v, 10, 64)
		if err != nil {
			s.logger.Error("invalid CpuBreathingRoom value", zap.String("value", *v))
			return nil, echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("invalid CpuBreathingRoom value: %s", *v))
		}
		neededVCPU = calculateHeadroom(neededVCPU, vPercent)
	}
	usageFreeMemoryBytesMin := 0.0
	if usageFreeMemoryBytes.Min != nil {
		usageFreeMemoryBytesMin = *usageFreeMemoryBytes.Min
	} else if usageFreeMemoryBytes.Avg != nil {
		usageFreeMemoryBytesMin = *usageFreeMemoryBytes.Avg
	}
	neededMemoryGb := currentInstanceRow.MemoryGb - (usageFreeMemoryBytesMin / (1024 * 1024 * 1024))
	if v, ok := preferences["MemoryBreathingRoom"]; ok {
		vPercent, err := strconv.ParseInt(*v, 10, 64)
		if err != nil {
			s.logger.Error("invalid MemoryBreathingRoom value", zap.String("value", *v))
			return nil, echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("invalid MemoryBreathingRoom value: %s", *v))
		}
		neededMemoryGb = calculateHeadroom(neededMemoryGb, vPercent)
	}
	neededNetworkThroughput := 0.0
	if usageNetworkThroughputBytes.Avg != nil {
		neededNetworkThroughput = *usageNetworkThroughputBytes.Avg
	}
	if v, ok := preferences["NetworkBreathingRoom"]; ok && v != nil {
		vPercent, err := strconv.ParseInt(*v, 10, 64)
		if err != nil {
			s.logger.Error("invalid NetworkBreathingRoom value", zap.String("value", *v))
			return nil, echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("invalid NetworkBreathingRoom value: %s", *v))
		}
		neededNetworkThroughput = calculateHeadroom(neededNetworkThroughput, vPercent)
	}

	neededStorageSize := int32(0)
	if rdsInstance.StorageSize != nil {
		usageFreeStorageBytesMin := 0.0
		if usageFreeStorageBytes.Min != nil {
			usageFreeStorageBytesMin = *usageFreeStorageBytes.Min
		} else if usageFreeStorageBytes.Avg != nil {
			usageFreeStorageBytesMin = *usageFreeStorageBytes.Avg
		}
		neededStorageSizeFloat := float64(*rdsInstance.StorageSize) - (usageFreeStorageBytesMin / (1024 * 1024 * 1024))
		if strings.Contains(strings.ToLower(rdsInstance.Engine), "aurora") {
			if usageVolumeBytesUsed.Max != nil {
				neededStorageSizeFloat = *usageVolumeBytesUsed.Max / (1024 * 1024 * 1024)
			} else if usageVolumeBytesUsed.Avg != nil {
				neededStorageSizeFloat = *usageVolumeBytesUsed.Avg / (1024 * 1024 * 1024)
			}
		}
		if v, ok := preferences["StorageSizeBreathingRoom"]; ok && v != nil {
			vPercent, err := strconv.ParseInt(*v, 10, 64)
			if err != nil {
				s.logger.Error("invalid StorageSizeBreathingRoom value", zap.String("value", *v))
				return nil, echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("invalid StorageSizeBreathingRoom value: %s", *v))
			}
			neededStorageSizeFloat = calculateHeadroom(neededStorageSizeFloat, vPercent)
		}
		neededStorageSize = int32(math.Ceil(neededStorageSizeFloat))
	}
	neededStorageIops := int32(0)
	if usageStorageIops.Avg != nil {
		neededStorageIopsFloat := *usageStorageIops.Avg
		if v, ok := preferences["StorageIopsBreathingRoom"]; ok && v != nil {
			vPercent, err := strconv.ParseInt(*v, 10, 64)
			if err != nil {
				s.logger.Error("invalid StorageIopsBreathingRoom value", zap.String("value", *v))
				return nil, echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("invalid StorageIopsBreathingRoom value: %s", *v))
			}
			neededStorageIopsFloat = calculateHeadroom(neededStorageIopsFloat, vPercent)
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
				return nil, echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("invalid StorageThroughputBreathingRoom value: %s", *v))
			}
			neededStorageThroughputMB = calculateHeadroom(neededStorageThroughputMB, vPercent)
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

	excluedBurstable := false
	if _, ok := instancePref["instance_type = ?"]; !ok {
		if value, ok := preferences["ExcludeBurstableInstances"]; ok && value != nil {
			if *value == "Yes" {
				excluedBurstable = true
				instancePref["NOT(instance_type like ?)"] = "db.t%"
			} else if *value == "if current resource is burstable" {
				if !strings.HasPrefix(rdsInstance.InstanceType, "db.t") {
					excluedBurstable = true
					instancePref["NOT(instance_type like ?)"] = "db.t%"
				}
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
			Architecture:  rightSizedInstanceRow.ProcessorArchitecture,
			Processor:     rightSizedInstanceRow.PhysicalProcessor,
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
	} else {
		newInstance = rdsInstance
	}
	if rightSizedStorageRow != nil {
		if recommended == nil {
			recommended = &entity.RightsizingAwsRds{
				Region:        region,
				InstanceType:  currentInstanceRow.InstanceType,
				Engine:        awsRdsDbTypeToAPIDbType(currentInstanceRow.DatabaseEngine, currentInstanceRow.DatabaseEdition),
				EngineVersion: rdsInstance.EngineVersion,
				ClusterType:   rdsInstance.ClusterType,
				Architecture:  currentInstanceRow.ProcessorArchitecture,
				Processor:     currentInstanceRow.PhysicalProcessor,
				VCPU:          int64(currentInstanceRow.VCpu),
				MemoryGb:      int64(currentInstanceRow.MemoryGb),
				Cost:          currentComputeCost + currentStorageCost,
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
		if rightSizedInstanceRow != nil {
			recommendedComputeCost, err := s.costSvc.GetRDSComputeCost(ctx, region, newInstance, metrics)
			if err != nil {
				s.logger.Error("failed to get rds instance cost", zap.Error(err))
				return nil, err
			}
			recommended.ComputeCost = recommendedComputeCost
		}

		recommendedStorageCost, err := s.costSvc.GetRDSStorageCost(ctx, region, newInstance, metrics)
		if err != nil {
			s.logger.Error("failed to get rds instance cost", zap.Error(err))
			return nil, err
		}
		recommended.StorageCost = recommendedStorageCost

		recommended.Cost = recommended.ComputeCost + recommended.StorageCost
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
		VolumeBytesUsed:        usageVolumeBytesUsed,

		Description: "",
	}

	if preferences["ExcludeUpsizingFeature"] != nil {
		if *preferences["ExcludeUpsizingFeature"] == "Yes" {
			if recommendation.Recommended != nil && recommendation.Recommended.Cost > recommendation.Current.Cost {
				recommendation.Recommended = &recommendation.Current
				recommendation.Description = "No recommendation available as upsizing feature is disabled"
				return &recommendation, nil
			}
		}
	}

	var computeDescription, storageDescription string
	if rightSizedInstanceRow != nil {
		computeDescription, err = s.generateRdsInstanceComputeDescription(rdsInstance, region, &currentInstanceRow,
			rightSizedInstanceRow, metrics, excluedBurstable, preferences, neededVCPU, neededMemoryGb, neededNetworkThroughput, usageAverageType)
		if err != nil {
			s.logger.Error("failed to generate rds instance compute description", zap.Error(err))
			return nil, err
		}
	}
	if rightSizedStorageRow != nil && recommended != nil {
		storageDescription, err = s.generateRdsInstanceStorageDescription(rdsInstance, region,
			*rdsInstance.StorageType, rdsInstance.StorageSize, rdsInstance.StorageIops, rdsInstance.StorageThroughput,
			*recommended.StorageType, recommended.StorageSize, recommended.StorageIops, recommended.StorageThroughput, metrics,
			preferences, neededStorageSize, neededStorageIops, neededStorageThroughputMB, usageAverageType)
	}

	recommendation.Description = computeDescription + "\n" + storageDescription
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

func (s *Service) generateRdsInstanceComputeDescription(rdsInstance entity.AwsRds, region string, currentInstanceType,
	rightSizedInstanceType *model.RDSDBInstance, metrics map[string][]types2.Datapoint, excludeBurstable bool,
	preferences map[string]*string, neededCPU, neededMemory, neededNetworkThroughput float64, usageAverageType UsageAverageType) (string, error) {
	usageCpuPercent := extractUsage(metrics["CPUUtilization"], usageAverageType)
	usageFreeMemoryBytes := extractUsage(metrics["FreeableMemory"], usageAverageType)
	usageNetworkThroughputBytes := extractUsage(sumMergeDatapoints(metrics["NetworkReceiveThroughput"], metrics["NetworkTransmitThroughput"]), usageAverageType)

	usage := fmt.Sprintf("- %s has %.1f vCPUs. Usage over the course of last week is ", currentInstanceType.InstanceType, currentInstanceType.VCpu)
	if usageCpuPercent.Min == nil && usageCpuPercent.Avg == nil && usageCpuPercent.Max == nil {
		usage += "not available."
	} else {
		if usageCpuPercent.Min != nil {
			usage += fmt.Sprintf("min=%.2f%%, ", *usageCpuPercent.Min)
		}
		if usageCpuPercent.Avg != nil {
			usage += fmt.Sprintf("avg=%.2f%%, ", *usageCpuPercent.Avg)
		}
		if usageCpuPercent.Max != nil {
			usage += fmt.Sprintf("max=%.2f%%, ", *usageCpuPercent.Max)
		}
		usage += fmt.Sprintf("so you only need %.1f vCPUs. %s has %d vCPUs.\n", neededCPU, rightSizedInstanceType.InstanceType, int32(rightSizedInstanceType.VCpu))
	}

	usage += fmt.Sprintf("- %s has %.1fGB Memory. Free Memory over the course of last week is ", currentInstanceType.InstanceType, currentInstanceType.MemoryGb)
	if usageFreeMemoryBytes.Min == nil && usageFreeMemoryBytes.Avg == nil && usageFreeMemoryBytes.Max == nil {
		usage += "not available."
	} else {
		if usageFreeMemoryBytes.Min != nil {
			usage += fmt.Sprintf("min=%.2fGB, ", *usageFreeMemoryBytes.Min/(1024.0*1024.0*1024.0))
		}
		if usageFreeMemoryBytes.Avg != nil {
			usage += fmt.Sprintf("avg=%.2fGB, ", *usageFreeMemoryBytes.Avg/(1024.0*1024.0*1024.0))
		}
		if usageFreeMemoryBytes.Max != nil {
			usage += fmt.Sprintf("max=%.2fGB, ", *usageFreeMemoryBytes.Max/(1024.0*1024.0*1024.0))
		}
		usage += fmt.Sprintf("so you only need %.1fGB Memory. %s has %.1fGB Memory.\n", neededMemory, rightSizedInstanceType.InstanceType, rightSizedInstanceType.MemoryGb)
	}

	usage += fmt.Sprintf("- %s's network performance is %s. Throughput over the course of last week is ", currentInstanceType.InstanceType, currentInstanceType.NetworkPerformance)
	if usageNetworkThroughputBytes.Min == nil && usageNetworkThroughputBytes.Avg == nil && usageNetworkThroughputBytes.Max == nil {
		usage += "not available."
	} else {
		if usageNetworkThroughputBytes.Min != nil {
			usage += fmt.Sprintf("min=%.2fMB, ", *usageNetworkThroughputBytes.Min/(1024*1024))
		}
		if usageNetworkThroughputBytes.Avg != nil {
			usage += fmt.Sprintf("avg=%.2fMB, ", *usageNetworkThroughputBytes.Avg/(1024*1024))
		}
		if usageNetworkThroughputBytes.Max != nil {
			usage += fmt.Sprintf("max=%.2fMB, ", *usageNetworkThroughputBytes.Max/(1024*1024))
		}
		usage += fmt.Sprintf("so you only need %.2fMB Throughput. %s has %s Throughput.\n", neededNetworkThroughput/(1024.0*1024.0), rightSizedInstanceType.InstanceType, rightSizedInstanceType.NetworkPerformance)
	}

	needs := ""
	for k, v := range preferences {
		if _, ok := aws_rds.PreferenceInstanceDBKey[k]; !ok {
			continue
		}
		if aws_rds.PreferenceInstanceDBKey[k] == "" {
			continue
		}
		if v == nil {
			vl := extractFromRdsInstance(rdsInstance, *currentInstanceType, region, k)
			needs += fmt.Sprintf("- You asked %s to be same as the current instance value which is %v\n", k, vl)
		} else {
			needs += fmt.Sprintf("- You asked %s to be %s\n", k, *v)
		}
	}

	prompt := fmt.Sprintf(`
I'm giving recommendation on aws rds db instance right sizing. Based on user's usage and needs I have concluded that the best option for them is to use %s instead of %s. I need help summarizing the explanation into 280 characters (it's not a tweet! dont use hashtag!) while keeping these rules:
- mention the requirements from user side.
- for those fields which are changing make sure you mention the change.

Here's usage data:
%s

User's needs:
%s
`, rightSizedInstanceType.InstanceType, currentInstanceType.InstanceType, usage, needs)
	if excludeBurstable {
		prompt += "\nBurstable instances are excluded."
	}
	resp, err := s.openaiSvc.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: openai.GPT4TurboPreview,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleUser,
					Content: prompt,
				},
			},
		},
	)

	if err != nil {
		return "", err
	}

	if len(resp.Choices) == 0 {
		return "", errors.New("empty choices")
	}
	return strings.TrimSpace(resp.Choices[0].Message.Content), nil
}

func (s *Service) generateRdsInstanceStorageDescription(rdsInstance entity.AwsRds, region string,
	currStorageType string, currStorageSize *int32, currStorageIops *int32, currStorageThroughput *float64,
	recStorageType string, recStorageSize *int32, recStorageIops *int32, recStorageThroughput *float64, metrics map[string][]types2.Datapoint,
	preferences map[string]*string, neededStorageSize int32, neededStorageIops int32, neededStorageThroughputMB float64, usageAverageType UsageAverageType) (string, error) {
	usageFreeStorageBytes := extractUsage(metrics["FreeStorageSpace"], usageAverageType)
	usageStorageIops := extractUsage(sumMergeDatapoints(metrics["ReadIOPS"], metrics["WriteIOPS"]), usageAverageType)
	usageStorageThroughputBytes := extractUsage(sumMergeDatapoints(metrics["ReadThroughput"], metrics["WriteThroughput"]), usageAverageType)
	usageStorageThroughputMB := entity.Usage{
		Avg: funcP(usageStorageThroughputBytes.Avg, usageStorageThroughputBytes.Avg, func(a, _ float64) float64 { return a / (1024 * 1024) }),
		Min: funcP(usageStorageThroughputBytes.Min, usageStorageThroughputBytes.Min, func(a, _ float64) float64 { return a / (1024 * 1024) }),
		Max: funcP(usageStorageThroughputBytes.Max, usageStorageThroughputBytes.Max, func(a, _ float64) float64 { return a / (1024 * 1024) }),
	}
	usageVolumeBytesUsed := extractUsage(metrics["VolumeBytesUsed"], usageAverageType)

	var usage string
	if strings.Contains(strings.ToLower(rdsInstance.Engine), "aurora") {
		if currStorageSize != nil && recStorageSize != nil && *currStorageSize != 0 && *recStorageSize != 0 {
			usage += fmt.Sprintf("- %s has %dGB Storage. Usage over the course of last week is ", currStorageType, *currStorageSize)
			if usageFreeStorageBytes.Max != nil {
				usage += fmt.Sprintf("min=%.2fGB, ", float64(*currStorageSize)-*usageFreeStorageBytes.Max/(1024*1024*1024))
			}
			if usageFreeStorageBytes.Avg != nil {
				usage += fmt.Sprintf("avg=%.2fGB, ", float64(*currStorageSize)-*usageFreeStorageBytes.Avg/(1024*1024*1024))
			}
			if usageFreeStorageBytes.Min != nil {
				usage += fmt.Sprintf("max=%.2fGB, ", float64(*currStorageSize)-*usageFreeStorageBytes.Min/(1024*1024*1024))
			}
			usage += fmt.Sprintf("so you only need %dGB Storage. %s has %dGB Storage.\n", neededStorageSize, recStorageType, recStorageSize)
		}
	} else {
		if currStorageSize != nil && recStorageSize != nil && *currStorageSize != 0 && *recStorageSize != 0 {
			usage += fmt.Sprintf("- %s has %dGB Storage. Usage over the course of last week is ", currStorageType, *currStorageSize)
			if usageVolumeBytesUsed.Min != nil {
				usage += fmt.Sprintf("min=%.2fGB, ", *usageVolumeBytesUsed.Min/(1024*1024*1024))
			}
			if usageVolumeBytesUsed.Avg != nil {
				usage += fmt.Sprintf("avg=%.2fGB, ", *usageVolumeBytesUsed.Avg/(1024*1024*1024))
			}
			if usageVolumeBytesUsed.Max != nil {
				usage += fmt.Sprintf("max=%.2fGB, ", *usageVolumeBytesUsed.Max/(1024*1024*1024))
			}
			usage += fmt.Sprintf("so you only need %dGB Storage. %s has %dGB Storage.\n", neededStorageSize, recStorageType, recStorageSize)
		}
	}
	if currStorageIops != nil && recStorageIops != nil && *currStorageIops != 0 && *recStorageIops != 0 {
		if getValueOrZero(usageStorageIops.Min) == 0 && getValueOrZero(usageStorageIops.Avg) == 0 && getValueOrZero(usageStorageIops.Max) == 0 {
			usage += fmt.Sprintf("- %s has %d IOPS. Usage over the course of last week is ", currStorageType, getValueOrZero(currStorageIops))
			if usageStorageIops.Min != nil {
				usage += fmt.Sprintf("min=%.2f, ", *usageStorageIops.Min)
			}
			if usageStorageIops.Avg != nil {
				usage += fmt.Sprintf("avg=%.2f, ", *usageStorageIops.Avg)
			}
			if usageStorageIops.Max != nil {
				usage += fmt.Sprintf("max=%.2f, ", *usageStorageIops.Max)
			}
			usage += fmt.Sprintf("so you only need %d io/s. %s has %d IOPS.\n", neededStorageIops, recStorageType, recStorageIops)
		} else {
			usage += fmt.Sprintf("- %s has %d IOPS. Usage data is not available. you need %d io/s. %s has %d IOPS.\n", currStorageType, getValueOrZero(currStorageIops), neededStorageIops, recStorageType, recStorageIops)
		}
	}
	if currStorageThroughput != nil && recStorageThroughput != nil && *currStorageThroughput != 0 && *recStorageThroughput != 0 {
		if getValueOrZero(usageStorageThroughputMB.Min) == 0 && getValueOrZero(usageStorageThroughputMB.Avg) == 0 && getValueOrZero(usageStorageThroughputMB.Max) == 0 {
			usage += fmt.Sprintf("- %s has %.1fMB Throughput. Usage over the course of last week is ", currStorageType, *currStorageThroughput)
			if usageStorageThroughputMB.Min != nil {
				usage += fmt.Sprintf("min=%.2fMB, ", *usageStorageThroughputMB.Min)
			}
			if usageStorageThroughputMB.Avg != nil {
				usage += fmt.Sprintf("avg=%.2fMB, ", *usageStorageThroughputMB.Avg)
			}
			if usageStorageThroughputMB.Max != nil {
				usage += fmt.Sprintf("max=%.2fMB, ", *usageStorageThroughputMB.Max)
			}
			usage += fmt.Sprintf("so you only need %.2f MB. %s has %.2fMB Throughput.\n", neededStorageThroughputMB, recStorageType, recStorageThroughput)
		} else {
			usage += fmt.Sprintf("- %s has %.1fMB Throughput. Usage data is not available. you only need %.2f MB. %s has %.2fMB Throughput.\n", currStorageType, getValueOrZero(currStorageThroughput), neededStorageThroughputMB, recStorageType, getValueOrZero(recStorageThroughput))
		}
	}

	needs := ""
	for k, v := range preferences {
		if _, ok := aws_rds.PreferenceStorageDBKey[k]; !ok {
			continue
		}
		if aws_rds.PreferenceStorageDBKey[k] == "" {
			continue
		}
		if v == nil {
			vl := extractFromRdsInstance(rdsInstance, model.RDSDBInstance{}, region, k)
			needs += fmt.Sprintf("- You asked %s to be same as the current instance value which is %v\n", k, vl)
		} else {
			needs += fmt.Sprintf("- You asked %s to be %s\n", k, *v)
		}
	}

	prompt := fmt.Sprintf(`
I'm giving recommendation on aws rds db instance storage right sizing. Based on user's usage and needs I have concluded that the best option for them is to use %s instead of %s. I need help summarizing the explanation into 280 characters (it's not a tweet! dont use hashtag!) while keeping these rules:
- mention the requirements from user side.
- for those fields which are changing make sure you mention the change.

Here's usage data:
%s

User's needs:
%s
`, recStorageType, currStorageType, usage, needs)
	resp, err := s.openaiSvc.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: openai.GPT4TurboPreview,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleUser,
					Content: prompt,
				},
			},
		},
	)

	if err != nil {
		return "", err
	}

	if len(resp.Choices) == 0 {
		return "", errors.New("empty choices")
	}
	return strings.TrimSpace(resp.Choices[0].Message.Content), nil
}
