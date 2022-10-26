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

func FSXStorageVirtualMachine(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := fsx.NewFromConfig(cfg)
	paginator := fsx.NewDescribeStorageVirtualMachinesPaginator(client, &fsx.DescribeStorageVirtualMachinesInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, item := range page.StorageVirtualMachines {
			values = append(values, Resource{
				ARN:  *item.ResourceARN,
				Name: *item.Name,
				Description: model.FSXStorageVirtualMachineDescription{
					StorageVirtualMachine: item,
				},
			})
		}
	}

	return values, nil
}

func FSXTask(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := fsx.NewFromConfig(cfg)
	paginator := fsx.NewDescribeDataRepositoryTasksPaginator(client, &fsx.DescribeDataRepositoryTasksInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, item := range page.DataRepositoryTasks {
			values = append(values, Resource{
				ARN:  *item.ResourceARN,
				Name: *item.TaskId,
				Description: model.FSXTaskDescription{
					Task: item,
				},
			})
		}
	}

	return values, nil
}

func FSXVolume(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := fsx.NewFromConfig(cfg)
	paginator := fsx.NewDescribeVolumesPaginator(client, &fsx.DescribeVolumesInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, item := range page.Volumes {
			values = append(values, Resource{
				ARN:  *item.ResourceARN,
				Name: *item.Name,
				Description: model.FSXVolumeDescription{
					Volume: item,
				},
			})
		}
	}

	return values, nil
}

func FSXSnapshot(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := fsx.NewFromConfig(cfg)
	paginator := fsx.NewDescribeSnapshotsPaginator(client, &fsx.DescribeSnapshotsInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, item := range page.Snapshots {
			values = append(values, Resource{
				ARN:  *item.ResourceARN,
				Name: *item.Name,
				Description: model.FSXSnapshotDescription{
					Snapshot: item,
				},
			})
		}
	}

	return values, nil
}
