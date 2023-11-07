package cost_estimator

type CostEstimatorFunc func(h *HttpHandler, resourceType string, resourceId string) (float64, error)

var azureResourceTypes = map[string]CostEstimatorFunc{
	"Microsoft.Compute/virtualMachines": GetComputeVirtualMachineCost,
	"Microsoft.Network/virtualNetworks": GetVirtualNetworkCost,
}

var awsResourceTypes = map[string]CostEstimatorFunc{
	"AWS::EC2::Instance":                        GetEC2InstanceCost,
	"AWS::EC2::Volume":                          GetEC2VolumeCost,
	"AWS::ElasticLoadBalancingV2::LoadBalancer": GetELBCost,
	"AWS::ElasticLoadBalancing::LoadBalancer":   GetELBCost,
	"AWS::RDS::DBInstance":                      GetRDSInstanceCost,
}
