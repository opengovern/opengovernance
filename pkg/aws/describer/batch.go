package describer

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/batch"
	"gitlab.com/keibiengine/keibi-engine/pkg/aws/model"
)

func BatchComputeEnvironment(ctx context.Context, cfg aws.Config, stream *StreamSender) ([]Resource, error) {
	client := batch.NewFromConfig(cfg)
	paginator := batch.NewDescribeComputeEnvironmentsPaginator(client, &batch.DescribeComputeEnvironmentsInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.ComputeEnvironments {
			resource := Resource{
				ARN:  *v.ComputeEnvironmentArn,
				Name: *v.ComputeEnvironmentName,
				Description: model.BatchComputeEnvironmentDescription{
					ComputeEnvironment: v,
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

func BatchJob(ctx context.Context, cfg aws.Config, stream *StreamSender) ([]Resource, error) {
	client := batch.NewFromConfig(cfg)
	paginator := batch.NewDescribeJobQueuesPaginator(client, &batch.DescribeJobQueuesInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, jq := range page.JobQueues {
			jobsPaginator := batch.NewListJobsPaginator(client, &batch.ListJobsInput{
				JobQueue: jq.JobQueueName,
			})
			for jobsPaginator.HasMorePages() {
				page, err := jobsPaginator.NextPage(ctx)
				if err != nil {
					return nil, err
				}

				for _, v := range page.JobSummaryList {
					resource := Resource{
						ARN:  *v.JobArn,
						Name: *v.JobName,
						Description: model.BatchJobDescription{
							Job: v,
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
