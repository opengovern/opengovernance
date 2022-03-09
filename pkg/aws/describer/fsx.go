package describer

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/fsx"
	"gitlab.com/keibiengine/keibi-engine/pkg/aws/model"
)

func FSXFileSystem(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := fsx.NewFromConfig(cfg)
	paginator := fsx.NewDescribeFileSystemsPaginator(client, &fsx.DescribeFileSystemsInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, item := range page.FileSystems {
			values = append(values, Resource{
				ARN:  *item.ResourceARN,
				Name: *item.FileSystemId,
				Description: model.FSXFileSystemDescription{
					FileSystem: item,
				},
			})
		}
	}

	return values, nil
}
