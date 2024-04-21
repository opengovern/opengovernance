package model

import (
	"gorm.io/gorm"
	"strconv"
	"strings"
)

type EC2InstanceType struct {
	gorm.Model

	InstanceType        string
	VCpu                int64
	MemoryGB            int64
	NetworkMaxBandwidth int64
	NetworkIsDedicated  bool
	PricePerUnit        float64

	PricePerUnitStr             string
	NetworkPerformance          string
	Memory                      string
	VCPUStr                     string
	TermType                    string
	PriceDescription            string
	EffectiveDate               string
	StartingRange               string
	EndingRange                 string
	Unit                        string
	Currency                    string
	RelatedTo                   string
	LeaseContractLength         string
	PurchaseOption              string
	OfferingClass               string
	ProductFamily               string
	ServiceCode                 string
	Location                    string
	LocationType                string
	CurrentGeneration           string
	InstanceFamily              string
	PhysicalProcessor           string
	ClockSpeed                  string
	Storage                     string
	ProcessorArchitecture       string
	StorageMedia                string
	VolumeType                  string
	MaxVolumeSize               string
	MaxIOPSVolume               string
	MaxIOPSBurstPerformance     string
	MaxThroughputVolume         string
	Provisioned                 string
	Tenancy                     string
	EBSOptimized                string
	OperatingSystem             string
	LicenseModel                string
	Group                       string
	GroupDescription            string
	TransferType                string
	FromLocation                string
	FromLocationType            string
	ToLocation                  string
	ToLocationType              string
	UsageType                   string
	Operation                   string
	AvailabilityZone            string
	CapacityStatus              string
	ClassicNetworkingSupport    string
	DedicatedEBSThroughput      string
	ECU                         string
	ElasticGraphicsType         string
	EnhancedNetworkingSupported string
	FromRegionCode              string
	GPU                         string
	GPUMemory                   string
	Instance                    string
	InstanceCapacity10xlarge    string
	InstanceCapacity12xlarge    string
	InstanceCapacity16xlarge    string
	InstanceCapacity18xlarge    string
	InstanceCapacity24xlarge    string
	InstanceCapacity2xlarge     string
	InstanceCapacity32xlarge    string
	InstanceCapacity4xlarge     string
	InstanceCapacity8xlarge     string
	InstanceCapacity9xlarge     string
	InstanceCapacityLarge       string
	InstanceCapacityMedium      string
	InstanceCapacityMetal       string
	InstanceCapacityxlarge      string
	InstanceSKU                 string
	IntelAVX2Available          string
	IntelAVXAvailable           string
	IntelTurboAvailable         string
	MarketOption                string
	NormalizationSizeFactor     string
	PhysicalCores               string
	PreInstalledSW              string
	ProcessorFeatures           string
	ProductType                 string
	RegionCode                  string
	ResourceType                string
	ServiceName                 string
	SnapshotArchiveFeeType      string
	ToRegionCode                string
	VolumeAPIName               string
	VPCNetworkingSupport        string
}

