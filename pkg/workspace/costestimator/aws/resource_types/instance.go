package terraform

import (
	"fmt"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/api"
	"strings"

	"github.com/shopspring/decimal"

	"github.com/kaytu-io/kaytu-engine/pkg/workspace/costestimator/aws/region"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/costestimator/price"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/costestimator/product"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/costestimator/query"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/costestimator/util"
)

// Instance represents an EC2 instance definition that can be cost-estimated.
type Instance struct {
	provider     *Provider
	region       region.Code
	instanceType string

	// tenancy describes the tenancy of an instance.
	// Valid values include: Shared, Dedicated, Host.
	// Note: only "Shared" and "Dedicated" are supported at the moment.
	tenancy string

	// operatingSystem denotes the OS that the instance is using that may affect pricing.
	// Valid values include: Linux, RHEL, SUSE, Windows.
	// Note: only "Linux" is supported at the moment.
	operatingSystem string

	// capacityStatus describes the status of capacity reservations.
	// Valid values include: Used, UnusedCapacityReservation, AllocatedCapacityReservation.
	// Note: only "Used" is supported at the moment.
	capacityStatus string

	// preInstalledSW denotes any pre-installed software that may affect pricing.
	// Valid values include: NA, SQL Std, SQL Web, SQL Ent.
	// Note: only "NA" (no pre-installed software) is supported at the moment.
	preInstalledSW string

	// Credit option for CPU usage. Valid values include standard or unlimited
	cpuCredits bool

	ebsOptimized     bool
	enableMonitoring bool

	// instanceCount number of instance provisionned.
	// Currently used by ASG
	instanceCount decimal.Decimal

	rootVolume *Volume
}

// instanceValues represents the structure of Terraform values for aws_instance resource.
type instanceValues struct {
	RegionCode       string
	InstanceType     string
	Tenancy          string
	AvailabilityZone string
	OperatingSystem  string

	EBSOptimized        bool
	EnableMonitoring    bool
	CreditSpecification []struct {
		CPUCredits string
	}

	RootBlockDevice []struct {
		VolumeType string
		VolumeSize float64
		IOPS       float64
	}
}

// decodeInstanceValues decodes and returns instanceValues from a Terraform values map.
func decodeInstanceValues(request api.GetEC2InstanceCostRequest) (*instanceValues, error) {
	operatingSystem, err := getInstanceOperatingSystem(request)
	if err != nil {
		return nil, err
	}
	enableMonitoring := false
	if request.Instance.LaunchTemplateData.Monitoring.Enabled != nil {
		if *request.Instance.LaunchTemplateData.Monitoring.Enabled {
			enableMonitoring = true
		}
	}
	cpuCredits := []struct {
		CPUCredits string
	}{{CPUCredits: *request.Instance.LaunchTemplateData.CreditSpecification.CpuCredits}}
	var rootBlockDevice []struct {
		VolumeType string
		VolumeSize float64
		IOPS       float64
	}
	for _, volume := range request.Instance.LaunchTemplateData.BlockDeviceMappings {
		rootBlockDevice = append(rootBlockDevice, struct {
			VolumeType string
			VolumeSize float64
			IOPS       float64
		}{
			VolumeType: string(volume.Ebs.VolumeType),
			VolumeSize: float64(*volume.Ebs.VolumeSize),
			IOPS:       float64(*volume.Ebs.Iops),
		})
	}
	return &instanceValues{
		RegionCode:       request.RegionCode,
		InstanceType:     string(request.Instance.Instance.InstanceType),
		Tenancy:          string(request.Instance.Instance.Placement.Tenancy),
		AvailabilityZone: request.RegionCode,
		OperatingSystem:  operatingSystem,

		EBSOptimized:        request.EBSOptimized,
		EnableMonitoring:    enableMonitoring,
		CreditSpecification: cpuCredits,

		RootBlockDevice: rootBlockDevice,
	}, nil
}

