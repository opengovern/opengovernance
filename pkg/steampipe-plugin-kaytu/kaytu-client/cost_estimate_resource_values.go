package kaytu_client

import (
	kaytuAws "github.com/kaytu-io/kaytu-aws-describer/pkg/kaytu-es-sdk"
	kaytuAzure "github.com/kaytu-io/kaytu-azure-describer/pkg/kaytu-es-sdk"
)

// getAwsEc2HostValues get resource values needed for cost estimate from model.EC2HostDescription
func getAwsEc2HostValues(resource Resource) (map[string]interface{}, error) {
	var valuesMap map[string]interface{}
	if v, ok := resource.Description.(kaytuAws.EC2Host); ok {
		valuesMap["availability_zone"] = ""
		if v.Description.Host.AvailabilityZone != nil {
			valuesMap["availability_zone"] = *v.Description.Host.AvailabilityZone
		}
		valuesMap["instance_type"] = ""
		valuesMap["instance_family"] = ""
		if v.Description.Host.HostProperties != nil {
			if v.Description.Host.HostProperties.InstanceType != nil {
				valuesMap["instance_type"] = *v.Description.Host.HostProperties.InstanceType
			}
			if v.Description.Host.HostProperties.InstanceFamily != nil {
				valuesMap["instance_family"] = *v.Description.Host.HostProperties.InstanceFamily
			}
		}
	}
	return valuesMap, nil
}

// getAwsLambdaFunctionValues get resource values needed for cost estimate from model.LambdaFunctionDescription
func getAwsLambdaFunctionValues(resource Resource) (map[string]interface{}, error) {
	var valuesMap map[string]interface{}
	if v, ok := resource.Description.(kaytuAws.LambdaFunction); ok {
		if v.Description.Function == nil {
			return nil, nil
		}
		if v.Description.Function.Configuration != nil {
			valuesMap["memory_size"] = ptrInt(v.Description.Function.Configuration.MemorySize)
			valuesMap["architectures"] = v.Description.Function.Configuration.Architectures
			if v.Description.Function.Configuration.EphemeralStorage != nil {
				valuesMap["ephemeral_storage"] = []map[string]interface{}{{"size": ptrInt(v.Description.Function.Configuration.EphemeralStorage.Size)}}
			}
		}
	}
	return valuesMap, nil
}

// getAwsEsDomainValues get resource values needed for cost estimate from model.ESDomainDescription
func getAwsEsDomainValues(resource Resource) (map[string]interface{}, error) {
	var valuesMap map[string]interface{}
	if v, ok := resource.Description.(kaytuAws.ESDomain); ok {
		if v.Description.Domain.ElasticsearchClusterConfig != nil {
			valuesMap["cluster_config"] = []map[string]interface{}{
				{
					"instance_type":            v.Description.Domain.ElasticsearchClusterConfig.InstanceType,
					"instance_count":           ptrInt(v.Description.Domain.ElasticsearchClusterConfig.InstanceCount),
					"dedicated_master_enabled": ptrBool(v.Description.Domain.ElasticsearchClusterConfig.DedicatedMasterEnabled),
					"dedicated_master_type":    v.Description.Domain.ElasticsearchClusterConfig.DedicatedMasterType,
					"dedicated_master_count":   ptrInt(v.Description.Domain.ElasticsearchClusterConfig.DedicatedMasterCount),
					"warm_enabled":             ptrBool(v.Description.Domain.ElasticsearchClusterConfig.WarmEnabled),
					"warm_type":                v.Description.Domain.ElasticsearchClusterConfig.WarmType,
					"warm_count":               ptrInt(v.Description.Domain.ElasticsearchClusterConfig.WarmCount),
				},
			}
		}

		if v.Description.Domain.EBSOptions != nil {
			valuesMap["ebs_options"] = []map[string]interface{}{
				{
					"ebs_enabled": ptrBool(v.Description.Domain.EBSOptions.EBSEnabled),
					"volume_type": v.Description.Domain.EBSOptions.VolumeType,
					"volume_size": ptrInt(v.Description.Domain.EBSOptions.VolumeSize),
					"iops":        ptrInt(v.Description.Domain.EBSOptions.Iops),
					"throughput":  ptrInt(v.Description.Domain.EBSOptions.Throughput),
				},
			}
		}
	}
	return valuesMap, nil
}

