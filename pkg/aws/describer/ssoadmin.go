package describer

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssoadmin"
	"gitlab.com/keibiengine/keibi-engine/pkg/aws/model"
)

func SSOAdminInstance(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := ssoadmin.NewFromConfig(cfg)
	paginator := ssoadmin.NewListInstancesPaginator(client, &ssoadmin.ListInstancesInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, v := range page.Instances {
			values = append(values, Resource{
				ARN:  *v.InstanceArn,
				Name: *v.InstanceArn,
				Description: model.SSOAdminInstanceDescription{
					Instance: v,
				},
			})
		}
	}
	return values, nil
}
