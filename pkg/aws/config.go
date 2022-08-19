package aws

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws/retry"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

// getConfig loads the AWS credentionals and returns the configuration to be used by the AWS services client.
// If the awsAccessKey is specified, the config will be created for the combination of awsAccessKey, awsSecretKey, awsSessionToken.
// Else it will use the default AWS SDK logic to load the configuration. See https://aws.github.io/aws-sdk-go-v2/docs/configuring-sdk/
// If assumeRoleArn is provided, it will use the evaluated configuration to then assume the specified role.
func GetConfig(ctx context.Context, awsAccessKey, awsSecretKey, awsSessionToken, assumeRoleArn string) (aws.Config, error) {
	retryer := config.WithRetryer(func() aws.Retryer {
		// Generally you will always want to return new instance of a Retryer. This will avoid a global rate limit
		// bucket being shared between across all service clients.
		return retry.AddWithMaxBackoffDelay(retry.NewStandard(), time.Second*5)
	})
	opts := []func(*config.LoadOptions) error{retryer}

	if awsAccessKey != "" {
		opts = append(opts, config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(awsAccessKey, awsSecretKey, awsSessionToken)))
	}

	cfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return aws.Config{}, fmt.Errorf("failed to load AWS config: %w", err)
	}

	if assumeRoleArn != "" {
		cfg, err = config.LoadDefaultConfig(context.Background(), retryer, config.WithCredentialsProvider(stscreds.NewAssumeRoleProvider(sts.NewFromConfig(cfg), assumeRoleArn)))
		if err != nil {
			return aws.Config{}, fmt.Errorf("failed to assume role: %w", err)
		}
	}

	return cfg, nil
}
