package cost_estimator

type CostEstimatorFunc func(h *HttpHandler, resourceType string, resourceId string) (float64, error)

var azureResourceTypes = map[string]CostEstimatorFunc{
	"microsoft_compute_virtualmachines": GetComputeVirtualMachineCost,
	"microsoft_compute_disks":           GetManagedStorageCost,
	"microsoft_network_loadbalancers":   GetLoadBalancerCost,
	"microsoft_network_virtualnetworks": GetVirtualNetworkCost,
}

var awsResourceTypes = map[string]CostEstimatorFunc{
	"aws_ec2_instance":                        GetEC2InstanceCost,
	"aws_ec2_volume":                          GetEC2VolumeCost,
	"aws_elasticloadbalancingv2_loadbalancer": GetELBCost,
	"aws_elasticloadbalancing_loadbalancer":   GetELBCost,
	"aws_rds_dbinstance":                      GetRDSInstanceCost,
}
