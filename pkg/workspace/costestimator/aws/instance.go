package aws

import (
	"fmt"
	"github.com/kaytu-io/kaytu-engine/pkg/cost-estimator/es"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/costestimator"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/db"
	"strings"
)

func EC2InstanceCostByResource(db *db.CostEstimatorDatabase, instance es.EC2InstanceResponse) (float64, error) {
	var cost float64
	description := instance.Hits.Hits[0].Source.Description
	operatingSystem, err := getInstanceOperatingSystem(instance)
	if err != nil {
		return 0, err
	}
	instanceCost, err := db.FindEC2InstancePrice(instance.Hits.Hits[0].Source.Region, "Used", string(description.Instance.InstanceType),
		getTenancy(string(description.Instance.Placement.Tenancy)), operatingSystem, "NA", "Hrs")
	if err != nil {
		return 0, err
	}
	cost += instanceCost.Price * costestimator.TimeInterval

	for _, volume := range description.LaunchTemplateData.BlockDeviceMappings {
		volumeCost, err := calcEC2VolumeCost(db, instance.Hits.Hits[0].Source.Region, string(volume.Ebs.VolumeType), *volume.Ebs.VolumeSize, *volume.Ebs.Iops)
		if err != nil {
			return 0, err
		}
		cost += volumeCost * costestimator.TimeInterval
	}

	if description.LaunchTemplateData.CreditSpecification.CpuCredits != nil {
		if *description.LaunchTemplateData.CreditSpecification.CpuCredits == "unlimited" {
			region := strings.ToUpper(strings.Split(instance.Hits.Hits[0].Source.Region, "-")[0])
			instType := strings.Split(string(description.Instance.InstanceType), ".")[0]
			usageType := fmt.Sprintf("%s-CPUCredits:%s", region, instType)
			cpuCreditsCost, err := db.FindEC2CpuCreditsPrice(instance.Hits.Hits[0].Source.Region, operatingSystem, usageType, "vCPU-Hours")
			if err != nil {
				return 0, err
			}
			cost += cpuCreditsCost.Price * costestimator.TimeInterval
		}
	}

	if description.LaunchTemplateData.Monitoring.Enabled != nil {
		if *description.LaunchTemplateData.Monitoring.Enabled {
			cloudWatch, err := db.FindAmazonCloudWatchPrice(instance.Hits.Hits[0].Source.Region, 0, "Metrics")
			if err != nil {
				return 0, err
			}
			days := getNumberOfDays()
			cost += (((cloudWatch.Price * 7) / float64(days)) / 24) * costestimator.TimeInterval //TODO: Change this default metrics number
		}
	}

	if description.LaunchTemplateData.EbsOptimized != nil {
		if *description.LaunchTemplateData.EbsOptimized {
			region := strings.ToUpper(strings.Split(instance.Hits.Hits[0].Source.Region, "-")[0])
			usageType := fmt.Sprintf("%s-EBSOptimized:%s", region, string(description.Instance.InstanceType))
			ebsCost, err := db.FindEbsOptimizedPrice(instance.Hits.Hits[0].Source.Region, string(description.Instance.InstanceType), usageType, "Hrs")
			if err != nil {
				return 0, err
			}
			cost += ebsCost.Price * costestimator.TimeInterval
		}
	}

	return cost, nil
}

func getTenancy(tenancy string) string {
	if tenancy == "default" {
		return "Shared"
	} else if tenancy == "dedicated" {
		return "Dedicated"
	} else if tenancy == "host" {
		return "Hosted"
	} else {
		return tenancy
	}
}

// getInstanceOperatingSystem get instance operating system
// not sure about this function, should check operating systems in our resources and in cost tables
func getInstanceOperatingSystem(instance es.EC2InstanceResponse) (string, error) {
	instanceTags := instance.Hits.Hits[0].Source.Description.Instance.Tags
	launchTableDataTags := instance.Hits.Hits[0].Source.Description.LaunchTemplateData.TagSpecifications[0].Tags
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
