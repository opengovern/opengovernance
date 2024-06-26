package recommendation

import (
	"fmt"
	"strconv"
	"strings"
)

type DiskLimitationsPerVm struct {
	MaxWriteIOPS       float64 `json:"max_write_iops"`
	MaxReadIOPS        float64 `json:"max_read_iops"`
	MaxWriteThroughput float64 `json:"max_write_throughput"` // MiBps
	MaxReadThroughput  float64 `json:"max_read_throughput"`  // MiBps
}

type DiskLimitations struct {
	WriteIOPS  float64 `json:"max_write_iops"`
	ReadIOPS   float64 `json:"max_read_iops"`
	Throughput float64 `json:"max_write_throughput"` // MiBps
}

var (
	DiskLimitationsPerGb = map[string]DiskLimitations{
		"pd-standard": {
			WriteIOPS:  1.5,
			ReadIOPS:   0.75,
			Throughput: 0.12,
		},
		"pd-balanced": {
			WriteIOPS:  6,
			ReadIOPS:   6,
			Throughput: 0.28,
		},
		"pd-ssd": {
			WriteIOPS:  30,
			ReadIOPS:   30,
			Throughput: 0.48,
		},
	}
)

// diskTypes sorted by cost per GB: pd-standard, pd-balanced, pd-extreme, pd-ssd
func findCheapestDiskType(machineFamily, machineType string, vCPUs int64, neededReadIops, neededWriteIops,
	neededReadThroughput, neededWriteThroughput float64, sizeGb int64) (string, error) {
	limitations := findLimitations(machineFamily, machineType, vCPUs)
	if len(limitations) == 0 {
		return "", fmt.Errorf("could not find limitations")
	}

	// pd-standard
	l := limitations["pd-standard"]
	maxReadIops := min(l.MaxReadIOPS, float64(sizeGb)*DiskLimitationsPerGb["pd-standard"].ReadIOPS)
	maxWriteIops := min(l.MaxWriteIOPS, float64(sizeGb)*DiskLimitationsPerGb["pd-standard"].WriteIOPS)
	maxReadThroughput := min(l.MaxReadThroughput, float64(sizeGb)*DiskLimitationsPerGb["pd-standard"].Throughput)
	maxWriteThroughput := min(l.MaxWriteThroughput, float64(sizeGb)*DiskLimitationsPerGb["pd-standard"].Throughput)
	if neededReadIops <= maxReadIops && neededWriteIops <= maxWriteIops &&
		neededReadThroughput <= maxReadThroughput && neededWriteThroughput <= maxWriteThroughput {
		return "pd-standard", nil
	}

	// pd-balanced
	l = limitations["pd-balanced"]
	maxReadIops = min(l.MaxReadIOPS, 3000+float64(sizeGb)*DiskLimitationsPerGb["pd-balanced"].ReadIOPS)
	maxWriteIops = min(l.MaxWriteIOPS, 3000+float64(sizeGb)*DiskLimitationsPerGb["pd-balanced"].WriteIOPS)
	maxReadThroughput = min(l.MaxReadThroughput, 140+float64(sizeGb)*DiskLimitationsPerGb["pd-balanced"].Throughput)
	maxWriteThroughput = min(l.MaxWriteThroughput, 140+float64(sizeGb)*DiskLimitationsPerGb["pd-balanced"].Throughput)
	if neededReadIops <= maxReadIops && neededWriteIops <= maxWriteIops &&
		neededReadThroughput <= maxReadThroughput && neededWriteThroughput <= maxWriteThroughput {
		return "pd-balanced", nil
	}

	// pd-extreme
	l = limitations["pd-extreme"]
	if neededReadIops <= l.MaxReadIOPS && neededWriteIops <= l.MaxWriteIOPS &&
		neededReadThroughput <= l.MaxReadThroughput && neededWriteThroughput <= l.MaxWriteThroughput {
		return "pd-extreme", nil
	}

	// pd-ssd
	l = limitations["pd-ssd"]
	maxReadIops = min(l.MaxReadIOPS, 6000+float64(sizeGb)*DiskLimitationsPerGb["pd-ssd"].ReadIOPS)
	maxWriteIops = min(l.MaxWriteIOPS, 6000+float64(sizeGb)*DiskLimitationsPerGb["pd-ssd"].WriteIOPS)
	maxReadThroughput = min(l.MaxReadThroughput, 240+float64(sizeGb)*DiskLimitationsPerGb["pd-ssd"].Throughput)
	maxWriteThroughput = min(l.MaxWriteThroughput, 240+float64(sizeGb)*DiskLimitationsPerGb["pd-ssd"].Throughput)
	if neededReadIops <= maxReadIops && neededWriteIops <= maxWriteIops &&
		neededReadThroughput <= maxReadThroughput && neededWriteThroughput <= maxWriteThroughput {
		return "pd-ssd", nil
	}
	return "", fmt.Errorf("could not find sutiable disk type")
}

func findLimitations(machineFamily, machineType string, vCPUs int64) map[string]DiskLimitationsPerVm {
	limitations := make(map[string]DiskLimitationsPerVm)
	if machineFamily == "n2" {
		limitations["pd-extreme"] = machineTypeDiskLimitations[machineFamily][machineType]["pd-extreme"]
		for k, v := range machineTypeDiskLimitationsPerCPURange[machineFamily] {
			r := strings.Split(k, "-")
			min, _ := strconv.ParseInt(r[0], 10, 64)
			if vCPUs >= min {
				if len(r) == 1 {
					limitations["pd-balanced"] = v["pd-balanced"]
					limitations["pd-ssd"] = v["pd-ssd"]
					limitations["pd-standard"] = v["pd-standard"]
					break
				} else {
					max, _ := strconv.ParseInt(r[1], 10, 64)
					if vCPUs <= max {
						limitations["pd-balanced"] = v["pd-balanced"]
						limitations["pd-ssd"] = v["pd-ssd"]
						limitations["pd-standard"] = v["pd-standard"]
						break
					}
				}
			}
		}
		return limitations
	}
	if l, ok := machineTypeDiskLimitations[machineFamily][machineType]; ok {
		return l
	}
	if l, ok := machineTypeDiskLimitationsPerCPU[machineFamily]; ok {
		for k, v := range l {
			if k == vCPUs {
				return v
			}
		}
	}
	if l, ok := machineTypeDiskLimitationsPerCPURange[machineFamily]; ok {
		for k, v := range l {
			r := strings.Split(k, "-")
			min, _ := strconv.ParseInt(r[0], 10, 64)
			if vCPUs >= min {
				if len(r) == 1 {
					return v
				}
				max, _ := strconv.ParseInt(r[1], 10, 64)
				if vCPUs <= max {
					return v
				}
			}
		}
	}
	return limitations
}

