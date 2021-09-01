package aws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
)

// getConfig loads the AWS credentionals and returns the configuration to be used by the AWS services client.
// If the region is specifed, the config will be created for that region.
// If the awsAccessKey is specified, the config will be created for the combination of awsAccessKey, awsSecretKey, awsSessionToken.
// Else it will use the default AWS SDK logic to load the configuration. See https://aws.github.io/aws-sdk-go-v2/docs/configuring-sdk/
func getConfig(ctx context.Context, region, awsAccessKey, awsSecretKey, awsSessionToken string) (aws.Config, error) {
	opts := []func(*config.LoadOptions) error{}

	if region != "" {
		opts = append(opts, config.WithRegion(region))
	}

	if awsAccessKey != "" {
		opts = append(opts, config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(awsAccessKey, awsSecretKey, awsSessionToken)))
	}

	cfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return aws.Config{}, fmt.Errorf("failed to load AWS config: %w", err)
	}

	return cfg, nil
}
