package recommendation

var (
	PreferenceDBKey = map[string]string{
		"Tenancy":                      "tenancy",
		"EBSOptimized":                 "ebs_optimized",
		"OperatingSystem":              "operating_system",
		"LicenseModel":                 "license_model",
		"Region":                       "region_code",
		"Hypervisor":                   "",
		"CurrentGeneration":            "current_generation",
		"PhysicalProcessor":            "physical_processor",
		"ClockSpeed":                   "clock_speed",
		"ProcessorArchitecture":        "processor_architecture",
		"SupportedArchitectures":       "",
		"ENASupported":                 "enhanced_networking_supported",
		"EncryptionInTransitSupported": "",
		"SupportedRootDeviceTypes":     "",
		"Cores":                        "",
		"Threads":                      "",
		"vCPU":                         "v_cpu",
		"MemoryGB":                     "memory_gb",
	}

	PreferenceSpecialCond = map[string]string{
		"vCPU":     ">=",
		"MemoryGB": ">=",
	}
)