package cost_estimator

type CostEstimatorFunc func(h *HttpHandler, resourceId string, timeInterval int) (float64, error)

var azureResourceTypes = map[string]CostEstimatorFunc{
	"Microsoft.Compute/virtualMachines": GetComputeVirtualMachineCost,
	"Microsoft.Network/virtualNetworks": GetVirtualNetworkCost,
}

var awsResourceTypes = map[string]CostEstimatorFunc{
	"AWS::EC2::Instance":   GetEC2InstanceCost,
	"AWS::RDS::DBInstance": GetRDSInstanceCost,
}
