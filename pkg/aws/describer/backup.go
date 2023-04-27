package describer

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/backup"
	"gitlab.com/keibiengine/keibi-engine/pkg/aws/model"
)

func BackupPlan(ctx context.Context, cfg aws.Config, stream *StreamSender) ([]Resource, error) {
	client := backup.NewFromConfig(cfg)
	paginator := backup.NewListBackupPlansPaginator(client, &backup.ListBackupPlansInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.BackupPlansList {
			resource := Resource{
				ARN:  *v.BackupPlanArn,
				Name: *v.BackupPlanName,
				Description: model.BackupPlanDescription{
					BackupPlan: v,
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

func BackupRecoveryPoint(ctx context.Context, cfg aws.Config, stream *StreamSender) ([]Resource, error) {
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

					resource := Resource{
						ARN:  *recoveryPoint.RecoveryPointArn,
						Name: nameFromArn(*out.RecoveryPointArn),
						Description: model.BackupRecoveryPointDescription{
							RecoveryPoint: out,
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
		}
	}

	return values, nil
}

func BackupProtectedResource(ctx context.Context, cfg aws.Config, stream *StreamSender) ([]Resource, error) {
	client := backup.NewFromConfig(cfg)
	paginator := backup.NewListProtectedResourcesPaginator(client, &backup.ListProtectedResourcesInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, resource := range page.Results {
			resource := Resource{
				ARN:  *resource.ResourceArn,
				Name: nameFromArn(*resource.ResourceArn),
				Description: model.BackupProtectedResourceDescription{
					ProtectedResource: resource,
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

func BackupSelection(ctx context.Context, cfg aws.Config, stream *StreamSender) ([]Resource, error) {
	describeCtx := GetDescribeContext(ctx)

	plans, err := BackupPlan(ctx, cfg, nil)
	if err != nil {
		return nil, err
	}

	client := backup.NewFromConfig(cfg)

	var values []Resource
	for _, plan := range plans {
		paginator := backup.NewListBackupSelectionsPaginator(client, &backup.ListBackupSelectionsInput{
			BackupPlanId: plan.Description.(model.BackupPlanDescription).BackupPlan.BackupPlanId,
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

				name := "arn:" + describeCtx.Partition + ":backup:" + describeCtx.Region + ":" + describeCtx.AccountID + ":backup-plan:" + *v.BackupPlanId + "/selection/" + *v.SelectionId
				resource := Resource{
					ARN:  name,
					Name: *v.SelectionName,
					Description: model.BackupSelectionDescription{
						BackupSelection: v,
						ListOfTags:      out.BackupSelection.ListOfTags,
						Resources:       out.BackupSelection.Resources,
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
	}

	return values, nil
}

func BackupVault(ctx context.Context, cfg aws.Config, stream *StreamSender) ([]Resource, error) {
	client := backup.NewFromConfig(cfg)
	paginator := backup.NewListBackupVaultsPaginator(client, &backup.ListBackupVaultsInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.BackupVaultList {
			notification := &backup.GetBackupVaultNotificationsOutput{}
			//TODO-Saleh getting rid of this to prevent error in describe job
			//notification, err := client.GetBackupVaultNotifications(ctx, &backup.GetBackupVaultNotificationsInput{
			//	BackupVaultName: v.BackupVaultName,
			//})
			//if err != nil {
			//	if isErr(err, "ResourceNotFoundException") || isErr(err, "InvalidParameter") {
			//		notification = &backup.GetBackupVaultNotificationsOutput{}
			//	} else {
			//		return nil, err
			//	}
			//}

			accessPolicy, err := client.GetBackupVaultAccessPolicy(ctx, &backup.GetBackupVaultAccessPolicyInput{
				BackupVaultName: v.BackupVaultName,
			})
			if err != nil {
				if isErr(err, "ResourceNotFoundException") || isErr(err, "InvalidParameter") {
					accessPolicy = &backup.GetBackupVaultAccessPolicyOutput{}
				} else {
					return nil, err
				}
			}

			resource := Resource{
				ARN:  *v.BackupVaultArn,
				Name: *v.BackupVaultName,
				Description: model.BackupVaultDescription{
					BackupVault:       v,
					Policy:            accessPolicy.Policy,
					BackupVaultEvents: notification.BackupVaultEvents,
					SNSTopicArn:       notification.SNSTopicArn,
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

func BackupFramework(ctx context.Context, cfg aws.Config, stream *StreamSender) ([]Resource, error) {
	client := backup.NewFromConfig(cfg)
	paginator := backup.NewListFrameworksPaginator(client, &backup.ListFrameworksInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.Frameworks {
			framework, err := client.DescribeFramework(ctx, &backup.DescribeFrameworkInput{
				FrameworkName: v.FrameworkName,
			})
			if err != nil {
				return nil, err
			}

			tags, err := client.ListTags(ctx, &backup.ListTagsInput{
				ResourceArn: v.FrameworkArn,
			})

			resource := Resource{
				ARN:  *v.FrameworkArn,
				Name: *v.FrameworkName,
				Description: model.BackupFrameworkDescription{
					Framework: *framework,
					Tags:      tags.Tags,
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

func BackupLegalHold(ctx context.Context, cfg aws.Config, stream *StreamSender) ([]Resource, error) {
	client := backup.NewFromConfig(cfg)
	paginator := backup.NewListLegalHoldsPaginator(client, &backup.ListLegalHoldsInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.LegalHolds {
			legalHold, err := client.GetLegalHold(ctx, &backup.GetLegalHoldInput{
				LegalHoldId: v.LegalHoldId,
			})
			if err != nil {
				return nil, err
			}

			resource := Resource{
				Name: *v.Title,
				ARN:  *v.LegalHoldArn,
				ID:   *v.LegalHoldId,
				Description: model.BackupLegalHoldDescription{
					LegalHold: *legalHold,
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
