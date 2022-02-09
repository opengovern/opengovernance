package describer

import (
	"context"
	"errors"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/organizations"
	"github.com/aws/aws-sdk-go-v2/service/organizations/types"
)

// IsAccountAMember Checks whether an account is a member of an organization or not.
func IsAccountAMember(cfg aws.Config, id string) bool {
	_, err := DescribeAccountById(cfg, id)
	return err == nil
}

// DescribeAccountById Retrieves AWS Organizations-related information about
// the specified (ID) account .
func DescribeAccountById(cfg aws.Config, id string) (*types.Account, error) {
	svc := organizations.NewFromConfig(cfg)

	req, err := svc.DescribeAccount(context.Background(), &organizations.DescribeAccountInput{AccountId: aws.String(id)})
	var notFoundErr *types.AWSOrganizationsNotInUseException
	if errors.As(err, &notFoundErr) {
		return nil, err
	}

	return req.Account, nil
}
