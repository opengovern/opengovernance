package db

import "github.com/shopspring/decimal"

// AwsEC2InstancePrice Service = AmazonEC2, ProductFamily = Compute Instance
type AwsEC2InstancePrice struct {
	SKU             string `gorm:"primaryKey"`
	EffectiveDate   int64
	RegionCode      string
	InstanceType    string
	Tenancy         string
	OperatingSystem string
	CapacityStatus  string
	PreInstalledSw  string
	UsageType       string
	PriceUnit       string
	Price           decimal.Decimal
}

// AwsEC2InstanceSystemOperationPrice Service = AmazonEC2, ProductFamily = System Operation
type AwsEC2InstanceSystemOperationPrice struct {
	SKU           string `gorm:"primaryKey"`
	EffectiveDate int64
	RegionCode    string
	VolumeAPIName string
	UsageType     string
	Currency      string
	PriceUnit     string
	Price         decimal.Decimal
}

// AwsEC2InstanceStoragePrice Service = AmazonEC2, ProductFamily = Storage
type AwsEC2InstanceStoragePrice struct {
	SKU           string `gorm:"primaryKey"`
	EffectiveDate int64
	RegionCode    string
	VolumeAPIName string
	PriceUnit     string
	Price         decimal.Decimal
}

// AwsCloudwatchPrice Service = AmazonCloudWatch
type AwsCloudwatchPrice struct {
	SKU           string `gorm:"primaryKey"`
	ProductFamily string
	EffectiveDate int64
	RegionCode    string
	BeginRange    int
	PriceUnit     string
	Price         decimal.Decimal
}

// AwsEC2CpuCreditsPrice Service = AmazonEC2, ProductFamily = CPU Credits
type AwsEC2CpuCreditsPrice struct {
	SKU             string `gorm:"primaryKey"`
	EffectiveDate   int64
	RegionCode      string
	OperatingSystem string
	UsageType       string
	PriceUnit       string
	Price           float64
}

// AwsElasticLoadBalancingPrice service = Elastic Load Balancing,
// ProductFamily = (Load Balancer-Gateway), (Load Balancer-Application), (Load Balancer-Network), (Load Balancer)
type AwsElasticLoadBalancingPrice struct {
	SKU           string `gorm:"primaryKey"`
	EffectiveDate int64
	ProductFamily string
	RegionCode    string
	UsageType     string
	PriceUnit     string
	Price         decimal.Decimal
}

// AwsRdsInstancePrice Service = AmazonRDS, ProductFamily = Database Instance
type AwsRdsInstancePrice struct {
	SKU              string `gorm:"primaryKey"`
	EffectiveDate    int64
	RegionCode       string
	InstanceType     string
	DatabaseEngine   string
	DatabaseEdition  string
	LicenseModel     string
	DeploymentOption string
	PriceUnit        string
	Price            decimal.Decimal
}

// AwsRdsStoragePrice Service = AmazonRDS, ProductFamily = Database Storage
type AwsRdsStoragePrice struct {
	SKU              string `gorm:"primaryKey"`
	EffectiveDate    int64
	RegionCode       string
	DeploymentOption string
	VolumeType       string
	PriceUnit        string
	Price            decimal.Decimal
}

// AwsRdsIopsPrice Service = AmazonRDS, ProductFamily = Provisioned IOPS
type AwsRdsIopsPrice struct {
	SKU              string `gorm:"primaryKey"`
	EffectiveDate    int64
	RegionCode       string
	DeploymentOption string
	PriceUnit        string
	Price            decimal.Decimal
}

// AzureVirtualMachinePrice Service = Virtual Machines, Family = Compute
type AzureVirtualMachinePrice struct {
	SKU           string `gorm:"primaryKey"`
	EffectiveDate int64
	ArmRegionName string
	ArmSkuName    string
	ProductName   string
	Priority      string
	SkuName       string
	PriceUnit     string
	Price         decimal.Decimal
}

// AzureManagedStoragePrice Product Name contains "Disk"
type AzureManagedStoragePrice struct {
	SKU           string `gorm:"primaryKey"`
	EffectiveDate int64
	ArmRegionName string
	SkuName       string
	Meter         string
	PriceUnit     string
	Price         decimal.Decimal
}

// AzureLoadBalancerPrice Service = Load Balancer, Family = Networking
type AzureLoadBalancerPrice struct {
	SKU           string `gorm:"primaryKey"`
	EffectiveDate int64
	ArmRegionName string
	MeterName     string
	PriceUnit     string
	Price         decimal.Decimal
}
