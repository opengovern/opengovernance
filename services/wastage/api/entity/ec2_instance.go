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

type Recommendation struct {
	Description string  `json:"description"`
	Saving      float64 `json:"saving"`
}

type EC2InstanceWastageRequest struct {
	Instance types.Instance                `json:"instance"`
	Volumes  []types.Volume                `json:"volumes"`
	Metrics  map[string][]types2.Datapoint `json:"metrics"`
	Region   string                        `json:"region"`
}

type EC2InstanceWastageResponse struct {
	CurrentCost     float64          `json:"currentCost"`
	TotalSavings    float64          `json:"totalSavings"`
	Recommendations []Recommendation `json:"recommendations"`
}
