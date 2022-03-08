package describer

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/eks"
	"github.com/aws/aws-sdk-go-v2/service/eks/types"
	"gitlab.com/keibiengine/keibi-engine/pkg/aws/model"
)

func EKSCluster(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	clusters, err := listEksClusters(ctx, cfg)
	if err != nil {
		return nil, err
	}

	client := eks.NewFromConfig(cfg)

	var values []Resource
	for _, cluster := range clusters {
		output, err := client.DescribeCluster(ctx, &eks.DescribeClusterInput{Name: aws.String(cluster)})
		if err != nil {
			return nil, err
		}

		values = append(values, Resource{
			ARN:  *output.Cluster.Arn,
			Name: *output.Cluster.Name,
			Description: model.EKSClusterDescription{
				Cluster: *output.Cluster,
			},
		})
	}

	return values, nil
}

func EKSAddon(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	clusters, err := listEksClusters(ctx, cfg)
	if err != nil {
		return nil, err
	}

	client := eks.NewFromConfig(cfg)

	var values []Resource
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

			values = append(values, Resource{
				ARN:  *output.Addon.AddonArn,
				Name: *output.Addon.AddonName,
				Description: model.EKSAddonDescription{
					Addon: *output.Addon,
				},
			})
		}
	}

	return values, nil
}

func EKSFargateProfile(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	clusters, err := listEksClusters(ctx, cfg)
	if err != nil {
		return nil, err
	}

	client := eks.NewFromConfig(cfg)

	var values []Resource
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

			values = append(values, Resource{
				ARN:         *output.FargateProfile.FargateProfileArn,
				Name:        *output.FargateProfile.FargateProfileName,
				Description: output.FargateProfile,
			})
		}
	}

	return values, nil
}

func EKSNodegroup(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	clusters, err := listEksClusters(ctx, cfg)
	if err != nil {
		return nil, err
	}

	client := eks.NewFromConfig(cfg)

	var values []Resource
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

			values = append(values, Resource{
				ARN:         *output.Nodegroup.NodegroupArn,
				Name:        *output.Nodegroup.NodegroupName,
				Description: output.Nodegroup,
			})
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

type EKSIdentityProviderConfigDescription struct {
	ConfigName             string
	ConfigType             string
	IdentityProviderConfig types.OidcIdentityProviderConfig
}

func EKSIdentityProviderConfig(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	clusters, err := listEksClusters(ctx, cfg)
	if err != nil {
		return nil, err
	}

	var values []Resource
	for _, cluster := range clusters {
		client := eks.NewFromConfig(cfg)
		paginator := eks.NewListIdentityProviderConfigsPaginator(client, &eks.ListIdentityProviderConfigsInput{
			ClusterName: &cluster,
		})

		for paginator.HasMorePages() {
			page, err := paginator.NextPage(ctx)
			if err != nil {
				return nil, err
			}

			for _, config := range page.IdentityProviderConfigs {
				output, err := client.DescribeIdentityProviderConfig(ctx, &eks.DescribeIdentityProviderConfigInput{
					ClusterName:            &cluster,
					IdentityProviderConfig: &config,
				})
				if err != nil {
					return nil, err
				}

				values = append(values, Resource{
					ARN:  *output.IdentityProviderConfig.Oidc.IdentityProviderConfigArn,
					Name: *config.Name,
					Description: EKSIdentityProviderConfigDescription{
						ConfigName:             *config.Name,
						ConfigType:             *config.Type,
						IdentityProviderConfig: *output.IdentityProviderConfig.Oidc,
					},
				})

			}
		}
	}

	return values, nil
}
