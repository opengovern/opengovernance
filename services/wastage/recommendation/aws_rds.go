package recommendation

import (
	"fmt"
	types2 "github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/kaytu-io/kaytu-engine/services/wastage/api/entity"
	"github.com/kaytu-io/kaytu-engine/services/wastage/db/model"
	"github.com/kaytu-io/kaytu-engine/services/wastage/recommendation/preferences/aws_rds"
	"strconv"
	"strings"
)

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

	currentInstanceTypeList, err := s.awsRDSDBInstanceRepo.ListByInstanceType(region, rdsInstance.InstanceType, "", "")
	if err != nil {
		return nil, err
	}
	if len(currentInstanceTypeList) == 0 {
		return nil, fmt.Errorf("rds instance type not found: %s", rdsInstance.InstanceType)
	}
	currentInstanceRow := currentInstanceTypeList[0]

	// TODO get current cost

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
		Cost:              0,
	}

	neededVCPU := *usageCpuPercent.Avg * float64(currentInstanceRow.VCpu)
	if v, ok := preferences["CpuBreathingRoom"]; ok {
		vPercent, err := strconv.ParseInt(*v, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid CpuBreathingRoom value: %s", *v)
		}
		neededVCPU = (1 + float64(vPercent)/100) * neededVCPU
	}
	neededMemoryGb := float64(currentInstanceRow.MemoryGb) - (*usageFreeMemoryBytes.Avg / 1e9)
	if v, ok := preferences["MemoryBreathingRoom"]; ok {
		vPercent, err := strconv.ParseInt(*v, 10, 64)
		if err != nil {
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
			return nil, fmt.Errorf("invalid NetworkBreathingRoom value: %s", *v)
		}
		neededNetworkThroughput = (1 + float64(vPercent)/100) * neededNetworkThroughput
	}
	neededStorageSize := float64(*rdsInstance.StorageSize) - (*usageFreeStorageBytes.Avg / 1e9)
	if v, ok := preferences["StorageSizeBreathingRoom"]; ok {
		vPercent, err := strconv.ParseInt(*v, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid StorageBreathingRoom value: %s", *v)
		}
		neededStorageSize = (1 + float64(vPercent)/100) * neededStorageSize
	}
	neededStorageIops := float64(*rdsInstance.StorageIops) - *usageStorageIops.Avg
	if v, ok := preferences["StorageIopsBreathingRoom"]; ok {
		vPercent, err := strconv.ParseInt(*v, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid StorageIopsBreathingRoom value: %s", *v)
		}
		neededStorageIops = (1 + float64(vPercent)/100) * neededStorageIops
	}
	neededStorageThroughput := float64(*rdsInstance.StorageThroughput) - *usageStorageThroughputBytes.Avg
	if v, ok := preferences["StorageThroughputBreathingRoom"]; ok {
		vPercent, err := strconv.ParseInt(*v, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid StorageThroughputBreathingRoom value: %s", *v)
		}
		neededStorageThroughput = (1 + float64(vPercent)/100) * neededStorageThroughput
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

	rightSizedInstanceRow, err := s.awsRDSDBInstanceRepo.GetCheapestByPref(instancePref)
	if err != nil {
		return nil, err
	}

	// Aurora instances storage preferences are very different from other RDS instances
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
		newInstance.Engine = rightSizedInstanceRow.DatabaseEngine

		// TODO get new cost

		recommended = &entity.RightsizingAwsRds{
			Region:            rightSizedInstanceRow.RegionCode,
			InstanceType:      rightSizedInstanceRow.InstanceType,
			Engine:            rightSizedInstanceRow.DatabaseEngine,
			EngineVersion:     newInstance.EngineVersion,
			ClusterType:       newInstance.ClusterType,
			VCPU:              rightSizedInstanceRow.VCpu,
			MemoryGb:          rightSizedInstanceRow.MemoryGb,
			StorageType:       nil,
			StorageSize:       nil,
			StorageIops:       nil,
			StorageThroughput: nil,
			Cost:              0,
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
