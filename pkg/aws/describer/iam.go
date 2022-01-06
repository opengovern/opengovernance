package describer

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/iam/types"
	"github.com/aws/aws-sdk-go-v2/service/organizations"
	orgtypes "github.com/aws/aws-sdk-go-v2/service/organizations/types"
)

const (
	organizationsNotInUseException = "AWSOrganizationsNotInUseException"
)

type IAMAccountDescription struct {
	Aliases      []string
	Organization *orgtypes.Organization
}

func IAMAccount(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := organizations.NewFromConfig(cfg)

	output, err := client.DescribeOrganization(ctx, &organizations.DescribeOrganizationInput{})
	if err != nil {
		if isErr(err, organizationsNotInUseException) {
			output = &organizations.DescribeOrganizationOutput{}
		} else {
			return nil, err
		}
	}

	iamClient := iam.NewFromConfig(cfg)
	paginator := iam.NewListAccountAliasesPaginator(iamClient, &iam.ListAccountAliasesInput{})

	var aliases []string
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		aliases = append(aliases, page.AccountAliases...)
	}

	return []Resource{
		{
			// No ID or ARN. Per Account Configuration
			Description: IAMAccountDescription{
				Aliases:      aliases,
				Organization: output.Organization,
			},
		},
	}, nil
}

func IAMAccessKey(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := iam.NewFromConfig(cfg)
	paginator := iam.NewListAccessKeysPaginator(client, &iam.ListAccessKeysInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.AccessKeyMetadata {
			values = append(values, Resource{
				ID:          *v.AccessKeyId,
				Description: v,
			})
		}
	}

	return values, nil
}

func IAMGroup(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := iam.NewFromConfig(cfg)
	paginator := iam.NewListGroupsPaginator(client, &iam.ListGroupsInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.Groups {
			values = append(values, Resource{
				ARN:         *v.Arn,
				Description: v,
			})
		}
	}

	return values, nil
}

func IAMInstanceProfile(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := iam.NewFromConfig(cfg)
	paginator := iam.NewListInstanceProfilesPaginator(client, &iam.ListInstanceProfilesInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.InstanceProfiles {
			values = append(values, Resource{
				ARN:         *v.Arn,
				Description: v,
			})
		}
	}

	return values, nil
}

func IAMManagedPolicy(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := iam.NewFromConfig(cfg)
	paginator := iam.NewListPoliciesPaginator(client, &iam.ListPoliciesInput{
		OnlyAttached: true,
	})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.Policies {
			values = append(values, Resource{
				ARN:         *v.Arn,
				Description: v,
			})
		}
	}

	return values, nil
}

func IAMOIDCProvider(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := iam.NewFromConfig(cfg)
	output, err := client.ListOpenIDConnectProviders(ctx, &iam.ListOpenIDConnectProvidersInput{})
	if err != nil {
		return nil, err
	}

	var values []Resource
	for _, v := range output.OpenIDConnectProviderList {
		values = append(values, Resource{
			ARN:         *v.Arn,
			Description: v,
		})
	}

	return values, nil
}

func IAMGroupPolicy(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	groups, err := IAMGroup(ctx, cfg)
	if err != nil {
		return nil, err
	}

	client := iam.NewFromConfig(cfg)

	var values []Resource
	for _, g := range groups {
		group := g.Description.(types.Group)
		err := PaginateRetrieveAll(func(prevToken *string) (nextToken *string, err error) {
			output, err := client.ListGroupPolicies(ctx, &iam.ListGroupPoliciesInput{
				GroupName: group.GroupName,
				Marker:    prevToken,
			})
			if err != nil {
				return nil, err
			}

			for _, policy := range output.PolicyNames {
				v, err := client.GetGroupPolicy(ctx, &iam.GetGroupPolicyInput{
					GroupName:  group.GroupName,
					PolicyName: aws.String(policy),
				})
				if err != nil {
					return nil, err
				}

				values = append(values, Resource{
					ID:          CompositeID(*v.GroupName, *v.PolicyName),
					Description: v,
				})
			}

			return output.Marker, nil
		})
		if err != nil {
			return nil, err
		}
	}

	return values, nil
}

