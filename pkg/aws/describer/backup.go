package describer

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/backup"
	"github.com/aws/aws-sdk-go-v2/service/backup/types"
)

func BackupBackupPlan(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := backup.NewFromConfig(cfg)
	paginator := backup.NewListBackupPlansPaginator(client, &backup.ListBackupPlansInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.BackupPlansList {
			values = append(values, Resource{
				ARN:         *v.BackupPlanArn,
				Description: v,
			})
		}
	}

	return values, nil
}

func BackupBackupSelection(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	plans, err := BackupBackupPlan(ctx, cfg)
	if err != nil {
		return nil, err
	}

	client := backup.NewFromConfig(cfg)

	var values []Resource
	for _, plan := range plans {
		paginator := backup.NewListBackupSelectionsPaginator(client, &backup.ListBackupSelectionsInput{
			BackupPlanId: plan.Description.(types.BackupPlansListMember).BackupPlanId,
		})

		for paginator.HasMorePages() {
			page, err := paginator.NextPage(ctx)
			if err != nil {
				return nil, err
			}

			for _, v := range page.BackupSelectionsList {
				values = append(values, Resource{
					ID:          CompositeID(*v.BackupPlanId, *v.SelectionId),
					Description: v,
				})
			}
		}
	}

	return values, nil
}

func BackupBackupVault(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := backup.NewFromConfig(cfg)
	paginator := backup.NewListBackupVaultsPaginator(client, &backup.ListBackupVaultsInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.BackupVaultList {
			values = append(values, Resource{
				ARN:         *v.BackupVaultArn,
				Description: v,
			})
		}
	}

	return values, nil
}
