package describer

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/efs"
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
			name := aws.ToString(v.Name)
			if name == "" {
				name = *v.AccessPointId
			}

			values = append(values, Resource{
				ARN:  *v.AccessPointArn,
				Name: name,
				Description: model.EFSAccessPointDescription{
					AccessPoint: v,
				},
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

	filesystems, err := EFSFileSystem(ctx, cfg)
	if err != nil {
		return nil, err
	}
	for _, fs := range filesystems {
		filesystem := fs.Description.(model.EFSFileSystemDescription)
		err := PaginateRetrieveAll(func(prevToken *string) (nextToken *string, err error) {
			output, err := client.DescribeMountTargets(ctx, &efs.DescribeMountTargetsInput{
				FileSystemId: filesystem.FileSystem.FileSystemId,
				Marker:       prevToken,
			})
			if err != nil {
				return nil, err
			}

			for _, v := range output.MountTargets {
				arn := fmt.Sprintf("arn:%s:elasticfilesystem:%s:%s:file-system/%s/mount-target/%s", describeCtx.Partition, describeCtx.Region, describeCtx.AccountID, *filesystem.FileSystem.FileSystemId, *v.MountTargetId)

				securityGroups, err := client.DescribeMountTargetSecurityGroups(ctx, &efs.DescribeMountTargetSecurityGroupsInput{
					MountTargetId: v.MountTargetId,
				})
				if err != nil {
					return nil, err
				}

				values = append(values, Resource{
					ARN: arn,
					ID:  *v.MountTargetId,
					Description: model.EFSMountTargetDescription{
						MountTarget:    v,
						SecurityGroups: securityGroups.SecurityGroups,
					},
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
