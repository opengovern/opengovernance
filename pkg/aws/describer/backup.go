package describer

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/backup"
	"github.com/aws/aws-sdk-go-v2/service/backup/types"
)

func BackupBackupPlan(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
	client := backup.NewFromConfig(cfg)
	paginator := backup.NewListBackupPlansPaginator(client, &backup.ListBackupPlansInput{})

	var values []interface{}
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.BackupPlansList {
			values = append(values, v)
		}
	}

	return values, nil
}

func BackupBackupSelection(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
	plans, err := BackupBackupPlan(ctx, cfg)
	if err != nil {
		return nil, err
	}

	client := backup.NewFromConfig(cfg)

	var values []interface{}
	for _, plan := range plans {
		paginator := backup.NewListBackupSelectionsPaginator(client, &backup.ListBackupSelectionsInput{
			BackupPlanId: plan.(types.BackupPlansListMember).BackupPlanId,
		})

		for paginator.HasMorePages() {
			page, err := paginator.NextPage(ctx)
			if err != nil {
				return nil, err
			}

			for _, v := range page.BackupSelectionsList {
				values = append(values, v)
			}
		}
	}

	return values, nil
}

func BackupBackupVault(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
	client := backup.NewFromConfig(cfg)
	paginator := backup.NewListBackupVaultsPaginator(client, &backup.ListBackupVaultsInput{})

	var values []interface{}
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.BackupVaultList {
			values = append(values, v)
		}
	}

	return values, nil
}
