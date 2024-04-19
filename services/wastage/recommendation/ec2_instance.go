package recommendation

import (
	"fmt"
	types2 "github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/kaytu-io/kaytu-engine/services/wastage/api/entity"
	"github.com/kaytu-io/kaytu-engine/services/wastage/db/model"
)

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

func (s *Service) EC2InstanceRecommendation(region string, instance entity.EC2Instance, volumes []entity.EC2Volume, metrics map[string][]types2.Datapoint, preferences map[string]*string) (*Recommendation, error) {
	averageCPUUtilization := averageOfDatapoints(metrics["CPUUtilization"])
	averageNetworkIn := averageOfDatapoints(metrics["NetworkIn"])
	averageNetworkOut := averageOfDatapoints(metrics["NetworkOut"])
	averageMemPercent := averageOfDatapoints(metrics["mem_used_percent"])

	i, err := s.ec2InstanceRepo.ListByInstanceType(string(instance.InstanceType))
	if err != nil {
		return nil, err
	}
	if len(i) == 0 {
		return nil, fmt.Errorf("instance type not found: %s", string(instance.InstanceType))
	}
	// Burst in CPU & Network
	// Network: UpTo
	// Memory: -> User , Arch , EbsOptimized , EnaSupport
	// Volume ===> Optimization

	vCPU := instance.ThreadsPerCore * instance.CoreCount
	neededCPU := float64(vCPU) * averageCPUUtilization / 100.0
	neededMemory := float64(i[0].MemoryGB) * averageMemPercent / 100.0

	pref := map[string]interface{}{}
	for k, v := range preferences {
		var vl interface{}
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

	instanceType, err := s.ec2InstanceRepo.GetCheapestByCoreAndNetwork(averageNetworkIn+averageNetworkOut, pref)
	if err != nil {
		return nil, err
	}

	if instanceType != nil {
		description := fmt.Sprintf("change your vms from %s to %s", instance.InstanceType, instanceType.InstanceType)
		instance.InstanceType = types.InstanceType(instanceType.InstanceType)
		return &Recommendation{
			Description:         description,
			NewInstance:         instance,
			NewVolumes:          volumes,
			CurrentInstanceType: &i[0],
			NewInstanceType:     instanceType,
			AvgNetworkBandwidth: fmt.Sprintf("%.0f Bytes", averageNetworkOut+averageNetworkIn),
			AvgCPUUsage:         fmt.Sprintf("%.1f vCPUs", neededCPU),
		}, nil
	}
	return nil, nil
}

func extractFromInstance(instance entity.EC2Instance, i model.EC2InstanceType, region string, k string) interface{} {
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
