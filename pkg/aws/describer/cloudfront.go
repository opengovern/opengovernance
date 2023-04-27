package describer

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudfront"
	"gitlab.com/keibiengine/keibi-engine/pkg/aws/model"
)

func CloudFrontDistribution(ctx context.Context, cfg aws.Config, stream *StreamSender) ([]Resource, error) {
	client := cloudfront.NewFromConfig(cfg)
	paginator := cloudfront.NewListDistributionsPaginator(client, &cloudfront.ListDistributionsInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, item := range page.DistributionList.Items {
			tags, err := client.ListTagsForResource(ctx, &cloudfront.ListTagsForResourceInput{
				Resource: item.ARN,
			})
			if err != nil {
				return nil, err
			}

			distribution, err := client.GetDistribution(ctx, &cloudfront.GetDistributionInput{
				Id: item.Id,
			})
			if err != nil {
				return nil, err
			}

			resource := Resource{
				ARN:  *item.ARN,
				Name: *item.Id,
				Description: model.CloudFrontDistributionDescription{
					Distribution: distribution.Distribution,
					ETag:         distribution.ETag,
					Tags:         tags.Tags.Items,
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

func GetCloudFrontDistribution(ctx context.Context, cfg aws.Config, id string) ([]Resource, error) {
	client := cloudfront.NewFromConfig(cfg)

	out, err := client.GetDistribution(ctx, &cloudfront.GetDistributionInput{Id: &id})
	if err != nil {
		return nil, err
	}
	item := out.Distribution

	var values []Resource
	tags, err := client.ListTagsForResource(ctx, &cloudfront.ListTagsForResourceInput{
		Resource: item.ARN,
	})
	if err != nil {
		return nil, err
	}

	distribution, err := client.GetDistribution(ctx, &cloudfront.GetDistributionInput{
		Id: item.Id,
	})
	if err != nil {
		return nil, err
	}

	values = append(values, Resource{
		ARN:  *item.ARN,
		Name: *item.Id,
		Description: model.CloudFrontDistributionDescription{
			Distribution: distribution.Distribution,
			ETag:         distribution.ETag,
			Tags:         tags.Tags.Items,
		},
	})

	return values, nil
}

func CloudFrontOriginAccessControl(ctx context.Context, cfg aws.Config, stream *StreamSender) ([]Resource, error) {
	describeCtx := GetDescribeContext(ctx)
	client := cloudfront.NewFromConfig(cfg)

	var values []Resource
	err := PaginateRetrieveAll(func(prevToken *string) (nextToken *string, err error) {
		output, err := client.ListOriginAccessControls(ctx, &cloudfront.ListOriginAccessControlsInput{
			Marker:   prevToken,
			MaxItems: aws.Int32(100),
		})
		if err != nil {
			return nil, err
		}

		for _, v := range output.OriginAccessControlList.Items {
			arn := fmt.Sprintf("arn:%s:cloudfront::%s:origin-access-control/%s", describeCtx.Partition, describeCtx.AccountID, *v.Id) //TODO: this is fake ARN, find out the real one's format
			tags, err := client.ListTagsForResource(ctx, &cloudfront.ListTagsForResourceInput{
				Resource: &arn,
			})
			if err != nil {
				return nil, err
			}

			resource := Resource{
				ARN:  arn,
				Name: *v.Id,
				Description: model.CloudFrontOriginAccessControlDescription{
					OriginAccessControl: v,
					Tags:                tags.Tags.Items,
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
		return output.OriginAccessControlList.NextMarker, nil
	})
	if err != nil {
		return nil, err
	}

	return values, nil
}

func CloudFrontCachePolicy(ctx context.Context, cfg aws.Config, stream *StreamSender) ([]Resource, error) {
	describeCtx := GetDescribeContext(ctx)
	client := cloudfront.NewFromConfig(cfg)

	var values []Resource
	err := PaginateRetrieveAll(func(prevToken *string) (nextToken *string, err error) {
		output, err := client.ListCachePolicies(ctx, &cloudfront.ListCachePoliciesInput{
			Marker:   prevToken,
			MaxItems: aws.Int32(1000),
		})
		if err != nil {
			return nil, err
		}

		for _, v := range output.CachePolicyList.Items {
			arn := fmt.Sprintf("arn:%s:cloudfront::%s:cache-policy/%s", describeCtx.Partition, describeCtx.AccountID, *v.CachePolicy.Id)

			cachePolicy, err := client.GetCachePolicy(ctx, &cloudfront.GetCachePolicyInput{
				Id: v.CachePolicy.Id,
			})
			if err != nil {
				return nil, err
			}

			resource := Resource{
				ARN: arn,
				ID:  *v.CachePolicy.Id,
				Description: model.CloudFrontCachePolicyDescription{
					CachePolicy: *cachePolicy,
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
		return output.CachePolicyList.NextMarker, nil
	})
	if err != nil {
		return nil, err
	}

	return values, nil
}

func CloudFrontFunction(ctx context.Context, cfg aws.Config, stream *StreamSender) ([]Resource, error) {
	//describeCtx := GetDescribeContext(ctx)
	client := cloudfront.NewFromConfig(cfg)

	var values []Resource
	err := PaginateRetrieveAll(func(prevToken *string) (nextToken *string, err error) {
		output, err := client.ListFunctions(ctx, &cloudfront.ListFunctionsInput{
			Marker:   prevToken,
			MaxItems: aws.Int32(1000),
		})
		if err != nil {
			return nil, err
		}

		for _, v := range output.FunctionList.Items {
			function, err := client.DescribeFunction(ctx, &cloudfront.DescribeFunctionInput{
				Name:  v.Name,
				Stage: v.FunctionMetadata.Stage,
			})
			if err != nil {
				return nil, err
			}

			resource := Resource{
				ARN:  *function.FunctionSummary.FunctionMetadata.FunctionARN,
				Name: *function.FunctionSummary.Name,
				Description: model.CloudFrontFunctionDescription{
					Function: *function,
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
		return output.FunctionList.NextMarker, nil
	})
	if err != nil {
		return nil, err
	}

	return values, nil
}

func CloudFrontOriginAccessIdentity(ctx context.Context, cfg aws.Config, stream *StreamSender) ([]Resource, error) {
	describeCtx := GetDescribeContext(ctx)
	client := cloudfront.NewFromConfig(cfg)
	var values []Resource
	paginator := cloudfront.NewListCloudFrontOriginAccessIdentitiesPaginator(client, &cloudfront.ListCloudFrontOriginAccessIdentitiesInput{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, item := range page.CloudFrontOriginAccessIdentityList.Items {
			arn := fmt.Sprintf("arn:%s:cloudfront::%s:origin-access-identity/%s", describeCtx.Partition, describeCtx.AccountID, *item.Id)

			originAccessIdentity, err := client.GetCloudFrontOriginAccessIdentity(ctx, &cloudfront.GetCloudFrontOriginAccessIdentityInput{
				Id: item.Id,
			})
			if err != nil {
				return nil, err
			}

			resource := Resource{
				ARN:  arn,
				Name: *item.Id,
				Description: model.CloudFrontOriginAccessIdentityDescription{
					OriginAccessIdentity: *originAccessIdentity,
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

func GetCloudFrontOriginAccessIdentity(ctx context.Context, cfg aws.Config, id string) ([]Resource, error) {
	describeCtx := GetDescribeContext(ctx)
	client := cloudfront.NewFromConfig(cfg)
	var values []Resource

	out, err := client.GetCloudFrontOriginAccessIdentity(ctx, &cloudfront.GetCloudFrontOriginAccessIdentityInput{
		Id: &id,
	})
	if err != nil {
		return nil, err
	}

	item := out.CloudFrontOriginAccessIdentity
	arn := fmt.Sprintf("arn:%s:cloudfront::%s:origin-access-identity/%s", describeCtx.Partition, describeCtx.AccountID, *item.Id)

	originAccessIdentity, err := client.GetCloudFrontOriginAccessIdentity(ctx, &cloudfront.GetCloudFrontOriginAccessIdentityInput{
		Id: item.Id,
	})
	if err != nil {
		return nil, err
	}

	values = append(values, Resource{
		ARN:  arn,
		Name: *item.Id,
		Description: model.CloudFrontOriginAccessIdentityDescription{
			OriginAccessIdentity: *originAccessIdentity,
		},
	})

	return values, nil
}

func CloudFrontOriginRequestPolicy(ctx context.Context, cfg aws.Config, stream *StreamSender) ([]Resource, error) {
	describeCtx := GetDescribeContext(ctx)
	client := cloudfront.NewFromConfig(cfg)
	var values []Resource
	err := PaginateRetrieveAll(func(prevToken *string) (nextToken *string, err error) {
		output, err := client.ListOriginRequestPolicies(ctx, &cloudfront.ListOriginRequestPoliciesInput{
			Marker:   prevToken,
			MaxItems: aws.Int32(1000),
		})
		if err != nil {
			return nil, err
		}

		for _, v := range output.OriginRequestPolicyList.Items {
			arn := fmt.Sprintf("arn:%s:cloudfront::%s:origin-request-policy/%s", describeCtx.Partition, describeCtx.AccountID, *v.OriginRequestPolicy.Id)

			policy, err := client.GetOriginRequestPolicy(ctx, &cloudfront.GetOriginRequestPolicyInput{
				Id: v.OriginRequestPolicy.Id,
			})
			if err != nil {
				return nil, err
			}

			resource := Resource{
				ARN: arn,
				ID:  *policy.OriginRequestPolicy.Id,
				Description: model.CloudFrontOriginRequestPolicyDescription{
					OriginRequestPolicy: *policy,
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
		return output.OriginRequestPolicyList.NextMarker, nil
	})
	if err != nil {
		return nil, err
	}

	return values, nil
}

func CloudFrontResponseHeadersPolicy(ctx context.Context, cfg aws.Config, stream *StreamSender) ([]Resource, error) {
	describeCtx := GetDescribeContext(ctx)
	client := cloudfront.NewFromConfig(cfg)
	var values []Resource
	err := PaginateRetrieveAll(func(prevToken *string) (nextToken *string, err error) {
		output, err := client.ListResponseHeadersPolicies(ctx, &cloudfront.ListResponseHeadersPoliciesInput{
			Marker:   prevToken,
			MaxItems: aws.Int32(1000),
		})
		if err != nil {
			return nil, err
		}

		for _, v := range output.ResponseHeadersPolicyList.Items {
			arn := fmt.Sprintf("arn:%s:cloudfront::%s:response-headers-policy/%s", describeCtx.Partition, describeCtx.AccountID, *v.ResponseHeadersPolicy.Id)

			policy, err := client.GetResponseHeadersPolicy(ctx, &cloudfront.GetResponseHeadersPolicyInput{
				Id: v.ResponseHeadersPolicy.Id,
			})
			if err != nil {
				return nil, err
			}

			resource := Resource{
				ARN: arn,
				ID:  *policy.ResponseHeadersPolicy.Id,
				Description: model.CloudFrontResponseHeadersPolicyDescription{
					ResponseHeadersPolicy: *policy,
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
		return output.ResponseHeadersPolicyList.NextMarker, nil
	})
	if err != nil {
		return nil, err
	}

	return values, nil
}
