package cost_estimator

type CostEstimatorFunc func(h *HttpHandler, resourceId string) (float64, error)

var azureResourceTypes = map[string]CostEstimatorFunc{
	"Microsoft.Compute/virtualMachines": GetComputeVirtualMachineCost,
}

var awsResourceTypes = map[string]CostEstimatorFunc{}
