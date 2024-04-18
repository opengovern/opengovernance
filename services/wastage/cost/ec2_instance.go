package cost

import (
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	types2 "github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/kaytu-io/kaytu-engine/pkg/httpclient"
	kaytu_client "github.com/kaytu-io/kaytu-engine/pkg/steampipe-plugin-kaytu/kaytu-client"
	awsResources "github.com/kaytu-io/pennywise-server/pkg/aws/resources"
	"github.com/kaytu-io/pennywise/pkg/cost"
	"github.com/kaytu-io/pennywise/pkg/schema"
	"net/http"
	"time"
)

func (s *Service) GetEC2InstanceCost(region string, instance types.Instance, volumes []types.Volume, metrics map[string][]types2.Datapoint) (float64, error) {
	req := schema.Submission{
		ID:        "submittion-1",
		CreatedAt: time.Now(),
		Resources: []schema.ResourceDef{},
	}

	var valuesMap awsResources.InstanceValues
	valuesMap.InstanceType = string(instance.InstanceType)
	if instance.Placement != nil {
		valuesMap.Tenancy = string(instance.Placement.Tenancy)
		valuesMap.AvailabilityZone = *instance.Placement.AvailabilityZone
		valuesMap.HostId = instance.Placement.HostId
	}
	valuesMap.EBSOptimized = instance.EbsOptimized
	if instance.Monitoring != nil {
		if instance.Monitoring.State == "disabled" || instance.Monitoring.State == "disabling" {
			valuesMap.EnableMonitoring = aws.Bool(false)
		} else {
			valuesMap.EnableMonitoring = aws.Bool(true)
		}
	}
	//if instance.CpuOptions != nil {
	//	valuesMap["credit_specification"] = []map[string]interface{}{{
	//		"cpu_credits": *instance.CpuOptions, //TODO - not sure
	//	}}
	//}
	var blockDevices []struct {
		DeviceName string  `mapstructure:"device_name"`
		VolumeType string  `mapstructure:"volume_type"`
		VolumeSize float64 `mapstructure:"volume_size"`
		IOPS       float64 `mapstructure:"iops"`
	}
	for _, v := range volumes {
		blockDevices = append(blockDevices, struct {
			DeviceName string  `mapstructure:"device_name"`
			VolumeType string  `mapstructure:"volume_type"`
			VolumeSize float64 `mapstructure:"volume_size"`
			IOPS       float64 `mapstructure:"iops"`
		}{
			DeviceName: *v.VolumeId,
			VolumeType: string(v.VolumeType),
			VolumeSize: float64(*v.Size),
			IOPS:       float64(*v.Iops),
		})
	}
	valuesMap.EbsBlockDevice = blockDevices
	valuesMap.LaunchTemplate = []struct {
		Id   *string `mapstructure:"id"`
		Name *string `mapstructure:"name"`
	}{}
	if instance.InstanceLifecycle == types.InstanceLifecycleTypeSpot {
		valuesMap.SpotPrice = "Spot"
	} else {
		valuesMap.SpotPrice = ""
	}

	os := "Linux"
	if instance.Platform != "" {
		os = string(instance.Platform)
	}
	valuesMap.Usage = struct {
		OperatingSystem               *string  `mapstructure:"operating_system"`
		ReservedInstanceType          *string  `mapstructure:"reserved_instance_type"`
		ReservedInstanceTerm          *string  `mapstructure:"reserved_instance_term"`
		ReservedInstancePaymentOption *string  `mapstructure:"reserved_instance_payment_option"`
		MonthlyCPUCreditHours         *int64   `mapstructure:"monthly_cpu_credit_hrs"`
		VcpuCount                     *int64   `mapstructure:"vcpu_count"`
		MonthlyHours                  *float64 `mapstructure:"monthly_hrs"`
	}{
		OperatingSystem: &os,
		//"reserved_instance_type": "",
		//"reserved_instance_term": "",
		//"reserved_instance_payment_option": "",
		//"monthly_cpu_credit_hrs": "",
		//"vcpu_count": "",
		MonthlyHours: aws.Float64(720),
	}

	jsonData, err := json.Marshal(instance)
	if err != nil {
		return 0, err
	}

	// Unmarshal the JSON to a map
	var result map[string]interface{}
	err = json.Unmarshal(jsonData, &result)
	if err != nil {
		return 0, err
	}

	req.Resources = append(req.Resources, schema.ResourceDef{
		Address:      *instance.InstanceId,
		Type:         kaytu_client.ResourceTypeConversion("aws::ec2::instance"),
		Name:         "",
		RegionCode:   region,
		ProviderName: schema.AWSProvider,
		Values:       result,
	})

	reqBody, err := json.Marshal(req)
	if err != nil {
		return 0, err
	}

	var response cost.State
	statusCode, err := httpclient.DoRequest("GET", s.pennywiseBaseUrl+"/api/v1/cost/submission", nil, reqBody, &response)
	if err != nil {
		return 0, err
	}

	if statusCode != http.StatusOK {
		return 0, fmt.Errorf("failed to get pennywise cost, status code = %d", statusCode)
	}

	resourceCost, err := response.Cost()
	if err != nil {
		return 0, err
	}

	return resourceCost.Decimal.InexactFloat64(), nil
}
