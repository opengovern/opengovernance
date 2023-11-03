package db

import (
	"github.com/kaytu-io/kaytu-util/pkg/source"
	"gorm.io/gorm"
	"time"
)

type StoreCostTableJobStatus string

const (
	StoreCostTableJobStatusProcessing StoreCostTableJobStatus = "PROCESSING"
	StoreCostTableJobStatusFailed     StoreCostTableJobStatus = "FAILED"
	StoreCostTableJobStatusSucceeded  StoreCostTableJobStatus = "SUCCEEDED"
)

type StoreCostTableJob struct {
	Id           uint `json:"id" sql:"AUTO_INCREMENT" gorm:"primary_key"`
	CreatedAt    time.Time
	UpdatedAt    time.Time      `gorm:"index:,sort:desc"`
	DeletedAt    gorm.DeletedAt `gorm:"index"`
	Connector    source.Type
	ErrorMessage string
	Status       StoreCostTableJobStatus
	Count        int64
}

// EC2InstanceCost Service = AmazonEC2, ProductFamily = Compute Instance
type EC2InstanceCost struct {
	SKU             string `gorm:"primaryKey"`
	RegionCode      string
	InstanceType    string
	Tenancy         string
	OperatingSystem string
	CapacityStatus  string
	PreInstalledSw  string
	UsageType       string
	CostUnit        string
	Cost            float64
}

// EC2InstanceSystemOperationCost Service = AmazonEC2, ProductFamily = System Operation
type EC2InstanceSystemOperationCost struct {
	SKU           string `gorm:"primaryKey"`
	RegionCode    string
	VolumeAPIName string
	UsageType     string
	CostUnit      string
	Cost          float64
}

// EC2InstanceStorageCost Service = AmazonEC2, ProductFamily = Storage
type EC2InstanceStorageCost struct {
	SKU           string `gorm:"primaryKey"`
	RegionCode    string
	VolumeAPIName string
	CostUnit      string
	Cost          float64
}

// AmazonCloudWatchCost Service = AmazonCloudWatch, ProductFamily = Metric
type AmazonCloudWatchCost struct {
	SKU           string `gorm:"primaryKey"`
	RegionCode    string
	StartingRange int
	CostUnit      string
	Cost          float64
}

// EC2CpuCreditsCost Service = AmazonEC2, ProductFamily = CPU Credits
type EC2CpuCreditsCost struct {
	SKU             string `gorm:"primaryKey"`
	RegionCode      string
	OperatingSystem string
	UsageType       string
	CostUnit        string
	Cost            float64
}
