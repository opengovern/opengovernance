package kaytu_client

import (
	"encoding/json"
	"github.com/kaytu-io/kaytu-aws-describer/aws/model"
)

// getAwsEc2Values get resource values needed for cost estimate from model.EC2HostDescription
func getAwsEc2Values(resource Resource) (map[string]interface{}, error) {
	description := resource.Description
	bytes, err := json.Marshal(description)
	if err != nil {
		return nil, err
	}
	var values model.EC2HostDescription
	err = json.Unmarshal(bytes, &values)

	var valuesMap map[string]interface{}
	valuesMap["availability_zone"] = ""
	if values.Host.AvailabilityZone != nil {
		valuesMap["availability_zone"] = *values.Host.AvailabilityZone
	}
	valuesMap["instance_type"] = ""
	valuesMap["instance_family"] = ""
	if values.Host.HostProperties != nil {
		if values.Host.HostProperties.InstanceType != nil {
			valuesMap["instance_type"] = *values.Host.HostProperties.InstanceType
		}
		if values.Host.HostProperties.InstanceFamily != nil {
			valuesMap["instance_family"] = *values.Host.HostProperties.InstanceFamily
		}
	}
	return valuesMap, nil
}

// getAwsLambdaFunctionValues get resource values needed for cost estimate from model.LambdaFunctionDescription
func getAwsLambdaFunctionValues(resource Resource) (map[string]interface{}, error) {
	description := resource.Description
	bytes, err := json.Marshal(description)
	if err != nil {
		return nil, err
	}
	var values model.LambdaFunctionDescription
	err = json.Unmarshal(bytes, &values)

	var valuesMap map[string]interface{}
	if values.Function == nil {
		return nil, nil
	}
	if values.Function.Configuration != nil {
		valuesMap["memory_size"] = ptrInt(values.Function.Configuration.MemorySize)
		valuesMap["architectures"] = values.Function.Configuration.Architectures
		if values.Function.Configuration.EphemeralStorage != nil {
			valuesMap["ephemeral_storage"] = []map[string]interface{}{{"size": ptrInt(values.Function.Configuration.EphemeralStorage.Size)}}
		}
	}
	return valuesMap, nil
}

// getAwsEsDomainValues get resource values needed for cost estimate from model.ESDomainDescription
func getAwsEsDomainValues(resource Resource) (map[string]interface{}, error) {
	description := resource.Description
	bytes, err := json.Marshal(description)
	if err != nil {
		return nil, err
	}
	var values model.ESDomainDescription
	err = json.Unmarshal(bytes, &values)

	var valuesMap map[string]interface{}
	if values.Domain.ElasticsearchClusterConfig != nil {
		valuesMap["cluster_config"] = []map[string]interface{}{
			{
				"instance_type":            values.Domain.ElasticsearchClusterConfig.InstanceType,
				"instance_count":           ptrInt(values.Domain.ElasticsearchClusterConfig.InstanceCount),
				"dedicated_master_enabled": ptrBool(values.Domain.ElasticsearchClusterConfig.DedicatedMasterEnabled),
				"dedicated_master_type":    values.Domain.ElasticsearchClusterConfig.DedicatedMasterType,
				"dedicated_master_count":   ptrInt(values.Domain.ElasticsearchClusterConfig.DedicatedMasterCount),
				"warm_enabled":             ptrBool(values.Domain.ElasticsearchClusterConfig.WarmEnabled),
				"warm_type":                values.Domain.ElasticsearchClusterConfig.WarmType,
				"warm_count":               ptrInt(values.Domain.ElasticsearchClusterConfig.WarmCount),
			},
		}
	}

	if values.Domain.EBSOptions != nil {
		valuesMap["ebs_options"] = []map[string]interface{}{
			{
				"ebs_enabled": ptrBool(values.Domain.EBSOptions.EBSEnabled),
				"volume_type": values.Domain.EBSOptions.VolumeType,
				"volume_size": ptrInt(values.Domain.EBSOptions.VolumeSize),
				"iops":        ptrInt(values.Domain.EBSOptions.Iops),
				"throughput":  ptrInt(values.Domain.EBSOptions.Throughput),
			},
		}
	}

	return valuesMap, nil
}

