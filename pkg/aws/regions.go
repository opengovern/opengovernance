package aws

import (
	"context"
	"encoding/json"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

func getAllRegionsInJSON(ctx context.Context, cfg aws.Config, includeDisabledRegions bool) (string, error) {
	regions, err := getAllRegions(ctx, cfg, includeDisabledRegions)
	if err != nil {
		return "", nil
	}
	j, err := json.MarshalIndent(regions, "", "  ")
	if err != nil {
		return "", err
	}

	return string(j), err
}

func getAllRegions(ctx context.Context, cfg aws.Config, includeDisabledRegions bool) ([]types.Region, error) {
	client := ec2.NewFromConfig(cfg)
	output, err := client.DescribeRegions(ctx, &ec2.DescribeRegionsInput{
		AllRegions: &includeDisabledRegions,
	})
	if err != nil {
		return nil, err
	}

	return output.Regions, nil
}
