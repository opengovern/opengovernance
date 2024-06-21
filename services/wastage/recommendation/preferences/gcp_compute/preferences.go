package gcp_compute

var (
	PreferenceInstanceKey = map[string]string{
		"Region":        "region",
		"vCPU":          "guest_cpus",
		"MemoryGB":      "memory_mb",
		"MachineFamily": "machine_family",
	}

	PreferenceInstanceSpecialCond = map[string]string{
		"vCPU":     ">=",
		"MemoryGB": ">=",
	}
)
