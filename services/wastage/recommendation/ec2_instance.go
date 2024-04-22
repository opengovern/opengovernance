package recommendation

import (
	"fmt"
	types2 "github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/kaytu-io/kaytu-engine/pkg/utils"
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

func (s *Service) EC2InstanceRecommendation(region string, instance entity.EC2Instance, volumes []entity.EC2Volume, metrics map[string][]types2.Datapoint, volumeMetrics map[string]map[string][]types2.Datapoint, preferences map[string]*string) (*Ec2InstanceRecommendation, error) {
	averageCPUUtilization := averageOfDatapoints(metrics["CPUUtilization"])
	averageNetworkIn := averageOfDatapoints(metrics["NetworkIn"])
	averageNetworkOut := averageOfDatapoints(metrics["NetworkOut"])
	averageEBSIn, averageEBSOut := 0.0, 0.0
	for _, v := range volumeMetrics {
		readBytesAvg, writeBytesAvg := averageOfDatapoints(v["VolumeReadBytes"]), averageOfDatapoints(v["VolumeWriteBytes"])
		averageEBSIn += readBytesAvg
		averageEBSOut += writeBytesAvg
	}
	averageEBSIn = averageEBSIn / float64(len(volumeMetrics))
	averageEBSOut = averageEBSOut / float64(len(volumeMetrics))

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

	instanceType, err := s.ec2InstanceRepo.GetCheapestByCoreAndNetwork(neededNetworkThroughput, pref)
	if err != nil {
		return nil, err
	}

	if instanceType != nil {
		description := fmt.Sprintf("change your vms from %s to %s", instance.InstanceType, instanceType.InstanceType)
		instance.InstanceType = types.InstanceType(instanceType.InstanceType)
		if instanceType.OperatingSystem == "Windows" {
			instance.Platform = types.PlatformValuesWindows
		} else {
			instance.Platform = ""
		}
		return &Ec2InstanceRecommendation{
			Description:              description,
			NewInstance:              instance,
			NewVolumes:               volumes,
			CurrentInstanceType:      &i[0],
			NewInstanceType:          instanceType,
			AvgNetworkBandwidth:      fmt.Sprintf("Avg: %.1f Megabit", (averageNetworkOut+averageNetworkIn)/1000000.0*8.0),
			AvgEBSBandwidth:          fmt.Sprintf("Avg: %.1f Megabit", (averageEBSOut+averageEBSIn)/1000000.0*8.0),
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
	averageThroughput = averageThroughput / 1000000.0
	size := float64(0)
	if volume.Size != nil {
		size = float64(*volume.Size)
	}

	iopsBreathingRoom := int64(0)
	if preferences["IOPSBreathingRoom"] != nil {
		iopsBreathingRoom, _ = strconv.ParseInt(*preferences["IOPSBreathingRoom"], 10, 32)
	}
	throughputBreathingRoom := int64(0)
	if preferences["ThroughputBreathingRoom"] != nil {
		throughputBreathingRoom, _ = strconv.ParseInt(*preferences["ThroughputBreathingRoom"], 10, 32)
	}
	neededIops := averageIops * (1 + float64(iopsBreathingRoom)/100.0)
	neededThroughput := averageThroughput * (1 + float64(throughputBreathingRoom)/100.0)
	neededSize := size
	var validTypes []types.VolumeType
	if v, ok := preferences["IOPS"]; ok {
		if v == nil && volume.Iops != nil {
			neededIops = float64(*volume.Iops)
		} else {
			neededIops, _ = strconv.ParseFloat(*v, 64)
		}
	}
	if v, ok := preferences["Throughput"]; ok {
		if v == nil && volume.Throughput != nil {
			neededThroughput = *volume.Throughput
		} else {
			neededThroughput, _ = strconv.ParseFloat(*v, 64)
		}
	}
	if v, ok := preferences["Size"]; ok {
		if v == nil && volume.Size != nil {
			neededSize = float64(*volume.Size)
		} else {
			neededSize, _ = strconv.ParseFloat(*v, 64)
		}
	} else if volume.Size != nil {
		neededSize = float64(*volume.Size)
	}

	if v, ok := preferences["VolumeFamily"]; ok {
		if preferences["VolumeFamily"] == nil {
			validTypes = []types.VolumeType{volume.VolumeType}
		} else {
			switch strings.ToLower(*v) {
			case "general purpose", "ssd", "solid state drive", "gp":
				validTypes = []types.VolumeType{types.VolumeTypeGp2, types.VolumeTypeGp3}
			case "io", "io optimized":
				validTypes = []types.VolumeType{types.VolumeTypeIo1, types.VolumeTypeIo2}
			case "hdd", "sc", "cold", "hard disk drive", "st":
				validTypes = []types.VolumeType{types.VolumeTypeSc1, types.VolumeTypeSt1}
			}
		}
	}

	if v, ok := preferences["VolumeType"]; ok {
		if preferences["VolumeType"] == nil {
			validTypes = []types.VolumeType{volume.VolumeType}
		} else {
			validTypes = []types.VolumeType{types.VolumeType(*v)}
		}
	}

	result := &EbsVolumeRecommendation{
		Description:                  "",
		NewVolume:                    volume,
		CurrentSize:                  int32(size),
		NewSize:                      int32(size),
		CurrentProvisionedIOPS:       volume.Iops,
		NewProvisionedIOPS:           nil,
		CurrentProvisionedThroughput: volume.Throughput,
		NewProvisionedThroughput:     nil,
		CurrentVolumeType:            volume.VolumeType,
		NewVolumeType:                "",
		AvgIOPS:                      averageIops,
		AvgThroughput:                averageThroughput,
	}

	newType, err := s.ebsVolumeRepo.GetMinimumVolumeTotalPrice(region, int32(neededSize), int32(neededIops), neededThroughput, validTypes)
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

	if int32(neededSize) != *volume.Size {
		hasResult = true
		result.NewVolume.Size = utils.GetPointer(int32(neededSize))
		result.NewSize = int32(neededSize)
		result.Description += fmt.Sprintf("- change volume size from %d to %d\n", *volume.Size, int32(neededSize))
	}

	if newType == types.VolumeTypeIo1 || newType == types.VolumeTypeIo2 {
		avgIOps := int32(neededIops)
		hasResult = true
		result.NewProvisionedIOPS = &avgIOps
		result.NewVolume.Iops = &avgIOps

		if volume.Iops == nil {
			result.Description += fmt.Sprintf("- add provisioned iops: %d\n", avgIOps)
		} else if avgIOps > *volume.Iops {
			result.Description += fmt.Sprintf("- increase provisioned iops from %d to %d\n", *volume.Iops, avgIOps)
		} else if avgIOps < *volume.Iops {
			result.Description += fmt.Sprintf("- decrease provisioned iops from %d to %d\n", *volume.Iops, avgIOps)
		} else {
			result.NewProvisionedIOPS = nil
			result.NewVolume.Iops = volume.Iops
		}
	}

	if newType == types.VolumeTypeGp3 {
		provIops := max(int32(neededIops), model.Gp3BaseIops)
		hasResult = true
		result.NewProvisionedIOPS = &provIops
		result.NewVolume.Iops = &provIops

		if volume.Iops == nil {
			result.Description += fmt.Sprintf("- add provisioned iops: %d\n", provIops)
		} else if provIops > *volume.Iops {
			result.Description += fmt.Sprintf("- increase provisioned iops from %d to %d\n", *volume.Iops, provIops)
		} else if provIops < *volume.Iops {
			result.Description += fmt.Sprintf("- decrease provisioned iops from %d to %d\n", *volume.Iops, provIops)
		} else {
			result.NewProvisionedIOPS = nil
			result.NewVolume.Iops = volume.Iops
		}
	}

	if newType == types.VolumeTypeGp3 {
		provThroughput := max(neededThroughput, model.Gp3BaseThroughput)

		hasResult = true
		result.NewProvisionedThroughput = &provThroughput
		result.NewVolume.Throughput = &provThroughput

		if volume.Throughput == nil {
			result.Description += fmt.Sprintf("- add provisioned throughput: %.2f\n", provThroughput)
		} else if provThroughput > *volume.Throughput {
			result.Description += fmt.Sprintf("- increase provisioned throughput from %.2f to %.2f\n", *volume.Throughput, provThroughput)
		} else if provThroughput < *volume.Throughput {
			result.Description += fmt.Sprintf("- decrease provisioned throughput from %.2f to %.2f\n", *volume.Throughput, provThroughput)
		} else {
			result.NewProvisionedThroughput = nil
			result.NewVolume.Throughput = volume.Throughput
		}
	}

	if !hasResult {
		return nil, nil
	}

	return result, nil
}
