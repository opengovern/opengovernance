package describer

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/fsx"
	"gitlab.com/keibiengine/keibi-engine/pkg/aws/model"
)

func FSXFileSystem(ctx context.Context, cfg aws.Config, stream *StreamSender) ([]Resource, error) {
	client := fsx.NewFromConfig(cfg)
	paginator := fsx.NewDescribeFileSystemsPaginator(client, &fsx.DescribeFileSystemsInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, item := range page.FileSystems {
			resource := Resource{
				ARN:  *item.ResourceARN,
				Name: *item.FileSystemId,
				Description: model.FSXFileSystemDescription{
					FileSystem: item,
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

func FSXStorageVirtualMachine(ctx context.Context, cfg aws.Config, stream *StreamSender) ([]Resource, error) {
	client := fsx.NewFromConfig(cfg)
	paginator := fsx.NewDescribeStorageVirtualMachinesPaginator(client, &fsx.DescribeStorageVirtualMachinesInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, item := range page.StorageVirtualMachines {
			resource := Resource{
				ARN:  *item.ResourceARN,
				Name: *item.Name,
				Description: model.FSXStorageVirtualMachineDescription{
					StorageVirtualMachine: item,
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

func FSXTask(ctx context.Context, cfg aws.Config, stream *StreamSender) ([]Resource, error) {
	client := fsx.NewFromConfig(cfg)
	paginator := fsx.NewDescribeDataRepositoryTasksPaginator(client, &fsx.DescribeDataRepositoryTasksInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, item := range page.DataRepositoryTasks {
			resource := Resource{
				ARN:  *item.ResourceARN,
				Name: *item.TaskId,
				Description: model.FSXTaskDescription{
					Task: item,
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

func FSXVolume(ctx context.Context, cfg aws.Config, stream *StreamSender) ([]Resource, error) {
	client := fsx.NewFromConfig(cfg)
	paginator := fsx.NewDescribeVolumesPaginator(client, &fsx.DescribeVolumesInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, item := range page.Volumes {
			resource := Resource{
				ARN:  *item.ResourceARN,
				Name: *item.Name,
				Description: model.FSXVolumeDescription{
					Volume: item,
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

func FSXSnapshot(ctx context.Context, cfg aws.Config, stream *StreamSender) ([]Resource, error) {
	client := fsx.NewFromConfig(cfg)
	paginator := fsx.NewDescribeSnapshotsPaginator(client, &fsx.DescribeSnapshotsInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, item := range page.Snapshots {
			resource := Resource{
				ARN:  *item.ResourceARN,
				Name: *item.Name,
				Description: model.FSXSnapshotDescription{
					Snapshot: item,
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
