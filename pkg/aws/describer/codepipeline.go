package describer

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/codepipeline"
	"gitlab.com/keibiengine/keibi-engine/pkg/aws/model"
)

func CodePipelinePipeline(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := codepipeline.NewFromConfig(cfg)
	paginator := codepipeline.NewListPipelinesPaginator(client, &codepipeline.ListPipelinesInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			if !isErr(err, "PipelineNotFoundException") {
				return nil, err
			}
			continue
		}

		for _, v := range page.Pipelines {
			pipeline, err := client.GetPipeline(ctx, &codepipeline.GetPipelineInput{
				Name: v.Name,
			})
			if err != nil {
				if !isErr(err, "PipelineNotFoundException") {
					return nil, err
				}
				continue
			}

			tags, err := client.ListTagsForResource(ctx, &codepipeline.ListTagsForResourceInput{
				ResourceArn: pipeline.Metadata.PipelineArn,
			})
			if err != nil {
				if !isErr(err, "InvalidParameter") {
					return nil, err
				}
				tags = &codepipeline.ListTagsForResourceOutput{}
			}

			values = append(values, Resource{
				ARN:  *pipeline.Metadata.PipelineArn,
				Name: *pipeline.Pipeline.Name,
				Description: model.CodePipelinePipelineDescription{
					Pipeline: *pipeline.Pipeline,
					Metadata: *pipeline.Metadata,
					Tags:     tags.Tags,
				},
			})
		}
	}

	return values, nil
}
