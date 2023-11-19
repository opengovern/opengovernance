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
	Volume     aws.EC2VolumeDescription
}

type GetLBCostRequest struct {
	RegionCode string
	LBType     string
}

type GetRDSInstanceRequest struct {
	RegionCode string
	DBInstance aws.RDSDBInstanceDescription
}

type GetAzureVmRequest struct {
	RegionCode string
	VM         azure.ComputeVirtualMachineDescription
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
	LoadBalancer     azure.LoadBalancerDescription
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
	RegionCode                 string
	SqlServerDB                azure.SqlDatabaseDescription
	MonthlyVCoreHours          int64
	ExtraDataStorageGB         float64
	LongTermRetentionStorageGB int64
	BackupStorageGB            int64
	ResourceId                 string
}
