package recommendation

import (
	"fmt"
	types2 "github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/kaytu-io/kaytu-engine/pkg/utils"
	"github.com/kaytu-io/kaytu-engine/services/wastage/api/entity"
	"github.com/kaytu-io/kaytu-engine/services/wastage/db/model"
	"math"
	"sort"
	"strconv"
	"strings"
)

func mergeDatapoints(in []types2.Datapoint, out []types2.Datapoint) []types2.Datapoint {
	if len(in) != len(out) {
		return nil
	}

	funcP := func(a, b *float64, f func(aa, bb float64) float64) *float64 {
		if a == nil && b == nil {
			return nil
		} else if a == nil {
			return b
		} else if b == nil {
			return a
		} else {
			tmp := f(*a, *b)
			return &tmp
		}
	}

	avg := func(aa, bb float64) float64 {
		return (aa + bb) / 2.0
	}
	sum := func(aa, bb float64) float64 {
		return aa + bb
	}

	dps := map[int64]*types2.Datapoint{}
	for _, dp := range in {
		dps[dp.Timestamp.Unix()] = &dp
	}
	for _, dp := range out {
		if dps[dp.Timestamp.Unix()] == nil {
			dps[dp.Timestamp.Unix()] = &dp
			break
		}

		dps[dp.Timestamp.Unix()].Average = funcP(dps[dp.Timestamp.Unix()].Average, dp.Average, avg)
		dps[dp.Timestamp.Unix()].Maximum = funcP(dps[dp.Timestamp.Unix()].Maximum, dp.Maximum, math.Max)
		dps[dp.Timestamp.Unix()].Minimum = funcP(dps[dp.Timestamp.Unix()].Minimum, dp.Minimum, math.Min)
		dps[dp.Timestamp.Unix()].SampleCount = funcP(dps[dp.Timestamp.Unix()].SampleCount, dp.SampleCount, sum)
		dps[dp.Timestamp.Unix()].Sum = funcP(dps[dp.Timestamp.Unix()].Sum, dp.Sum, sum)
	}

	var dpArr []types2.Datapoint
	for _, dp := range dps {
		dpArr = append(dpArr, *dp)
	}
	sort.Slice(dpArr, func(i, j int) bool {
		return dpArr[i].Timestamp.Unix() < dpArr[j].Timestamp.Unix()
	})
	return dpArr
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

func minOfDatapoints(datapoints []types2.Datapoint) float64 {
	if len(datapoints) == 0 {
		return 0.0
	}

	minV := math.MaxFloat64
	for _, dp := range datapoints {
		if dp.Minimum == nil {
			continue
		}
		minV = min(minV, *dp.Minimum)
	}
	return minV
}

func maxOfDatapoints(datapoints []types2.Datapoint) float64 {
	if len(datapoints) == 0 {
		return 0.0
	}

	maxV := 0.0
	for _, dp := range datapoints {
		if dp.Maximum == nil {
			continue
		}
		maxV = min(maxV, *dp.Maximum)
	}
	return maxV
}

func (s *Service) EC2InstanceRecommendation(region string, instance entity.EC2Instance, volumes []entity.EC2Volume, metrics map[string][]types2.Datapoint, volumeMetrics map[string]map[string][]types2.Datapoint, preferences map[string]*string) (*Ec2InstanceRecommendation, error) {
	minCPU, avgCPU, maxCPU := minOfDatapoints(metrics["CPUUtilization"]), averageOfDatapoints(metrics["CPUUtilization"]), maxOfDatapoints(metrics["CPUUtilization"])
	networkDatapoints := mergeDatapoints(metrics["NetworkIn"], metrics["NetworkOut"])
	minNetwork, avgNetwork, maxNetwork := minOfDatapoints(networkDatapoints), averageOfDatapoints(networkDatapoints), maxOfDatapoints(networkDatapoints)

	avgEBSThroughput := 0.0
	minEBSThroughput := math.MaxFloat64
	maxEBSThroughput := 0.0
	for _, v := range volumeMetrics {
		ebsThroughput := mergeDatapoints(v["VolumeReadBytes"], v["VolumeWriteBytes"])

		avgEBSThroughput += averageOfDatapoints(ebsThroughput)
		minEBSThroughput = min(minEBSThroughput, minOfDatapoints(ebsThroughput))
		maxEBSThroughput = max(maxEBSThroughput, maxOfDatapoints(ebsThroughput))
	}
	avgEBSThroughput = avgEBSThroughput / float64(len(volumeMetrics))

	maxMemPercent := maxOfDatapoints(metrics["mem_used_percent"])
	maxMemUsagePercentage := "Not available"
	if len(metrics["mem_used_percent"]) > 0 {
		maxMemUsagePercentage = fmt.Sprintf("Max: %.1f%%", maxMemPercent)
	}

	currentInstanceTypeList, err := s.ec2InstanceRepo.ListByInstanceType(string(instance.InstanceType), instance.Platform, region)
	if err != nil {
		return nil, err
	}
	if len(currentInstanceTypeList) == 0 {
		return nil, fmt.Errorf("instance type not found: %s", string(instance.InstanceType))
	}
	currentInstanceType := currentInstanceTypeList[0]

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
	neededCPU := float64(vCPU) * (avgCPU + float64(cpuBreathingRoom)) / 100.0
	neededMemory := float64(currentInstanceType.MemoryGB) * (maxMemPercent + float64(memoryBreathingRoom)) / 100.0
	neededNetworkThroughput := avgNetwork
	if preferences["NetworkBreathingRoom"] != nil {
		room, _ := strconv.ParseInt(*preferences["NetworkBreathingRoom"], 10, 64)
		neededNetworkThroughput += neededNetworkThroughput * float64(room) / 100.0
	}

	pref := map[string]any{}
	for k, v := range preferences {
		var vl any
		if v == nil {
			vl = extractFromInstance(instance, currentInstanceType, region, k)
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

	rightSizedInstanceType, err := s.ec2InstanceRepo.GetCheapestByCoreAndNetwork(neededNetworkThroughput, pref)
	if err != nil {
		return nil, err
	}
	avgCPUUsage := fmt.Sprintf("Avg: %.1f%%, Min: %.1f%%, Max: %.1f%%", avgCPU, minCPU, maxCPU)
	avgNetworkBandwidth := fmt.Sprintf("Avg: %.1f Megabit, Min: %.1f Megabit, Max: %.1f Megabit", avgNetwork/1000000.0*8.0,
		minNetwork/1000000.0*8.0, maxNetwork/1000000.0*8.0)

	avgEbsBandwidth := fmt.Sprintf("Avg: %.1f Megabit, Min: %.1f Megabit, Max: %.1f Megabit", (avgEBSThroughput)/1000000.0*8.0,
		(minEBSThroughput)/1000000.0*8.0, (maxEBSThroughput)/1000000.0*8.0)

	if rightSizedInstanceType != nil {
		instance.InstanceType = types.InstanceType(rightSizedInstanceType.InstanceType)
		return &Ec2InstanceRecommendation{
			Description:              generateDescription(instance, region, &currentInstanceType, rightSizedInstanceType, metrics, preferences, neededCPU, neededMemory, neededNetworkThroughput),
			NewInstance:              instance,
			NewVolumes:               volumes,
			CurrentInstanceType:      &currentInstanceType,
			NewInstanceType:          rightSizedInstanceType,
			AvgNetworkBandwidth:      avgNetworkBandwidth,
			AvgEBSBandwidth:          avgEbsBandwidth,
			AvgCPUUsage:              avgCPUUsage,
			MaxMemoryUsagePercentage: maxMemUsagePercentage,
		}, nil
	}
	return nil, nil
}

func generateDescription(
	instance entity.EC2Instance,
	region string,
	currentInstanceType, rightSizedInstanceType *model.EC2InstanceType,
	metrics map[string][]types2.Datapoint,
	preferences map[string]*string,
	neededCPU, neededMemory, neededNetworkThroughput float64,
) string {
	minCPU, avgCPU, maxCPU := minOfDatapoints(metrics["CPUUtilization"]), averageOfDatapoints(metrics["CPUUtilization"]), maxOfDatapoints(metrics["CPUUtilization"])
	minMemory, avgMemory, maxMemory := minOfDatapoints(metrics["mem_used_percent"]), averageOfDatapoints(metrics["mem_used_percent"]), maxOfDatapoints(metrics["mem_used_percent"])
	networkDatapoints := mergeDatapoints(metrics["NetworkIn"], metrics["NetworkOut"])
	minNetwork, avgNetwork, maxNetwork := minOfDatapoints(networkDatapoints), averageOfDatapoints(networkDatapoints), maxOfDatapoints(networkDatapoints)

	description := ""
	description += fmt.Sprintf("Currently the workload is running on %s instance type. right sized suggested instance type is %s\n", instance.InstanceType, rightSizedInstanceType.InstanceType)
	description += fmt.Sprintf("Currently the workload has %d vCPUs. Usage over the course of last week is min=%.2f%%, avg=%.2f%%, max=%.2f%%, so you only need %.2f vCPUs and the right sized one has %d vCPUs.\n", currentInstanceType.VCpu, minCPU, avgCPU, maxCPU, neededCPU, rightSizedInstanceType.VCpu)
	if len(metrics["mem_used_percent"]) > 0 {
		description += fmt.Sprintf("Currently the workload has %dGB Memory. Usage over the course of last week is min=%.2f%%, avg=%.2f%%, max=%.2f%%, so you only need %.2fGB Memory and the right sized one has %dGB Memory.\n", currentInstanceType.MemoryGB, minMemory, avgMemory, maxMemory, neededMemory, rightSizedInstanceType.MemoryGB)
	} else {
		description += fmt.Sprintf("Currently the workload has %dGB Memory. Usage is not available. You need to install CloudWatch Agent on your instance to get this data. The right sized one has %dGB Memory.\n", currentInstanceType.MemoryGB, rightSizedInstanceType.MemoryGB)
	}
	description += fmt.Sprintf("Currently the workload's network performance is %s. Throughput over the course of last week is min=%.2f MB/s, avg=%.2f MB/s, max=%.2f MB/s, so you only need %.2f MB/s and the right sized one has %s.\n", currentInstanceType.NetworkPerformance, minNetwork/1000000.0, avgNetwork/1000000.0, maxNetwork/1000000.0, neededNetworkThroughput/1000000.0, rightSizedInstanceType.NetworkPerformance)

	for k, v := range preferences {
		if PreferenceDBKey[k] == "" {
			continue
		}
		if v == nil {
			vl := extractFromInstance(instance, *currentInstanceType, region, k)
			description += fmt.Sprintf("You asked %s to be same as the current instance value which is %v\n", k, vl)
		} else {
			description += fmt.Sprintf("You asked %s to be %s\n", k, *v)
		}
	}

	description += fmt.Sprintf("based on these, the suggested right sized option is to go with %s instance type\n", rightSizedInstanceType.InstanceType)
	return description
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
		return instance.Platform
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
		return i.PhysicalProcessorArch
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
	averageSizeUsage := averageOfDatapoints(metrics["disk_used_percent"])

	size := float64(0)
	if size == 0 && volume.Size != nil {
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
	sizeBreathingRoom := int64(0)
	if preferences["SizeBreathingRoom"] != nil {
		sizeBreathingRoom, _ = strconv.ParseInt(*preferences["SizeBreathingRoom"], 10, 32)
	}
	neededIops := averageIops * (1 + float64(iopsBreathingRoom)/100.0)
	neededThroughput := averageThroughput * (1 + float64(throughputBreathingRoom)/100.0)
	neededSize := size
	if _, ok := metrics["disk_used_percent"]; ok {
		neededSize = max(1, neededSize*(averageSizeUsage/100.0))
		neededSize = neededSize * (1 + float64(sizeBreathingRoom)/100.0)
	}

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
		NewSize:                      int32(neededSize),
		CurrentProvisionedIOPS:       volume.Iops,
		NewProvisionedIOPS:           nil,
		CurrentProvisionedThroughput: volume.Throughput,
		NewProvisionedThroughput:     nil,
		CurrentVolumeType:            volume.VolumeType,
		NewVolumeType:                "",
		AvgIOPS:                      averageIops,
		AvgThroughput:                averageThroughput,
	}

	newType, newBaselineIops, newBaselineThroughput, err := s.ebsVolumeRepo.GetCheapestTypeWithSpecs(region, int32(neededSize), int32(neededIops), neededThroughput, validTypes)
	if err != nil {
		if strings.Contains(err.Error(), "no feasible volume types found") {
			return nil, nil
		}
		return nil, err
	}
	result.NewBaselineIOPS = utils.GetPointer(newBaselineIops)
	result.NewBaselineThroughput = utils.GetPointer(newBaselineThroughput)

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
		provIops := max(int32(neededIops)-model.Gp3BaseIops, 0)
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
		provThroughput := max(neededThroughput-model.Gp3BaseThroughput, 0)

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
