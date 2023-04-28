package describer

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/imagebuilder"
	"gitlab.com/keibiengine/keibi-engine/pkg/aws/model"
)

func ImageBuilderImage(ctx context.Context, cfg aws.Config, stream *StreamSender) ([]Resource, error) {
	client := imagebuilder.NewFromConfig(cfg)
	paginator := imagebuilder.NewListImagesPaginator(client, &imagebuilder.ListImagesInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.ImageVersionList {
			image, err := client.GetImage(ctx, &imagebuilder.GetImageInput{
				ImageBuildVersionArn: v.Arn,
			})
			if err != nil {
				return nil, err
			}

			resource := Resource{
				ARN:  *image.Image.Arn,
				Name: *image.Image.Name,
				Description: model.ImageBuilderImageDescription{
					Image: *image.Image,
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