// getAwsOpenSearchDomainValues get resource values needed for cost estimate from model.OpenSearchDomainDescription
func getAwsOpenSearchDomainValues(resource Resource) (map[string]interface{}, error) {
	var valuesMap map[string]interface{}

	if v, ok := resource.Description.(kaytuAws.OpenSearchDomain); ok {

		if v.Description.Domain.ClusterConfig != nil {
			valuesMap["cluster_config"] = []map[string]interface{}{
				{
					"instance_type":            v.Description.Domain.ClusterConfig.InstanceType,
					"instance_count":           ptrInt(v.Description.Domain.ClusterConfig.InstanceCount),
					"dedicated_master_enabled": ptrBool(v.Description.Domain.ClusterConfig.DedicatedMasterEnabled),
					"dedicated_master_type":    v.Description.Domain.ClusterConfig.DedicatedMasterType,
					"dedicated_master_count":   ptrInt(v.Description.Domain.ClusterConfig.DedicatedMasterCount),
					"warm_enabled":             ptrBool(v.Description.Domain.ClusterConfig.WarmEnabled),
					"warm_type":                v.Description.Domain.ClusterConfig.WarmType,
					"warm_count":               ptrInt(v.Description.Domain.ClusterConfig.WarmCount),
				},
			}
		}

		if v.Description.Domain.EBSOptions != nil {
			valuesMap["ebs_options"] = []map[string]interface{}{
				{
					"ebs_enabled": ptrBool(v.Description.Domain.EBSOptions.EBSEnabled),
					"volume_type": v.Description.Domain.EBSOptions.VolumeType,
					"volume_size": ptrInt(v.Description.Domain.EBSOptions.VolumeSize),
					"iops":        ptrInt(v.Description.Domain.EBSOptions.Iops),
					"throughput":  ptrInt(v.Description.Domain.EBSOptions.Throughput),
				},
			}
		}
	}

	return valuesMap, nil
}

// getAwsNatGatewayValues get resource values needed for cost estimate
func getAwsNatGatewayValues(resource Resource) (map[string]interface{}, error) {
	return map[string]interface{}{}, nil
}

// getAwsFSXFileSystemValues get resource values needed for cost estimate from model.FSXFileSystemDescription
func getAwsFSXFileSystemValues(resource Resource) (map[string]interface{}, error) {
	var valuesMap map[string]interface{}
	if v, ok := resource.Description.(kaytuAws.FSXFileSystem); ok {
		valuesMap["storage_capacity"] = ptrInt2(v.Description.FileSystem.StorageCapacity)
		valuesMap["storage_type"] = v.Description.FileSystem.StorageType
		if v.Description.FileSystem.LustreConfiguration != nil {
			valuesMap["deployment_type"] = v.Description.FileSystem.LustreConfiguration.DeploymentType
			valuesMap["data_compression_type"] = v.Description.FileSystem.LustreConfiguration.DataCompressionType
		}
		if v.Description.FileSystem.OpenZFSConfiguration != nil {
			valuesMap["throughput_capacity"] = ptrInt2(v.Description.FileSystem.OpenZFSConfiguration.ThroughputCapacity)
			valuesMap["automatic_backup_retention_days"] = ptrInt2(v.Description.FileSystem.OpenZFSConfiguration.AutomaticBackupRetentionDays)
		}
		if v.Description.FileSystem.OntapConfiguration != nil {
			if v.Description.FileSystem.OntapConfiguration.DiskIopsConfiguration != nil {
				valuesMap["disk_iops_configuration"] = []map[string]interface{}{{
					"iops": ptrInt642(v.Description.FileSystem.OntapConfiguration.DiskIopsConfiguration.Iops),
					"mode": v.Description.FileSystem.OntapConfiguration.DiskIopsConfiguration.Mode,
				}}
			}
		}
	}
	return valuesMap, nil
}

