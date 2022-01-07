package describer

import (
	"context"
	"github.com/aws/aws-sdk-go/aws/awserr"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/backup"
	"github.com/aws/aws-sdk-go-v2/service/backup/types"
)

type BackupPlanDescription struct {
	BackupPlan types.BackupPlansListMember
}

func BackupPlan(ctx context.Context, cfg aws.Config) ([]Resource, error) {
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
				Description: BackupPlanDescription{
					BackupPlan: v,
				},
			})
		}
	}

	return values, nil
}

type BackupRecoveryPointDescription struct {
	RecoveryPoint *backup.DescribeRecoveryPointOutput
}

func BackupRecoveryPoint(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := backup.NewFromConfig(cfg)
	paginator := backup.NewListBackupVaultsPaginator(client, &backup.ListBackupVaultsInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, item := range page.BackupVaultList {
			recoveryPointPaginator := backup.NewListRecoveryPointsByBackupVaultPaginator(client,
				&backup.ListRecoveryPointsByBackupVaultInput{
					BackupVaultName: item.BackupVaultName,
				})

			for recoveryPointPaginator.HasMorePages() {
				page, err := recoveryPointPaginator.NextPage(ctx)
				if err != nil {
					return nil, err
				}

				for _, recoveryPoint := range page.RecoveryPoints {
					out, err := client.DescribeRecoveryPoint(ctx, &backup.DescribeRecoveryPointInput{
						BackupVaultName:  recoveryPoint.BackupVaultName,
						RecoveryPointArn: recoveryPoint.RecoveryPointArn,
					})
					if err != nil {
						return nil, err
					}

					values = append(values, Resource{
						ARN: *recoveryPoint.RecoveryPointArn,
						Description: BackupRecoveryPointDescription{
							RecoveryPoint: out,
						},
					})
				}
			}
		}
	}

	return values, nil
}

type BackupProtectedResourceDescription struct {
	ProtectedResource types.ProtectedResource
}

func BackupProtectedResource(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := backup.NewFromConfig(cfg)
	paginator := backup.NewListProtectedResourcesPaginator(client, &backup.ListProtectedResourcesInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, resource := range page.Results {
			values = append(values, Resource{
				ARN: *resource.ResourceArn,
				Description: BackupProtectedResourceDescription{
					ProtectedResource: resource,
				},
			})
		}
	}

	return values, nil
}

type BackupSelectionDescription struct {
	BackupSelection types.BackupSelectionsListMember
	ListOfTags      []types.Condition
	Resources       []string
}

func BackupSelection(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	plans, err := BackupPlan(ctx, cfg)
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
				out, err := client.GetBackupSelection(ctx, &backup.GetBackupSelectionInput{
					BackupPlanId: v.BackupPlanId,
					SelectionId:  v.SelectionId,
				})
				if err != nil {
					return nil, err
				}

				values = append(values, Resource{
					ID: CompositeID(*v.BackupPlanId, *v.SelectionId),
					Description: BackupSelectionDescription{
						BackupSelection: v,
						ListOfTags:      out.BackupSelection.ListOfTags,
						Resources:       out.BackupSelection.Resources,
					},
				})
			}
		}
	}

	return values, nil
}

type BackupVaultDescription struct {
	BackupVault       types.BackupVaultListMember
	Policy            *string
	BackupVaultEvents []types.BackupVaultEvent
	SNSTopicArn       *string
}

func BackupVault(ctx context.Context, cfg aws.Config) ([]Resource, error) {
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
				Description: BackupVaultDescription{
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