func IAMUserPolicy(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	users, err := IAMUser(ctx, cfg)
	if err != nil {
		return nil, err
	}

	client := iam.NewFromConfig(cfg)

	var values []Resource
	for _, u := range users {
		user := u.Description.(types.User)
		err := PaginateRetrieveAll(func(prevToken *string) (nextToken *string, err error) {
			output, err := client.ListUserPolicies(ctx, &iam.ListUserPoliciesInput{
				UserName: user.UserName,
				Marker:   prevToken,
			})
			if err != nil {
				return nil, err
			}

			for _, policy := range output.PolicyNames {
				v, err := client.GetUserPolicy(ctx, &iam.GetUserPolicyInput{
					UserName:   user.UserName,
					PolicyName: aws.String(policy),
				})
				if err != nil {
					return nil, err
				}

				values = append(values, Resource{
					ID:          CompositeID(*v.UserName, *v.PolicyName),
					Description: v,
				})
			}

			return output.Marker, nil
		})
		if err != nil {
			return nil, err
		}
	}

	return values, nil
}

func IAMRolePolicy(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	roles, err := IAMRole(ctx, cfg)
	if err != nil {
		return nil, err
	}

	client := iam.NewFromConfig(cfg)

	var values []Resource

	for _, r := range roles {
		role := r.Description.(types.Role)
		err := PaginateRetrieveAll(func(prevToken *string) (nextToken *string, err error) {
			output, err := client.ListRolePolicies(ctx, &iam.ListRolePoliciesInput{
				RoleName: role.RoleName,
				Marker:   prevToken,
			})
			if err != nil {
				return nil, err
			}

			for _, policy := range output.PolicyNames {
				v, err := client.GetRolePolicy(ctx, &iam.GetRolePolicyInput{
					RoleName:   role.RoleName,
					PolicyName: aws.String(policy),
				})
				if err != nil {
					return nil, err
				}

				values = append(values, Resource{
					ID:          CompositeID(*v.RoleName, *v.PolicyName),
					Description: v,
				})
			}

			return output.Marker, nil
		})
		if err != nil {
			return nil, err
		}
	}

	return values, nil
}

func IAMRole(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := iam.NewFromConfig(cfg)
	paginator := iam.NewListRolesPaginator(client, &iam.ListRolesInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.Roles {
			values = append(values, Resource{
				ARN:         *v.Arn,
				Description: v,
			})
		}
	}

	return values, nil
}

func IAMSAMLProvider(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := iam.NewFromConfig(cfg)
	output, err := client.ListSAMLProviders(ctx, &iam.ListSAMLProvidersInput{})
	if err != nil {
		return nil, err
	}

	var values []Resource
	for _, v := range output.SAMLProviderList {
		values = append(values, Resource{
			ARN:         *v.Arn,
			Description: v,
		})
	}

	return values, nil
}

func IAMServerCertificate(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := iam.NewFromConfig(cfg)
	paginator := iam.NewListServerCertificatesPaginator(client, &iam.ListServerCertificatesInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.ServerCertificateMetadataList {
			values = append(values, Resource{
				ARN:         *v.Arn,
				Description: v,
			})
		}
	}

	return values, nil
}

func IAMUser(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := iam.NewFromConfig(cfg)
	paginator := iam.NewListUsersPaginator(client, &iam.ListUsersInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.Users {
			values = append(values, Resource{
				ARN:         *v.Arn,
				Description: v,
			})
		}
	}

	return values, nil
}

func IAMVirtualMFADevice(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := iam.NewFromConfig(cfg)
	output, err := client.ListMFADevices(ctx, &iam.ListMFADevicesInput{})
	if err != nil {
		return nil, err
	}

	var values []Resource
	for _, v := range output.MFADevices {
		values = append(values, Resource{
			ID:          *v.SerialNumber,
			Description: v,
		})
	}

	return values, nil
}
