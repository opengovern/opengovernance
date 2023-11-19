package api

import (
	aws "github.com/kaytu-io/kaytu-aws-describer/aws/model"
	azure "github.com/kaytu-io/kaytu-azure-describer/azure/model"
)

type GetEC2InstanceCostRequest struct {
	RegionCode string
	Instance   aws.EC2InstanceDescription
}

type GetEC2VolumeCostRequest struct {
	RegionCode string
	Type       string
	Size       float64
	IOPs       float64
}

type GetLBCostRequest struct {
	RegionCode string
	LBType     string
}

type GetRDSInstanceRequest struct {
	RegionCode           string
	InstanceEngine       string
	InstanceLicenseModel string
	InstanceMultiAZ      bool
	AllocatedStorage     float64
	StorageType          string
	IOPs                 float64
}

type GetAzureVmRequest struct {
	RegionCode      string
	VMSize          string
	OperatingSystem string
}

type GetAzureManagedStorageRequest struct {
	RegionCode      string
	SkuName         string
	DiskSize        float64
	BurstingEnabled bool
	DiskThroughput  float64
	DiskIOPs        float64
}

type GetAzureLoadBalancerRequest struct {
	RegionCode       string
	DailyDataProceed *int64 // (GB)
	SkuName          string
	SkuTier          string
	RulesNumber      int32
}

type GetAzureVirtualNetworkRequest struct {
	RegionCode            string
	PeeringLocations      []string
	MonthlyDataTransferGB *float64
}

type GetAzureVirtualNetworkPeeringRequest struct {
	SourceLocation        string
	DestinationLocation   string
	MonthlyDataTransferGB *float64
}

type GetAzureSqlServersDatabasesRequest struct {
	RegionCode  string
	SqlServerDB azure.SqlDatabaseDescription
	// MonthlyVCoreHours represents a usage param that allows users to define how many hours of usage a serverless sql database instance uses.
	MonthlyVCoreHours int64
	// ExtraDataStorageGB represents a usage cost of additional backup storage used by the sql database.
	ExtraDataStorageGB float64
	// LongTermRetentionStorageGB defines a usage param that allows users to define how many GB of cold storage the database uses.
	// This is storage that can be kept for up to 10 years.
	LongTermRetentionStorageGB int64
	// BackupStorageGB defines a usage param that allows users to define how many GB Point-In-Time Restore (PITR) backup storage the database uses.
	BackupStorageGB int64
	ResourceId      string
}