// getAwsEksNodeGroupValues get resource values needed for cost estimate from model.EKSNodegroupDescription
func getAwsEksNodeGroupValues(resource Resource) (map[string]interface{}, error) {
	var valuesMap map[string]interface{}
	if v, ok := resource.Description.(kaytuAws.EKSNodegroup); ok {
		if v.Description.Nodegroup.ScalingConfig != nil {
			valuesMap["scaling_config"] = []map[string]interface{}{
				{
					"min_size":     ptrInt2(v.Description.Nodegroup.ScalingConfig.MinSize),
					"desired_size": ptrInt2(v.Description.Nodegroup.ScalingConfig.DesiredSize),
				},
			}
		}
		valuesMap["instance_types"] = v.Description.Nodegroup.InstanceTypes
		valuesMap["disk_size"] = ptrInt2(v.Description.Nodegroup.DiskSize)
		if v.Description.Nodegroup.LaunchTemplate != nil {
			valuesMap["launch_template"] = []map[string]interface{}{
				{
					"id":      ptrStr2(v.Description.Nodegroup.LaunchTemplate.Id),
					"Name":    ptrStr2(v.Description.Nodegroup.LaunchTemplate.Name),
					"Version": ptrStr2(v.Description.Nodegroup.LaunchTemplate.Version),
				},
			}
		}
	}
	return valuesMap, nil
}

// getAwsEksNodeGroupValues get resource values needed for cost estimate from model.EKSNodegroupDescription
func getAwsEksClusterValues(resource Resource) (map[string]interface{}, error) {
	return map[string]interface{}{}, nil
}

// getAwsEc2EipValues get resource values needed for cost estimate from model.EC2EIPDescription
func getAwsEc2EipValues(resource Resource) (map[string]interface{}, error) {
	if v, ok := resource.Description.(kaytuAws.EC2EIP); ok {
		return map[string]interface{}{
			"customer_owned_ipv4_pool": ptrStr2(v.Description.Address.CustomerOwnedIpv4Pool),
			"instance":                 ptrStr2(v.Description.Address.InstanceId),
			"network_interface":        ptrStr2(v.Description.Address.NetworkInterfaceId),
		}, nil
	}

	return nil, nil
}

// getAwsElastiCacheReplicationGroupValues get resource values needed for cost estimate from model.ElastiCacheReplicationGroupDescription
func getAwsElastiCacheReplicationGroupValues(resource Resource) (map[string]interface{}, error) {
	var valuesMap map[string]interface{}
	if v, ok := resource.Description.(kaytuAws.ElastiCacheReplicationGroup); ok {
		valuesMap["node_type"] = ptrStr2(v.Description.ReplicationGroup.CacheNodeType)
		valuesMap["engine"] = "" // TODO find engine
		valuesMap["num_node_groups"] = len(v.Description.ReplicationGroup.NodeGroups)
		var replicas int
		for _, ng := range v.Description.ReplicationGroup.NodeGroups {
			replicas += len(ng.NodeGroupMembers)
		}
		valuesMap["replicas_per_node_group"] = replicas / len(v.Description.ReplicationGroup.NodeGroups)
		valuesMap["num_cache_clusters"] = len(v.Description.ReplicationGroup.MemberClusters)
		valuesMap["snapshot_retention_limit"] = ptrInt2(v.Description.ReplicationGroup.SnapshotRetentionLimit)

		if v.Description.ReplicationGroup.GlobalReplicationGroupInfo != nil {
			valuesMap["global_replication_group_id"] = ptrStr2(v.Description.ReplicationGroup.GlobalReplicationGroupInfo.GlobalReplicationGroupId)
		}
	}
	return valuesMap, nil
}

// getAwsElastiCacheClusterValues get resource values needed for cost estimate from model.ElastiCacheClusterDescription
func getAwsElastiCacheClusterValues(resource Resource) (map[string]interface{}, error) {
	var valuesMap map[string]interface{}
	if v, ok := resource.Description.(kaytuAws.ElastiCacheCluster); ok {
		valuesMap["node_type"] = ptrStr2(v.Description.Cluster.CacheNodeType)
		valuesMap["availability_zone"] = ptrStr2(v.Description.Cluster.PreferredAvailabilityZone)
		valuesMap["engine"] = ptrStr2(v.Description.Cluster.Engine)
		valuesMap["replication_group_id"] = ptrStr2(v.Description.Cluster.ReplicationGroupId)
		valuesMap["num_cache_nodes"] = len(v.Description.Cluster.CacheNodes)
		valuesMap["snapshot_retention_limit"] = ptrInt2(v.Description.Cluster.SnapshotRetentionLimit)
	}
	return valuesMap, nil
}

