package describer

import (
	"context"
	"errors"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/aws-sdk-go-v2/service/s3control"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/aws/smithy-go"
)

// S3Bucket describe S3 buckets.
// ListBuckets returns buckets in all regions. However, this function categorizes the buckets based
// on their location constaint, aka the regions they reside in.
func S3Bucket(ctx context.Context, cfg aws.Config, regions []string) (map[string][]interface{}, error) {
	regionalValues := make(map[string][]interface{}, len(regions))
	for _, r := range regions {
		regionalValues[r] = make([]interface{}, 0)
	}

	client := s3.NewFromConfig(cfg)
	output, err := client.ListBuckets(ctx, &s3.ListBucketsInput{})
	if err != nil {
		return nil, err
	}

	for _, bucket := range output.Buckets {
		output, err := client.GetBucketLocation(ctx, &s3.GetBucketLocationInput{
			Bucket: bucket.Name,
		})
		if err != nil {
			return nil, err
		}

		bRegion := string(output.LocationConstraint)
		if bRegion == "" {
			// Buckets in Region us-east-1 have a LocationConstraint of null.
			bRegion = "us-east-1"
		}

		if _, ok := regionalValues[bRegion]; ok {
			regionalValues[bRegion] = append(regionalValues[bRegion], bucket)
		}
	}

	return regionalValues, nil
}

// S3BucketPolicy describes bucket policies for each bucket. The BucketPolicy can only be queried from the
// reigon it resides in. That is despite the fact that ListBuckets returns all buckets regardless of the region.
func S3BucketPolicy(ctx context.Context, cfg aws.Config, regions []string) (map[string][]interface{}, error) {
	reigonalBuckets, err := S3Bucket(ctx, cfg, regions)
	if err != nil {
		return nil, err
	}

	regionalBucketPolicies := make(map[string][]interface{}, len(regions))
	for region, buckets := range reigonalBuckets {
		client := s3.NewFromConfig(cfg, func(o *s3.Options) { o.Region = region })

		for _, b := range buckets {
			bucket := b.(types.Bucket)
			output, err := client.GetBucketPolicy(ctx, &s3.GetBucketPolicyInput{
				Bucket: bucket.Name,
			})
			if err != nil {
				var ae smithy.APIError
				if errors.As(err, &ae) && (ae.ErrorCode() == "NoSuchBucketPolicy") {
					continue
				}

				return nil, err
			}

			regionalBucketPolicies[region] = append(regionalBucketPolicies[region], output.Policy)
		}
	}

	return regionalBucketPolicies, nil
}

func S3AccessPoint(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
	stsClient := sts.NewFromConfig(cfg)
	output, err := stsClient.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return nil, err
	}

	client := s3control.NewFromConfig(cfg)
	paginator := s3control.NewListAccessPointsPaginator(client, &s3control.ListAccessPointsInput{
		AccountId: output.Account,
	})

	var values []interface{}
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.AccessPointList {
			values = append(values, v)
		}
	}

	return values, nil
}

func S3StorageLens(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
	stsClient := sts.NewFromConfig(cfg)
	output, err := stsClient.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return nil, err
	}

	client := s3control.NewFromConfig(cfg)
	paginator := s3control.NewListStorageLensConfigurationsPaginator(client, &s3control.ListStorageLensConfigurationsInput{
		AccountId: output.Account,
	})

	var values []interface{}
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.StorageLensConfigurationList {
			values = append(values, v)
		}
	}

	return values, nil
}
