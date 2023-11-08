package db

// EC2InstancePrice Service = AmazonEC2, ProductFamily = Compute Instance
type EC2InstancePrice struct {
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
	Price           float64
}

// EC2InstanceSystemOperationPrice Service = AmazonEC2, ProductFamily = System Operation
type EC2InstanceSystemOperationPrice struct {
	SKU           string `gorm:"primaryKey"`
	EffectiveDate int64
	RegionCode    string
	VolumeAPIName string
	UsageType     string
	PriceUnit     string
	Price         float64
}

// EC2InstanceStoragePrice Service = AmazonEC2, ProductFamily = Storage
type EC2InstanceStoragePrice struct {
	SKU           string `gorm:"primaryKey"`
	EffectiveDate int64
	RegionCode    string
	VolumeAPIName string
	PriceUnit     string
	Price         float64
}

// AmazonCloudWatchPrice Service = AmazonCloudWatch
type AmazonCloudWatchPrice struct {
	SKU           string `gorm:"primaryKey"`
	ProductFamily string
	EffectiveDate int64
	RegionCode    string
	BeginRange    int
	PriceUnit     string
	Price         float64
}

// EC2CpuCreditsPrice Service = AmazonEC2, ProductFamily = CPU Credits
type EC2CpuCreditsPrice struct {
	SKU             string `gorm:"primaryKey"`
	EffectiveDate   int64
	RegionCode      string
	OperatingSystem string
	UsageType       string
	PriceUnit       string
	Price           float64
}

// LBPrice service = Elastic Load Balancing,
// ProductFamily = (Load Balancer-Gateway), (Load Balancer-Application), (Load Balancer-Network), (Load Balancer)
type LBPrice struct {
	SKU           string `gorm:"primaryKey"`
	ProductFamily string
	UsageType     string
	PriceUnit     string
	Price         float64
}

// RDSDBInstancePrice Service = AmazonRDS, ProductFamily = Database Instance
type RDSDBInstancePrice struct {
	SKU              string `gorm:"primaryKey"`
	EffectiveDate    int64
	RegionCode       string
	InstanceType     string
	DatabaseEngine   string
	DatabaseEdition  string
	LicenseModel     string
	DeploymentOption string
	PriceUnit        string
	Price            float64
}

// RDSDBStoragePrice Service = AmazonRDS, ProductFamily = Database Storage
type RDSDBStoragePrice struct {
	SKU              string `gorm:"primaryKey"`
	EffectiveDate    int64
	RegionCode       string
	DeploymentOption string
	VolumeType       string
	PriceUnit        string
	Price            float64
}

// RDSDBIopsPrice Service = AmazonRDS, ProductFamily = Provisioned IOPS
type RDSDBIopsPrice struct {
	SKU              string `gorm:"primaryKey"`
	EffectiveDate    int64
	RegionCode       string
	DeploymentOption string
	PriceUnit        string
	Price            float64
}
