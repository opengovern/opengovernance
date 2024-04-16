package cost

import (
	"encoding/json"
	"fmt"
	types2 "github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/kaytu-io/kaytu-aws-describer/aws/model"
	"github.com/kaytu-io/kaytu-engine/pkg/httpclient"
	kaytu_client "github.com/kaytu-io/kaytu-engine/pkg/steampipe-plugin-kaytu/kaytu-client"
	"github.com/kaytu-io/pennywise/pkg/cost"
	"github.com/kaytu-io/pennywise/pkg/schema"
	"net/http"
	"time"
)

func (s *Service) GetEC2InstanceCost(region string, instance model.EC2InstanceDescription, volumes []model.EC2VolumeDescription, metrics map[string][]types2.Datapoint) (float64, error) {
	req := schema.Submission{
		ID:        "submittion-1",
		CreatedAt: time.Now(),
		Resources: []schema.ResourceDef{},
	}

	valuesMap := map[string]interface{}{}
	valuesMap["instance_type"] = instance.Instance.InstanceType
	if instance.Instance.Placement != nil {
		valuesMap["tenancy"] = instance.Instance.Placement.Tenancy
		valuesMap["availability_zone"] = *instance.Instance.Placement.AvailabilityZone
	}
	valuesMap["ebs_optimized"] = *instance.Instance.EbsOptimized
	if instance.Instance.Monitoring != nil {
		if instance.Instance.Monitoring.State == "disabled" || instance.Instance.Monitoring.State == "disabling" {
			valuesMap["monitoring"] = false
		} else {
			valuesMap["monitoring"] = true
		}
	}
	if instance.Instance.CpuOptions != nil {
		valuesMap["credit_specification"] = []map[string]interface{}{{
			"cpu_credits": *instance.Instance.CpuOptions, //TODO - not sure
		}}
	}
	var blockDevices []map[string]interface{}
	for _, v := range volumes {
		blockDevices = append(blockDevices, map[string]interface{}{
			"device_name": *v.Volume.VolumeId,
			"volume_type": v.Volume.VolumeType,
			"volume_size": *v.Volume.Size,
			"iops":        *v.Volume.Iops,
		})
	}
	valuesMap["ebs_block_device"] = blockDevices
	valuesMap["launch_template"] = []map[string]interface{}{
		{
			"id":   instance.LaunchTemplateData.KeyName,
			"name": instance.LaunchTemplateData.KeyName,
		},
	}
	if instance.Instance.InstanceLifecycle == types.InstanceLifecycleTypeSpot {
		valuesMap["spot_price"] = "Spot"
	}
	// valuesMap["host_id"] = WTF??
	os := "Linux"
	if instance.Instance.Platform != "" {
		os = string(instance.Instance.Platform)
	}
	valuesMap["pennywise_usage"] = []map[string]interface{}{{
		"operating_system": os,
		//"reserved_instance_type": "",
		//"reserved_instance_term": "",
		//"reserved_instance_payment_option": "",
		//"monthly_cpu_credit_hrs": "",
		//"vcpu_count": "",
		"monthly_hrs": "720",
	}}

	req.Resources = append(req.Resources, schema.ResourceDef{
		Address:      *instance.Instance.InstanceId,
		Type:         kaytu_client.ResourceTypeConversion("aws::ec2::instance"),
		Name:         "",
		RegionCode:   region,
		ProviderName: schema.AWSProvider,
		Values:       valuesMap,
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
