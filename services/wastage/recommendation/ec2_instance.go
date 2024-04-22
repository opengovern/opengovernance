package recommendation

import (
	"fmt"
	types2 "github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/kaytu-io/kaytu-engine/services/wastage/api/entity"
	"github.com/kaytu-io/kaytu-engine/services/wastage/db/model"
	"strconv"
	"strings"
)

func maxOfDatapoints(datapoints []types2.Datapoint) float64 {
	if len(datapoints) == 0 {
		return 0.0
	}

	avg := float64(0)
	for _, dp := range datapoints {
		if dp.Maximum == nil {
			if dp.Average == nil {
				continue
			}
			avg += *dp.Average
			continue
		}
		avg += *dp.Maximum
	}
	avg = avg / float64(len(datapoints))
	return avg
}

func averageOfDatapoints(datapoints []types2.Datapoint) float64 {
	if len(datapoints) == 0 {
		return 0.0
	}

	avg := float64(0)
	for _, dp := range datapoints {
		if dp.Average == nil {
			continue
		}
		avg += *dp.Average
	}
	avg = avg / float64(len(datapoints))
	return avg
}

func (s *Service) EC2InstanceRecommendation(region string, instance entity.EC2Instance, volumes []entity.EC2Volume, metrics map[string][]types2.Datapoint, preferences map[string]*string) (*Ec2InstanceRecommendation, error) {
	averageCPUUtilization := averageOfDatapoints(metrics["CPUUtilization"])
	averageNetworkIn := averageOfDatapoints(metrics["NetworkIn"])
	averageNetworkOut := averageOfDatapoints(metrics["NetworkOut"])
	maxMemPercent := maxOfDatapoints(metrics["mem_used_percent"])
	maxMemUsagePercentage := "NA"
	if len(metrics["mem_used_percent"]) > 0 {
		maxMemUsagePercentage = fmt.Sprintf("Max: %.1f%%", maxMemPercent)
	}

	i, err := s.ec2InstanceRepo.ListByInstanceType(string(instance.InstanceType))
	if err != nil {
		return nil, err
	}
	if len(i) == 0 {
		return nil, fmt.Errorf("instance type not found: %s", string(instance.InstanceType))
	}

	//TODO Burst in CPU & Network
	//TODO Network: UpTo

	vCPU := instance.ThreadsPerCore * instance.CoreCount
	cpuBreathingRoom := int64(0)
	if preferences["CPUBreathingRoom"] != nil {
		cpuBreathingRoom, _ = strconv.ParseInt(*preferences["CPUBreathingRoom"], 10, 64)
	}
	memoryBreathingRoom := int64(0)
	if preferences["MemoryBreathingRoom"] != nil {
		memoryBreathingRoom, _ = strconv.ParseInt(*preferences["MemoryBreathingRoom"], 10, 64)
	}
	neededCPU := float64(vCPU) * (averageCPUUtilization + float64(cpuBreathingRoom)) / 100.0
	neededMemory := float64(i[0].MemoryGB) * (maxMemPercent + float64(memoryBreathingRoom)) / 100.0
	neededNetworkThroughput := averageNetworkIn + averageNetworkOut
	if preferences["NetworkBreathingRoom"] != nil {
		room, _ := strconv.ParseInt(*preferences["NetworkBreathingRoom"], 10, 64)
		neededNetworkThroughput += neededNetworkThroughput * float64(room) / 100.0
	}

	pref := map[string]any{}
	for k, v := range preferences {
		var vl any
		if v == nil {
			vl = extractFromInstance(instance, i[0], region, k)
		} else {
			vl = *v
		}
		if PreferenceDBKey[k] == "" {
			continue
		}

		cond := "="
		if sc, ok := PreferenceSpecialCond[k]; ok {
			cond = sc
		}
		pref[fmt.Sprintf("%s %s ?", PreferenceDBKey[k], cond)] = vl
	}
	if _, ok := preferences["vCPU"]; !ok {
		pref["v_cpu >= ?"] = neededCPU
	}
	if _, ok := metrics["mem_used_percent"]; ok {
		if _, ok := preferences["MemoryGB"]; !ok {
			pref["memory_gb >= ?"] = neededMemory
		}
	}
	//os := "Linux"
	//if instance.Platform != "" {
	//	os = string(instance.Platform)
	//}
	//currInstanceType, err := s.ec2InstanceRepo.GetCurrentInstanceType(string(instance.InstanceType), string(instance.Placement.Tenancy), os)

	instanceType, err := s.ec2InstanceRepo.GetCheapestByCoreAndNetwork(neededNetworkThroughput, pref)
	if err != nil {
		return nil, err
	}

	if instanceType != nil {
		description := fmt.Sprintf("change your vms from %s to %s", instance.InstanceType, instanceType.InstanceType)
		instance.InstanceType = types.InstanceType(instanceType.InstanceType)
		//if instanceType.OperatingSystem == "Windows" {
		//	instance.Platform = types.PlatformValuesWindows
		//} else {
		//	instance.Platform = ""
		//}
		return &Ec2InstanceRecommendation{
			Description:              description,
			NewInstance:              instance,
			NewVolumes:               volumes,
			CurrentInstanceType:      &i[0],
			NewInstanceType:          instanceType,
			AvgNetworkBandwidth:      fmt.Sprintf("Avg: %.1f Megabit", (averageNetworkOut+averageNetworkIn)/1000000.0*8.0),
			AvgCPUUsage:              fmt.Sprintf("Avg: %.1f%%", averageCPUUtilization),
			MaxMemoryUsagePercentage: maxMemUsagePercentage,
		}, nil
	}
	return nil, nil
}

