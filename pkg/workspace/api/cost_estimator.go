package api

import (
	aws "github.com/kaytu-io/kaytu-aws-describer/aws/model"
	azure "github.com/kaytu-io/kaytu-azure-describer/azure/model"
)

type GetEC2InstanceCostRequest struct {
	ResourceId string
	RegionCode string
	Instance   aws.EC2InstanceDescription
}

type GetEC2VolumeCostRequest struct {
	ResourceId string
	RegionCode string
	Volume     aws.EC2VolumeDescription
}

type GetLBCostRequest struct {
	ResourceId string
	RegionCode string
	LBType     string
}

type GetRDSInstanceRequest struct {
	ResourceId string
	RegionCode string
	DBInstance aws.RDSDBInstanceDescription
}

type GetAzureVmRequest struct {
	ResourceId      string
	RegionCode      string
	VMSize          string
	OperatingSystem string
}

type GetAzureManagedStorageRequest struct {
	ResourceId      string
	RegionCode      string
	SkuName         string
	DiskSize        float64
	BurstingEnabled bool
	DiskThroughput  float64
	DiskIOPs        float64
}

type GetAzureLoadBalancerRequest struct {
	ResourceId       string
	RegionCode       string
	DailyDataProceed *int64 // (GB)
	SkuName          string
	SkuTier          string
	RulesNumber      int32
}

type GetAzureVirtualNetworkRequest struct {
	ResourceId            string
	RegionCode            string
	PeeringLocations      []string
	MonthlyDataTransferGB *float64
}

type GetAzureVirtualNetworkPeeringRequest struct {
	ResourceId            string
	SourceLocation        string
	DestinationLocation   string
	MonthlyDataTransferGB *float64
}

type GetAzureSqlServersDatabasesRequest struct {
	ResourceId                 string
	RegionCode                 string
	SqlServerDB                azure.SqlDatabaseDescription
	MonthlyVCoreHours          int64
	ExtraDataStorageGB         float64
	LongTermRetentionStorageGB int64
	BackupStorageGB            int64
}
