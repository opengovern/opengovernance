package describer

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/drs"
	"gitlab.com/keibiengine/keibi-engine/pkg/aws/model"
)

func DRSSourceServer(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := drs.NewFromConfig(cfg)
	paginator := drs.NewDescribeSourceServersPaginator(client, &drs.DescribeSourceServersInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.Items {
			values = append(values, Resource{
				ARN:  *v.Arn,
				Name: *v.SourceServerID,
				Description: model.DRSSourceServerDescription{
					SourceServer: v,
				},
			})
		}
	}

	return values, nil
}

func DRSRecoveryInstance(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := drs.NewFromConfig(cfg)
	paginator := drs.NewDescribeRecoveryInstancesPaginator(client, &drs.DescribeRecoveryInstancesInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.Items {
			values = append(values, Resource{
				ARN:  *v.Arn,
				Name: *v.RecoveryInstanceID,
				Description: model.DRSRecoveryInstanceDescription{
					RecoveryInstance: v,
				},
			})
		}
	}

	return values, nil
}
