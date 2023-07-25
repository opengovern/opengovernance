package onboard

import (
	"context"
	"errors"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/organizations/types"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/aws/smithy-go"
	keibiaws "github.com/kaytu-io/kaytu-aws-describer/aws"
	"github.com/kaytu-io/kaytu-aws-describer/aws/describer"
	"github.com/kaytu-io/kaytu-engine/pkg/describe"
	"github.com/kaytu-io/kaytu-engine/pkg/utils"
	"go.uber.org/zap"
)

var PermissionError = errors.New("PermissionError")

type awsAccount struct {
	AccountID    string
	AccountName  *string
	Organization *types.Organization
	Account      *types.Account
}

func currentAwsAccount(ctx context.Context, logger *zap.Logger, cfg aws.Config) (*awsAccount, error) {
	stsClient := sts.NewFromConfig(cfg)
	account, err := stsClient.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return nil, err
	}

	orgs, err := describer.OrganizationOrganization(ctx, cfg)
	if err != nil {
		if !ignoreAwsOrgError(err) {
			logger.Warn("failed to get organization", zap.Error(err))
			return nil, err
		}
	}

	acc, err := describer.OrganizationAccount(ctx, cfg, *account.Account)
	if err != nil {
		if !ignoreAwsOrgError(err) {
			logger.Warn("failed to get account", zap.Error(err))
			return nil, err
		}
	}
	accountName := account.UserId
	if acc != nil {
		accountName = acc.Name
	}

	return &awsAccount{
		AccountID:    *account.Account,
		AccountName:  accountName,
		Organization: orgs,
		Account:      acc,
	}, nil
}

func getAWSCredentialsMetadata(ctx context.Context, logger *zap.Logger, config describe.AWSAccountConfig) (*AWSCredentialMetadata, error) {
	creds, err := keibiaws.GetConfig(ctx, config.AccessKey, config.SecretKey, "", "", nil)
	if err != nil {
		return nil, err
	}
	if creds.Region == "" {
		creds.Region = "us-east-1"
	}

	accID, err := describer.STSAccount(ctx, creds)
	if err != nil {
		logger.Warn("failed to get account id", zap.Error(err))
		return nil, err
	}

	iamClient := iam.NewFromConfig(creds)
	user, err := iamClient.GetUser(ctx, &iam.GetUserInput{})
	if err != nil {
		logger.Warn("failed to get user", zap.Error(err))
		return nil, err
	}
	paginator := iam.NewListAttachedUserPoliciesPaginator(iamClient, &iam.ListAttachedUserPoliciesInput{
		UserName: user.User.UserName,
	})

	policyARNs := make([]string, 0)
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			logger.Warn("failed to get attached policies", zap.Error(err))
			return nil, err
		}
		for _, policy := range page.AttachedPolicies {
			policyARNs = append(policyARNs, *policy.PolicyArn)
		}
	}

	metadata := AWSCredentialMetadata{
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
		logger.Warn("failed to get access keys", zap.Error(err))
		return nil, err
	}

	organization, err := describer.OrganizationOrganization(ctx, creds)
	if err != nil {
		if !ignoreAwsOrgError(err) {
			logger.Warn("failed to get organization", zap.Error(err))
			return nil, err
		}
	}

	if organization != nil {
		metadata.OrganizationID = organization.Id
		metadata.OrganizationMasterAccountEmail = organization.MasterAccountEmail
		metadata.OrganizationMasterAccountId = organization.MasterAccountId
		accounts, err := discoverAWSAccounts(ctx, creds)
		if err != nil {
			logger.Warn("failed to get accounts", zap.Error(err))
			return nil, err
		}
		metadata.OrganizationDiscoveredAccountCount = utils.GetPointer(len(accounts))
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
