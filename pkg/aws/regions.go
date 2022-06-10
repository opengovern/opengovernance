package aws

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go/aws/endpoints"
)

func CheckDescribeRegionsPermission(accessKey, secretKey string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cfg, err := GetConfig(ctx, accessKey, secretKey, "", "")
	if err != nil {
		return err
	}

	cfgClone := cfg.Copy()
	cfgClone.Region = "us-east-1"

	_, err = getAllRegions(ctx, cfgClone, false)
	if err != nil {
		return err
	}
	return nil
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