// getAwsEfsFileSystemValues get resource values needed for cost estimate from model.EFSFileSystemDescription
func getAwsEfsFileSystemValues(resource Resource) (map[string]interface{}, error) {
	var valuesMap map[string]interface{}
	if v, ok := resource.Description.(kaytuAws.EFSFileSystem); ok {
		valuesMap["availability_zone_name"] = ptrStr2(v.Description.FileSystem.AvailabilityZoneName)
		//valuesMap["lifecycle_policy"] = nil // TODO
		valuesMap["throughput_mode"] = v.Description.FileSystem.ThroughputMode
		valuesMap["provisioned_throughput_in_mibps"] = ptrFloat2(v.Description.FileSystem.ProvisionedThroughputInMibps)
	}
	return valuesMap, nil
}

// getAwsEbsSnapshotValues get resource values needed for cost estimate from model.EC2VolumeSnapshotDescription
func getAwsEbsSnapshotValues(resource Resource) (map[string]interface{}, error) {
	if v, ok := resource.Description.(kaytuAws.EC2VolumeSnapshot); ok {
		return map[string]interface{}{
			"volume_size": v.Description.Snapshot.VolumeSize,
		}, nil
	}
	return nil, nil
}

// getAwsEbsVolumeValues get resource values needed for cost estimate from model.EC2VolumeDescription
func getAwsEbsVolumeValues(resource Resource) (map[string]interface{}, error) {
	if v, ok := resource.Description.(kaytuAws.EC2Volume); ok {
		return map[string]interface{}{
			"availability_zone": ptrStr2(v.Description.Volume.AvailabilityZone),
			"type":              v.Description.Volume.VolumeType,
			"size":              ptrInt2(v.Description.Volume.Size),
			"iops":              ptrInt2(v.Description.Volume.Iops),
			"throughput":        ptrInt2(v.Description.Volume.Throughput),
		}, nil
	}

	return nil, nil
}

// getAwsRdsDbInstanceValues get resource values needed for cost estimate from model.RDSDBInstanceDescription
func getAwsRdsDbInstanceValues(resource Resource) (map[string]interface{}, error) {
	var valuesMap map[string]interface{}

	if v, ok := resource.Description.(kaytuAws.RDSDBInstance); ok {
		valuesMap["instance_class"] = ptrStr2(v.Description.DBInstance.DBInstanceClass)
		valuesMap["availability_zone"] = ptrStr2(v.Description.DBInstance.AvailabilityZone)
		valuesMap["engine"] = ptrStr2(v.Description.DBInstance.Engine)
		valuesMap["license_model"] = ptrStr2(v.Description.DBInstance.LicenseModel)
		valuesMap["multi_az"] = ptrBool2(v.Description.DBInstance.MultiAZ)
		valuesMap["allocated_storage"] = ptrInt2(v.Description.DBInstance.AllocatedStorage)
		valuesMap["storage_type"] = ptrStr2(v.Description.DBInstance.StorageType)
		valuesMap["iops"] = ptrInt2(v.Description.DBInstance.Iops)
	}

	return valuesMap, nil
}

// getAwsAutoscalingGroupValues get resource values needed for cost estimate from model.AutoScalingGroupDescription
func getAwsAutoscalingGroupValues(resource Resource) (map[string]interface{}, error) {
	var valuesMap map[string]interface{}
	if v, ok := resource.Description.(kaytuAws.AutoScalingGroup); ok {
		valuesMap["availability_zones"] = v.Description.AutoScalingGroup.AvailabilityZones
		valuesMap["launch_configuration"] = v.Description.AutoScalingGroup.LaunchConfigurationName
	}
	return valuesMap, nil
}

// getAwsEc2InstanceValues get resource values needed for cost estimate from model.EC2InstanceDescription
func getAwsEc2InstanceValues(resource Resource) (map[string]interface{}, error) {
	var valuesMap map[string]interface{}

	if v, ok := resource.Description.(kaytuAws.EC2Instance); ok {
		valuesMap["instance_type"] = v.Description.Instance.InstanceType
		if v.Description.Instance.Placement != nil {
			valuesMap["tenancy"] = v.Description.Instance.Placement.Tenancy
			valuesMap["availability_zone"] = ptrStr2(v.Description.Instance.Placement.AvailabilityZone)
		}
		if v.Description.Instance.Platform == "" {
			valuesMap["operating_system"] = "Linux"
		} else {
			valuesMap["operating_system"] = v.Description.Instance.Platform
		}
		valuesMap["ebs_optimized"] = ptrBool2(v.Description.Instance.EbsOptimized)
		if v.Description.Instance.Monitoring != nil {
			if v.Description.Instance.Monitoring.State == "disabled" || v.Description.Instance.Monitoring.State == "disabling" {
				valuesMap["monitoring"] = false
			} else {
				valuesMap["monitoring"] = true
			}
		}
		if v.Description.Instance.CpuOptions != nil {
			valuesMap["credit_specification"] = []map[string]interface{}{{
				"cpu_credits": ptrInt2(v.Description.Instance.CpuOptions.CoreCount), // TODO not sure
			}}
		}
		// valuesMap["root_block_device"] // TODO must get from another resource
	}
	return valuesMap, nil
}

