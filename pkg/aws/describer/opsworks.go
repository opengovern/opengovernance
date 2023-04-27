package describer

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/opsworkscm"
	"gitlab.com/keibiengine/keibi-engine/pkg/aws/model"
)

func OpsWorksCMServer(ctx context.Context, cfg aws.Config, stream *StreamSender) ([]Resource, error) {
	client := opsworkscm.NewFromConfig(cfg)
	paginator := opsworkscm.NewDescribeServersPaginator(client, &opsworkscm.DescribeServersInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.Servers {
			tags, err := client.ListTagsForResource(ctx, &opsworkscm.ListTagsForResourceInput{
				ResourceArn: v.ServerArn,
			})
			if err != nil {
				return nil, err
			}

			resource := Resource{
				ARN:  *v.ServerArn,
				Name: *v.ServerName,
				Description: model.OpsWorksCMServerDescription{
					Server: v,
					Tags:   tags.Tags,
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

	return values, nil
}
