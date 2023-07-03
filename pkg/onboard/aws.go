package onboard

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/organizations/types"
	"github.com/aws/smithy-go"
	keibiaws "github.com/kaytu-io/kaytu-aws-describer/aws"
	"github.com/kaytu-io/kaytu-aws-describer/aws/describer"
	"github.com/kaytu-io/kaytu-engine/pkg/describe"
	"github.com/kaytu-io/kaytu-util/pkg/source"
)

var PermissionError = errors.New("PermissionError")

type awsAccount struct {
	AccountID    string
	AccountName  *string
	Organization *types.Organization
	Account      *types.Account
}

func currentAwsAccount(ctx context.Context, cfg aws.Config) (*awsAccount, error) {
	accID, err := describer.STSAccount(ctx, cfg)
	if err != nil {
		return nil, err
	}

	iamClient := iam.NewFromConfig(cfg)
	user, err := iamClient.GetUser(ctx, &iam.GetUserInput{})
	if err != nil {
		fmt.Printf("failed to get user: %v", err)
		return nil, err
	}

	orgs, err := describer.OrganizationOrganization(ctx, cfg)
	if err != nil {
		if !ignoreAwsOrgError(err) {
			return nil, err
		}
	}

	acc, err := describer.OrganizationAccount(ctx, cfg, accID)
	if err != nil {
		if !ignoreAwsOrgError(err) {
			return nil, err
		}
	}

	return &awsAccount{
		AccountID:    accID,
		AccountName:  user.User.UserName,
		Organization: orgs,
		Account:      acc,
	}, nil
}

func getAWSCredentialsMetadata(ctx context.Context, config describe.AWSAccountConfig) (*source.AWSCredentialMetadata, error) {
	creds, err := keibiaws.GetConfig(ctx, config.AccessKey, config.SecretKey, "", "")
	if err != nil {
		return nil, err
	}
	if creds.Region == "" {
		creds.Region = "us-east-1"
	}

	accID, err := describer.STSAccount(ctx, creds)
	if err != nil {
		return nil, err
	}

	iamClient := iam.NewFromConfig(creds)
	user, err := iamClient.GetUser(ctx, &iam.GetUserInput{})
	if err != nil {
		fmt.Printf("failed to get user: %v", err)
		return nil, err
	}
	paginator := iam.NewListAttachedUserPoliciesPaginator(iamClient, &iam.ListAttachedUserPoliciesInput{
		UserName: user.User.UserName,
	})

	policyARNs := make([]string, 0)
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			fmt.Printf("failed to get policy page: %v", err)
			return nil, err
		}
		for _, policy := range page.AttachedPolicies {
			policyARNs = append(policyARNs, *policy.PolicyArn)
		}
	}

	metadata := source.AWSCredentialMetadata{
		AccountID:        accID,
		IamUserName:      user.User.UserName,
		AttachedPolicies: policyARNs,
	}

	accessKeys, err := iamClient.ListAccessKeys(ctx, &iam.ListAccessKeysInput{
		UserName: user.User.UserName,
	})
	for _, key := range accessKeys.AccessKeyMetadata {
		if *key.AccessKeyId == config.AccessKey && key.CreateDate != nil {
			metadata.IamApiKeyCreationDate = *key.CreateDate
		}
	}
	if err != nil {
		fmt.Printf("failed to get access keys: %v", err)
		return nil, err
	}

	organization, err := describer.OrganizationOrganization(ctx, creds)
	if err != nil {
		if !ignoreAwsOrgError(err) {
			return nil, err
		}
	}

	if organization != nil {
		metadata.OrganizationID = organization.Id
		metadata.OrganizationMasterAccountEmail = organization.MasterAccountEmail
		metadata.OrganizationMasterAccountId = organization.MasterAccountId
	}

	return &metadata, nil

}

func ignoreAwsOrgError(err error) bool {
	var ae smithy.APIError
	return errors.As(err, &ae) &&
		(ae.ErrorCode() == (&types.AWSOrganizationsNotInUseException{}).ErrorCode() ||
			ae.ErrorCode() == (&types.AccessDeniedException{}).ErrorCode())
}

func discoverAWSAccounts(ctx context.Context, cfg aws.Config) ([]awsAccount, error) {
	orgs, err := describer.OrganizationOrganization(ctx, cfg)
	if err != nil {
		if !ignoreAwsOrgError(err) {
			return nil, err
		}
	}

	accounts, err := describer.OrganizationAccounts(ctx, cfg)
	if err != nil {
		if !ignoreAwsOrgError(err) {
			return nil, err
		}
	}

	awsAccounts := make([]awsAccount, 0)
	for _, account := range accounts {
		if account.Id == nil {
			continue
		}
		localAccount := account
		awsAccounts = append(awsAccounts, awsAccount{
			AccountID:    *localAccount.Id,
			AccountName:  localAccount.Name,
			Organization: orgs,
			Account:      &localAccount,
		})
	}

	return awsAccounts, nil
}
