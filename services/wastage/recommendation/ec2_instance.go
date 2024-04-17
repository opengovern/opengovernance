package recommendation

import (
	"fmt"
	types2 "github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/kaytu-io/kaytu-aws-describer/aws/model"
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

func (s *Service) EC2InstanceRecommendation(client *ec2.Client, region string, instance model.EC2InstanceDescription,
	volumes []model.EC2VolumeDescription, metrics map[string][]types2.Datapoint) ([]Recommendation, error) {
	averageCPUUtilization := averageOfDatapoints(metrics["CPUUtilization"])
	averageNetworkIn := averageOfDatapoints(metrics["NetworkIn"])
	averageNetworkOut := averageOfDatapoints(metrics["NetworkOut"])
	averageNetworkPacketsIn := averageOfDatapoints(metrics["NetworkPacketsIn"])
	averageNetworkPacketsOut := averageOfDatapoints(metrics["NetworkPacketsOut"])
	fmt.Println(
		*instance.Instance.CpuOptions.ThreadsPerCore,
		*instance.Instance.CpuOptions.CoreCount,
		averageCPUUtilization,
		averageNetworkIn,
		averageNetworkOut,
		averageNetworkPacketsIn,
		averageNetworkPacketsOut,
	)
	vCPU := *instance.Instance.CpuOptions.ThreadsPerCore * *instance.Instance.CpuOptions.CoreCount
	neededCPU := float64(vCPU) * averageCPUUtilization / 100.0
	instanceType, err := s.ec2InstanceRepo.GetCheapestByCoreAndNetwork(neededCPU, averageNetworkIn+averageNetworkOut)
	if err != nil {
		return nil, err
	}

	fmt.Println(instanceType.InstanceType)

	//for _, instanceType := range out.InstanceTypes {
	//
	//}

	return nil, nil
}
