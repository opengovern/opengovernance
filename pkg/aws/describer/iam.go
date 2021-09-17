package describer

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/iam/types"
)

func IAMAccessKey(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
	client := iam.NewFromConfig(cfg)
	paginator := iam.NewListAccessKeysPaginator(client, &iam.ListAccessKeysInput{})

	var values []interface{}
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.AccessKeyMetadata {
			values = append(values, v)
		}
	}

	return values, nil
}

func IAMGroup(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
	client := iam.NewFromConfig(cfg)
	paginator := iam.NewListGroupsPaginator(client, &iam.ListGroupsInput{})

	var values []interface{}
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.Groups {
			values = append(values, v)
		}
	}

	return values, nil
}

func IAMInstanceProfile(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
	client := iam.NewFromConfig(cfg)
	paginator := iam.NewListInstanceProfilesPaginator(client, &iam.ListInstanceProfilesInput{})

	var values []interface{}
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.InstanceProfiles {
			values = append(values, v)
		}
	}

	return values, nil
}

func IAMManagedPolicy(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
	client := iam.NewFromConfig(cfg)
	paginator := iam.NewListPoliciesPaginator(client, &iam.ListPoliciesInput{
		OnlyAttached: true,
	})

	var values []interface{}
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.Policies {
			values = append(values, v)
		}
	}

	return values, nil
}

func IAMOIDCProvider(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
	client := iam.NewFromConfig(cfg)
	output, err := client.ListOpenIDConnectProviders(ctx, &iam.ListOpenIDConnectProvidersInput{})
	if err != nil {
		return nil, err
	}

	var values []interface{}
	for _, v := range output.OpenIDConnectProviderList {
		values = append(values, v)
	}

	return values, nil
}

func IAMGroupPolicy(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
	groups, err := IAMGroup(ctx, cfg)
	if err != nil {
		return nil, err
	}

	client := iam.NewFromConfig(cfg)

	var values []interface{}
	for _, g := range groups {
		group := g.(types.Group)
		err := PaginateRetrieveAll(func(prevToken *string) (nextToken *string, err error) {
			output, err := client.ListGroupPolicies(ctx, &iam.ListGroupPoliciesInput{
				GroupName: group.GroupName,
				Marker:    prevToken,
			})
			if err != nil {
				return nil, err
			}

			for _, policy := range output.PolicyNames {
				pOutput, err := client.GetGroupPolicy(ctx, &iam.GetGroupPolicyInput{
					GroupName:  group.GroupName,
					PolicyName: aws.String(policy),
				})
				if err != nil {
					return nil, err
				}

				values = append(values, pOutput)
			}

			return output.Marker, nil
		})
		if err != nil {
			return nil, err
		}
	}

	return values, nil
}

func IAMUserPolicy(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
	users, err := IAMUser(ctx, cfg)
	if err != nil {
		return nil, err
	}

	client := iam.NewFromConfig(cfg)

	var values []interface{}
	for _, u := range users {
		user := u.(types.User)
		err := PaginateRetrieveAll(func(prevToken *string) (nextToken *string, err error) {
			output, err := client.ListUserPolicies(ctx, &iam.ListUserPoliciesInput{
				UserName: user.UserName,
				Marker:   prevToken,
			})
			if err != nil {
				return nil, err
			}

			for _, policy := range output.PolicyNames {
				pOutput, err := client.GetUserPolicy(ctx, &iam.GetUserPolicyInput{
					UserName:   user.UserName,
					PolicyName: aws.String(policy),
				})
				if err != nil {
					return nil, err
				}

				values = append(values, pOutput)
			}

			return output.Marker, nil
		})
		if err != nil {
			return nil, err
		}
	}

	return values, nil
}

func IAMRolePolicy(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
	roles, err := IAMRole(ctx, cfg)
	if err != nil {
		return nil, err
	}

	client := iam.NewFromConfig(cfg)

	var values []interface{}

	for _, r := range roles {
		role := r.(types.Role)
		err := PaginateRetrieveAll(func(prevToken *string) (nextToken *string, err error) {
			output, err := client.ListRolePolicies(ctx, &iam.ListRolePoliciesInput{
				RoleName: role.RoleName,
				Marker:   prevToken,
			})
			if err != nil {
				return nil, err
			}

			for _, policy := range output.PolicyNames {
				pOutput, err := client.GetRolePolicy(ctx, &iam.GetRolePolicyInput{
					RoleName:   role.RoleName,
					PolicyName: aws.String(policy),
				})
				if err != nil {
					return nil, err
				}

				values = append(values, pOutput)
			}

			return output.Marker, nil
		})
		if err != nil {
			return nil, err
		}
	}

	return values, nil
}

func IAMRole(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
	client := iam.NewFromConfig(cfg)
	paginator := iam.NewListRolesPaginator(client, &iam.ListRolesInput{})

	var values []interface{}
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.Roles {
			values = append(values, v)
		}
	}

	return values, nil
}

func IAMSAMLProvider(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
	client := iam.NewFromConfig(cfg)
	output, err := client.ListSAMLProviders(ctx, &iam.ListSAMLProvidersInput{})
	if err != nil {
		return nil, err
	}

	var values []interface{}
	for _, v := range output.SAMLProviderList {
		values = append(values, v)
	}

	return values, nil
}

func IAMServerCertificate(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
	client := iam.NewFromConfig(cfg)
	paginator := iam.NewListServerCertificatesPaginator(client, &iam.ListServerCertificatesInput{})

	var values []interface{}
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.ServerCertificateMetadataList {
			values = append(values, v)
		}
	}

	return values, nil
}

func IAMUser(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
	client := iam.NewFromConfig(cfg)
	paginator := iam.NewListUsersPaginator(client, &iam.ListUsersInput{})

	var values []interface{}
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.Users {
			values = append(values, v)
		}
	}

	return values, nil
}

func IAMVirtualMFADevice(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
	client := iam.NewFromConfig(cfg)
	output, err := client.ListMFADevices(ctx, &iam.ListMFADevicesInput{})
	if err != nil {
		return nil, err
	}

	var values []interface{}
	for _, v := range output.MFADevices {
		values = append(values, v)
	}

	return values, nil
}
