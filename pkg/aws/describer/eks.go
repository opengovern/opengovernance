package describer

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/eks"
)

func EKSCluster(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
	clusters, err := listEksClusters(ctx, cfg)
	if err != nil {
		return nil, err
	}

	client := eks.NewFromConfig(cfg)

	var values []interface{}
	for _, cluster := range clusters {
		output, err := client.DescribeCluster(ctx, &eks.DescribeClusterInput{Name: aws.String(cluster)})
		if err != nil {
			return nil, err
		}

		values = append(values, output.Cluster)
	}

	return values, nil
}

func EKSAddon(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
	clusters, err := listEksClusters(ctx, cfg)
	if err != nil {
		return nil, err
	}

	client := eks.NewFromConfig(cfg)

	var values []interface{}
	for _, cluster := range clusters {
		var addons []string

		paginator := eks.NewListAddonsPaginator(client, &eks.ListAddonsInput{ClusterName: aws.String(cluster)})
		for paginator.HasMorePages() {
			page, err := paginator.NextPage(ctx)
			if err != nil {
				return nil, err
			}

			addons = append(addons, page.Addons...)
		}

		for _, addon := range addons {
			output, err := client.DescribeAddon(ctx, &eks.DescribeAddonInput{
				AddonName:   aws.String(addon),
				ClusterName: aws.String(cluster),
			})
			if err != nil {
				return nil, err
			}

			values = append(values, output.Addon)
		}
	}

	return values, nil
}

func EKSFargateProfile(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
	clusters, err := listEksClusters(ctx, cfg)
	if err != nil {
		return nil, err
	}

	client := eks.NewFromConfig(cfg)

	var values []interface{}
	for _, cluster := range clusters {
		var profiles []string

		paginator := eks.NewListFargateProfilesPaginator(client, &eks.ListFargateProfilesInput{ClusterName: aws.String(cluster)})
		for paginator.HasMorePages() {
			page, err := paginator.NextPage(ctx)
			if err != nil {
				return nil, err
			}

			profiles = append(profiles, page.FargateProfileNames...)
		}

		for _, profile := range profiles {
			output, err := client.DescribeFargateProfile(ctx, &eks.DescribeFargateProfileInput{
				FargateProfileName: aws.String(profile),
				ClusterName:        aws.String(cluster),
			})
			if err != nil {
				return nil, err
			}

			values = append(values, output.FargateProfile)
		}
	}

	return values, nil
}

func EKSNodegroup(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
	clusters, err := listEksClusters(ctx, cfg)
	if err != nil {
		return nil, err
	}

	client := eks.NewFromConfig(cfg)

	var values []interface{}
	for _, cluster := range clusters {
		var groups []string
		paginator := eks.NewListNodegroupsPaginator(client, &eks.ListNodegroupsInput{ClusterName: aws.String(cluster)})

		for paginator.HasMorePages() {
			page, err := paginator.NextPage(ctx)
			if err != nil {
				return nil, err
			}

			groups = append(groups, page.Nodegroups...)
		}

		for _, profile := range groups {
			output, err := client.DescribeNodegroup(ctx, &eks.DescribeNodegroupInput{
				NodegroupName: aws.String(profile),
				ClusterName:   aws.String(cluster),
			})
			if err != nil {
				return nil, err
			}

			values = append(values, output.Nodegroup)
		}
	}

	return values, nil
}

func listEksClusters(ctx context.Context, cfg aws.Config) ([]string, error) {
	client := eks.NewFromConfig(cfg)
	paginator := eks.NewListClustersPaginator(client, &eks.ListClustersInput{})

	var values []string
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		values = append(values, page.Clusters...)
	}

	return values, nil
}
