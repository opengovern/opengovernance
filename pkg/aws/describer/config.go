package describer

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/configservice"
	"github.com/aws/aws-sdk-go-v2/service/configservice/types"
)

type ConfigConfigurationRecorderDescription struct {
	ConfigurationRecorder        types.ConfigurationRecorder
	ConfigurationRecordersStatus types.ConfigurationRecorderStatus
}

func ConfigConfigurationRecorder(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	var values []Resource

	client := configservice.NewFromConfig(cfg)
	out, err := client.DescribeConfigurationRecorders(ctx, &configservice.DescribeConfigurationRecordersInput{})
	if err != nil {
		return nil, err
	}

	for _, item := range out.ConfigurationRecorders {
		status, err := client.DescribeConfigurationRecorderStatus(ctx, &configservice.DescribeConfigurationRecorderStatusInput{
			ConfigurationRecorderNames: []string{*item.Name},
		})
		if err != nil {
			return nil, err
		}

		values = append(values, Resource{
			ARN: *item.RoleARN,
			Description: ConfigConfigurationRecorderDescription{
				ConfigurationRecorder:        item,
				ConfigurationRecordersStatus: status.ConfigurationRecordersStatus[0],
			},
		})
	}

	return values, nil
}