func extractFromInstance(instance entity.EC2Instance, i model.EC2InstanceType, region string, k string) any {
	switch k {
	case "Tenancy":
		return i.Tenancy
	case "EBSOptimized":
		if instance.EbsOptimized {
			return "Yes"
		} else {
			return "No"
		}
	case "OperatingSystem":
		return i.OperatingSystem
	case "LicenseModel":
		return i.LicenseModel
	case "Region":
		return region
	case "Hypervisor":
		return "" //TODO
	case "CurrentGeneration":
		return i.CurrentGeneration
	case "PhysicalProcessor":
		return i.PhysicalProcessor
	case "ClockSpeed":
		return i.ClockSpeed
	case "ProcessorArchitecture":
		return i.ProcessorArchitecture
	case "SupportedArchitectures":
		return "" //TODO
	case "ENASupported":
		return i.EnhancedNetworkingSupported
	case "EncryptionInTransitSupported":
		return "" //TODO
	case "SupportedRootDeviceTypes":
		return "" //TODO
	case "Cores":
		return "" //TODO
	case "Threads":
		return "" //TODO
	case "vCPU":
		return i.VCpu
	case "MemoryGB":
		return i.MemoryGB
	}
	return ""
}

func (s *Service) EBSVolumeRecommendation(region string, volume entity.EC2Volume, metrics map[string][]types2.Datapoint, preferences map[string]*string) (*EbsVolumeRecommendation, error) {
	averageIops := averageOfDatapoints(metrics["VolumeReadOps"]) + averageOfDatapoints(metrics["VolumeWriteOps"])
	averageThroughput := averageOfDatapoints(metrics["VolumeReadBytes"]) + averageOfDatapoints(metrics["VolumeWriteBytes"])

	size := int32(0)
	if volume.Size != nil {
		size = *volume.Size
	}

	result := &EbsVolumeRecommendation{
		Description:                  "",
		NewVolume:                    volume,
		CurrentSize:                  size,
		NewSize:                      size,
		CurrentProvisionedIOPS:       volume.Iops,
		NewProvisionedIOPS:           nil,
		CurrentProvisionedThroughput: volume.Throughput,
		NewProvisionedThroughput:     nil,
		CurrentVolumeType:            volume.VolumeType,
		NewVolumeType:                "",
		AvgIOPS:                      averageIops,
		AvgThroughput:                averageThroughput / 1000000.0,
	}

	newType, err := s.ebsVolumeRepo.GetMinimumVolumeTotalPrice(region, size, int32(averageIops), int32(averageThroughput/1000000.0))
	if err != nil {
		if strings.Contains(err.Error(), "no feasible volume types found") {
			return nil, nil
		}
		return nil, err
	}

	hasResult := false

	if newType != volume.VolumeType {
		hasResult = true
		result.NewVolumeType = newType
		result.NewVolume.VolumeType = newType
		result.Description = fmt.Sprintf("- change your volume from %s to %s\n", volume.VolumeType, newType)
	}

	if newType == types.VolumeTypeIo1 || newType == types.VolumeTypeIo2 {
		avgIOps := int32(averageIops)
		if volume.Iops == nil {
			hasResult = true
			result.NewProvisionedIOPS = &avgIOps
			result.NewVolume.Iops = &avgIOps
			result.Description += fmt.Sprintf("- add provisioned iops: %d\n", avgIOps)
		} else if avgIOps > *volume.Iops {
			hasResult = true
			result.NewProvisionedIOPS = &avgIOps
			result.NewVolume.Iops = &avgIOps
			result.Description += fmt.Sprintf("- increase provisioned iops from %d to %d\n", *volume.Iops, avgIOps)
		} else if avgIOps < *volume.Iops {
			hasResult = true
			result.NewProvisionedIOPS = &avgIOps
			result.NewVolume.Iops = &avgIOps
			result.Description += fmt.Sprintf("- decrease provisioned iops from %d to %d\n", *volume.Iops, avgIOps)
		}
	}

	if newType == types.VolumeTypeGp3 && averageIops > model.Gp3BaseIops {
		provIops := int32(averageIops) - model.Gp3BaseIops
		if volume.Iops == nil {
			hasResult = true
			result.NewProvisionedIOPS = &provIops
			result.NewVolume.Iops = &provIops
			result.Description += fmt.Sprintf("- add provisioned iops: %d\n", provIops)
		} else if provIops > *volume.Iops {
			hasResult = true
			result.NewProvisionedIOPS = &provIops
			result.NewVolume.Iops = &provIops
			result.Description += fmt.Sprintf("- increase provisioned iops from %d to %d\n", *volume.Iops, provIops)
		} else if provIops < *volume.Iops {
			hasResult = true
			result.NewProvisionedIOPS = &provIops
			result.NewVolume.Iops = &provIops
			result.Description += fmt.Sprintf("- decrease provisioned iops from %d to %d\n", *volume.Iops, provIops)
		}
	}

	if newType == types.VolumeTypeGp3 && averageThroughput > model.Gp3BaseThroughput {
		provThroughput := int32(averageThroughput) - model.Gp3BaseThroughput
		if volume.Throughput == nil {
			hasResult = true
			result.NewProvisionedThroughput = &provThroughput
			result.NewVolume.Throughput = &provThroughput
			result.Description += fmt.Sprintf("- add provisioned throughput: %d\n", provThroughput)
		} else if provThroughput > *volume.Throughput {
			hasResult = true
			result.NewProvisionedThroughput = &provThroughput
			result.NewVolume.Throughput = &provThroughput
			result.Description += fmt.Sprintf("- increase provisioned throughput from %d to %d\n", *volume.Throughput, provThroughput)
		} else if provThroughput < *volume.Throughput {
			hasResult = true
			result.NewProvisionedThroughput = &provThroughput
			result.NewVolume.Throughput = &provThroughput
			result.Description += fmt.Sprintf("- decrease provisioned throughput from %d to %d\n", *volume.Throughput, provThroughput)
		}
	}

	if !hasResult {
		return nil, nil
	}

	return result, nil
}