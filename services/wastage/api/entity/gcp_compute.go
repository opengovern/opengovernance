package entity

import types2 "github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"

type GcpComputeInstance struct {
	HashedInstanceId string `json:"hashedInstanceId"`
	Zone             string `json:"zone"`
	MachineType      string `json:"machineType"`
}

type RightsizingGcpComputeInstance struct {
	Zone          string `json:"zone"`
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
	RequestId      *string                       `json:"requestId"`
	CliVersion     *string                       `json:"cliVersion"`
	Identification map[string]string             `json:"identification"`
	Instance       GcpComputeInstance            `json:"instance"`
	Metrics        map[string][]types2.Datapoint `json:"metrics"`
	Region         string                        `json:"region"`
	Preferences    map[string]*string            `json:"preferences"`
	Loading        bool                          `json:"loading"`
}

type GcpComputeInstanceWastageResponse struct {
	RightSizing GcpComputeInstanceRightsizingRecommendation `json:"rightSizing"`
}
