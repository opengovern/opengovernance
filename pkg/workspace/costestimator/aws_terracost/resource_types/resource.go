package terraform

import (
	"fmt"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/api"

	"github.com/kaytu-io/kaytu-engine/pkg/workspace/costestimator/aws_terracost/region"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/costestimator/query"
)

// Provider is an implementation of the terraform.Provider, used to extract component queries from
// terraform resources.
type Provider struct {
	key    string
	region region.Code
}

// NewProvider returns a new Provider with the provided default region and a query key.
func NewProvider(key string, regionCode region.Code) (*Provider, error) {
	if !regionCode.Valid() {
		return nil, fmt.Errorf("invalid AWS region: %q", regionCode)
	}
	return &Provider{key: key, region: regionCode}, nil
}

// Name returns the Provider's common name.
func (p *Provider) Name() string { return p.key }

// ResourceComponents returns Component queries for a given terraform.Resource.
func (p *Provider) ResourceComponents(resourceType string, request any) ([]query.Component, error) {
	switch resourceType {
	case "aws_instance":
		var instanceRequest api.GetEC2InstanceCostRequest
		if req, ok := request.(api.GetEC2InstanceCostRequest); ok {
			instanceRequest = req
		} else {
			return nil, fmt.Errorf("could not parse request")
		}
		vals, err := decodeInstanceValues(instanceRequest)
		if err != nil {
			return nil, err
		}
		return p.newInstance(*vals).Components(), nil
	case "aws_autoscaling_group":
		return nil, nil
		//vals, err := decodeAutoscalingGroupValues(tfRes.Values)
		//if err != nil {
		//	return nil
		//}
		//return p.newAutoscalingGroup(rss, vals).Components()
	case "aws_db_instance":
		var dbInstanceRequest api.GetRDSInstanceRequest
		if req, ok := request.(api.GetRDSInstanceRequest); ok {
			dbInstanceRequest = req
		} else {
			return nil, fmt.Errorf("could not parse request")
		}
		vals := decodeDBInstanceValues(dbInstanceRequest)
		return p.newDBInstance(vals).Components(), nil
	case "aws_ebs_volume":
		var ebsVolumeRequest api.GetEC2VolumeCostRequest
		if req, ok := request.(api.GetEC2VolumeCostRequest); ok {
			ebsVolumeRequest = req
		} else {
			return nil, fmt.Errorf("could not parse request")
		}
		vals := decodeVolumeValues(ebsVolumeRequest)
		return p.newVolume(vals).Components(), nil
	case "aws_efs_file_system":
		return nil, nil
		//vals, err := decodeEFSFileSystemValues(tfRes.Values)
		//if err != nil {
		//	return nil
		//}
		//return p.newEFSFileSystem(rss, vals).Components()
	case "aws_elasticache_cluster":
		return nil, nil
		//vals, err := decodeElastiCacheValues(tfRes.Values)
		//if err != nil {
		//	return nil
		//}
		//return p.newElastiCache(vals).Components()
	case "aws_elasticache_replication_group":
		return nil, nil
		//vals, err := decodeElastiCacheReplicationValues(tfRes.Values)
		//if err != nil {
		//	return nil
		//}
		//return p.newElastiCacheReplication(vals).Components()
	case "aws_eip":
		return nil, nil
		//vals, err := decodeElasticIPValues(tfRes.Values)
		//if err != nil {
		//	return nil
		//}
		//return p.newElasticIP(vals).Components()
	case "aws_elb":
		// ELB Classic does not have any special configuration.
		vals := lbValues{LoadBalancerType: "classic"}
		return p.newLB(vals).Components(), nil
	case "aws_eks_cluster":
		return nil, nil
		//vals, err := decodeEKSClusterValues(tfRes.Values)
		//if err != nil {
		//	return nil
		//}
		//return p.newEKSCluster(vals).Components()
	case "aws_eks_node_group":
		return nil, nil
		//vals, err := decodeEKSNodeGroupValues(tfRes.Values)
		//if err != nil {
		//	return nil
		//}
		//return p.newEKSNodeGroup(rss, vals).Components()
	case "aws_fsx_lustre_file_system":
		return nil, nil
		//vals, err := decodeFSxLustreFileSystemValues(tfRes.Values)
		//if err != nil {
		//	return nil
		//}
		//return p.newFSxLustreFileSystem(rss, vals).Components()
	case "aws_fsx_ontap_file_system":
		return nil, nil
		//vals, err := decodeFSxOntapFileSystemValues(tfRes.Values)
		//if err != nil {
		//	return nil
		//}
		//return p.newFSxOntapFileSystem(rss, vals).Components()
	case "aws_fsx_openzfs_file_system":
		return nil, nil
		//vals, err := decodeFSxOpenzfsFileSystemValues(tfRes.Values)
		//if err != nil {
		//	return nil
		//}
		//return p.newFSxOpenzfsFileSystem(rss, vals).Components()
	case "aws_fsx_windows_file_system":
		return nil, nil
		//vals, err := decodeFSxWindowsFileSystemValues(tfRes.Values)
		//if err != nil {
		//	return nil
		//}
		//return p.newFSxWindowsFileSystem(rss, vals).Components()
	case "aws_lb", "aws_alb":
		var loadBalancerRequest api.GetLBCostRequest
		if req, ok := request.(api.GetLBCostRequest); ok {
			loadBalancerRequest = req
		} else {
			return nil, fmt.Errorf("could not parse request")
		}
		vals := decodeLBValues(loadBalancerRequest)
		return p.newLB(vals).Components(), nil
	case "aws_nat_gateway":
		return nil, nil
		//vals, err := decodeNatGatewayValues(tfRes.Values)
		//if err != nil {
		//	return nil
		//}
		//return p.newNatGateway(vals).Components()
	default:
		return nil, nil
	}
}
