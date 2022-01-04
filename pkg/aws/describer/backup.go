package describer

import (
	"context"
	"github.com/aws/aws-sdk-go/aws/awserr"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/backup"
	"github.com/aws/aws-sdk-go-v2/service/backup/types"
)

type BackupBackupPlanDescription struct {
	BackupPlan types.BackupPlansListMember
}

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
				ARN: *v.BackupPlanArn,
				Description: BackupBackupPlanDescription{
					BackupPlan: v,
				},
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

type BackupBackupVaultDescription struct {
	BackupVault       types.BackupVaultListMember
	Policy            *string
	BackupVaultEvents []types.BackupVaultEvent
	SNSTopicArn       *string
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
			notification, err := client.GetBackupVaultNotifications(ctx, &backup.GetBackupVaultNotificationsInput{
				BackupVaultName: v.BackupVaultName,
			})
			if err != nil {
				if a, ok := err.(awserr.Error); ok {
					if a.Code() == "ResourceNotFoundException" || a.Code() == "InvalidParameter" {
						notification = &backup.GetBackupVaultNotificationsOutput{}
					} else {
						return nil, err
					}
				}
			}

			accessPolicy, err := client.GetBackupVaultAccessPolicy(ctx, &backup.GetBackupVaultAccessPolicyInput{
				BackupVaultName: v.BackupVaultName,
			})
			if a, ok := err.(awserr.Error); ok {
				if a.Code() == "ResourceNotFoundException" || a.Code() == "InvalidParameter" {
					accessPolicy = &backup.GetBackupVaultAccessPolicyOutput{}
				} else {
					return nil, err
				}
			}

			values = append(values, Resource{
				ARN: *v.BackupVaultArn,
				Description: BackupBackupVaultDescription{
					BackupVault:       v,
					Policy:            accessPolicy.Policy,
					BackupVaultEvents: notification.BackupVaultEvents,
					SNSTopicArn:       notification.SNSTopicArn,
				},
			})
		}
	}

	return values, nil
}
