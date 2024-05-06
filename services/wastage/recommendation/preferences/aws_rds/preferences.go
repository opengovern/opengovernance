package aws_rds

var (
	PreferenceInstanceDBKey = map[string]string{
		"Region":         "region_code",
		"vCPU":           "v_cpu",
		"MemoryGB":       "memory_gb",
		"InstanceType":   "instance_type",
		"Engine":         "database_engine",
		"ClusterType":    "deployment_option",
		"InstanceFamily": "instance_family",
		"LicenseModel":   "license_model",
	}

	PreferenceInstanceSpecialCond = map[string]string{
		"vCPU":     ">=",
		"MemoryGB": ">=",
	}

	PreferenceStorageDBKey = map[string]string{
		"StorageType": "volume_type",
	}

	PreferenceStorageSpecialCond = map[string]string{}
)
