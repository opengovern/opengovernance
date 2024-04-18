package recommendation

import (
	"fmt"
	types2 "github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

func averageOfDatapoints(datapoints []types2.Datapoint) float64 {
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

func (s *Service) EC2InstanceRecommendation(region string, instance types.Instance,
	volumes []types.Volume, metrics map[string][]types2.Datapoint) ([]Recommendation, error) {
	averageCPUUtilization := averageOfDatapoints(metrics["CPUUtilization"])
	averageNetworkIn := averageOfDatapoints(metrics["NetworkIn"])
	averageNetworkOut := averageOfDatapoints(metrics["NetworkOut"])

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

	vCPU := *instance.CpuOptions.ThreadsPerCore * *instance.CpuOptions.CoreCount
	neededCPU := float64(vCPU) * averageCPUUtilization / 100.0
	instanceType, err := s.ec2InstanceRepo.GetCheapestByCoreAndNetwork(neededCPU, i[0].MemoryGB, averageNetworkIn+averageNetworkOut, "Linux", region)
	if err != nil {
		return nil, err
	}

	var recoms []Recommendation
	if instanceType != nil {
		recoms = append(recoms, Recommendation{
			Description: fmt.Sprintf("change your vms from %s to %s", instance.InstanceType, instanceType.InstanceType),
			NewInstance: instance,
			NewVolumes:  volumes,
		})
		instance.InstanceType = types.InstanceType(instanceType.InstanceType)
	} else {
		fmt.Println("instance type not found")
	}
	return recoms, nil
}
