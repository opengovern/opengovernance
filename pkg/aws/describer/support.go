package describer

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/support"
	"github.com/aws/aws-sdk-go-v2/service/support/types"
)

// SupportServices returns the specified services in an aws region
func SupportServices(ctx context.Context, cfg aws.Config) ([]types.Service, error) {
	svc := support.NewFromConfig(cfg)

	req, err := svc.DescribeServices(ctx, &support.DescribeServicesInput{})
	if err != nil {
		return nil, err
	}

	return req.Services, nil
}
