package describer

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/drs"
	"gitlab.com/keibiengine/keibi-engine/pkg/aws/model"
)

func DRSSourceServer(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := drs.NewFromConfig(cfg)
	paginator := drs.NewDescribeSourceServersPaginator(client, &drs.DescribeSourceServersInput{
		MaxResults: 100,
	})

	var values []Resource
	pageNo := 0
	for paginator.HasMorePages() && pageNo < 5 {
		pageNo++
		page, err := paginator.NextPage(ctx)
		if err != nil {
			if !isErr(err, "UninitializedAccountException") {
				return nil, err
			}
			continue
		}

		for _, v := range page.Items {
			values = append(values, Resource{
				ARN:  *v.Arn,
				Name: *v.SourceServerID,
				Description: model.DRSSourceServerDescription{
					SourceServer: v,
				},
			})
		}
	}

	return values, nil
}

func DRSRecoveryInstance(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := drs.NewFromConfig(cfg)
	paginator := drs.NewDescribeRecoveryInstancesPaginator(client, &drs.DescribeRecoveryInstancesInput{
		MaxResults: 100,
	})

	var values []Resource
	pageNo := 0
	for paginator.HasMorePages() && pageNo < 5 {
		pageNo++
		page, err := paginator.NextPage(ctx)
		if err != nil {
			if !isErr(err, "UninitializedAccountException") {
				return nil, err
			}
			continue
		}

		for _, v := range page.Items {
			values = append(values, Resource{
				ARN:  *v.Arn,
				Name: *v.RecoveryInstanceID,
				Description: model.DRSRecoveryInstanceDescription{
					RecoveryInstance: v,
				},
			})
		}
	}

	return values, nil
}

func DRSJob(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := drs.NewFromConfig(cfg)
	paginator := drs.NewDescribeJobsPaginator(client, &drs.DescribeJobsInput{
		MaxResults: 100,
	})

	var values []Resource
	pageNo := 0
	for paginator.HasMorePages() && pageNo < 5 {
		pageNo++
		page, err := paginator.NextPage(ctx)
		if err != nil {
			if !isErr(err, "UninitializedAccountException") {
				return nil, err
			}
			continue
		}

		for _, v := range page.Items {
			values = append(values, Resource{
				ARN: *v.Arn,
				ID:  *v.JobID,
				Description: model.DRSJobDescription{
					Job: v,
				},
			})
		}
	}

	return values, nil
}

func DRSRecoverySnapshot(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := drs.NewFromConfig(cfg)
	paginator := drs.NewDescribeSourceServersPaginator(client, &drs.DescribeSourceServersInput{
		MaxResults: 100,
	})

	var values []Resource
	sourceServerPageNo := 0
	for paginator.HasMorePages() && sourceServerPageNo < 5 {
		sourceServerPageNo++
		page, err := paginator.NextPage(ctx)
		if err != nil {
			if !isErr(err, "UninitializedAccountException") {
				return nil, err
			}
			continue
		}

		for _, sourceServer := range page.Items {
			recoverySnapshotPaginator := drs.NewDescribeRecoverySnapshotsPaginator(client, &drs.DescribeRecoverySnapshotsInput{
				MaxResults:     100,
				SourceServerID: sourceServer.SourceServerID,
			})
			recoverySnapshotPageNo := 0
			for recoverySnapshotPaginator.HasMorePages() && recoverySnapshotPageNo < 5 {
				recoverySnapshotPageNo++

				recoverySnapshotPage, err := recoverySnapshotPaginator.NextPage(ctx)
				if err != nil {
					return nil, err
				}

				for _, recoverySnapshot := range recoverySnapshotPage.Items {
					values = append(values, Resource{
						ID: *recoverySnapshot.SnapshotID,
						Description: model.DRSRecoverySnapshotDescription{
							RecoverySnapshot: recoverySnapshot,
						},
					})
				}
			}
		}
	}

	return values, nil
}