// getInstanceOperatingSystem get instance operating system
// not sure about this function, should check operating systems in our resources and in cost tables
func getInstanceOperatingSystem(request api.GetEC2InstanceCostRequest) (string, error) {
	instanceTags := request.Instance.Instance.Tags
	launchTableDataTags := request.Instance.LaunchTemplateData.TagSpecifications[0].Tags
	var operatingSystem string
	for _, tag := range instanceTags {
		if *tag.Key == "wk_gbs_interpreted_os_type" {
			operatingSystem = *tag.Value
			break
		}
	}
	if operatingSystem == "" {
		for _, tag := range launchTableDataTags {
			if *tag.Key == "wk_gbs_interpreted_os_type" {
				operatingSystem = *tag.Value
				break
			}
		}
	}
	if operatingSystem == "" {
		return "", fmt.Errorf("could not find operating system")
	}
	if strings.Contains(operatingSystem, "Linux") {
		return "Linux", nil
	} else if strings.Contains(operatingSystem, "Windows") { // Make sure
		return "Windows", nil
	} else {
		return operatingSystem, nil
	}
}

// newInstance creates a new Instance from instanceValues.
func (p *Provider) newInstance(vals instanceValues) *Instance {
	inst := &Instance{
		provider: p,
		region:   region.Code(vals.RegionCode),
		tenancy:  "Shared",

		// Note: every Instance is estimated as a Linux without pre-installed S/W
		operatingSystem: vals.OperatingSystem,
		capacityStatus:  "Used",
		preInstalledSW:  "NA",
		instanceCount:   decimal.NewFromInt(1),

		instanceType: vals.InstanceType,
	}

	if reg := region.NewFromZone(vals.AvailabilityZone); reg.Valid() {
		inst.region = reg
	}

	if vals.Tenancy == "dedicated" {
		inst.tenancy = "Dedicated"
	} else if vals.Tenancy == "host" {
		inst.tenancy = "Hosted"
	}

	if vals.EBSOptimized {
		inst.ebsOptimized = true
	}

	if len(vals.CreditSpecification) > 0 {
		creditspec := vals.CreditSpecification[0]
		if creditspec.CPUCredits == "unlimited" {
			inst.cpuCredits = true
		}
	}

	if vals.EnableMonitoring {
		inst.enableMonitoring = true
	}

	volVals := volumeValues{AvailabilityZone: vals.AvailabilityZone}
	if len(vals.RootBlockDevice) > 0 {
		rbd := vals.RootBlockDevice[0]
		volVals.Type = rbd.VolumeType
		volVals.Size = rbd.VolumeSize
		volVals.IOPS = rbd.IOPS
	}
	inst.rootVolume = p.newVolume(volVals)

	return inst
}

// Components returns the price component queries that make up this Instance.
func (inst *Instance) Components() []query.Component {
	components := []query.Component{inst.computeComponent()}

	if inst.rootVolume != nil {
		for _, comp := range inst.rootVolume.Components() {
			comp.Name = "Root volume: " + comp.Name
			components = append(components, comp)
		}
	}

	if inst.cpuCredits {
		components = append(components, inst.cpuCreditCostComponent())
	}

	if inst.enableMonitoring {
		components = append(components, inst.detailedMonitoringCostComponent())
	}

	if inst.ebsOptimized {
		components = append(components, inst.ebsOptimizedCostComponent())
	}

	return components
}

func (inst *Instance) cpuCreditCostComponent() query.Component {

	// Used to generate the UsageType
	region := strings.ToUpper(strings.Split(inst.region.String(), "-")[0])
	instType := strings.Split(inst.instanceType, ".")[0]

	return query.Component{
		Name:           "CPUCreditCost",
		Details:        []string{inst.operatingSystem, "on-demand", inst.instanceType},
		HourlyQuantity: inst.instanceCount,
		ProductFilter: &product.Filter{
			Provider: util.StringPtr(inst.provider.key),
			Service:  util.StringPtr("AmazonEC2"),
			Family:   util.StringPtr("CPU Credits"),
			Location: util.StringPtr(inst.region.String()),
			AttributeFilters: []*product.AttributeFilter{
				{Key: "OperatingSystem", Value: util.StringPtr(inst.operatingSystem)},
				{Key: "UsageType", Value: util.StringPtr(fmt.Sprintf("%s-CPUCredits:%s", region, instType))},
			},
		},
		PriceFilter: &price.Filter{
			Unit: util.StringPtr("vCPU-Hours"),
			AttributeFilters: []*price.AttributeFilter{
				{Key: "TermType", Value: util.StringPtr("OnDemand")},
			},
		},
	}
}

