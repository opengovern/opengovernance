package describer

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/organizations"
	"github.com/aws/aws-sdk-go-v2/service/organizations/types"
)

// OrganizationOrganization Retrieves information about the organization that the
// user's account belongs to.
func OrganizationOrganization(ctx context.Context, cfg aws.Config) (*types.Organization, error) {
	svc := organizations.NewFromConfig(cfg)

	req, err := svc.DescribeOrganization(ctx, &organizations.DescribeOrganizationInput{})
	if err != nil {
		return nil, err
	}

	return req.Organization, nil
}

// OrganizationAccount Retrieves AWS Organizations-related information about
// the specified (ID) account .
func OrganizationAccount(ctx context.Context, cfg aws.Config, id string) (*types.Account, error) {
	svc := organizations.NewFromConfig(cfg)

	req, err := svc.DescribeAccount(ctx, &organizations.DescribeAccountInput{AccountId: aws.String(id)})
	if err != nil {
		return nil, err
	}

	return req.Account, nil
}

// DescribeOrganization Retrieves information about the organization that the
// user's account belongs to.
func OrganizationAccounts(ctx context.Context, cfg aws.Config) ([]types.Account, error) {
	client := organizations.NewFromConfig(cfg)

	paginator := organizations.NewListAccountsPaginator(client, &organizations.ListAccountsInput{})

	var values []types.Account
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		values = append(values, page.Accounts...)
	}

	return values, nil
}
