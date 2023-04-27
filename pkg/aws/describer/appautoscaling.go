package describer

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/applicationautoscaling"
	"github.com/aws/aws-sdk-go-v2/service/applicationautoscaling/types"
	"gitlab.com/keibiengine/keibi-engine/pkg/aws/model"
)

func ApplicationAutoScalingTarget(ctx context.Context, cfg aws.Config, stream *StreamSender) ([]Resource, error) {
	client := applicationautoscaling.NewFromConfig(cfg)

	describeCtx := GetDescribeContext(ctx)

	var values []Resource
	for _, serviceNameSpace := range types.ServiceNamespaceEcs.Values() {
		paginator := applicationautoscaling.NewDescribeScalableTargetsPaginator(client, &applicationautoscaling.DescribeScalableTargetsInput{
			ServiceNamespace: serviceNameSpace,
		})

		for paginator.HasMorePages() {
			page, err := paginator.NextPage(ctx)
			if err != nil {
				return nil, err
			}

			for _, item := range page.ScalableTargets {
				arn := "arn:" + describeCtx.Partition + ":application-autoscaling:" + describeCtx.Region + ":" + describeCtx.AccountID + ":service-namespace:" + string(item.ServiceNamespace) + "/target/" + *item.ResourceId

				resource := Resource{
					ARN:  arn,
					Name: *item.ResourceId,
					Description: model.ApplicationAutoScalingTargetDescription{
						ScalableTarget: item,
					},
				}
				if stream != nil {
					if err := (*stream)(resource); err != nil {
						return nil, err
					}
				} else {
					values = append(values, resource)
				}
			}
		}
	}

	return values, nil
}
