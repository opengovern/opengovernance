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
	"github.com/sashabaranov/go-openai"
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
			neededStorageSizeFloat = math.Ceil((1 + float64(vPercent)/100) * neededStorageSizeFloat)
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
			neededStorageIopsFloat = math.Ceil((1 + float64(vPercent)/100) * neededStorageIopsFloat)
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
		} else {
			if !strings.HasPrefix(rdsInstance.InstanceType, "db.t") {
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
				Engine:        awsRdsDbTypeToAPIDbType(currentInstanceRow.DatabaseEngine, currentInstanceRow.DatabaseEdition),
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
		fmt.Println("New Instance", newInstance)
		fmt.Println("New Instance Storage Type", *newInstance.StorageType)
		fmt.Println("New Instance Storage Size", *newInstance.StorageSize)
		fmt.Println("New Instance Storage IOPS", *newInstance.StorageIops)
		fmt.Println("New Instance Storage Throughput", *newInstance.StorageThroughput)
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

	var computeDescription, storageDescription string
	if rightSizedInstanceRow != nil {
		computeDescription, err = s.generateRdsInstanceComputeDescription(rdsInstance, region, &currentInstanceRow,
			rightSizedInstanceRow, metrics, preferences, neededVCPU, neededMemoryGb, neededNetworkThroughput, usageAverageType)
		if err != nil {
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
	rightSizedInstanceType *model.RDSDBInstance, metrics map[string][]types2.Datapoint,
	preferences map[string]*string, neededCPU, neededMemory, neededNetworkThroughput float64, usageAverageType UsageAverageType) (string, error) {
	usageCpuPercent := extractUsage(metrics["CPUUtilization"], usageAverageType)
	usageFreeMemoryBytes := extractUsage(metrics["FreeableMemory"], usageAverageType)
	usageNetworkThroughputBytes := extractUsage(sumMergeDatapoints(metrics["NetworkReceiveThroughput"], metrics["NetworkTransmitThroughput"]), usageAverageType)

	usage := fmt.Sprintf("- %s has %.1f vCPUs. Usage over the course of last week is min=%.2f%%, avg=%.2f%%, max=%.2f%%, so you only need %.1f vCPUs. %s has %d vCPUs.\n", currentInstanceType.InstanceType, currentInstanceType.VCpu, *usageCpuPercent.Min, *usageCpuPercent.Avg, *usageCpuPercent.Max, neededCPU, rightSizedInstanceType.InstanceType, int32(rightSizedInstanceType.VCpu))
	usage += fmt.Sprintf("- %s has %.1fGB Memory. Free Memory over the course of last week is min=%.2fGB, avg=%.2fGB, max=%.2fGB, so you only need %.1fGB Memory. %s has %.1fGB Memory.\n", currentInstanceType.InstanceType, currentInstanceType.MemoryGb, *usageFreeMemoryBytes.Min/(1024.0*1024.0*1024.0), *usageFreeMemoryBytes.Avg/(1024.0*1024.0*1024.0), *usageFreeMemoryBytes.Max/(1024.0*1024.0*1024.0), neededMemory, rightSizedInstanceType.InstanceType, rightSizedInstanceType.MemoryGb)

	usage += fmt.Sprintf("- %s's network performance is %s. Throughput over the course of last week is min=%.2f MB/s, avg=%.2f MB/s, max=%.2f MB/s, so you only need %.2f MB/s. %s has %s.\n", currentInstanceType.InstanceType, currentInstanceType.NetworkPerformance, *usageNetworkThroughputBytes.Min/(1024.0*1024.0), *usageNetworkThroughputBytes.Avg/(1024*1024), *usageNetworkThroughputBytes.Max/(1024.0*1024.0), neededNetworkThroughput/(1024.0*1024.0), rightSizedInstanceType.InstanceType, rightSizedInstanceType.NetworkPerformance)

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
I'm giving recommendation on aws rds db instance right sizing. Based on user's usage and needs I have concluded that the best option for them is to use %s instead of %s. I need help summarizing the explanation into 3 lines while keeping these rules:
- mention the requirements from user side.
- for those fields which are changing make sure you mention the change.

Here's usage data:
%s

User's needs:
%s
`, rightSizedInstanceType.InstanceType, currentInstanceType.InstanceType, usage, needs)
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

	var usage string
	if currStorageSize != nil && recStorageSize != nil && *currStorageSize != 0 && *recStorageSize != 0 {
		usage += fmt.Sprintf("- %s has %dGB Storage. Usage over the course of last week is min=%.2fGB, avg=%.2fGB, max=%.2fGB, so you only need %dGB Storage. %s has %dGB Storage.\n", currStorageType, *currStorageSize, float64(*currStorageSize)-*usageFreeStorageBytes.Max/(1024*1024*1024), float64(*currStorageSize)-*usageFreeStorageBytes.Avg/(1024*1024*1024), float64(*currStorageSize)-*usageFreeStorageBytes.Min/(1024*1024*1024), neededStorageSize, recStorageType, recStorageSize)
	}
	if currStorageIops != nil && recStorageIops != nil && *currStorageIops != 0 && *recStorageIops != 0 {
		if *usageStorageIops.Min == 0 && *usageStorageIops.Avg == 0 && *usageStorageIops.Max == 0 {
			usage += fmt.Sprintf("- %s has %d IOPS. Usage over the course of last week is min=%.2f io/s, avg=%.2f io/s, max=%.2f io/s, so you only need %d io/s. %s has %d IOPS.\n", currStorageType, *currStorageIops, *usageStorageIops.Min, *usageStorageIops.Avg, *usageStorageIops.Max, neededStorageIops, recStorageType, recStorageIops)
		} else {
			usage += fmt.Sprintf("- %s has %d IOPS. Usage data is not available. you need %d io/s. %s has %d IOPS.\n", currStorageType, *currStorageIops, neededStorageIops, recStorageType, recStorageIops)
		}
	}
	if currStorageThroughput != nil && recStorageThroughput != nil && *currStorageThroughput != 0 && *recStorageThroughput != 0 {
		if *usageStorageThroughputMB.Min == 0 && *usageStorageThroughputMB.Avg == 0 && *usageStorageThroughputMB.Max == 0 {
			usage += fmt.Sprintf("- %s has %.1fMB Throughput. Usage over the course of last week is min=%.2fMB, avg=%.2fMB, max=%.2fMB, so you only need %.2f MB. %s has %.2fMB Throughput.\n", currStorageType, *currStorageThroughput, *usageStorageThroughputMB.Min, *usageStorageThroughputMB.Avg, *usageStorageThroughputMB.Max, neededStorageThroughputMB, recStorageType, *recStorageThroughput)
		} else {
			usage += fmt.Sprintf("- %s has %.1fMB Throughput. Usage data is not available. you only need %.2f MB. %s has %.2fMB Throughput.\n", currStorageType, *currStorageThroughput, neededStorageThroughputMB, recStorageType, *recStorageThroughput)
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
I'm giving recommendation on aws rds db instance storage right sizing. Based on user's usage and needs I have concluded that the best option for them is to use %s instead of %s. I need help summarizing the explanation into 3 lines while keeping these rules:
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