// MachineTypeDiskLimitations is a map of machine types to disk types to disk limitations.
var (
	machineTypeDiskLimitations = map[string]map[string]map[string]DiskLimitationsPerVm{
		"a3": {
			"a3-megagpu-8g": {
				"pd-ssd": {
					MaxWriteIOPS:       80000,
					MaxReadIOPS:        80000,
					MaxWriteThroughput: 1200,
					MaxReadThroughput:  1200,
				},
				"pd-balanced": {
					MaxWriteIOPS:       80000,
					MaxReadIOPS:        80000,
					MaxWriteThroughput: 1200,
					MaxReadThroughput:  1200,
				},
			},
			"a3-highgpu-8g": {
				"pd-ssd": {
					MaxWriteIOPS:       80000,
					MaxReadIOPS:        80000,
					MaxWriteThroughput: 1200,
					MaxReadThroughput:  1200,
				},
				"pd-balanced": {
					MaxWriteIOPS:       80000,
					MaxReadIOPS:        80000,
					MaxWriteThroughput: 1200,
					MaxReadThroughput:  1200,
				},
			},
		},
		"a2": {
			"a2-highgpu-1g": {
				"pd-balanced": {
					MaxWriteIOPS:       15000,
					MaxReadIOPS:        15000,
					MaxWriteThroughput: 800,
					MaxReadThroughput:  800,
				},
				"pd-ssd": {
					MaxWriteIOPS:       15000,
					MaxReadIOPS:        15000,
					MaxWriteThroughput: 800,
					MaxReadThroughput:  800,
				},
				"pd-standard": {
					MaxWriteIOPS:       15000,
					MaxReadIOPS:        5000,
					MaxWriteThroughput: 400,
					MaxReadThroughput:  800,
				},
			},
			"a2-highgpu-2g": {
				"pd-balanced": {
					MaxWriteIOPS:       20000,
					MaxReadIOPS:        20000,
					MaxWriteThroughput: 1200,
					MaxReadThroughput:  1200,
				},
				"pd-ssd": {
					MaxWriteIOPS:       25000,
					MaxReadIOPS:        25000,
					MaxWriteThroughput: 1200,
					MaxReadThroughput:  1200,
				},
				"pd-standard": {
					MaxWriteIOPS:       15000,
					MaxReadIOPS:        5000,
					MaxWriteThroughput: 400,
					MaxReadThroughput:  1200,
				},
			},
			"a2-highgpu-4g": {
				"pd-balanced": {
					MaxWriteIOPS:       50000,
					MaxReadIOPS:        50000,
					MaxWriteThroughput: 1200,
					MaxReadThroughput:  1200,
				},
				"pd-ssd": {
					MaxWriteIOPS:       60000,
					MaxReadIOPS:        60000,
					MaxWriteThroughput: 1200,
					MaxReadThroughput:  1200,
				},
				"pd-standard": {
					MaxWriteIOPS:       15000,
					MaxReadIOPS:        5000,
					MaxWriteThroughput: 400,
					MaxReadThroughput:  1200,
				},
			},
			"a2-highgpu-8g": {
				"pd-balanced": {
					MaxWriteIOPS:       80000,
					MaxReadIOPS:        80000,
					MaxWriteThroughput: 1200,
					MaxReadThroughput:  1200,
				},
				"pd-ssd": {
					MaxWriteIOPS:       100000,
					MaxReadIOPS:        100000,
					MaxWriteThroughput: 1200,
					MaxReadThroughput:  1200,
				},
				"pd-standard": {
					MaxWriteIOPS:       15000,
					MaxReadIOPS:        5000,
					MaxWriteThroughput: 400,
					MaxReadThroughput:  1200,
				},
			},
			"a2-megagpu-16g": {
				"pd-balanced": {
					MaxWriteIOPS:       80000,
					MaxReadIOPS:        80000,
					MaxWriteThroughput: 1200,
					MaxReadThroughput:  1200,
				},
				"pd-ssd": {
					MaxWriteIOPS:       100000,
					MaxReadIOPS:        100000,
					MaxWriteThroughput: 1200,
					MaxReadThroughput:  1200,
				},
				"pd-standard": {
					MaxWriteIOPS:       15000,
					MaxReadIOPS:        5000,
					MaxWriteThroughput: 400,
					MaxReadThroughput:  1200,
				},
			},
			"a2-ultragpu-1g": {
				"pd-balanced": {
					MaxWriteIOPS:       15000,
					MaxReadIOPS:        15000,
					MaxWriteThroughput: 800,
					MaxReadThroughput:  800,
				},
				"pd-ssd": {
					MaxWriteIOPS:       15000,
					MaxReadIOPS:        15000,
					MaxWriteThroughput: 800,
					MaxReadThroughput:  800,
				},
				"pd-standard": {
					MaxWriteIOPS:       15000,
					MaxReadIOPS:        5000,
					MaxWriteThroughput: 400,
					MaxReadThroughput:  800,
				},
			},
			"a2-ultragpu-2g": {
				"pd-balanced": {
					MaxWriteIOPS:       20000,
					MaxReadIOPS:        20000,
					MaxWriteThroughput: 1200,
					MaxReadThroughput:  1200,
				},
				"pd-ssd": {
					MaxWriteIOPS:       25000,
					MaxReadIOPS:        25000,
					MaxWriteThroughput: 1200,
					MaxReadThroughput:  1200,
				},
				"pd-standard": {
					MaxWriteIOPS:       15000,
					MaxReadIOPS:        5000,
					MaxWriteThroughput: 400,
					MaxReadThroughput:  1200,
				},
			},
			"a2-ultragpu-4g": {
				"pd-balanced": {
					MaxWriteIOPS:       50000,
					MaxReadIOPS:        50000,
					MaxWriteThroughput: 1200,
					MaxReadThroughput:  1200,
				},
				"pd-ssd": {
					MaxWriteIOPS:       60000,
					MaxReadIOPS:        60000,
					MaxWriteThroughput: 1200,
					MaxReadThroughput:  1200,
				},
				"pd-standard": {
					MaxWriteIOPS:       15000,
					MaxReadIOPS:        5000,
					MaxWriteThroughput: 400,
					MaxReadThroughput:  1200,
				},
			},
			"a2-ultragpu-8g": {
				"pd-balanced": {
					MaxWriteIOPS:       80000,
					MaxReadIOPS:        80000,
					MaxWriteThroughput: 1200,
					MaxReadThroughput:  1200,
				},
				"pd-ssd": {
					MaxWriteIOPS:       100000,
					MaxReadIOPS:        100000,
					MaxWriteThroughput: 1200,
					MaxReadThroughput:  1200,
				},
				"pd-standard": {
					MaxWriteIOPS:       15000,
					MaxReadIOPS:        5000,
					MaxWriteThroughput: 400,
					MaxReadThroughput:  1200,
				},
			},
		},
		"g2": {
			"g2-standard-4": {
				"pd-balanced": {
					MaxWriteIOPS:       15000,
					MaxReadIOPS:        15000,
					MaxWriteThroughput: 240,
					MaxReadThroughput:  240,
				},
				"pd-ssd": {
					MaxWriteIOPS:       15000,
					MaxReadIOPS:        15000,
					MaxWriteThroughput: 240,
					MaxReadThroughput:  240,
				},
			},
			"g2-standard-8": {
				"pd-balanced": {
					MaxWriteIOPS:       15000,
					MaxReadIOPS:        15000,
					MaxWriteThroughput: 800,
					MaxReadThroughput:  800,
				},
				"pd-ssd": {
					MaxWriteIOPS:       15000,
					MaxReadIOPS:        15000,
					MaxWriteThroughput: 800,
					MaxReadThroughput:  800,
				},
			},
			"g2-standard-12": {
				"pd-balanced": {
					MaxWriteIOPS: 15000,

					MaxReadIOPS:        15000,
					MaxWriteThroughput: 800,
					MaxReadThroughput:  800,
				},
				"pd-ssd": {
					MaxWriteIOPS:       15000,
					MaxReadIOPS:        15000,
					MaxWriteThroughput: 800,
					MaxReadThroughput:  800,
				},
			},
			"g2-standard-16": {
				"pd-balanced": {
					MaxWriteIOPS:       20000,
					MaxReadIOPS:        20000,
					MaxWriteThroughput: 1200,
					MaxReadThroughput:  1200,
				},
				"pd-ssd": {
					MaxWriteIOPS:       25000,
					MaxReadIOPS:        25000,
					MaxWriteThroughput: 1200,
					MaxReadThroughput:  1200,
				},
			},
			"g2-standard-24": {
				"pd-balanced": {
					MaxWriteIOPS:       20000,
					MaxReadIOPS:        20000,
					MaxWriteThroughput: 1200,
					MaxReadThroughput:  1200,
				},
				"pd-ssd": {
					MaxWriteIOPS:       25000,
					MaxReadIOPS:        25000,
					MaxWriteThroughput: 1200,
					MaxReadThroughput:  1200,
				},
			},
			"g2-standard-32": {
				"pd-balanced": {
					MaxWriteIOPS:       50000,
					MaxReadIOPS:        50000,
					MaxWriteThroughput: 1200,
					MaxReadThroughput:  1200,
				},
				"pd-ssd": {
					MaxWriteIOPS:       60000,
					MaxReadIOPS:        60000,
					MaxWriteThroughput: 1200,
					MaxReadThroughput:  1200,
				},
			},
			"g2-standard-48": {
				"pd-balanced": {
					MaxWriteIOPS:       50000,
					MaxReadIOPS:        50000,
					MaxWriteThroughput: 1200,
					MaxReadThroughput:  1200,
				},
				"pd-ssd": {
					MaxWriteIOPS:       60000,
					MaxReadIOPS:        60000,
					MaxWriteThroughput: 1200,
					MaxReadThroughput:  1200,
				},
			},
			"g2-standard-96": {
				"pd-balanced": {
					MaxWriteIOPS:       80000,
					MaxReadIOPS:        80000,
					MaxWriteThroughput: 1200,
					MaxReadThroughput:  1200,
				},
				"pd-ssd": {
					MaxWriteIOPS:       100000,
					MaxReadIOPS:        100000,
					MaxWriteThroughput: 1200,
					MaxReadThroughput:  1200,
				},
			},
		},
		"m1": {
			"m1-megamem-96": {
				"pd-balanced": {
					MaxWriteIOPS:       80000,
					MaxReadIOPS:        80000,
					MaxWriteThroughput: 1200,
					MaxReadThroughput:  1200,
				},
				"pd-ssd": {
					MaxWriteIOPS:       90000,
					MaxReadIOPS:        90000,
					MaxWriteThroughput: 1200,
					MaxReadThroughput:  1200,
				},
				"pd-standard": {
					MaxWriteIOPS:       15000,
					MaxReadIOPS:        7500,
					MaxWriteThroughput: 400,
					MaxReadThroughput:  1200,
				},
				"pd-extreme": {
					MaxWriteIOPS:       90000,
					MaxReadIOPS:        90000,
					MaxWriteThroughput: 2200,
					MaxReadThroughput:  2200,
				},
			},
			"m1-ultramem-40": {
				"pd-balanced": {
					MaxWriteIOPS:       60000,
					MaxReadIOPS:        60000,
					MaxWriteThroughput: 1200,
					MaxReadThroughput:  1200,
				},
				"pd-ssd": {
					MaxWriteIOPS:       60000,
					MaxReadIOPS:        60000,
					MaxWriteThroughput: 1200,
					MaxReadThroughput:  1200,
				},
				"pd-standard": {
					MaxWriteIOPS:       15000,
					MaxReadIOPS:        7500,
					MaxWriteThroughput: 400,
					MaxReadThroughput:  1200,
				},
			},
			"m1-ultramem-80": {
				"pd-balanced": {
					MaxWriteIOPS:       70000,
					MaxReadIOPS:        70000,
					MaxWriteThroughput: 1200,
					MaxReadThroughput:  1200,
				},
				"pd-ssd": {
					MaxWriteIOPS:       70000,
					MaxReadIOPS:        70000,
					MaxWriteThroughput: 1200,
					MaxReadThroughput:  1200,
				},
				"pd-standard": {
					MaxWriteIOPS:       15000,
					MaxReadIOPS:        7500,
					MaxWriteThroughput: 400,
					MaxReadThroughput:  1200,
				},
			},
			"m1-ultramem-160": {
				"pd-balanced": {
					MaxWriteIOPS:       70000,
					MaxReadIOPS:        70000,
					MaxWriteThroughput: 1200,
					MaxReadThroughput:  1200,
				},
				"pd-ssd": {
					MaxWriteIOPS:       70000,
					MaxReadIOPS:        70000,
					MaxWriteThroughput: 1200,
					MaxReadThroughput:  1200,
				},
				"pd-standard": {
					MaxWriteIOPS:       15000,
					MaxReadIOPS:        7500,
					MaxWriteThroughput: 400,
					MaxReadThroughput:  1200,
				},
			},
		},
		"m2": {
			"m2-megamem-416": {
				"pd-balanced": {
					MaxWriteIOPS:       40000,
					MaxReadIOPS:        40000,
					MaxWriteThroughput: 1200,
					MaxReadThroughput:  1200,
				},
				"pd-ssd": {
					MaxWriteIOPS:       40000,
					MaxReadIOPS:        40000,
					MaxWriteThroughput: 1200,
					MaxReadThroughput:  1200,
				},
				"pd-standard": {
					MaxWriteIOPS:       15000,
					MaxReadIOPS:        7500,
					MaxWriteThroughput: 400,
					MaxReadThroughput:  1200,
				},
				"pd-extreme": {
					MaxWriteIOPS: 40000,
					MaxReadIOPS:  40000,

					MaxWriteThroughput: 1200,
					MaxReadThroughput:  1200,
				},
			},
			"m2-ultramem-208": {
				"pd-balanced": {
					MaxWriteIOPS:       40000,
					MaxReadIOPS:        40000,
					MaxWriteThroughput: 1200,
					MaxReadThroughput:  1200,
				},
				"pd-ssd": {
					MaxWriteIOPS:       40000,
					MaxReadIOPS:        40000,
					MaxWriteThroughput: 1200,
					MaxReadThroughput:  1200,
				},
				"pd-standard": {
					MaxWriteIOPS:       15000,
					MaxReadIOPS:        7500,
					MaxWriteThroughput: 400,
					MaxReadThroughput:  1200,
				},
				"pd-extreme": {
					MaxWriteIOPS:       40000,
					MaxReadIOPS:        40000,
					MaxWriteThroughput: 1200,
					MaxReadThroughput:  1200,
				},
			},
			"m2-ultramem-416": {
				"pd-balanced": {
					MaxWriteIOPS:       40000,
					MaxReadIOPS:        40000,
					MaxWriteThroughput: 1200,
					MaxReadThroughput:  1200,
				},
				"pd-ssd": {
					MaxWriteIOPS:       40000,
					MaxReadIOPS:        40000,
					MaxWriteThroughput: 1200,
					MaxReadThroughput:  1200,
				},
				"pd-standard": {
					MaxWriteIOPS:       15000,
					MaxReadIOPS:        7500,
					MaxWriteThroughput: 400,
					MaxReadThroughput:  1200,
				},
				"pd-extreme": {
					MaxWriteIOPS:       40000,
					MaxReadIOPS:        40000,
					MaxWriteThroughput: 1200,
					MaxReadThroughput:  1200,
				},
			},
			"m2-hypermem-416": {
				"pd-balanced": {
					MaxWriteIOPS:       40000,
					MaxReadIOPS:        40000,
					MaxWriteThroughput: 1200,
					MaxReadThroughput:  1200,
				},
				"pd-ssd": {
					MaxWriteIOPS:       40000,
					MaxReadIOPS:        40000,
					MaxWriteThroughput: 1200,
					MaxReadThroughput:  1200,
				},
				"pd-standard": {
					MaxWriteIOPS:       15000,
					MaxReadIOPS:        7500,
					MaxWriteThroughput: 400,
					MaxReadThroughput:  1200,
				},
				"pd-extreme": {
					MaxWriteIOPS:       40000,
					MaxReadIOPS:        40000,
					MaxWriteThroughput: 1200,
					MaxReadThroughput:  1200,
				},
			},
		},
		"m3": {
			"m3-megamem-64": {
				"pd-balanced": {
					MaxWriteIOPS:       40000,
					MaxReadIOPS:        40000,
					MaxWriteThroughput: 1200,
					MaxReadThroughput:  1200,
				},
				"pd-ssd": {
					MaxWriteIOPS:       40000,
					MaxReadIOPS:        40000,
					MaxWriteThroughput: 1200,
					MaxReadThroughput:  1200,
				},
				"pd-extreme": {
					MaxWriteIOPS:       40000,
					MaxReadIOPS:        40000,
					MaxWriteThroughput: 1200,
					MaxReadThroughput:  2200,
				},
			},
			"m3-megamem-128": {
				"pd-balanced": {
					MaxWriteIOPS:       80000,
					MaxReadIOPS:        80000,
					MaxWriteThroughput: 1200,
					MaxReadThroughput:  1200,
				},
				"pd-ssd": {
					MaxWriteIOPS:       80000,
					MaxReadIOPS:        80000,
					MaxWriteThroughput: 1200,
					MaxReadThroughput:  1200,
				},
				"pd-extreme": {
					MaxWriteIOPS:       80000,
					MaxReadIOPS:        80000,
					MaxWriteThroughput: 1700,
					MaxReadThroughput:  2200,
				},
			},
			"m3-ultramem-32": {
				"pd-balanced": {
					MaxWriteIOPS:       40000,
					MaxReadIOPS:        40000,
					MaxWriteThroughput: 1200,
					MaxReadThroughput:  1200,
				},
				"pd-ssd": {
					MaxWriteIOPS:       40000,
					MaxReadIOPS:        40000,
					MaxWriteThroughput: 1200,
					MaxReadThroughput:  1200,
				},
				"pd-extreme": {
					MaxWriteIOPS:       40000,
					MaxReadIOPS:        40000,
					MaxWriteThroughput: 1200,
					MaxReadThroughput:  2200,
				},
			},
			"m3-ultramem-64": {
				"pd-balanced": {
					MaxWriteIOPS:       40000,
					MaxReadIOPS:        40000,
					MaxWriteThroughput: 1200,
					MaxReadThroughput:  1200,
				},
				"pd-ssd": {
					MaxWriteIOPS:       40000,
					MaxReadIOPS:        40000,
					MaxWriteThroughput: 1200,
					MaxReadThroughput:  1200,
				},
				"pd-extreme": {
					MaxWriteIOPS:       40000,
					MaxReadIOPS:        40000,
					MaxWriteThroughput: 1200,
					MaxReadThroughput:  2200,
				},
			},
			"m3-ultramem-128": {
				"pd-balanced": {
					MaxWriteIOPS:       80000,
					MaxReadIOPS:        80000,
					MaxWriteThroughput: 1200,
					MaxReadThroughput:  1200,
				},
				"pd-ssd": {
					MaxWriteIOPS:       80000,
					MaxReadIOPS:        80000,
					MaxWriteThroughput: 1200,
					MaxReadThroughput:  1200,
				},
				"pd-extreme": {
					MaxWriteIOPS: 80000,

					MaxReadIOPS:        80000,
					MaxWriteThroughput: 1700,
					MaxReadThroughput:  2200,
				},
			},
		},
		"n2": {
			"n2-standard-64": {
				"pd-extreme": {
					MaxWriteIOPS:       120000,
					MaxReadIOPS:        120000,
					MaxWriteThroughput: 3000,
					MaxReadThroughput:  4000,
				},
			},
			"n2-standard-80": {
				"pd-extreme": {
					MaxWriteIOPS:       120000,
					MaxReadIOPS:        120000,
					MaxWriteThroughput: 3000,
					MaxReadThroughput:  4000,
				},
			},
			"n2-standard-96": {
				"pd-extreme": {
					MaxWriteIOPS:       120000,
					MaxReadIOPS:        120000,
					MaxWriteThroughput: 3000,
					MaxReadThroughput:  4000,
				},
			},
			"n2-standard-128": {
				"pd-extreme": {
					MaxWriteIOPS:       120000,
					MaxReadIOPS:        120000,
					MaxWriteThroughput: 3000,
					MaxReadThroughput:  4000,
				},
			},
			"n2-highmem-64": {
				"pd-extreme": {
					MaxWriteIOPS:       120000,
					MaxReadIOPS:        120000,
					MaxWriteThroughput: 3000,
					MaxReadThroughput:  4000,
				},
			},
			"n2-highmem-80": {
				"pd-extreme": {
					MaxWriteIOPS:       120000,
					MaxReadIOPS:        120000,
					MaxWriteThroughput: 3000,
					MaxReadThroughput:  4000,
				},
			},
			"n2-highmem-96": {
				"pd-extreme": {
					MaxWriteIOPS:       120000,
					MaxReadIOPS:        120000,
					MaxWriteThroughput: 3000,
					MaxReadThroughput:  4000,
				},
			},
			"n2-highmem-128": {
				"pd-extreme": {
					MaxWriteIOPS:       120000,
					MaxReadIOPS:        120000,
					MaxWriteThroughput: 3000,
					MaxReadThroughput:  4000,
				},
			},
			"n2-highcpu-64": {
				"pd-extreme": {
					MaxWriteIOPS:       120000,
					MaxReadIOPS:        120000,
					MaxWriteThroughput: 3000,
					MaxReadThroughput:  4000,
				},
			},
			"n2-highcpu-80": {
				"pd-extreme": {
					MaxWriteIOPS:       120000,
					MaxReadIOPS:        120000,
					MaxWriteThroughput: 3000,
					MaxReadThroughput:  4000,
				},
			},
			"n2-highcpu-96": {
				"pd-extreme": {
					MaxWriteIOPS:       120000,
					MaxReadIOPS:        120000,
					MaxWriteThroughput: 3000,
					MaxReadThroughput:  4000,
				},
			},
		},
	}
	machineTypeDiskLimitationsPerCPU = map[string]map[int64]map[string]DiskLimitationsPerVm{
		"c2": {
			4: {
				"pd-balanced": {
					MaxWriteIOPS:       4000,
					MaxReadIOPS:        4000,
					MaxWriteThroughput: 240,
					MaxReadThroughput:  240,
				},
				"pd-ssd": {
					MaxWriteIOPS:       4000,
					MaxReadIOPS:        4000,
					MaxWriteThroughput: 240,
					MaxReadThroughput:  240,
				},
				"pd-standard": {
					MaxWriteIOPS:       4000,
					MaxReadIOPS:        3000,
					MaxWriteThroughput: 240,
					MaxReadThroughput:  240,
				},
			},
			8: {
				"pd-balanced": {
					MaxWriteIOPS:       4000,
					MaxReadIOPS:        4000,
					MaxWriteThroughput: 240,
					MaxReadThroughput:  240,
				},
				"pd-ssd": {
					MaxWriteIOPS:       4000,
					MaxReadIOPS:        4000,
					MaxWriteThroughput: 240,
					MaxReadThroughput:  240,
				},
				"pd-standard": {
					MaxWriteIOPS:       4000,
					MaxReadIOPS:        3000,
					MaxWriteThroughput: 240,
					MaxReadThroughput:  240,
				},
			},
			16: {
				"pd-balanced": {
					MaxWriteIOPS:       4000,
					MaxReadIOPS:        8000,
					MaxWriteThroughput: 480,
					MaxReadThroughput:  600,
				},
				"pd-ssd": {
					MaxWriteIOPS:       4000,
					MaxReadIOPS:        8000,
					MaxWriteThroughput: 480,
					MaxReadThroughput:  600,
				},
				"pd-standard": {
					MaxWriteIOPS:       4000,
					MaxReadIOPS:        3000,
					MaxWriteThroughput: 240,
					MaxReadThroughput:  240,
				},
			},
			30: {
				"pd-balanced": {
					MaxWriteIOPS:       8000,
					MaxReadIOPS:        15000,
					MaxWriteThroughput: 480,
					MaxReadThroughput:  600,
				},
				"pd-ssd": {
					MaxWriteIOPS:       8000,
					MaxReadIOPS:        15000,
					MaxWriteThroughput: 480,
					MaxReadThroughput:  600,
				},
				"pd-standard": {
					MaxWriteIOPS:       8000,
					MaxReadIOPS:        3000,
					MaxWriteThroughput: 240,
					MaxReadThroughput:  240,
				},
			},
			60: {
				"pd-balanced": {
					MaxWriteIOPS:       15000,
					MaxReadIOPS:        15000,
					MaxWriteThroughput: 800,
					MaxReadThroughput:  1200,
				},
				"pd-ssd": {
					MaxWriteIOPS:       15000,
					MaxReadIOPS:        30000,
					MaxWriteThroughput: 800,
					MaxReadThroughput:  1200,
				},
				"pd-standard": {
					MaxWriteIOPS:       15000,
					MaxReadIOPS:        3000,
					MaxWriteThroughput: 240,
					MaxReadThroughput:  240,
				},
			},
		},
		"c2d": {
			2: {
				"pd-balanced": {
					MaxWriteIOPS:       4590,
					MaxReadIOPS:        4080,
					MaxWriteThroughput: 245,
					MaxReadThroughput:  245,
				},
				"pd-ssd": {
					MaxWriteIOPS:       4590,
					MaxReadIOPS:        4080,
					MaxWriteThroughput: 245,
					MaxReadThroughput:  245,
				},
				"pd-standard": {
					MaxWriteIOPS:       4590,
					MaxReadIOPS:        3060,
					MaxWriteThroughput: 245,
					MaxReadThroughput:  245,
				},
			},
			4: {
				"pd-balanced": {
					MaxWriteIOPS:       4590,
					MaxReadIOPS:        4080,
					MaxWriteThroughput: 245,
					MaxReadThroughput:  245,
				},
				"pd-ssd": {
					MaxWriteIOPS:       4590,
					MaxReadIOPS:        4080,
					MaxWriteThroughput: 245,
					MaxReadThroughput:  245,
				},
				"pd-standard": {
					MaxWriteIOPS:       4590,
					MaxReadIOPS:        3060,
					MaxWriteThroughput: 245,
					MaxReadThroughput:  245,
				},
			},
			8: {
				"pd-balanced": {
					MaxWriteIOPS:       4590,
					MaxReadIOPS:        4080,
					MaxWriteThroughput: 245,
					MaxReadThroughput:  245,
				},
				"pd-ssd": {
					MaxWriteIOPS:       4590,
					MaxReadIOPS:        4080,
					MaxWriteThroughput: 245,
					MaxReadThroughput:  245,
				},
				"pd-standard": {
					MaxWriteIOPS:       4590,
					MaxReadIOPS:        3060,
					MaxWriteThroughput: 245,
					MaxReadThroughput:  245,
				},
			},
			16: {
				"pd-balanced": {
					MaxWriteIOPS:       4590,
					MaxReadIOPS:        8160,
					MaxWriteThroughput: 245,
					MaxReadThroughput:  326,
				},
				"pd-ssd": {
					MaxWriteIOPS:       4590,
					MaxReadIOPS:        8160,
					MaxWriteThroughput: 245,
					MaxReadThroughput:  326,
				},
				"pd-standard": {
					MaxWriteIOPS:       4590,
					MaxReadIOPS:        3060,
					MaxWriteThroughput: 245,
					MaxReadThroughput:  245,
				},
			},
			32: {
				"pd-balanced": {

					MaxWriteIOPS:       8160,
					MaxReadIOPS:        15300,
					MaxWriteThroughput: 245,
					MaxReadThroughput:  612,
				},
				"pd-ssd": {
					MaxWriteIOPS:       8160,
					MaxReadIOPS:        15300,
					MaxWriteThroughput: 245,
					MaxReadThroughput:  612,
				},
				"pd-standard": {
					MaxWriteIOPS:       8160,
					MaxReadIOPS:        3060,
					MaxWriteThroughput: 245,
					MaxReadThroughput:  245,
				},
			},
			56: {
				"pd-balanced": {
					MaxWriteIOPS:       8160,
					MaxReadIOPS:        15300,
					MaxWriteThroughput: 245,
					MaxReadThroughput:  612,
				},
				"pd-ssd": {
					MaxWriteIOPS:       8160,
					MaxReadIOPS:        15300,
					MaxWriteThroughput: 245,
					MaxReadThroughput:  612,
				},
				"pd-standard": {
					MaxWriteIOPS:       8160,
					MaxReadIOPS:        3060,
					MaxWriteThroughput: 245,
					MaxReadThroughput:  245,
				},
			},
			112: {
				"pd-balanced": {
					MaxWriteIOPS:       15300,
					MaxReadIOPS:        30600,
					MaxWriteThroughput: 408,
					MaxReadThroughput:  1224,
				},
				"pd-ssd": {
					MaxWriteIOPS:       15300,
					MaxReadIOPS:        30600,
					MaxWriteThroughput: 408,
					MaxReadThroughput:  1224,
				},
				"pd-standard": {
					MaxWriteIOPS:       15300,
					MaxReadIOPS:        3060,
					MaxWriteThroughput: 245,
					MaxReadThroughput:  245,
				},
			},
		},
		"c3d": {
			4: {
				"pd-balanced": {
					MaxWriteIOPS:       15000,
					MaxReadIOPS:        15000,
					MaxWriteThroughput: 240,
					MaxReadThroughput:  240,
				},
				"pd-ssd": {
					MaxWriteIOPS:       15000,
					MaxReadIOPS:        15000,
					MaxWriteThroughput: 240,
					MaxReadThroughput:  240,
				},
			},
			8: {
				"pd-balanced": {
					MaxWriteIOPS:       25000,
					MaxReadIOPS:        25000,
					MaxWriteThroughput: 800,
					MaxReadThroughput:  800,
				},
				"pd-ssd": {
					MaxWriteIOPS:       25000,
					MaxReadIOPS:        25000,
					MaxWriteThroughput: 800,
					MaxReadThroughput:  800,
				},
			},
			16: {
				"pd-balanced": {
					MaxWriteIOPS:       25000,
					MaxReadIOPS:        25000,
					MaxWriteThroughput: 1200,
					MaxReadThroughput:  1200,
				},
				"pd-ssd": {
					MaxWriteIOPS:       25000,
					MaxReadIOPS:        25000,
					MaxWriteThroughput: 1200,
					MaxReadThroughput:  1200,
				},
			},
			30: {
				"pd-balanced": {
					MaxWriteIOPS:       50000,
					MaxReadIOPS:        50000,
					MaxWriteThroughput: 1200,
					MaxReadThroughput:  1200,
				},
				"pd-ssd": {
					MaxWriteIOPS: 50000,
					MaxReadIOPS:  50000,

					MaxWriteThroughput: 1200,
					MaxReadThroughput:  1200,
				},
			},
			60: {
				"pd-balanced": {
					MaxWriteIOPS:       80000,
					MaxReadIOPS:        80000,
					MaxWriteThroughput: 1200,
					MaxReadThroughput:  1200,
				},
				"pd-ssd": {
					MaxWriteIOPS: 80000,

					MaxReadIOPS:        80000,
					MaxWriteThroughput: 1200,
					MaxReadThroughput:  1200,
				},
			},
			90: {
				"pd-balanced": {
					MaxWriteIOPS:       80000,
					MaxReadIOPS:        80000,
					MaxWriteThroughput: 1200,
					MaxReadThroughput:  1200,
				},
				"pd-ssd": {
					MaxWriteIOPS:       80000,
					MaxReadIOPS:        80000,
					MaxWriteThroughput: 1200,
					MaxReadThroughput:  1200,
				},
			},
			180: {
				"pd-balanced": {
					MaxWriteIOPS:       80000,
					MaxReadIOPS:        80000,
					MaxWriteThroughput: 2200,
					MaxReadThroughput:  2200,
				},
				"pd-ssd": {
					MaxWriteIOPS:       80000,
					MaxReadIOPS:        80000,
					MaxWriteThroughput: 2200,
					MaxReadThroughput:  2200,
				},
			},
			360: {
				"pd-balanced": {
					MaxWriteIOPS:       80000,
					MaxReadIOPS:        80000,
					MaxWriteThroughput: 2200,
					MaxReadThroughput:  2200,
				},
				"pd-ssd": {
					MaxWriteIOPS: 80000,

					MaxReadIOPS:        80000,
					MaxWriteThroughput: 2200,
					MaxReadThroughput:  2200,
				},
			},
		},
		"c3": {
			4: {
				"pd-balanced": {
					MaxWriteIOPS:       15000,
					MaxReadIOPS:        15000,
					MaxWriteThroughput: 240,
					MaxReadThroughput:  240,
				},
				"pd-ssd": {
					MaxWriteIOPS:       15000,
					MaxReadIOPS:        15000,
					MaxWriteThroughput: 240,
					MaxReadThroughput:  240,
				},
			},
			8: {
				"pd-balanced": {
					MaxWriteIOPS:       15000,
					MaxReadIOPS:        15000,
					MaxWriteThroughput: 240,
					MaxReadThroughput:  240,
				},
				"pd-ssd": {
					MaxWriteIOPS:       15000,
					MaxReadIOPS:        15000,
					MaxWriteThroughput: 240,
					MaxReadThroughput:  240,
				},
			},
			22: {
				"pd-balanced": {

					MaxWriteIOPS:       20000,
					MaxReadIOPS:        20000,
					MaxWriteThroughput: 800,
					MaxReadThroughput:  800,
				},
				"pd-ssd": {
					MaxWriteIOPS:       25000,
					MaxReadIOPS:        25000,
					MaxWriteThroughput: 800,
					MaxReadThroughput:  800,
				},
			},
			44: {
				"pd-balanced": {
					MaxWriteIOPS:       50000,
					MaxReadIOPS:        50000,
					MaxWriteThroughput: 1200,
					MaxReadThroughput:  1200,
				},
				"pd-ssd": {
					MaxWriteIOPS:       60000,
					MaxReadIOPS:        60000,
					MaxWriteThroughput: 1200,
					MaxReadThroughput:  1200,
				},
			},
			88: {
				"pd-balanced": {
					MaxWriteIOPS:       80000,
					MaxReadIOPS:        80000,
					MaxWriteThroughput: 1200,
					MaxReadThroughput:  1200,
				},
				"pd-ssd": {
					MaxWriteIOPS:       80000,
					MaxReadIOPS:        80000,
					MaxWriteThroughput: 1200,
					MaxReadThroughput:  1200,
				},
			},
			176: {
				"pd-balanced": {
					MaxWriteIOPS:       80000,
					MaxReadIOPS:        80000,
					MaxWriteThroughput: 1200,
					MaxReadThroughput:  1200,
				},
				"pd-ssd": {
					MaxWriteIOPS:       80000,
					MaxReadIOPS:        80000,
					MaxWriteThroughput: 1200,
					MaxReadThroughput:  1200,
				},
			},
		},
		"z3": {
			88: {
				"pd-balanced": {
					MaxWriteIOPS:       80000,
					MaxReadIOPS:        80000,
					MaxWriteThroughput: 1200,
					MaxReadThroughput:  1200,
				},
				"pd-ssd": {
					MaxWriteIOPS:       100000,
					MaxReadIOPS:        100000,
					MaxWriteThroughput: 1200,
					MaxReadThroughput:  1200,
				},
			},
			176: {
				"pd-balanced": {
					MaxWriteIOPS:       80000,
					MaxReadIOPS:        80000,
					MaxWriteThroughput: 1200,
					MaxReadThroughput:  1200,
				},
				"pd-ssd": {
					MaxWriteIOPS:       100000,
					MaxReadIOPS:        100000,
					MaxWriteThroughput: 1200,
					MaxReadThroughput:  1200,
				},
			},
		},
		"h3": {
			88: {
				"pd-balanced": {
					MaxWriteIOPS:       15000,
					MaxReadIOPS:        15000,
					MaxWriteThroughput: 240,
					MaxReadThroughput:  240,
				},
			},
		},
	}
	machineTypeDiskLimitationsPerCPURange = map[string]map[string]map[string]DiskLimitationsPerVm{
		"e2": {
			"e2-medium": {
				"pd-balanced": {
					MaxWriteIOPS:       10000,
					MaxReadIOPS:        12000,
					MaxWriteThroughput: 200,
					MaxReadThroughput:  200,
				},
				"pd-ssd": {
					MaxWriteIOPS:       10000,
					MaxReadIOPS:        12000,
					MaxWriteThroughput: 200,
					MaxReadThroughput:  200,
				},
				"pd-standard": {
					MaxWriteIOPS: 10000,

					MaxReadIOPS:        1000,
					MaxWriteThroughput: 200,
					MaxReadThroughput:  200,
				},
			},
			"2-7": {
				"pd-balanced": {
					MaxWriteIOPS:       15000,
					MaxReadIOPS:        15000,
					MaxWriteThroughput: 240,
					MaxReadThroughput:  240,
				},
				"pd-ssd": {
					MaxWriteIOPS:       15000,
					MaxReadIOPS:        15000,
					MaxWriteThroughput: 240,
					MaxReadThroughput:  240,
				},
				"pd-standard": {
					MaxWriteIOPS:       15000,
					MaxReadIOPS:        3000,
					MaxWriteThroughput: 240,
					MaxReadThroughput:  240,
				},
			},
			"8-15": {
				"pd-balanced": {
					MaxWriteIOPS:       15000,
					MaxReadIOPS:        15000,
					MaxWriteThroughput: 800,
					MaxReadThroughput:  800,
				},
				"pd-ssd": {
					MaxWriteIOPS:       15000,
					MaxReadIOPS:        15000,
					MaxWriteThroughput: 800,
					MaxReadThroughput:  800,
				},
				"pd-standard": {
					MaxWriteIOPS:       15000,
					MaxReadIOPS:        5000,
					MaxWriteThroughput: 400,
					MaxReadThroughput:  800,
				},
			},
			"16-31": {
				"pd-balanced": {
					MaxWriteIOPS: 20000,

					MaxReadIOPS:        20000,
					MaxWriteThroughput: 1000,
					MaxReadThroughput:  1200,
				},
				"pd-ssd": {
					MaxWriteIOPS:       25000,
					MaxReadIOPS:        25000,
					MaxWriteThroughput: 1000,
					MaxReadThroughput:  1200,
				},
				"pd-standard": {
					MaxWriteIOPS:       15000,
					MaxReadIOPS:        7500,
					MaxWriteThroughput: 400,
					MaxReadThroughput:  1200,
				},
			},
			"32-": {
				"pd-balanced": {
					MaxWriteIOPS:       50000,
					MaxReadIOPS:        50000,
					MaxWriteThroughput: 1000,
					MaxReadThroughput:  1200,
				},
				"pd-ssd": {
					MaxWriteIOPS: 60000,
					MaxReadIOPS:  60000,

					MaxWriteThroughput: 1000,
					MaxReadThroughput:  1200,
				},
				"pd-standard": {
					MaxWriteIOPS:       15000,
					MaxReadIOPS:        7500,
					MaxWriteThroughput: 400,
					MaxReadThroughput:  1200,
				},
			},
		},
		"n1": {
			"1-1": {
				"pd-balanced": {
					MaxWriteIOPS:       15000,
					MaxReadIOPS:        15000,
					MaxWriteThroughput: 204,
					MaxReadThroughput:  240,
				},
				"pd-ssd": {
					MaxWriteIOPS: 15000,
					MaxReadIOPS:  15000,

					MaxWriteThroughput: 204,
					MaxReadThroughput:  240,
				},
				"pd-standard": {
					MaxWriteIOPS:       15000,
					MaxReadIOPS:        3000,
					MaxWriteThroughput: 204,
					MaxReadThroughput:  240,
				},
			},
			"2-7": {
				"pd-balanced": {
					MaxWriteIOPS:       15000,
					MaxReadIOPS:        15000,
					MaxWriteThroughput: 240,
					MaxReadThroughput:  240,
				},
				"pd-ssd": {
					MaxWriteIOPS:       15000,
					MaxReadIOPS:        15000,
					MaxWriteThroughput: 240,
					MaxReadThroughput:  240,
				},
				"pd-standard": {
					MaxWriteIOPS:       15000,
					MaxReadIOPS:        3000,
					MaxWriteThroughput: 240,
					MaxReadThroughput:  240,
				},
			},
			"8-15": {
				"pd-balanced": {
					MaxWriteIOPS:       15000,
					MaxReadIOPS:        15000,
					MaxWriteThroughput: 800,
					MaxReadThroughput:  800,
				},
				"pd-ssd": {
					MaxWriteIOPS:       15000,
					MaxReadIOPS:        15000,
					MaxWriteThroughput: 800,
					MaxReadThroughput:  800,
				},
				"pd-standard": {
					MaxWriteIOPS:       15000,
					MaxReadIOPS:        5000,
					MaxWriteThroughput: 400,
					MaxReadThroughput:  800,
				},
			},
			"16-31": {
				"pd-balanced": {
					MaxWriteIOPS:       20000,
					MaxReadIOPS:        20000,
					MaxWriteThroughput: 1000,
					MaxReadThroughput:  1200,
				},
				"pd-ssd": {
					MaxWriteIOPS:       25000,
					MaxReadIOPS:        25000,
					MaxWriteThroughput: 1000,
					MaxReadThroughput:  1200,
				},
				"pd-standard": {
					MaxWriteIOPS:       15000,
					MaxReadIOPS:        7500,
					MaxWriteThroughput: 400,
					MaxReadThroughput:  1200,
				},
			},
			"32-63": {
				"pd-balanced": {
					MaxWriteIOPS:       50000,
					MaxReadIOPS:        50000,
					MaxWriteThroughput: 1000,
					MaxReadThroughput:  1200,
				},
				"pd-ssd": {

					MaxWriteIOPS:       60000,
					MaxReadIOPS:        60000,
					MaxWriteThroughput: 1000,
					MaxReadThroughput:  1200,
				},
				"pd-standard": {
					MaxWriteIOPS:       15000,
					MaxReadIOPS:        7500,
					MaxWriteThroughput: 400,
					MaxReadThroughput:  1200,
				},
			},
			"64-": {
				"pd-balanced": {
					MaxWriteIOPS:       80000,
					MaxReadIOPS:        80000,
					MaxWriteThroughput: 1000,
					MaxReadThroughput:  1200,
				},
				"pd-ssd": {
					MaxWriteIOPS:       100000,
					MaxReadIOPS:        100000,
					MaxWriteThroughput: 1000,
					MaxReadThroughput:  1200,
				},
				"pd-standard": {
					MaxWriteIOPS:       15000,
					MaxReadIOPS:        7500,
					MaxWriteThroughput: 400,
					MaxReadThroughput:  1200,
				},
			},
		},
		"n2": {
			"2-7": {
				"pd-balanced": {
					MaxWriteIOPS:       15000,
					MaxReadIOPS:        15000,
					MaxWriteThroughput: 240,
					MaxReadThroughput:  240,
				},
				"pd-ssd": {
					MaxWriteIOPS:       15000,
					MaxReadIOPS:        15000,
					MaxWriteThroughput: 240,
					MaxReadThroughput:  240,
				},
				"pd-standard": {
					MaxWriteIOPS:       15000,
					MaxReadIOPS:        3000,
					MaxWriteThroughput: 240,
					MaxReadThroughput:  240,
				},
			},
			"8-15": {
				"pd-balanced": {
					MaxWriteIOPS:       15000,
					MaxReadIOPS:        15000,
					MaxWriteThroughput: 800,
					MaxReadThroughput:  800,
				},
				"pd-ssd": {
					MaxWriteIOPS: 15000,

					MaxReadIOPS:        15000,
					MaxWriteThroughput: 800,
					MaxReadThroughput:  800,
				},
				"pd-standard": {
					MaxWriteIOPS:       15000,
					MaxReadIOPS:        5000,
					MaxWriteThroughput: 400,
					MaxReadThroughput:  800,
				},
			},
			"16-31": {
				"pd-balanced": {
					MaxWriteIOPS:       20000,
					MaxReadIOPS:        20000,
					MaxWriteThroughput: 1200,
					MaxReadThroughput:  1200,
				},
				"pd-ssd": {
					MaxWriteIOPS:       25000,
					MaxReadIOPS:        25000,
					MaxWriteThroughput: 1200,
					MaxReadThroughput:  1200,
				},
				"pd-standard": {
					MaxWriteIOPS:       15000,
					MaxReadIOPS:        7500,
					MaxWriteThroughput: 400,
					MaxReadThroughput:  1200,
				},
			},
			"32-63": {
				"pd-balanced": {
					MaxWriteIOPS:       50000,
					MaxReadIOPS:        50000,
					MaxWriteThroughput: 1200,
					MaxReadThroughput:  1200,
				},
				"pd-ssd": {
					MaxWriteIOPS:       60000,
					MaxReadIOPS:        60000,
					MaxWriteThroughput: 1200,
					MaxReadThroughput:  1200,
				},
				"pd-standard": {
					MaxWriteIOPS:       15000,
					MaxReadIOPS:        7500,
					MaxWriteThroughput: 400,
					MaxReadThroughput:  1200,
				},
			},
			"64-": {

				"pd-balanced": {
					MaxWriteIOPS:       80000,
					MaxReadIOPS:        80000,
					MaxWriteThroughput: 1200,
					MaxReadThroughput:  1200,
				},
				"pd-ssd": {
					MaxWriteIOPS:       100000,
					MaxReadIOPS:        100000,
					MaxWriteThroughput: 1200,
					MaxReadThroughput:  1200,
				},
				"pd-standard": {
					MaxWriteIOPS:       15000,
					MaxReadIOPS:        7500,
					MaxWriteThroughput: 400,
					MaxReadThroughput:  1200,
				},
			},
		},
		"n2d": {
			"2-7": {
				"pd-balanced": {
					MaxWriteIOPS:       15000,
					MaxReadIOPS:        15000,
					MaxWriteThroughput: 240,
					MaxReadThroughput:  240,
				},
				"pd-ssd": {
					MaxWriteIOPS:       15000,
					MaxReadIOPS:        15000,
					MaxWriteThroughput: 240,
					MaxReadThroughput:  240,
				},
				"pd-standard": {
					MaxWriteIOPS:       15000,
					MaxReadIOPS:        3000,
					MaxWriteThroughput: 240,
					MaxReadThroughput:  240,
				},
			},
			"8-15": {
				"pd-balanced": {
					MaxWriteIOPS:       15000,
					MaxReadIOPS:        15000,
					MaxWriteThroughput: 800,
					MaxReadThroughput:  800,
				},
				"pd-ssd": {
					MaxWriteIOPS:       15000,
					MaxReadIOPS:        15000,
					MaxWriteThroughput: 800,
					MaxReadThroughput:  800,
				},

				"pd-standard": {
					MaxWriteIOPS:       15000,
					MaxReadIOPS:        5000,
					MaxWriteThroughput: 400,
					MaxReadThroughput:  800,
				},
			},
			"16-31": {
				"pd-balanced": {
					MaxWriteIOPS:       20000,
					MaxReadIOPS:        20000,
					MaxWriteThroughput: 1200,
					MaxReadThroughput:  1200,
				},
				"pd-ssd": {
					MaxWriteIOPS:       25000,
					MaxReadIOPS:        25000,
					MaxWriteThroughput: 1200,
					MaxReadThroughput:  1200,
				},
				"pd-standard": {
					MaxWriteIOPS:       15000,
					MaxReadIOPS:        7500,
					MaxWriteThroughput: 400,
					MaxReadThroughput:  1200,
				},
			},
			"32-63": {
				"pd-balanced": {
					MaxWriteIOPS:       50000,
					MaxReadIOPS:        50000,
					MaxWriteThroughput: 1200,
					MaxReadThroughput:  1200,
				},
				"pd-ssd": {
					MaxWriteIOPS:       60000,
					MaxReadIOPS:        60000,
					MaxWriteThroughput: 1200,
					MaxReadThroughput:  1200,
				},

				"pd-standard": {
					MaxWriteIOPS:       15000,
					MaxReadIOPS:        7500,
					MaxWriteThroughput: 400,
					MaxReadThroughput:  1200,
				},
			},
		},
		"t2d": {
			"1-1": {
				"pd-balanced": {
					MaxWriteIOPS:       15000,
					MaxReadIOPS:        15000,
					MaxWriteThroughput: 204,
					MaxReadThroughput:  240,
				},
				"pd-ssd": {
					MaxWriteIOPS:       15000,
					MaxReadIOPS:        15000,
					MaxWriteThroughput: 204,
					MaxReadThroughput:  240,
				},
				"pd-standard": {
					MaxWriteIOPS:       15000,
					MaxReadIOPS:        3000,
					MaxWriteThroughput: 204,
					MaxReadThroughput:  240,
				},
			},
			"2-7": {
				"pd-balanced": {
					MaxWriteIOPS:       15000,
					MaxReadIOPS:        15000,
					MaxWriteThroughput: 240,
					MaxReadThroughput:  240,
				},
				"pd-ssd": {
					MaxWriteIOPS:       15000,
					MaxReadIOPS:        15000,
					MaxWriteThroughput: 240,
					MaxReadThroughput:  240,
				},
				"pd-standard": {
					MaxWriteIOPS:       15000,
					MaxReadIOPS:        3000,
					MaxWriteThroughput: 240,
					MaxReadThroughput:  240,
				},
			},
			"8-15": {
				"pd-balanced": {
					MaxWriteIOPS:       15000,
					MaxReadIOPS:        15000,
					MaxWriteThroughput: 800,
					MaxReadThroughput:  800,
				},
				"pd-ssd": {
					MaxWriteIOPS:       15000,
					MaxReadIOPS:        15000,
					MaxWriteThroughput: 800,
					MaxReadThroughput:  800,
				},
				"pd-standard": {
					MaxWriteIOPS:       15000,
					MaxReadIOPS:        5000,
					MaxWriteThroughput: 400,
					MaxReadThroughput:  800,
				},
			},
			"16-31": {
				"pd-balanced": {
					MaxWriteIOPS: 20000,

					MaxReadIOPS:        20000,
					MaxWriteThroughput: 1200,
					MaxReadThroughput:  1200,
				},
				"pd-ssd": {
					MaxWriteIOPS:       25000,
					MaxReadIOPS:        25000,
					MaxWriteThroughput: 1200,
					MaxReadThroughput:  1200,
				},
				"pd-standard": {
					MaxWriteIOPS: 15000,
					MaxReadIOPS:  7500,

					MaxWriteThroughput: 400,
					MaxReadThroughput:  1200,
				},
			},
			"32-60": {
				"pd-balanced": {
					MaxWriteIOPS:       50000,
					MaxReadIOPS:        50000,
					MaxWriteThroughput: 1200,
					MaxReadThroughput:  1200,
				},
				"pd-ssd": {
					MaxWriteIOPS:       60000,
					MaxReadIOPS:        60000,
					MaxWriteThroughput: 1200,
					MaxReadThroughput:  1200,
				},
				"pd-standard": {
					MaxWriteIOPS:       15000,
					MaxReadIOPS:        7500,
					MaxWriteThroughput: 400,
					MaxReadThroughput:  1200,
				},
			},
		},
		"t2a": {
			// pd-balance 1	20,000	20,000	204	240
			//2-7	20,000	20,000	240	240
			//8-15	25,000	25,000	800	800
			//16-31	25,000	25,000	1,200	1,200
			//32-47	60,000	60,000	1,200	1,200
			//48	80,000	80,000	1,800	1,800
			// pd-ssd 1	20,000	20,000	204	240
			//2-7	20,000	20,000	240	240
			//8-15	25,000	25,000	800	800
			//16-31	25,000	25,000	1,200	1,200
			//32-47	60,000	60,000	1,200	1,200
			//48	80,000	80,000	1,800	1,800
			// pd-standard 1	15,000	1,000	204	240
			//2-3	15,000	2,400	240	240
			//4-7	15,000	3,000	240	240
			//8-15	15,000	5,000	400	800
			//16 or more	15,000	7,500	400	1,200
			"1-1": {
				"pd-balanced": {
					MaxWriteIOPS:       20000,
					MaxReadIOPS:        20000,
					MaxWriteThroughput: 204,
					MaxReadThroughput:  240,
				},
				"pd-ssd": {

					MaxWriteIOPS:       20000,
					MaxReadIOPS:        20000,
					MaxWriteThroughput: 204,

					MaxReadThroughput: 240,
				},
				"pd-standard": {
					MaxWriteIOPS:       15000,
					MaxReadIOPS:        1000,
					MaxWriteThroughput: 204,
					MaxReadThroughput:  240,
				},
			},
			"2-3": {
				"pd-balanced": {
					MaxWriteIOPS:       20000,
					MaxReadIOPS:        20000,
					MaxWriteThroughput: 240,
					MaxReadThroughput:  240,
				},
				"pd-ssd": {
					MaxWriteIOPS: 20000,

					MaxReadIOPS:        20000,
					MaxWriteThroughput: 240,
					MaxReadThroughput:  240,
				},
				"pd-standard": {
					MaxWriteIOPS:       15000,
					MaxReadIOPS:        2400,
					MaxWriteThroughput: 240,
					MaxReadThroughput:  240,
				},
			},
			"4-7": {
				"pd-balanced": {
					MaxWriteIOPS:       20000,
					MaxReadIOPS:        20000,
					MaxWriteThroughput: 240,
					MaxReadThroughput:  240,
				},
				"pd-ssd": {
					MaxWriteIOPS:       20000,
					MaxReadIOPS:        20000,
					MaxWriteThroughput: 240,
					MaxReadThroughput:  240,
				},
				"pd-standard": {
					MaxWriteIOPS:       15000,
					MaxReadIOPS:        3000,
					MaxWriteThroughput: 240,
					MaxReadThroughput:  240,
				},
			},
			"8-15": {
				"pd-balanced": {
					MaxWriteIOPS: 25000,

					MaxReadIOPS:        25000,
					MaxWriteThroughput: 800,
					MaxReadThroughput:  800,
				},
				"pd-ssd": {
					MaxWriteIOPS:       25000,
					MaxReadIOPS:        25000,
					MaxWriteThroughput: 800,
					MaxReadThroughput:  800,
				},
				"pd-standard": {
					MaxWriteIOPS:       15000,
					MaxReadIOPS:        5000,
					MaxWriteThroughput: 400,
					MaxReadThroughput:  800,
				},
			},
			"16-31": {

				"pd-balanced": {
					MaxWriteIOPS: 25000,

					MaxReadIOPS:        25000,
					MaxWriteThroughput: 1200,
					MaxReadThroughput:  1200,
				},

				"pd-ssd": {
					MaxWriteIOPS:       25000,
					MaxReadIOPS:        25000,
					MaxWriteThroughput: 1200,
					MaxReadThroughput:  1200,
				},
				"pd-standard": {
					MaxWriteIOPS:       15000,
					MaxReadIOPS:        7500,
					MaxWriteThroughput: 400,
					MaxReadThroughput:  1200,
				},
			},
			"32-47": {
				"pd-balanced": {

					MaxWriteIOPS:       60000,
					MaxReadIOPS:        60000,
					MaxWriteThroughput: 1200,
					MaxReadThroughput:  1200,
				},
				"pd-ssd": {
					MaxWriteIOPS:       60000,
					MaxReadIOPS:        60000,
					MaxWriteThroughput: 1200,
					MaxReadThroughput:  1200,
				},
				"pd-standard": {
					MaxWriteIOPS:       15000,
					MaxReadIOPS:        7500,
					MaxWriteThroughput: 400,
					MaxReadThroughput:  1200,
				},
			},
			"48-48": {
				"pd-balanced": {
					MaxWriteIOPS: 80000,
					MaxReadIOPS:  80000,

					MaxWriteThroughput: 1800,
					MaxReadThroughput:  1800,
				},
				"pd-ssd": {
					MaxWriteIOPS:       80000,
					MaxReadIOPS:        80000,
					MaxWriteThroughput: 1800,
					MaxReadThroughput:  1800,
				},
				"pd-standard": {
					MaxWriteIOPS:       15000,
					MaxReadIOPS:        7500,
					MaxWriteThroughput: 400,
					MaxReadThroughput:  1200,
				},
			},
		},
	}
)
