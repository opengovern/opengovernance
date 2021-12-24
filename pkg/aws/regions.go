package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go/aws/endpoints"
)

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

func partitionOf(region string) (string, bool) {
	resolver := endpoints.DefaultResolver()
	partitions := resolver.(endpoints.EnumPartitions).Partitions()

	for _, p := range partitions {
		for r := range p.Regions() {
			if r == region {
				return p.ID(), true
			}
		}
	}

	return "", false
}
