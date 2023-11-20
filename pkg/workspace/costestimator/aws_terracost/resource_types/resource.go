package terraform

import (
	"fmt"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/api"
	"go.uber.org/zap"

	"github.com/kaytu-io/kaytu-engine/pkg/workspace/costestimator/query"
)

// Provider is an implementation of the terraform.Provider, used to extract component queries from
// terraform resources.
type Provider struct {
	key string
}

// NewProvider returns a new Provider with the provided default region and a query key.
func NewProvider(key string) (*Provider, error) {
	return &Provider{key: key}, nil
}

// Name returns the Provider's common name.
func (p *Provider) Name() string { return p.key }

// ResourceComponents returns Component queries for a given terraform.Resource.
func (p *Provider) ResourceComponents(logger *zap.Logger, resourceType string, request any) ([]query.Component, error) {
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
		if req, ok := request.(map[string]interface{}); ok {
			dbInstanceRequest = api.GetRDSInstanceRequest{
				RegionCode:           req["RegionCode"].(string),
				InstanceEngine:       req["InstanceEngine"].(string),
				InstanceLicenseModel: req["InstanceLicenseModel"].(string),
				InstanceMultiAZ:      req["InstanceMultiAZ"].(bool),
				AllocatedStorage:     req["AllocatedStorage"].(float64),
				StorageType:          req["StorageType"].(string),
				IOPs:                 req["IOPs"].(float64),
			}
		} else {
			return nil, fmt.Errorf("could not parse request")
		}
		vals := decodeDBInstanceValues(dbInstanceRequest)
		return p.newDBInstance(vals).Components(), nil
	case "aws_ebs_volume":
		var ebsVolumeRequest api.GetEC2VolumeCostRequest
		if req, ok := request.(map[string]interface{}); ok {
			ebsVolumeRequest = api.GetEC2VolumeCostRequest{
				RegionCode: req["RegionCode"].(string),
				Type:       req["Type"].(string),
				Size:       req["Size"].(float64),
				IOPs:       req["IOPs"].(float64),
			}
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
		var ebsVolumeRequest api.GetLBCostRequest
		if req, ok := request.(map[string]interface{}); ok {
			ebsVolumeRequest = api.GetLBCostRequest{
				RegionCode: req["RegionCode"].(string),
				LBType:     req["LBType"].(string),
			}
		} else {
			return nil, fmt.Errorf("could not parse request")
		}
		vals := decodeLBValues(ebsVolumeRequest)
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
		if req, ok := request.(map[string]interface{}); ok {
			loadBalancerRequest = api.GetLBCostRequest{
				RegionCode: req["RegionCode"].(string),
				LBType:     req["LBType"].(string),
			}
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
