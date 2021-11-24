package describer

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/efs"
	"github.com/aws/aws-sdk-go-v2/service/efs/types"
)

func EFSAccessPoint(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
	client := efs.NewFromConfig(cfg)
	paginator := efs.NewDescribeAccessPointsPaginator(client, &efs.DescribeAccessPointsInput{})

	var values []interface{}
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.AccessPoints {
			values = append(values, v)
		}
	}

	return values, nil
}

func EFSFileSystem(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
	client := efs.NewFromConfig(cfg)
	paginator := efs.NewDescribeFileSystemsPaginator(client, &efs.DescribeFileSystemsInput{})

	var values []interface{}
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.FileSystems {
			values = append(values, v)
		}
	}

	return values, nil
}

func EFSMountTarget(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
	client := efs.NewFromConfig(cfg)

	var values []interface{}

	accessPoints, err := EFSAccessPoint(ctx, cfg)
	if err != nil {
		return nil, err
	}
	for _, ap := range accessPoints {
		accessPoint := ap.(types.AccessPointDescription)
		err := PaginateRetrieveAll(func(prevToken *string) (nextToken *string, err error) {
			output, err := client.DescribeMountTargets(ctx, &efs.DescribeMountTargetsInput{
				AccessPointId: accessPoint.AccessPointId,
				Marker:        prevToken,
			})
			if err != nil {
				return nil, err
			}

			for _, v := range output.MountTargets {
				values = append(values, v)
			}
			return output.NextMarker, nil
		})
		if err != nil {
			return nil, err
		}
	}

	filesystems, err := EFSFileSystem(ctx, cfg)
	if err != nil {
		return nil, err
	}
	for _, fs := range filesystems {
		filesystem := fs.(types.FileSystemDescription)
		err := PaginateRetrieveAll(func(prevToken *string) (nextToken *string, err error) {
			output, err := client.DescribeMountTargets(ctx, &efs.DescribeMountTargetsInput{
				FileSystemId: filesystem.FileSystemId,
				Marker:       prevToken,
			})
			if err != nil {
				return nil, err
			}

			for _, v := range output.MountTargets {
				values = append(values, v)
			}
			return output.NextMarker, nil
		})
		if err != nil {
			return nil, err
		}
	}

	return values, nil
}
