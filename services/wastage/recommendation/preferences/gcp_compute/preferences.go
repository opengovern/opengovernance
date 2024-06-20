package gcp_compute

var (
	PreferenceInstanceKey = map[string]string{
		"Region":        "region_code",
		"vCPU":          "guest_cpus",
		"MemoryGB":      "memory_mb",
		"Zone":          "zone",
		"MachineFamily": "machine_family",
		"MachineType":   "machine_type",
	}

	PreferenceInstanceSpecialCond = map[string]string{
		"vCPU":     ">=",
		"MemoryGB": ">=",
	}
)
