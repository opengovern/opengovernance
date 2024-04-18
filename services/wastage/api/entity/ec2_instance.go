package entity

import (
	types2 "github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

type AWSCredential struct {
	AccountID string `json:"accountID"`
	AccessKey string `json:"accessKey"`
	SecretKey string `json:"secretKey"`
}

type Limitations struct {
	MemoryGB     *int64 `json:"memoryGB"`
	ENASupport   *bool  `json:"ENASupport"`
	EBSOptimized *bool  `json:"EBSOptimized"`
}

type EC2InstanceWastageRequest struct {
	Instance types.Instance                `json:"instance"`
	Volumes  []types.Volume                `json:"volumes"`
	Metrics  map[string][]types2.Datapoint `json:"metrics"`
	Region   string                        `json:"region"`
}

type RightSizingRecommendation struct {
	TargetInstanceType string  `json:"targetInstanceType"`
	Saving             float64 `json:"saving"`

	AvgCPUUsage string `json:"avgCPUUsage"`
	TargetCores string `json:"targetCores"`

	AvgNetworkBandwidth      string `json:"avgNetworkBandwidth"`
	TargetNetworkPerformance string `json:"targetNetworkBandwidth"`

	CurrentMemory string `json:"currentMemory"`
	TargetMemory  string `json:"targetMemory"`
}

type EC2InstanceWastageResponse struct {
	CurrentCost  float64                   `json:"currentCost"`
	TotalSavings float64                   `json:"totalSavings"`
	RightSizing  RightSizingRecommendation `json:"rightSizing"`
}
