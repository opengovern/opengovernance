package describer

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/batch"
	"gitlab.com/keibiengine/keibi-engine/pkg/aws/model"
)

func BatchComputeEnvironment(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := batch.NewFromConfig(cfg)
	paginator := batch.NewDescribeComputeEnvironmentsPaginator(client, &batch.DescribeComputeEnvironmentsInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.ComputeEnvironments {
			values = append(values, Resource{
				ARN:  *v.ComputeEnvironmentArn,
				Name: *v.ComputeEnvironmentName,
				Description: model.BatchComputeEnvironmentDescription{
					ComputeEnvironment: v,
				},
			})
		}
	}

	return values, nil
}

func BatchJob(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := batch.NewFromConfig(cfg)
	paginator := batch.NewListJobsPaginator(client, &batch.ListJobsInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.JobSummaryList {
			values = append(values, Resource{
				ARN:  *v.JobArn,
				Name: *v.JobName,
				Description: model.BatchJobDescription{
					Job: v,
				},
			})
		}
	}

	return values, nil
}