func (inst *Instance) detailedMonitoringCostComponent() query.Component {
	var defaultEC2InstanceMetricCount = decimal.NewFromInt(7)
	quantity := defaultEC2InstanceMetricCount.Mul(inst.instanceCount)

	return query.Component{
		Name:            "EC2 detailed monitoring",
		Details:         []string{"on-demand", "monitoring"},
		MonthlyQuantity: quantity,
		ProductFilter: &product.Filter{
			Provider:         util.StringPtr(inst.provider.key),
			Service:          util.StringPtr("AmazonCloudWatch"),
			Family:           util.StringPtr("Metric"),
			Location:         util.StringPtr(inst.region.String()),
			AttributeFilters: []*product.AttributeFilter{},
		},
		PriceFilter: &price.Filter{
			Unit: util.StringPtr("Metrics"),
			AttributeFilters: []*price.AttributeFilter{
				{Key: "TermType", Value: util.StringPtr("OnDemand")},
				{Key: "StartingRange", Value: util.StringPtr("0")},
			},
		},
	}
}

func (inst *Instance) ebsOptimizedCostComponent() query.Component {

	// Used to generate the UsageType
	region := strings.ToUpper(strings.Split(inst.region.String(), "-")[0])
	return query.Component{
		Name:           "EBS-optimized usage",
		Details:        []string{"EBS", "Optimizes", inst.instanceType},
		HourlyQuantity: inst.instanceCount,
		ProductFilter: &product.Filter{
			Provider: util.StringPtr(inst.provider.key),
			Service:  util.StringPtr("AmazonEC2"),
			Family:   util.StringPtr("Compute Instance"),
			Location: util.StringPtr(inst.region.String()),
			AttributeFilters: []*product.AttributeFilter{
				{Key: "InstanceType", Value: util.StringPtr(inst.instanceType)},
				{Key: "UsageType", Value: util.StringPtr(fmt.Sprintf("%s-EBSOptimized:%s", region, inst.instanceType))},
			},
		},
		PriceFilter: &price.Filter{
			Unit: util.StringPtr("Hrs"),
			AttributeFilters: []*price.AttributeFilter{
				{Key: "TermType", Value: util.StringPtr("OnDemand")},
			},
		},
	}
}

func (inst *Instance) computeComponent() query.Component {
	return query.Component{
		Name:           "Compute",
		Details:        []string{inst.operatingSystem, "on-demand", inst.instanceType},
		HourlyQuantity: inst.instanceCount,
		ProductFilter: &product.Filter{
			Provider: util.StringPtr(inst.provider.key),
			Service:  util.StringPtr("AmazonEC2"),
			Family:   util.StringPtr("Compute Instance"),
			Location: util.StringPtr(inst.region.String()),
			AttributeFilters: []*product.AttributeFilter{
				{Key: "CapacityStatus", Value: util.StringPtr(inst.capacityStatus)},
				{Key: "InstanceType", Value: util.StringPtr(inst.instanceType)},
				{Key: "Tenancy", Value: util.StringPtr(inst.tenancy)},
				{Key: "OperatingSystem", Value: util.StringPtr(inst.operatingSystem)},
				{Key: "PreInstalledSW", Value: util.StringPtr(inst.preInstalledSW)},
			},
		},
		PriceFilter: &price.Filter{
			Unit: util.StringPtr("Hrs"),
			AttributeFilters: []*price.AttributeFilter{
				{Key: "TermType", Value: util.StringPtr("OnDemand")},
			},
		},
	}
}
