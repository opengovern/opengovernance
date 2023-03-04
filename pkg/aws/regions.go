package aws

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go/aws/endpoints"
)

const (
	SecurityAuditPolicyARN = "arn:aws:iam::aws:policy/SecurityAudit"
)

func CheckSecurityAuditPermission(accessKey, secretKey string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cfg, err := GetConfig(ctx, accessKey, secretKey, "", "")
	if err != nil {
		fmt.Printf("failed to get config: %v", err)
		return err
	}

	cfgClone := cfg.Copy()
	if cfgClone.Region == "" {
		cfgClone.Region = "us-east-1"
	}

	iamClient := iam.NewFromConfig(cfgClone)
	user, err := iamClient.GetUser(ctx, &iam.GetUserInput{})
	if err != nil {
		fmt.Printf("failed to get user: %v", err)
		return err
	}
	paginator := iam.NewListAttachedUserPoliciesPaginator(iamClient, &iam.ListAttachedUserPoliciesInput{
		UserName: user.User.UserName,
	})

	policyARNs := make([]string, 0)
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			fmt.Printf("failed to get policy page: %v", err)
			return err
		}
		for _, policy := range page.AttachedPolicies {
			policyARNs = append(policyARNs, *policy.PolicyArn)
		}
	}

	for _, policyARN := range policyARNs {
		if policyARN == SecurityAuditPolicyARN {
			return nil
		}
	}

	return fmt.Errorf("SecurityAudit policy is not attached to the user")
}

func CheckGetUserPermission(accessKey, secretKey string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cfg, err := GetConfig(ctx, accessKey, secretKey, "", "")
	if err != nil {
		fmt.Printf("failed to get config: %v", err)
		return err
	}

	cfgClone := cfg.Copy()
	if cfgClone.Region == "" {
		cfgClone.Region = "us-east-1"
	}

	iamClient := iam.NewFromConfig(cfgClone)
	_, err = iamClient.GetUser(ctx, &iam.GetUserInput{})
	if err != nil {
		fmt.Printf("failed to get user: %v", err)
		return err
	}

	return nil
}

func CheckDescribeRegionsPermission(accessKey, secretKey string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
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
