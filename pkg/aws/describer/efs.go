package describer

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/efs"
	"github.com/aws/aws-sdk-go-v2/service/efs/types"
)

const (
	efsPolicyNotFound = "PolicyNotFound"
)

func EFSAccessPoint(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := efs.NewFromConfig(cfg)
	paginator := efs.NewDescribeAccessPointsPaginator(client, &efs.DescribeAccessPointsInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.AccessPoints {
			values = append(values, Resource{
				ARN:         *v.AccessPointArn,
				Description: v,
			})
		}
	}

	return values, nil
}

type EFSFileSystemDescription struct {
	FileSystem types.FileSystemDescription
	Policy     *string
}

func EFSFileSystem(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := efs.NewFromConfig(cfg)
	paginator := efs.NewDescribeFileSystemsPaginator(client, &efs.DescribeFileSystemsInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.FileSystems {
			output, err := client.DescribeFileSystemPolicy(ctx, &efs.DescribeFileSystemPolicyInput{
				FileSystemId: v.FileSystemId,
			})
			if err != nil {
				if !isErr(err, efsPolicyNotFound) {
					return nil, err
				}

				output = &efs.DescribeFileSystemPolicyOutput{}
			}

			values = append(values, Resource{
				ARN: *v.FileSystemArn,
				Description: EFSFileSystemDescription{
					FileSystem: v,
					Policy:     output.Policy,
				},
			})
		}
	}

	return values, nil
}

func EFSMountTarget(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := efs.NewFromConfig(cfg)

	var values []Resource

	accessPoints, err := EFSAccessPoint(ctx, cfg)
	if err != nil {
		return nil, err
	}
	for _, ap := range accessPoints {
		accessPoint := ap.Description.(types.AccessPointDescription)
		err := PaginateRetrieveAll(func(prevToken *string) (nextToken *string, err error) {
			output, err := client.DescribeMountTargets(ctx, &efs.DescribeMountTargetsInput{
				AccessPointId: accessPoint.AccessPointId,
				Marker:        prevToken,
			})
			if err != nil {
				return nil, err
			}

			for _, v := range output.MountTargets {
				values = append(values, Resource{
					ID:          *v.MountTargetId,
					Description: v,
				})
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
		filesystem := fs.Description.(types.FileSystemDescription)
		err := PaginateRetrieveAll(func(prevToken *string) (nextToken *string, err error) {
			output, err := client.DescribeMountTargets(ctx, &efs.DescribeMountTargetsInput{
				FileSystemId: filesystem.FileSystemId,
				Marker:       prevToken,
			})
			if err != nil {
				return nil, err
			}

			for _, v := range output.MountTargets {
				values = append(values, Resource{
					ID:          *v.FileSystemId,
					Description: v,
				})
			}
			return output.NextMarker, nil
		})
		if err != nil {
			return nil, err
		}
	}

	return values, nil
}
