package describer

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/aws-sdk-go-v2/service/ssm/types"
)

type SSMManagedInstanceDescription struct {
	InstanceInformation types.InstanceInformation
}

func SSMManagedInstance(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := ssm.NewFromConfig(cfg)
	paginator := ssm.NewDescribeInstanceInformationPaginator(client, &ssm.DescribeInstanceInformationInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, item := range page.InstanceInformationList {
			values = append(values, Resource{
				ID: *item.InstanceId,
				Description: SSMManagedInstanceDescription{
					InstanceInformation: item,
				},
			})
		}
	}
	return values, nil
}

func SSMAssociation(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := ssm.NewFromConfig(cfg)
	paginator := ssm.NewListAssociationsPaginator(client, &ssm.ListAssociationsInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.Associations {
			values = append(values, Resource{
				ID:          *v.AssociationId,
				Description: v,
			})
		}
	}

	return values, nil
}

func SSMDocument(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := ssm.NewFromConfig(cfg)
	paginator := ssm.NewListDocumentsPaginator(client, &ssm.ListDocumentsInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.DocumentIdentifiers {
			values = append(values, Resource{
				ID:          *v.Name,
				Description: v,
			})
		}
	}

	return values, nil
}

func SSMMaintenanceWindow(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := ssm.NewFromConfig(cfg)
	paginator := ssm.NewDescribeMaintenanceWindowsPaginator(client, &ssm.DescribeMaintenanceWindowsInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.WindowIdentities {
			values = append(values, Resource{
				ID:          *v.WindowId,
				Description: v,
			})
		}
	}

	return values, nil
}

func SSMMaintenanceWindowTarget(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	windows, err := SSMMaintenanceWindow(ctx, cfg)
	if err != nil {
		return nil, err
	}

	client := ssm.NewFromConfig(cfg)

	var values []Resource
	for _, w := range windows {
		window := w.Description.(types.MaintenanceWindowIdentity)
		paginator := ssm.NewDescribeMaintenanceWindowTargetsPaginator(client, &ssm.DescribeMaintenanceWindowTargetsInput{
			WindowId: window.WindowId,
		})

		for paginator.HasMorePages() {
			page, err := paginator.NextPage(ctx)
			if err != nil {
				return nil, err
			}

			for _, v := range page.Targets {
				values = append(values, Resource{
					ID:          *v.WindowTargetId,
					Description: v,
				})
			}
		}
	}

	return values, nil
}

func SSMMaintenanceWindowTask(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	windows, err := SSMMaintenanceWindow(ctx, cfg)
	if err != nil {
		return nil, err
	}

	client := ssm.NewFromConfig(cfg)

	var values []Resource
	for _, w := range windows {
		window := w.Description.(types.MaintenanceWindowIdentity)
		paginator := ssm.NewDescribeMaintenanceWindowTasksPaginator(client, &ssm.DescribeMaintenanceWindowTasksInput{
			WindowId: window.WindowId,
		})

		for paginator.HasMorePages() {
			page, err := paginator.NextPage(ctx)
			if err != nil {
				return nil, err
			}

			for _, v := range page.Tasks {
				values = append(values, Resource{
					ARN:         *v.TaskArn,
					Description: v,
				})
			}
		}
	}

	return values, nil
}

func SSMParameter(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := ssm.NewFromConfig(cfg)
	paginator := ssm.NewDescribeParametersPaginator(client, &ssm.DescribeParametersInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.Parameters {
			values = append(values, Resource{
				ID:          *v.Name,
				Description: v,
			})
		}
	}

	return values, nil
}

func SSMPatchBaseline(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := ssm.NewFromConfig(cfg)
	paginator := ssm.NewDescribePatchBaselinesPaginator(client, &ssm.DescribePatchBaselinesInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.BaselineIdentities {
			values = append(values, Resource{
				ID:          *v.BaselineId,
				Description: v,
			})
		}
	}

	return values, nil
}

func SSMResourceDataSync(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := ssm.NewFromConfig(cfg)
	paginator := ssm.NewListResourceDataSyncPaginator(client, &ssm.ListResourceDataSyncInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.ResourceDataSyncItems {
			values = append(values, Resource{
				ID:          *v.SyncName,
				Description: v,
			})
		}
	}

	return values, nil
}
