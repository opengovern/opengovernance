package describer

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/organizations"
	"github.com/aws/aws-sdk-go-v2/service/organizations/types"
)

// DescribeAccountByID Retrieves AWS Organizations-related information about
// the specified (ID) account .
func DescribeAccountByID(ctx context.Context, cfg aws.Config, id string) (*types.Account, error) {
	svc := organizations.NewFromConfig(cfg)

	req, err := svc.DescribeAccount(ctx, &organizations.DescribeAccountInput{AccountId: aws.String(id)})
	if err != nil {
		return nil, err
	}

	return req.Account, nil
}

// DescribeOrganization Retrieves information about the organization that the
// user's account belongs to.
func DescribeOrganization(ctx context.Context, cfg aws.Config) (*types.Organization, error) {
	svc := organizations.NewFromConfig(cfg)

	req, err := svc.DescribeOrganization(ctx, &organizations.DescribeOrganizationInput{})
	if err != nil {
		return nil, err
	}

	return req.Organization, nil
}
