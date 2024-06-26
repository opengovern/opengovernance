package entity

import "time"

type GcpComputeInstance struct {
	HashedInstanceId string `json:"hashedInstanceId"`
	Zone             string `json:"zone"`
	MachineType      string `json:"machineType"`
}

type GcpComputeDisk struct {
	HashedDiskId    string `json:"hashedDiskId"`
	Zone            string `json:"zone"`
	Region          string `json:"region"`
	DiskType        string `json:"diskType"`
	DiskSize        *int64 `json:"diskSize"`
	ProvisionedIops *int64 `json:"provisionedIops"`
}

type RightsizingGcpComputeInstance struct {
	Zone          string `json:"zone"`
	Region        string `json:"region"`
	MachineType   string `json:"machineType"`
	MachineFamily string `json:"machineFamily"`
	CPU           int64  `json:"cpu"`
	MemoryMb      int64  `json:"memoryMb"`

	Cost float64 `json:"cost"`
}

type GcpComputeInstanceRightsizingRecommendation struct {
	Current     RightsizingGcpComputeInstance  `json:"current"`
	Recommended *RightsizingGcpComputeInstance `json:"recommended"`

	CPU    Usage `json:"cpu"`
	Memory Usage `json:"memory"`

	Description string `json:"description"`
}

type GcpComputeInstanceWastageRequest struct {
	RequestId        *string                           `json:"requestId"`
	CliVersion       *string                           `json:"cliVersion"`
	Identification   map[string]string                 `json:"identification"`
	Instance         GcpComputeInstance                `json:"instance"`
	Disks            []GcpComputeDisk                  `json:"disks"`
	Metrics          map[string][]Datapoint            `json:"metrics"`
	DiskCapacityUsed map[string]map[string][]Datapoint `json:"diskCapacityUsed"`
	Region           string                            `json:"region"`
	Preferences      map[string]*string                `json:"preferences"`
	Loading          bool                              `json:"loading"`
}

type RightsizingGcpComputeDisk struct {
	Zone     string  `json:"zone"`
	Region   string  `json:"region"`
	DiskType string  `json:"diskType"`
	DiskSize *int64  `json:"diskSize"`
	Cost     float64 `json:"cost"`
}

type GcpComputeDiskRecommendation struct {
	Current     RightsizingGcpComputeDisk
	Recommended *RightsizingGcpComputeDisk

	Iops       Usage `json:"iops"`
	Throughput Usage `json:"throughput"`

	Description string `json:"description"`
}

type GcpComputeInstanceWastageResponse struct {
	RightSizing       GcpComputeInstanceRightsizingRecommendation `json:"rightSizing"`
	VolumeRightSizing map[string]GcpComputeDiskRecommendation     `json:"volumes"`
}

type Datapoint struct {
	StartTime time.Time
	EndTime   time.Time
	Value     float64
}
