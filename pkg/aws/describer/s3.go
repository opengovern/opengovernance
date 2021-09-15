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

// TODO: Handle global resources

func S3Bucket(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
	client := s3.NewFromConfig(cfg)
	output, err := client.ListBuckets(ctx, &s3.ListBucketsInput{})
	if err != nil {
		return nil, err
	}

	var values []interface{}
	for _, v := range output.Buckets {
		output, err := client.GetBucketLocation(ctx, &s3.GetBucketLocationInput{
			Bucket: v.Name,
		})
		if err != nil {
			return nil, err
		}

		if cfg.Region != string(output.LocationConstraint) &&
			!(output.LocationConstraint == "" && cfg.Region == "us-east-1") {
			continue
		}

		values = append(values, v)
	}

	return values, nil
}

func S3BucketPolicy(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
	buckets, err := S3Bucket(ctx, cfg)
	if err != nil {
		return nil, err
	}

	client := s3.NewFromConfig(cfg)

	var values []interface{}
	for _, b := range buckets {
		bucket := b.(types.Bucket)

		output1, err := client.GetBucketPolicy(ctx, &s3.GetBucketPolicyInput{
			Bucket: bucket.Name,
		})
		if err != nil {
			var ae smithy.APIError
			if errors.As(err, &ae) && ae.ErrorCode() == "NoSuchBucketPolicy" {
				continue
			}
			return nil, err
		}

		values = append(values, output1.Policy)
	}

	return values, nil
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
