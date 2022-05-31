package describer

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/configservice"
	"gitlab.com/keibiengine/keibi-engine/pkg/aws/model"
)

func ConfigConfigurationRecorder(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	describeCtx := GetDescribeContext(ctx)

	client := configservice.NewFromConfig(cfg)
	out, err := client.DescribeConfigurationRecorders(ctx, &configservice.DescribeConfigurationRecordersInput{})
	if err != nil {
		return nil, err
	}

	var values []Resource
	for _, item := range out.ConfigurationRecorders {
		status, err := client.DescribeConfigurationRecorderStatus(ctx, &configservice.DescribeConfigurationRecorderStatusInput{
			ConfigurationRecorderNames: []string{*item.Name},
		})
		if err != nil {
			return nil, err
		}

		arn := "arn:" + describeCtx.Partition + ":config:" + describeCtx.Region + ":" + describeCtx.AccountID + ":config-recorder" + "/" + *item.Name
		values = append(values, Resource{
			ARN:  arn,
			Name: *item.Name,
			Description: model.ConfigConfigurationRecorderDescription{
				ConfigurationRecorder:        item,
				ConfigurationRecordersStatus: status.ConfigurationRecordersStatus[0],
			},
		})
	}

	return values, nil
}