func (v *EC2InstanceType) PopulateFromMap(columns map[string]int, row []string) {
	for col, index := range columns {
		switch col {
		case "TermType":
			v.TermType = row[index]
		case "PriceDescription":
			v.PriceDescription = row[index]
		case "EffectiveDate":
			v.EffectiveDate = row[index]
		case "StartingRange":
			v.StartingRange = row[index]
		case "EndingRange":
			v.EndingRange = row[index]
		case "Unit":
			v.Unit = row[index]
		case "PricePerUnit":
			v.PricePerUnit, _ = strconv.ParseFloat(row[index], 64)
			v.PricePerUnitStr = row[index]
		case "Currency":
			v.Currency = row[index]
		case "RelatedTo":
			v.RelatedTo = row[index]
		case "LeaseContractLength":
			v.LeaseContractLength = row[index]
		case "PurchaseOption":
			v.PurchaseOption = row[index]
		case "OfferingClass":
			v.OfferingClass = row[index]
		case "Product Family":
			v.ProductFamily = row[index]
		case "serviceCode":
			v.ServiceCode = row[index]
		case "Location":
			v.Location = row[index]
		case "Location Type":
			v.LocationType = row[index]
		case "Instance Type":
			v.InstanceType = row[index]
		case "Current Generation":
			v.CurrentGeneration = row[index]
		case "Instance Family":
			v.InstanceFamily = row[index]
		case "vCPU":
			v.VCpu, _ = strconv.ParseInt(row[index], 10, 64)
			v.VCPUStr = row[index]
		case "Physical Processor":
			v.PhysicalProcessor = row[index]
		case "Clock Speed":
			v.ClockSpeed = row[index]
		case "Memory":
			v.MemoryGB = parseMemory(row[index])
			v.Memory = row[index]
		case "Storage":
			v.Storage = row[index]
		case "Network Performance":
			v.NetworkPerformance = row[index]
			bandwidth, upTo := parseNetworkPerformance(row[index])
			v.NetworkMaxBandwidth = bandwidth
			v.NetworkIsDedicated = !upTo
		case "Processor Architecture":
			v.ProcessorArchitecture = row[index]
		case "Storage Media":
			v.StorageMedia = row[index]
		case "Volume Type":
			v.VolumeType = row[index]
		case "Max Volume Size":
			v.MaxVolumeSize = row[index]
		case "Max IOPS/volume":
			v.MaxIOPSVolume = row[index]
		case "Max IOPS Burst Performance":
			v.MaxIOPSBurstPerformance = row[index]
		case "Max throughput/volume":
			v.MaxThroughputVolume = row[index]
		case "Provisioned":
			v.Provisioned = row[index]
		case "Tenancy":
			v.Tenancy = row[index]
		case "EBS Optimized":
			v.EBSOptimized = row[index]
		case "Operating System":
			v.OperatingSystem = row[index]
		case "License Model":
			v.LicenseModel = row[index]
		case "Group":
			v.Group = row[index]
		case "Group Description":
			v.GroupDescription = row[index]
		case "Transfer Type":
			v.TransferType = row[index]
		case "From Location":
			v.FromLocation = row[index]
		case "From Location Type":
			v.FromLocationType = row[index]
		case "To Location":
			v.ToLocation = row[index]
		case "To Location Type":
			v.ToLocationType = row[index]
		case "usageType":
			v.UsageType = row[index]
		case "operation":
			v.Operation = row[index]
		case "AvailabilityZone":
			v.AvailabilityZone = row[index]
		case "CapacityStatus":
			v.CapacityStatus = row[index]
		case "ClassicNetworkingSupport":
			v.ClassicNetworkingSupport = row[index]
		case "Dedicated EBS Throughput":
			v.DedicatedEBSThroughput = row[index]
		case "ECU":
			v.ECU = row[index]
		case "Elastic Graphics Type":
			v.ElasticGraphicsType = row[index]
		case "Enhanced Networking Supported":
			v.EnhancedNetworkingSupported = row[index]
		case "From Region Code":
			v.FromRegionCode = row[index]
		case "GPU":
			v.GPU = row[index]
		case "GPU Memory":
			v.GPUMemory = row[index]
		case "Instance":
			v.Instance = row[index]
		case "Instance Capacity - 10xlarge":
			v.InstanceCapacity10xlarge = row[index]
		case "Instance Capacity - 12xlarge":
			v.InstanceCapacity12xlarge = row[index]
		case "Instance Capacity - 16xlarge":
			v.InstanceCapacity16xlarge = row[index]
		case "Instance Capacity - 18xlarge":
			v.InstanceCapacity18xlarge = row[index]
		case "Instance Capacity - 24xlarge":
			v.InstanceCapacity24xlarge = row[index]
		case "Instance Capacity - 2xlarge":
			v.InstanceCapacity2xlarge = row[index]
		case "Instance Capacity - 32xlarge":
			v.InstanceCapacity32xlarge = row[index]
		case "Instance Capacity - 4xlarge":
			v.InstanceCapacity4xlarge = row[index]
		case "Instance Capacity - 8xlarge":
			v.InstanceCapacity8xlarge = row[index]
		case "Instance Capacity - 9xlarge":
			v.InstanceCapacity9xlarge = row[index]
		case "Instance Capacity - large":
			v.InstanceCapacityLarge = row[index]
		case "Instance Capacity - medium":
			v.InstanceCapacityMedium = row[index]
		case "Instance Capacity - metal":
			v.InstanceCapacityMetal = row[index]
		case "Instance Capacity - xlarge":
			v.InstanceCapacityxlarge = row[index]
		case "instanceSKU":
			v.InstanceSKU = row[index]
		case "Intel AVX2 Available":
			v.IntelAVX2Available = row[index]
		case "Intel AVX Available":
			v.IntelAVXAvailable = row[index]
		case "Intel Turbo Available":
			v.IntelTurboAvailable = row[index]
		case "MarketOption":
			v.MarketOption = row[index]
		case "Normalization Size Factor":
			v.NormalizationSizeFactor = row[index]
		case "Physical Cores":
			v.PhysicalCores = row[index]
		case "Pre Installed S/W":
			v.PreInstalledSW = row[index]
		case "Processor Features":
			v.ProcessorFeatures = row[index]
		case "Product Type":
			v.ProductType = row[index]
		case "Region Code":
			v.RegionCode = row[index]
		case "Resource Type":
			v.ResourceType = row[index]
		case "serviceName":
			v.ServiceName = row[index]
		case "SnapshotArchiveFeeType":
			v.SnapshotArchiveFeeType = row[index]
		case "To Region Code":
			v.ToRegionCode = row[index]
		case "Volume API Name":
			v.VolumeAPIName = row[index]
		case "VPCNetworkingSupport":
			v.VPCNetworkingSupport = row[index]
		}
	}
}

func parseMemory(str string) int64 {
	str = strings.TrimSpace(strings.ToLower(str))
	if str == "na" {
		return -1
	}
	str = strings.TrimSuffix(str, " gib")
	n, _ := strconv.ParseInt(str, 10, 64)
	return n
}

func parseNetworkPerformance(v string) (int64, bool) {
	v = strings.ToLower(v)
	upTo := strings.HasPrefix(v, "up to ")
	v = strings.TrimPrefix(v, "up to ")

	factor := int64(0)
	if strings.HasSuffix(v, "gigabit") {
		factor = 1000000000 / 8
		v = strings.TrimSuffix(v, " gigabit")
	} else if strings.HasSuffix(v, "megabit") {
		factor = 1000000 / 8
		v = strings.TrimSuffix(v, " megabit")
	}
	b, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return 0, false
	}
	return b * factor, upTo
}
