package describer

import (
	"context"
	"errors"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/aws/aws-sdk-go-v2/service/ecr/types"
	"github.com/aws/aws-sdk-go-v2/service/ecrpublic"
	"github.com/aws/smithy-go"
)

func ECRPublicRepository(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	// Only supported in US-EAST-1
	if !strings.EqualFold(cfg.Region, "us-east-1") {
		return []Resource{}, nil
	}

	client := ecrpublic.NewFromConfig(cfg)
	paginator := ecrpublic.NewDescribeRepositoriesPaginator(client, &ecrpublic.DescribeRepositoriesInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.Repositories {
			values = append(values, Resource{
				ARN:         *v.RepositoryArn,
				Description: v,
			})
		}
	}

	return values, nil
}

func ECRRepository(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := ecr.NewFromConfig(cfg)
	paginator := ecr.NewDescribeRepositoriesPaginator(client, &ecr.DescribeRepositoriesInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.Repositories {
			values = append(values, Resource{
				ARN:         *v.RepositoryArn,
				Description: v,
			})
		}
	}

	return values, nil
}

func ECRRegistryPolicy(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := ecr.NewFromConfig(cfg)
	output, err := client.GetRegistryPolicy(ctx, &ecr.GetRegistryPolicyInput{})
	if err != nil {
		var ae smithy.APIError
		e := types.RegistryPolicyNotFoundException{}
		if errors.As(err, &ae) && ae.ErrorCode() == e.ErrorCode() {
			return []Resource{}, nil
		}
		return nil, err
	}

	return []Resource{{
		ID:          *output.RegistryId,
		Description: output,
	}}, nil
}

func ECRRegistry(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := ecr.NewFromConfig(cfg)
	output, err := client.DescribeRegistry(ctx, &ecr.DescribeRegistryInput{})
	if err != nil {
		return nil, err
	}

	return []Resource{{
		ID:          *output.RegistryId,
		Description: output,
	}}, nil
}