// getAwsLoadBalancerValues get resource values needed for cost estimate
func getAwsLoadBalancerValues(resource Resource) (map[string]interface{}, error) {
	var valuesMap map[string]interface{}

	valuesMap["load_balancer_type"] = "classic"

	return valuesMap, nil
}

// getAwsLoadBalancer2Values get resource values needed for cost estimate from model.ElasticLoadBalancingV2LoadBalancerDescription
func getAwsLoadBalancer2Values(resource Resource) (map[string]interface{}, error) {
	if v, ok := resource.Description.(kaytuAws.ElasticLoadBalancingV2LoadBalancer); ok {
		return map[string]interface{}{
			"load_balancer_type": v.Description.LoadBalancer.Type,
		}, nil
	}

	return nil, nil
}

func getAzureComputeSnapshotValues(resource Resource) (map[string]interface{}, error) {
	if v, ok := resource.Description.(kaytuAzure.ComputeSnapshots); ok {
		return map[string]interface{}{
			"disk_size_gb": ptrInt2(v.Description.Snapshot.Properties.DiskSizeGB),
			"location":     ptrStr2(v.Description.Snapshot.Location),
		}, nil
	} else {
		return resource.Description.(map[string]interface{}), nil
	}
}

func getAzureComputeDiskValues(resource Resource) (map[string]interface{}, error) {
	if v, ok := resource.Description.(kaytuAzure.ComputeDisk); ok {
		return map[string]interface{}{
			"storage_account_type":       *v.Description.Disk.SKU.Name,
			"location":                   ptrStr2(v.Description.Disk.Location),
			"disk_size_gb":               ptrInt2(v.Description.Disk.Properties.DiskSizeGB),
			"on_demand_bursting_enabled": ptrBool2(v.Description.Disk.Properties.BurstingEnabled),
			"disk_mbps_read_write":       ptrInt642(v.Description.Disk.Properties.DiskMBpsReadWrite),
			"disk_iops_read_write":       ptrInt642(v.Description.Disk.Properties.DiskIOPSReadWrite),
		}, nil
	}

	return resource.Description.(map[string]interface{}), nil
}

func getAzureLoadBalancerValues(resource Resource) (map[string]interface{}, error) {
	if v, ok := resource.Description.(kaytuAzure.LoadBalancer); ok {
		return map[string]interface{}{
			"sku":          *v.Description.LoadBalancer.SKU.Name,
			"location":     ptrStr2(v.Description.LoadBalancer.Location),
			"rules_number": len(v.Description.LoadBalancer.Properties.InboundNatRules) + len(v.Description.LoadBalancer.Properties.LoadBalancingRules) + len(v.Description.LoadBalancer.Properties.OutboundRules),
			"sku_tier":     *v.Description.LoadBalancer.SKU.Tier,
		}, nil
	}

	return resource.Description.(map[string]interface{}), nil
}

func ptrBool(pointer *bool) interface{} {
	if pointer == nil {
		return nil
	} else {
		return *pointer
	}
}

func ptrInt(pointer *int32) interface{} {
	if pointer == nil {
		return nil
	} else {
		return *pointer
	}
}

func ptrStr(pointer *string) interface{} {
	if pointer == nil {
		return nil
	} else {
		return *pointer
	}
}

func ptrBool2(pointer *bool) bool {
	if pointer == nil {
		return false
	} else {
		return *pointer
	}
}

func ptrInt2(pointer *int32) int32 {
	if pointer == nil {
		return 0
	} else {
		return *pointer
	}
}

func ptrInt642(pointer *int64) int64 {
	if pointer == nil {
		return 0
	} else {
		return *pointer
	}
}

func ptrStr2(pointer *string) string {
	if pointer == nil {
		return ""
	} else {
		return *pointer
	}
}

func ptrFloat2(pointer *float64) float64 {
	if pointer == nil {
		return 0
	} else {
		return *pointer
	}
}
