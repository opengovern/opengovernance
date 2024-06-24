package model

import (
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"time"
)

type UsageV2 struct {
	//We don't use gorm.Model since we need the indices on CreatedAt and UpdatedAt
	ID        uint           `gorm:"primarykey"`
	CreatedAt time.Time      `gorm:"index"`
	UpdatedAt time.Time      `gorm:"index"`
	DeletedAt gorm.DeletedAt `gorm:"index"`

	RequestId      *string
	ResponseId     *string
	ApiEndpoint    string `gorm:"index"`
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

	KubernetesCurrentCPURequest     float64 `json:"kubernetesCurrentCPU"`
	KubernetesRecommendedCPURequest float64 `json:"kubernetesRecommendedCPU"`
	KubernetesCPURequestSavings     float64 `json:"kubernetesCPUSavings"`

	KubernetesCurrentMemoryRequest     float64 `json:"kubernetesCurrentMemoryRequest"`
	KubernetesRecommendedMemoryRequest float64 `json:"kubernetesRecommendedMemoryRequest"`
	KubernetesMemoryRequestSavings     float64 `json:"KubernetesMemoryRequestSavings"`

	GCPComputeInstanceCurrentCost     float64 `json:"gcpComputeInstanceCurrentCost"`
	GCPComputeInstanceRecommendedCost float64 `json:"gcpComputeInstanceRecommendedCost"`
	GCPComputeInstanceSavings         float64 `json:"gcpComputeInstanceSavings"`
}
