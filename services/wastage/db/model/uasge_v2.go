package model

import (
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type UsageV2 struct {
	gorm.Model

	RequestId      *string
	ResponseId     *string
	ApiEndpoint    string
	Request        datatypes.JSON
	Response       datatypes.JSON
	FailureMessage *string
	Latency        *float64 //Seconds
	CliVersion     *string
	Statistics     datatypes.JSON
}

type Statistics struct {
	AccountID   string `json:"accountID"`
	OrgEmail    string `json:"orgEmail"`
	ResourceID  string `json:"resourceID"`
	Auth0UserId string `json:"auth0UserId"`

	CurrentCost     float64 `json:"currentCost"`
	RecommendedCost float64 `json:"recommendedCost"`
	Savings         float64 `json:"savings"`

	EC2InstanceCurrentCost     float64 `json:"ec2InstanceCurrentCost"`
	EC2InstanceRecommendedCost float64 `json:"ec2InstanceRecommendedCost"`
	EC2InstanceSavings         float64 `json:"ec2InstanceSavings"`

	EBSCurrentCost     float64 `json:"ebsCurrentCost"`
	EBSRecommendedCost float64 `json:"ebsRecommendedCost"`
	EBSSavings         float64 `json:"ebsSavings"`
	EBSVolumeCount     int     `json:"ebsVolumeCount"`

	RDSInstanceCurrentCost     float64 `json:"rdsInstanceCurrentCost"`
	RDSInstanceRecommendedCost float64 `json:"rdsInstanceRecommendedCost"`
	RDSInstanceSavings         float64 `json:"rdsInstanceSavings"`
}
