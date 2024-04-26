package recommendation

import (
	"context"
	"errors"
	"fmt"
	types2 "github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/kaytu-io/kaytu-engine/pkg/utils"
	"github.com/kaytu-io/kaytu-engine/services/wastage/api/entity"
	"github.com/kaytu-io/kaytu-engine/services/wastage/db/model"
	"github.com/sashabaranov/go-openai"
	"math"
	"sort"
	"strconv"
	"strings"
)

func mergeDatapoints(in []types2.Datapoint, out []types2.Datapoint) []types2.Datapoint {
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
		maxV = max(maxV, *dp.Maximum)
	}
	return maxV
}

func extractUsage(dps []types2.Datapoint) entity.Usage {
	minV, avgV, maxV := minOfDatapoints(dps), averageOfDatapoints(dps), maxOfDatapoints(dps)
	return entity.Usage{
		Avg: &avgV,
		Min: &minV,
		Max: &maxV,
	}
}

func (s *Service) EC2InstanceRecommendation(
	region string,
	instance entity.EC2Instance,
	volumes []entity.EC2Volume,
	metrics map[string][]types2.Datapoint,
	volumeMetrics map[string]map[string][]types2.Datapoint,
	preferences map[string]*string,
) (*entity.RightSizingRecommendation, error) {
	networkDatapoints := mergeDatapoints(metrics["NetworkIn"], metrics["NetworkOut"])
	cpuUsage := extractUsage(metrics["CPUUtilization"])
	memoryUsage := extractUsage(metrics["mem_used_percent"])
	networkUsage := extractUsage(networkDatapoints)

	var ebsDatapoints []types2.Datapoint
	for _, v := range volumeMetrics {
		ebsDatapoints = mergeDatapoints(mergeDatapoints(v["VolumeReadBytes"], v["VolumeWriteBytes"]), ebsDatapoints)
	}
	ebsThroughputUsage := extractUsage(ebsDatapoints)

	currentInstanceTypeList, err := s.ec2InstanceRepo.ListByInstanceType(string(instance.InstanceType), instance.Platform, instance.UsageOperation, region)
	if err != nil {
		return nil, err
	}
	if len(currentInstanceTypeList) == 0 {
		return nil, fmt.Errorf("instance type not found: %s", string(instance.InstanceType))
	}
	currentInstanceType := currentInstanceTypeList[0]
	currentCost, err := s.costSvc.GetEC2InstanceCost(region, instance, volumes, metrics)
	if err != nil {
		return nil, err
	}
	current := entity.RightsizingEC2Instance{
		InstanceType:      currentInstanceType.InstanceType,
		Processor:         currentInstanceType.PhysicalProcessor,
		Architecture:      currentInstanceType.PhysicalProcessorArch,
		VCPU:              currentInstanceType.VCpu,
		Memory:            currentInstanceType.MemoryGB,
		EBSBandwidth:      currentInstanceType.DedicatedEBSThroughput,
		NetworkThroughput: currentInstanceType.NetworkPerformance,
		ENASupported:      currentInstanceType.EnhancedNetworkingSupported,
		Cost:              currentCost,
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
	neededCPU := float64(vCPU) * (*cpuUsage.Avg + float64(cpuBreathingRoom)) / 100.0
	neededMemory := float64(currentInstanceType.MemoryGB) * (*memoryUsage.Max + float64(memoryBreathingRoom)) / 100.0
	neededNetworkThroughput := *networkUsage.Avg
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
	if value, ok := preferences["ExcludeBurstableInstances"]; ok && value != nil {
		if *value == "Yes" {
			pref["NOT(instance_type like ?)"] = "t%"
		}
	}
	if value, ok := preferences["UsageOperation"]; ok && value != nil {
		if v, ok := UsageOperationHumanToMachine[*value]; ok {
			pref["operation = ?"] = v
		} else {
			delete(pref, "operation = ?")
		}
	}
	// if operation is not provided, limit the results to one with no pre-installed software
	if _, ok := pref["operation = ?"]; !ok {
		pref["pre_installed_sw = ?"] = "NA"
	}

	var recommended *entity.RightsizingEC2Instance
	rightSizedInstanceType, err := s.ec2InstanceRepo.GetCheapestByCoreAndNetwork(neededNetworkThroughput, pref)
	if err != nil {
		return nil, err
	}
	if rightSizedInstanceType != nil {
		newInstance := instance
		newInstance.InstanceType = types.InstanceType(rightSizedInstanceType.InstanceType)
		newInstance.UsageOperation = rightSizedInstanceType.Operation
		if newInstance.Placement == nil {
			newInstance.Placement = &entity.EC2Placement{}
		}
		if rightSizedInstanceType.Tenancy == "Dedicated" {
			newInstance.Placement.Tenancy = types.TenancyDedicated
		} else if rightSizedInstanceType.Tenancy == "Host" {
			newInstance.Placement.Tenancy = types.TenancyHost
		} else {
			newInstance.Placement.Tenancy = types.TenancyDefault
		}
		recommendedCost, err := s.costSvc.GetEC2InstanceCost(region, newInstance, volumes, metrics)
		if err != nil {
			return nil, err
		}

		recommended = &entity.RightsizingEC2Instance{
			InstanceType:      rightSizedInstanceType.InstanceType,
			Processor:         rightSizedInstanceType.PhysicalProcessor,
			Architecture:      rightSizedInstanceType.PhysicalProcessorArch,
			VCPU:              rightSizedInstanceType.VCpu,
			Memory:            rightSizedInstanceType.MemoryGB,
			EBSBandwidth:      rightSizedInstanceType.DedicatedEBSThroughput,
			NetworkThroughput: rightSizedInstanceType.NetworkPerformance,
			ENASupported:      rightSizedInstanceType.EnhancedNetworkingSupported,
			Cost:              recommendedCost,
		}
	}

	recommendation := entity.RightSizingRecommendation{
		Current:           current,
		Recommended:       recommended,
		VCPU:              cpuUsage,
		EBSBandwidth:      ebsThroughputUsage,
		NetworkThroughput: networkUsage,
		Description:       "",
	}
	if len(metrics["mem_used_percent"]) > 0 {
		recommendation.Memory = memoryUsage
	}

	if rightSizedInstanceType != nil {
		recommendation.Description, _ = s.generateDescription(instance, region, &currentInstanceType, rightSizedInstanceType, metrics, preferences, neededCPU, neededMemory, neededNetworkThroughput)
	}

	return &recommendation, nil
}

func (s *Service) generateDescription(instance entity.EC2Instance, region string, currentInstanceType, rightSizedInstanceType *model.EC2InstanceType, metrics map[string][]types2.Datapoint, preferences map[string]*string, neededCPU, neededMemory, neededNetworkThroughput float64) (string, error) {
	minCPU, avgCPU, maxCPU := minOfDatapoints(metrics["CPUUtilization"]), averageOfDatapoints(metrics["CPUUtilization"]), maxOfDatapoints(metrics["CPUUtilization"])
	minMemory, avgMemory, maxMemory := minOfDatapoints(metrics["mem_used_percent"]), averageOfDatapoints(metrics["mem_used_percent"]), maxOfDatapoints(metrics["mem_used_percent"])
	networkDatapoints := mergeDatapoints(metrics["NetworkIn"], metrics["NetworkOut"])
	minNetwork, avgNetwork, maxNetwork := minOfDatapoints(networkDatapoints), averageOfDatapoints(networkDatapoints), maxOfDatapoints(networkDatapoints)

	usage := fmt.Sprintf("- %s has %d vCPUs. Usage over the course of last week is min=%.2f%%, avg=%.2f%%, max=%.2f%%, so you only need %.2f vCPUs. %s has %d vCPUs.\n", currentInstanceType.InstanceType, currentInstanceType.VCpu, minCPU, avgCPU, maxCPU, neededCPU, rightSizedInstanceType.InstanceType, rightSizedInstanceType.VCpu)
	if len(metrics["mem_used_percent"]) > 0 {
		usage += fmt.Sprintf("- %s has %dGB Memory. Usage over the course of last week is min=%.2f%%, avg=%.2f%%, max=%.2f%%, so you only need %.2fGB Memory. %s has %dGB Memory.\n", currentInstanceType.InstanceType, currentInstanceType.MemoryGB, minMemory, avgMemory, maxMemory, neededMemory, rightSizedInstanceType.InstanceType, rightSizedInstanceType.MemoryGB)
	} else {
		usage += fmt.Sprintf("- %s has %dGB Memory. Usage is not available. You need to install CloudWatch Agent on your instance to get this data. %s has %dGB Memory.\n", currentInstanceType.InstanceType, currentInstanceType.MemoryGB, rightSizedInstanceType.InstanceType, rightSizedInstanceType.MemoryGB)
	}
	usage += fmt.Sprintf("- %s's network performance is %s. Throughput over the course of last week is min=%.2f MB/s, avg=%.2f MB/s, max=%.2f MB/s, so you only need %.2f MB/s. %s has %s.\n", currentInstanceType.InstanceType, currentInstanceType.NetworkPerformance, minNetwork/1000000.0, avgNetwork/1000000.0, maxNetwork/1000000.0, neededNetworkThroughput/1000000.0, rightSizedInstanceType.InstanceType, rightSizedInstanceType.NetworkPerformance)

	needs := ""
	for k, v := range preferences {
		if PreferenceDBKey[k] == "" {
			continue
		}
		if v == nil {
			vl := extractFromInstance(instance, *currentInstanceType, region, k)
			needs += fmt.Sprintf("- You asked %s to be same as the current instance value which is %v\n", k, vl)
		} else {
			needs += fmt.Sprintf("- You asked %s to be %s\n", k, *v)
		}
	}

	prompt := fmt.Sprintf(`
I'm giving recommendation on ec2 instance right sizing. Based on user's usage and needs I have concluded that the best option for him is to use %s instead of %s. I need help summarizing the explanation into 3 lines while keeping these rules:
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

func extractFromInstance(instance entity.EC2Instance, i model.EC2InstanceType, region string, k string) any {
	switch k {
	case "InstanceFamily":
		switch instance.Tenancy {
		case types.TenancyDefault:
			return "Shared"
		case types.TenancyDedicated:
			return "Dedicated"
		case types.TenancyHost:
			return "Host"
		default:
			return ""
		}
	case "UsageOperation":
		return instance.UsageOperation
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

func (s *Service) EBSVolumeRecommendation(region string, volume entity.EC2Volume, metrics map[string][]types2.Datapoint, preferences map[string]*string) (*entity.EBSVolumeRecommendation, error) {
	iopsUsage := extractUsage(mergeDatapoints(metrics["VolumeReadOps"], metrics["VolumeWriteOps"]))
	throughputUsage := extractUsage(mergeDatapoints(metrics["VolumeReadBytes"], metrics["VolumeWriteBytes"]))
	sizeUsage := extractUsage(metrics["disk_used_percent"])

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
	neededIops := *iopsUsage.Avg * (1 + float64(iopsBreathingRoom)/100.0)
	neededThroughput := *throughputUsage.Avg * (1 + float64(throughputBreathingRoom)/100.0)
	neededSize := size
	if _, ok := metrics["disk_used_percent"]; ok {
		neededSize = max(1, neededSize*(*sizeUsage.Avg/100.0))
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

	volumeCost, err := s.costSvc.GetEBSVolumeCost(region, volume, metrics)
	if err != nil {
		return nil, err
	}

	var result = &entity.EBSVolumeRecommendation{
		Current: entity.RightsizingEBSVolume{
			Tier:                  volume.VolumeType,
			VolumeSize:            volume.Size,
			BaselineIOPS:          0, //TODO-Saleh
			ProvisionedIOPS:       volume.Iops,
			BaselineThroughput:    0, //TODO-Saleh
			ProvisionedThroughput: volume.Throughput,
			Cost:                  volumeCost,
		},
		Recommended: nil,
		IOPS:        iopsUsage,
		Throughput:  throughputUsage,
		Description: "",
	}

	newType, newBaselineIops, newBaselineThroughput, err := s.ebsVolumeRepo.GetCheapestTypeWithSpecs(region, int32(neededSize), int32(neededIops), neededThroughput/1000000, validTypes)
	if err != nil {
		if strings.Contains(err.Error(), "no feasible volume types found") {
			return result, nil
		}
		return nil, err
	}

	result.Recommended = &entity.RightsizingEBSVolume{
		Tier:                  "",
		VolumeSize:            utils.GetPointer(int32(neededSize)),
		BaselineIOPS:          newBaselineIops,
		ProvisionedIOPS:       nil,
		BaselineThroughput:    newBaselineThroughput,
		ProvisionedThroughput: nil,
		Cost:                  0,
	}
	newVolume := volume

	if newType != volume.VolumeType {
		result.Recommended.Tier = newType
		newVolume.VolumeType = newType
		result.Description = fmt.Sprintf("- change your volume from %s to %s\n", volume.VolumeType, newType)
	}

	if int32(neededSize) != *volume.Size {
		result.Recommended.VolumeSize = utils.GetPointer(int32(neededSize))
		newVolume.Size = utils.GetPointer(int32(neededSize))
		result.Description += fmt.Sprintf("- change volume size from %d to %d\n", *volume.Size, int32(neededSize))
	}

	if newType == types.VolumeTypeIo1 || newType == types.VolumeTypeIo2 {
		avgIOps := int32(neededIops)
		result.Recommended.ProvisionedIOPS = &avgIOps
		newVolume.Iops = &avgIOps

		if volume.Iops == nil {
			result.Description += fmt.Sprintf("- add provisioned iops: %d\n", avgIOps)
		} else if avgIOps > *volume.Iops {
			result.Description += fmt.Sprintf("- increase provisioned iops from %d to %d\n", *volume.Iops, avgIOps)
		} else if avgIOps < *volume.Iops {
			result.Description += fmt.Sprintf("- decrease provisioned iops from %d to %d\n", *volume.Iops, avgIOps)
		} else {
			result.Recommended.ProvisionedIOPS = nil
			newVolume.Iops = volume.Iops
		}
	}

	if newType == types.VolumeTypeGp3 {
		provIops := max(int32(neededIops)-model.Gp3BaseIops, 0)
		result.Recommended.ProvisionedIOPS = &provIops
		newVolume.Iops = &provIops

		if volume.Iops == nil {
			result.Description += fmt.Sprintf("- add provisioned iops: %d\n", provIops)
		} else if provIops > *volume.Iops {
			result.Description += fmt.Sprintf("- increase provisioned iops from %d to %d\n", *volume.Iops, provIops)
		} else if provIops < *volume.Iops {
			result.Description += fmt.Sprintf("- decrease provisioned iops from %d to %d\n", *volume.Iops, provIops)
		} else {
			result.Recommended.ProvisionedIOPS = nil
			newVolume.Iops = volume.Iops
		}
	}

	if newType == types.VolumeTypeGp3 {
		provThroughput := max(neededThroughput-model.Gp3BaseThroughput, 0)
		result.Recommended.ProvisionedThroughput = &provThroughput
		newVolume.Throughput = &provThroughput

		if volume.Throughput == nil {
			result.Description += fmt.Sprintf("- add provisioned throughput: %.2f\n", provThroughput)
		} else if provThroughput > *volume.Throughput {
			result.Description += fmt.Sprintf("- increase provisioned throughput from %.2f to %.2f\n", *volume.Throughput, provThroughput)
		} else if provThroughput < *volume.Throughput {
			result.Description += fmt.Sprintf("- decrease provisioned throughput from %.2f to %.2f\n", *volume.Throughput, provThroughput)
		} else {
			result.Recommended.ProvisionedThroughput = nil
			newVolume.Throughput = volume.Throughput
		}
	}

	newVolumeCost, err := s.costSvc.GetEBSVolumeCost(region, newVolume, metrics)
	if err != nil {
		return nil, err
	}
	result.Recommended.Cost = newVolumeCost

	return result, nil
}
