package describer

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/efs"
	"github.com/aws/aws-sdk-go-v2/service/efs/types"
	"gitlab.com/keibiengine/keibi-engine/pkg/aws/model"
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

			// Doc: The name of the access point. This is the value of the Name tag.
			name := aws.ToString(v.Name)
			if name == "" {
				name = *v.AccessPointId
			}

			values = append(values, Resource{
				ARN:         *v.AccessPointArn,
				Name:        name,
				Description: v,
			})
		}
	}

	return values, nil
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

			// Doc: You can add tags to a file system, including a Name tag. For more information,
			// see CreateFileSystem. If the file system has a Name tag, Amazon EFS returns the
			// value in this field.
			name := aws.ToString(v.Name)
			if name == "" {
				name = *v.FileSystemId
			}

			values = append(values, Resource{
				ARN:  *v.FileSystemArn,
				Name: name,
				Description: model.EFSFileSystemDescription{
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

	describeCtx := GetDescribeContext(ctx)

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
				arn := "arn:" + describeCtx.Partition + ":elasticfilesystem:" + describeCtx.Region + ":" + describeCtx.AccountID + ":file-system/" + *v.FileSystemId + "/mount-target/" + *v.MountTargetId
				values = append(values, Resource{
					ARN:         arn,
					Name:        *v.MountTargetId,
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
					Name:        *v.AvailabilityZoneName,
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
