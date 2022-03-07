package describer

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/applicationautoscaling"
	"github.com/aws/aws-sdk-go-v2/service/applicationautoscaling/types"
)

type ApplicationAutoScalingTargetDescription struct {
	ScalableTarget types.ScalableTarget
}

func ApplicationAutoScalingTarget(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := applicationautoscaling.NewFromConfig(cfg)

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
				values = append(values, Resource{
					ID:   *item.ResourceId,
					Name: *item.ResourceId,
					Description: ApplicationAutoScalingTargetDescription{
						ScalableTarget: item,
					},
				})
			}
		}
	}

	return values, nil
}
