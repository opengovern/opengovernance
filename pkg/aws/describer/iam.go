package describer

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
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

// TODO: Adds or updates an inline policy document that is embedded in the specified IAM user, group, or role.
func IAMPolicy(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
	client := iam.NewFromConfig(cfg)
	paginator := iam.NewListPoliciesPaginator(client, &iam.ListPoliciesInput{})

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

// OMIT: Part of IAMRole
// func IAMServiceLinkedRole(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
// }

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

// OMIT: Adds the specified user to the specified group.
// Doesn't really make sense to list it
// func IAMUserToGroupAddition(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
// }

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
