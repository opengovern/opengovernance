package ec2instance

var (
	PreferenceDBKey = map[string]string{
		"Tenancy":                      "tenancy",
		"EBSOptimized":                 "ebs_optimized",
		"OperatingSystem":              "operating_system_family",
		"LicenseModel":                 "license_model",
		"Region":                       "region_code",
		"Hypervisor":                   "",
		"CurrentGeneration":            "current_generation",
		"PhysicalProcessor":            "physical_processor",
		"ClockSpeed":                   "clock_speed",
		"ProcessorArchitecture":        "physical_processor_arch",
		"SupportedArchitectures":       "",
		"ENASupport":                   "enhanced_networking_supported",
		"EncryptionInTransitSupported": "",
		"SupportedRootDeviceTypes":     "",
		"Cores":                        "",
		"Threads":                      "",
		"vCPU":                         "v_cpu",
		"MemoryGB":                     "memory_gb",
		"InstanceFamily":               "instance_family",
		"UsageOperation":               "operation",
	}

	PreferenceSpecialCond = map[string]string{
		"vCPU":     ">=",
		"MemoryGB": ">=",
	}

	UsageOperationHumanToMachine = map[string]string{
		"Linux/UNIX":                       "RunInstances",
		"Red Hat BYOL Linux":               "RunInstances:00g0",
		"Red Hat Enterprise Linux":         "RunInstances:0010",
		"Red Hat Enterprise Linux with HA": "RunInstances:1010",
		"Red Hat Enterprise Linux with SQL Server Standard and HA":   "RunInstances:1014",
		"Red Hat Enterprise Linux with SQL Server Enterprise and HA": "RunInstances:1110",
		"Red Hat Enterprise Linux with SQL Server Standard":          "RunInstances:0014",
		"Red Hat Enterprise Linux with SQL Server Web":               "RunInstances:0210",
		"Red Hat Enterprise Linux with SQL Server Enterprise":        "RunInstances:0110",
		"SQL Server Enterprise":                                      "RunInstances:0100",
		"SQL Server Standard":                                        "RunInstances:0004",
		"SQL Server Web":                                             "RunInstances:0200",
		"SUSE Linux":                                                 "RunInstances:000g",
		"Ubuntu Pro":                                                 "RunInstances:0g00",
		"Windows":                                                    "RunInstances:0002",
		"Windows BYOL":                                               "RunInstances:0800",
		"Windows with SQL Server Enterprise":                         "RunInstances:0102",
		"Windows with SQL Server Standard":                           "RunInstances:0006",
		"Windows with SQL Server Web":                                "RunInstances:0202",
	}
)