// getAwsOpenSearchDomainValues get resource values needed for cost estimate from model.OpenSearchDomainDescription
func getAwsOpenSearchDomainValues(resource Resource) (map[string]interface{}, error) {
	description := resource.Description
	bytes, err := json.Marshal(description)
	if err != nil {
		return nil, err
	}
	var values model.OpenSearchDomainDescription
	err = json.Unmarshal(bytes, &values)

	var valuesMap map[string]interface{}
	if values.Domain.ClusterConfig != nil {
		valuesMap["cluster_config"] = []map[string]interface{}{
			{
				"instance_type":            values.Domain.ClusterConfig.InstanceType,
				"instance_count":           ptrInt(values.Domain.ClusterConfig.InstanceCount),
				"dedicated_master_enabled": ptrBool(values.Domain.ClusterConfig.DedicatedMasterEnabled),
				"dedicated_master_type":    values.Domain.ClusterConfig.DedicatedMasterType,
				"dedicated_master_count":   ptrInt(values.Domain.ClusterConfig.DedicatedMasterCount),
				"warm_enabled":             ptrBool(values.Domain.ClusterConfig.WarmEnabled),
				"warm_type":                values.Domain.ClusterConfig.WarmType,
				"warm_count":               ptrInt(values.Domain.ClusterConfig.WarmCount),
			},
		}
	}

	if values.Domain.EBSOptions != nil {
		valuesMap["ebs_options"] = []map[string]interface{}{
			{
				"ebs_enabled": ptrBool(values.Domain.EBSOptions.EBSEnabled),
				"volume_type": values.Domain.EBSOptions.VolumeType,
				"volume_size": ptrInt(values.Domain.EBSOptions.VolumeSize),
				"iops":        ptrInt(values.Domain.EBSOptions.Iops),
				"throughput":  ptrInt(values.Domain.EBSOptions.Throughput),
			},
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
	description := resource.Description
	bytes, err := json.Marshal(description)
	if err != nil {
		return nil, err
	}
	var values model.FSXFileSystemDescription
	err = json.Unmarshal(bytes, &values)

	var valuesMap map[string]interface{}

	valuesMap["storage_capacity"] = ptrInt2(values.FileSystem.StorageCapacity)
	valuesMap["storage_type"] = values.FileSystem.StorageType
	if values.FileSystem.LustreConfiguration != nil {
		valuesMap["deployment_type"] = values.FileSystem.LustreConfiguration.DeploymentType
		valuesMap["data_compression_type"] = values.FileSystem.LustreConfiguration.DataCompressionType
	}
	if values.FileSystem.OpenZFSConfiguration != nil {
		valuesMap["throughput_capacity"] = ptrInt2(values.FileSystem.OpenZFSConfiguration.ThroughputCapacity)
		valuesMap["automatic_backup_retention_days"] = ptrInt2(values.FileSystem.OpenZFSConfiguration.AutomaticBackupRetentionDays)
	}
	if values.FileSystem.OntapConfiguration != nil {
		if values.FileSystem.OntapConfiguration.DiskIopsConfiguration != nil {
			valuesMap["disk_iops_configuration"] = []map[string]interface{}{{
				"iops": ptrInt642(values.FileSystem.OntapConfiguration.DiskIopsConfiguration.Iops),
				"mode": values.FileSystem.OntapConfiguration.DiskIopsConfiguration.Mode,
			}}
		}
	}

	return valuesMap, nil
}

// getAwsEksNodeGroupValues get resource values needed for cost estimate from model.EKSNodegroupDescription
func getAwsEksNodeGroupValues(resource Resource) (map[string]interface{}, error) {
	description := resource.Description
	bytes, err := json.Marshal(description)
	if err != nil {
		return nil, err
	}
	var values model.EKSNodegroupDescription
	err = json.Unmarshal(bytes, &values)

	var valuesMap map[string]interface{}

	if values.Nodegroup.ScalingConfig != nil {
		valuesMap["scaling_config"] = []map[string]interface{}{
			{
				"min_size":     ptrInt2(values.Nodegroup.ScalingConfig.MinSize),
				"desired_size": ptrInt2(values.Nodegroup.ScalingConfig.DesiredSize),
			},
		}
	}
	valuesMap["instance_types"] = values.Nodegroup.InstanceTypes
	valuesMap["disk_size"] = ptrInt2(values.Nodegroup.DiskSize)
	if values.Nodegroup.LaunchTemplate != nil {
		valuesMap["launch_template"] = []map[string]interface{}{
			{
				"id":      ptrStr2(values.Nodegroup.LaunchTemplate.Id),
				"Name":    ptrStr2(values.Nodegroup.LaunchTemplate.Name),
				"Version": ptrStr2(values.Nodegroup.LaunchTemplate.Version),
			},
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
	description := resource.Description
	bytes, err := json.Marshal(description)
	if err != nil {
		return nil, err
	}
	var values model.EC2EIPDescription
	err = json.Unmarshal(bytes, &values)

	var valuesMap map[string]interface{}

	valuesMap["customer_owned_ipv4_pool"] = ptrStr2(values.Address.CustomerOwnedIpv4Pool)
	valuesMap["instance"] = ptrStr2(values.Address.InstanceId)
	valuesMap["network_interface"] = ptrStr2(values.Address.NetworkInterfaceId)

	return valuesMap, nil
}

// getAwsElastiCacheReplicationGroupValues get resource values needed for cost estimate from model.ElastiCacheReplicationGroupDescription
func getAwsElastiCacheReplicationGroupValues(resource Resource) (map[string]interface{}, error) {
	description := resource.Description
	bytes, err := json.Marshal(description)
	if err != nil {
		return nil, err
	}
	var values model.ElastiCacheReplicationGroupDescription
	err = json.Unmarshal(bytes, &values)

	var valuesMap map[string]interface{}

	valuesMap["node_type"] = ptrStr2(values.ReplicationGroup.CacheNodeType)
	valuesMap["engine"] = "" // TODO find engine
	valuesMap["num_node_groups"] = len(values.ReplicationGroup.NodeGroups)
	var replicas int
	for _, ng := range values.ReplicationGroup.NodeGroups {
		replicas += len(ng.NodeGroupMembers)
	}
	valuesMap["replicas_per_node_group"] = replicas / len(values.ReplicationGroup.NodeGroups)
	valuesMap["num_cache_clusters"] = len(values.ReplicationGroup.MemberClusters)
	valuesMap["snapshot_retention_limit"] = ptrInt2(values.ReplicationGroup.SnapshotRetentionLimit)

	if values.ReplicationGroup.GlobalReplicationGroupInfo != nil {
		valuesMap["global_replication_group_id"] = ptrStr2(values.ReplicationGroup.GlobalReplicationGroupInfo.GlobalReplicationGroupId)
	}

	return valuesMap, nil
}

// getAwsElastiCacheClusterValues get resource values needed for cost estimate from model.ElastiCacheClusterDescription
func getAwsElastiCacheClusterValues(resource Resource) (map[string]interface{}, error) {
	description := resource.Description
	bytes, err := json.Marshal(description)
	if err != nil {
		return nil, err
	}
	var values model.ElastiCacheClusterDescription
	err = json.Unmarshal(bytes, &values)

	var valuesMap map[string]interface{}

	valuesMap["node_type"] = ptrStr2(values.Cluster.CacheNodeType)
	valuesMap["availability_zone"] = ptrStr2(values.Cluster.PreferredAvailabilityZone)
	valuesMap["engine"] = ptrStr2(values.Cluster.Engine)
	valuesMap["replication_group_id"] = ptrStr2(values.Cluster.ReplicationGroupId)
	valuesMap["num_cache_nodes"] = len(values.Cluster.CacheNodes)
	valuesMap["snapshot_retention_limit"] = ptrInt2(values.Cluster.SnapshotRetentionLimit)

	return valuesMap, nil
}

// getAwsEfsFileSystemValues get resource values needed for cost estimate from model.EFSFileSystemDescription
func getAwsEfsFileSystemValues(resource Resource) (map[string]interface{}, error) {
	description := resource.Description
	bytes, err := json.Marshal(description)
	if err != nil {
		return nil, err
	}
	var values model.EFSFileSystemDescription
	err = json.Unmarshal(bytes, &values)

	var valuesMap map[string]interface{}

	valuesMap["availability_zone_name"] = ptrStr2(values.FileSystem.AvailabilityZoneName)
	//valuesMap["lifecycle_policy"] = nil // TODO
	valuesMap["throughput_mode"] = values.FileSystem.ThroughputMode
	valuesMap["provisioned_throughput_in_mibps"] = ptrFloat2(values.FileSystem.ProvisionedThroughputInMibps)

	return valuesMap, nil
}

// getAwsEbsSnapshotValues get resource values needed for cost estimate from model.EC2VolumeSnapshotDescription
func getAwsEbsSnapshotValues(resource Resource) (map[string]interface{}, error) {
	description := resource.Description
	bytes, err := json.Marshal(description)
	if err != nil {
		return nil, err
	}
	var values model.EC2VolumeSnapshotDescription
	err = json.Unmarshal(bytes, &values)

	var valuesMap map[string]interface{}

	valuesMap["volume_size"] = ptrInt2(values.Snapshot.VolumeSize)

	return valuesMap, nil
}

// getAwsEbsVolumeValues get resource values needed for cost estimate from model.EC2VolumeDescription
func getAwsEbsVolumeValues(resource Resource) (map[string]interface{}, error) {
	description := resource.Description
	bytes, err := json.Marshal(description)
	if err != nil {
		return nil, err
	}
	var values model.EC2VolumeDescription
	err = json.Unmarshal(bytes, &values)

	var valuesMap map[string]interface{}

	valuesMap["availability_zone"] = ptrStr2(values.Volume.AvailabilityZone)
	valuesMap["type"] = values.Volume.VolumeType
	valuesMap["size"] = ptrInt2(values.Volume.Size)
	valuesMap["iops"] = ptrInt2(values.Volume.Iops)

	return valuesMap, nil
}

// getAwsRdsDbInstanceValues get resource values needed for cost estimate from model.RDSDBInstanceDescription
func getAwsRdsDbInstanceValues(resource Resource) (map[string]interface{}, error) {
	description := resource.Description
	bytes, err := json.Marshal(description)
	if err != nil {
		return nil, err
	}
	var values model.RDSDBInstanceDescription
	err = json.Unmarshal(bytes, &values)

	var valuesMap map[string]interface{}

	valuesMap["instance_class"] = ptrStr2(values.DBInstance.DBInstanceClass)
	valuesMap["availability_zone"] = ptrStr2(values.DBInstance.AvailabilityZone)
	valuesMap["engine"] = ptrStr2(values.DBInstance.Engine)
	valuesMap["license_model"] = ptrStr2(values.DBInstance.LicenseModel)
	valuesMap["multi_az"] = ptrBool2(values.DBInstance.MultiAZ)
	valuesMap["allocated_storage"] = ptrInt2(values.DBInstance.AllocatedStorage)
	valuesMap["storage_type"] = ptrStr2(values.DBInstance.StorageType)
	valuesMap["iops"] = ptrInt2(values.DBInstance.Iops)

	return valuesMap, nil
}

// getAwsAutoscalingGroupValues get resource values needed for cost estimate from model.AutoScalingGroupDescription
func getAwsAutoscalingGroupValues(resource Resource) (map[string]interface{}, error) {
	description := resource.Description
	bytes, err := json.Marshal(description)
	if err != nil {
		return nil, err
	}
	var values model.AutoScalingGroupDescription
	err = json.Unmarshal(bytes, &values)

	var valuesMap map[string]interface{}

	valuesMap["availability_zones"] = values.AutoScalingGroup.AvailabilityZones
	valuesMap["launch_configuration"] = values.AutoScalingGroup.LaunchConfigurationName

	return valuesMap, nil
}

// getAwsEc2InstanceValues get resource values needed for cost estimate from model.EC2InstanceDescription
func getAwsEc2InstanceValues(resource Resource) (map[string]interface{}, error) {
	description := resource.Description
	bytes, err := json.Marshal(description)
	if err != nil {
		return nil, err
	}
	var values model.EC2InstanceDescription
	err = json.Unmarshal(bytes, &values)

	var valuesMap map[string]interface{}

	valuesMap["instance_type"] = values.Instance.InstanceType
	if values.Instance.Placement != nil {
		valuesMap["tenancy"] = values.Instance.Placement.Tenancy
		valuesMap["availability_zone"] = ptrStr2(values.Instance.Placement.AvailabilityZone)
	}
	if values.Instance.Platform == "" {
		valuesMap["operating_system"] = "Linux"
	} else {
		valuesMap["operating_system"] = values.Instance.Platform
	}
	valuesMap["ebs_optimized"] = ptrBool2(values.Instance.EbsOptimized)
	if values.Instance.Monitoring != nil {
		if values.Instance.Monitoring.State == "disabled" || values.Instance.Monitoring.State == "disabling" {
			valuesMap["monitoring"] = false
		} else {
			valuesMap["monitoring"] = true
		}
	}
	if values.Instance.CpuOptions != nil {
		valuesMap["credit_specification"] = []map[string]interface{}{{
			"cpu_credits": ptrInt2(values.Instance.CpuOptions.CoreCount), // TODO not sure
		}}
	}
	// valuesMap["root_block_device"] // TODO must get from another resource

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
	description := resource.Description
	bytes, err := json.Marshal(description)
	if err != nil {
		return nil, err
	}
	var values model.ElasticLoadBalancingV2LoadBalancerDescription
	err = json.Unmarshal(bytes, &values)

	var valuesMap map[string]interface{}

	valuesMap["load_balancer_type"] = values.LoadBalancer.Type

	return valuesMap, nil
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
