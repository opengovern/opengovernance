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
