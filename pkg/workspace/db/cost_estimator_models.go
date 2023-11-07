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

// EC2CpuCreditsCost Service = AmazonEC2, ProductFamily = CPU Credits
type EC2CpuCreditsCost struct {
	SKU             string `gorm:"primaryKey"`
	EffectiveDate   int64
	RegionCode      string
	OperatingSystem string
	UsageType       string
	PriceUnit       string
	Price           float64
}

// RDSDBInstancePrice Service = AmazonRDS, ProductFamily = Database Instance
type RDSDBInstancePrice struct {
	SKU              string `gorm:"primaryKey"`
	EffectiveDate    int64
	region           string
	instanceType     string
	databaseEngine   string
	databaseEdition  string
	licenseModel     string
	deploymentOption string
	storageType      string
	UsageType        string
	PriceUnit        string
	Price            float64
}
