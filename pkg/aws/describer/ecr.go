package describer

import (
	"context"
	"errors"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/aws/aws-sdk-go-v2/service/ecr/types"
	"github.com/aws/aws-sdk-go-v2/service/ecrpublic"
	public_types "github.com/aws/aws-sdk-go-v2/service/ecrpublic/types"
	"github.com/aws/smithy-go"
	"gitlab.com/keibiengine/keibi-engine/pkg/aws/model"
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
			if isErr(err, "RepositoryNotFoundException") || isErr(err, "RepositoryPolicyNotFoundException") || isErr(err, "LifecyclePolicyNotFoundException") {
				continue
			}
			return nil, err
		}

		for _, v := range page.Repositories {
			var imageDetails []public_types.ImageDetail
			imagePaginator := ecrpublic.NewDescribeImagesPaginator(client, &ecrpublic.DescribeImagesInput{
				RepositoryName: v.RepositoryName,
			})
			for imagePaginator.HasMorePages() {
				imagePage, err := imagePaginator.NextPage(ctx)
				if err != nil {
					if isErr(err, "RepositoryNotFoundException") || isErr(err, "RepositoryPolicyNotFoundException") || isErr(err, "LifecyclePolicyNotFoundException") {
						continue
					}
					return nil, err
				}
				imageDetails = append(imageDetails, imagePage.ImageDetails...)
			}

			policyOutput, err := client.GetRepositoryPolicy(ctx, &ecrpublic.GetRepositoryPolicyInput{
				RepositoryName: v.RepositoryName,
			})
			if err != nil {
				if !isErr(err, "RepositoryNotFoundException") && !isErr(err, "RepositoryPolicyNotFoundException") && !isErr(err, "LifecyclePolicyNotFoundException") {
					return nil, err
				}
			}

			tagsOutput, err := client.ListTagsForResource(ctx, &ecrpublic.ListTagsForResourceInput{
				ResourceArn: v.RepositoryArn,
			})
			if err != nil {
				if !isErr(err, "RepositoryNotFoundException") && !isErr(err, "RepositoryPolicyNotFoundException") && !isErr(err, "LifecyclePolicyNotFoundException") {
					return nil, err
				} else {
					tagsOutput = &ecrpublic.ListTagsForResourceOutput{}
				}
			}

			values = append(values, Resource{
				ARN:  *v.RepositoryArn,
				Name: *v.RepositoryName,
				Description: model.ECRPublicRepositoryDescription{
					PublicRepository: v,
					ImageDetails:     imageDetails,
					Policy:           policyOutput,
					Tags:             tagsOutput.Tags,
				},
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
			if isErr(err, "RepositoryNotFoundException") || isErr(err, "RepositoryPolicyNotFoundException") || isErr(err, "LifecyclePolicyNotFoundException") {
				continue
			}
			return nil, err
		}

		for _, v := range page.Repositories {
			lifeCyclePolicyOutput, err := client.GetLifecyclePolicy(ctx, &ecr.GetLifecyclePolicyInput{
				RepositoryName: v.RepositoryName,
			})
			if err != nil {
				if !isErr(err, "RepositoryNotFoundException") && !isErr(err, "RepositoryPolicyNotFoundException") && !isErr(err, "LifecyclePolicyNotFoundException") {
					return nil, err
				}
			}

			var imageDetails []types.ImageDetail
			imagePaginator := ecr.NewDescribeImagesPaginator(client, &ecr.DescribeImagesInput{
				RepositoryName: v.RepositoryName,
			})
			for imagePaginator.HasMorePages() {
				imagePage, err := imagePaginator.NextPage(ctx)
				if err != nil {
					if isErr(err, "RepositoryNotFoundException") || isErr(err, "RepositoryPolicyNotFoundException") || isErr(err, "LifecyclePolicyNotFoundException") {
						continue
					}
					return nil, err
				}
				imageDetails = append(imageDetails, imagePage.ImageDetails...)
			}

			policyOutput, err := client.GetRepositoryPolicy(ctx, &ecr.GetRepositoryPolicyInput{
				RepositoryName: v.RepositoryName,
			})
			if err != nil {
				if !isErr(err, "RepositoryNotFoundException") && !isErr(err, "RepositoryPolicyNotFoundException") && !isErr(err, "LifecyclePolicyNotFoundException") {
					return nil, err
				}
			}

			tagsOutput, err := client.ListTagsForResource(ctx, &ecr.ListTagsForResourceInput{
				ResourceArn: v.RepositoryArn,
			})
			if err != nil {
				if !isErr(err, "RepositoryNotFoundException") && !isErr(err, "RepositoryPolicyNotFoundException") && !isErr(err, "LifecyclePolicyNotFoundException") {
					return nil, err
				} else {
					tagsOutput = &ecr.ListTagsForResourceOutput{}
				}
			}

			values = append(values, Resource{
				ARN:  *v.RepositoryArn,
				Name: *v.RepositoryName,
				Description: model.ECRRepositoryDescription{
					Repository:      v,
					LifecyclePolicy: lifeCyclePolicyOutput,
					ImageDetails:    imageDetails,
					Policy:          policyOutput,
					Tags:            tagsOutput.Tags,
				},
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
		Name:        *output.RegistryId,
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
		Name:        *output.RegistryId,
		Description: output,
	}}, nil
}
