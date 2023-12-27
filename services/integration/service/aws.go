package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	awsOfficial "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/organizations/types"
	"github.com/aws/smithy-go"
	"github.com/google/uuid"
	"github.com/kaytu-io/kaytu-aws-describer/aws"
	"github.com/kaytu-io/kaytu-aws-describer/aws/describer"
	"github.com/kaytu-io/kaytu-engine/pkg/auth/api"
	"github.com/kaytu-io/kaytu-engine/pkg/describe"
	"github.com/kaytu-io/kaytu-engine/pkg/httpclient"
	"github.com/kaytu-io/kaytu-engine/pkg/metadata/models"
	"github.com/kaytu-io/kaytu-engine/services/integration/api/entity"
	"github.com/kaytu-io/kaytu-engine/services/integration/model"
	"github.com/kaytu-io/kaytu-util/pkg/fp"
	"github.com/kaytu-io/kaytu-util/pkg/source"
	"go.uber.org/zap"
)

// NewAWS create a credential instance for aws
func (h Credential) NewAWS(
	ctx context.Context,
	name string,
	metadata *model.AWSCredentialMetadata,
	credentialType model.CredentialType,
	version int,
	config *entity.AWSCredentialConfig,
) (*model.Credential, error) {
	id := uuid.New()

	jsonMetadata, err := json.Marshal(metadata)
	if err != nil {
		return nil, err
	}
	crd := &model.Credential{
		ID:             id,
		Name:           &name,
		ConnectorType:  source.CloudAWS,
		Secret:         fmt.Sprintf("sources/%s/%s", strings.ToLower(string(source.CloudAWS)), id),
		CredentialType: credentialType,
		Metadata:       jsonMetadata,
		Version:        version,
	}
	if credentialType == model.CredentialTypeManualAwsOrganization {
		crd.AutoOnboardEnabled = true
	}

	secretBytes, err := h.kms.Encrypt(config.AsMap(), h.keyARN)
	if err != nil {
		return nil, err
	}
	crd.Secret = string(secretBytes)

	return crd, nil
}

func (Credential) AWSMetadata(ctx context.Context, config describe.AWSAccountConfig) (*model.AWSCredentialMetadata, error) {
	creds, err := aws.GetConfig(ctx, config.AccessKey, config.SecretKey, "", "", nil)
	if err != nil {
		return nil, err
	}

	if creds.Region == "" {
		creds.Region = "us-east-1"
	}

	iamClient := iam.NewFromConfig(creds)

	user, err := iamClient.GetUser(ctx, &iam.GetUserInput{})
	if err != nil {
		return nil, err
	}

	paginator := iam.NewListAttachedUserPoliciesPaginator(iamClient, &iam.ListAttachedUserPoliciesInput{
		UserName: user.User.UserName,
	})

	policyARNs := make([]string, 0)
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, policy := range page.AttachedPolicies {
			policyARNs = append(policyARNs, *policy.PolicyArn)
		}
	}

	accessKeys, err := iamClient.ListAccessKeys(ctx, &iam.ListAccessKeysInput{
		UserName: user.User.UserName,
	})
	if err != nil {
		return nil, err
	}

	creds, err = aws.GetConfig(ctx, config.AccessKey, config.SecretKey, "", config.AssumeAdminRoleName, nil)
	if err != nil {
		return nil, err
	}

	accID, err := describer.STSAccount(ctx, creds)
	if err != nil {
		return nil, err
	}

	metadata := model.AWSCredentialMetadata{
		AccountID:        accID,
		IamUserName:      user.User.UserName,
		AttachedPolicies: policyARNs,
	}

	for _, key := range accessKeys.AccessKeyMetadata {
		if *key.AccessKeyId == config.AccessKey && key.CreateDate != nil {
			metadata.IamApiKeyCreationDate = *key.CreateDate
		}
	}

	organization, err := describer.OrganizationOrganization(ctx, creds)
	if err != nil {
		var ae smithy.APIError
		if !errors.As(err, &ae) ||
			(ae.ErrorCode() != (&types.AWSOrganizationsNotInUseException{}).ErrorCode() &&
				ae.ErrorCode() != (&types.AccessDeniedException{}).ErrorCode()) {
			return nil, err
		}
	}

	if organization != nil {
		metadata.OrganizationID = organization.Id
		metadata.OrganizationMasterAccountEmail = organization.MasterAccountEmail
		metadata.OrganizationMasterAccountId = organization.MasterAccountId
		accounts, err := DiscoverAWSAccounts(ctx, creds)
		if err != nil {
			return nil, err
		}
		metadata.OrganizationDiscoveredAccountCount = fp.Optional[int](len(accounts))
	}

	return &metadata, nil
}

func DiscoverAWSAccounts(ctx context.Context, cfg awsOfficial.Config) ([]model.AWSAccount, error) {
	orgs, err := describer.OrganizationOrganization(ctx, cfg)
	if err != nil {
		var ae smithy.APIError
		if !errors.As(err, &ae) ||
			(ae.ErrorCode() != (&types.AWSOrganizationsNotInUseException{}).ErrorCode() &&
				ae.ErrorCode() != (&types.AccessDeniedException{}).ErrorCode()) {
			return nil, err
		}
	}

	accounts, err := describer.OrganizationAccounts(ctx, cfg)
	if err != nil {
		var ae smithy.APIError
		if !errors.As(err, &ae) ||
			(ae.ErrorCode() != (&types.AWSOrganizationsNotInUseException{}).ErrorCode() &&
				ae.ErrorCode() != (&types.AccessDeniedException{}).ErrorCode()) {
			return nil, err
		}
	}

	awsAccounts := make([]model.AWSAccount, 0)
	for _, account := range accounts {
		if account.Id == nil {
			continue
		}
		localAccount := account
		awsAccounts = append(awsAccounts, model.AWSAccount{
			AccountID:    *localAccount.Id,
			AccountName:  localAccount.Name,
			Organization: orgs,
			Account:      &localAccount,
		})
	}

	return awsAccounts, nil
}

func (h Credential) AWSHealthCheck(ctx context.Context, cred *model.Credential) (bool, error) {
	config, err := h.kms.Decrypt(cred.Secret, h.keyARN)
	if err != nil {
		return false, err
	}

	awsConfig, err := entity.AWSCredentialV2ConfigFromMap(config)
	if err != nil {
		return false, err
	}

	sdkCnf, err := h.GetAWSSDKConfig(generateRoleARN(awsConfig.AccountID, awsConfig.AssumeRoleName), awsConfig.ExternalId)
	if err != nil {
		return false, err
	}

	org, accounts, err := h.GetOrgAccounts(sdkCnf)
	if err != nil {
		return false, err
	}

	metadata, err := h.ExtractCredentialMetadata(awsConfig.AccountID, org, accounts)
	if err != nil {
		return false, err
	}

	jsonMetadata, err := json.Marshal(metadata)
	if err != nil {
		return false, err
	}
	cred.Metadata = jsonMetadata

	iamClient := iam.NewFromConfig(sdkCnf)
	paginator := iam.NewListAttachedRolePoliciesPaginator(iamClient, &iam.ListAttachedRolePoliciesInput{
		RoleName: &awsConfig.AssumeRoleName,
	})

	var policyARNs []string
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return false, err
		}
		for _, policy := range page.AttachedPolicies {
			policyARNs = append(policyARNs, *policy.PolicyArn)
		}
	}

	spendAttached := true
	awsSpendDiscovery, err := h.meta.Client.GetConfigMetadata(&httpclient.Context{UserRole: api.InternalRole}, models.MetadataKeySpendDiscoveryAWSPolicyARNs)
	if err != nil {
		if err != nil {
			return false, err
		}
	}

	for _, policyARN := range strings.Split(awsSpendDiscovery.GetValue().(string), ",") {
		policyARN = strings.ReplaceAll(policyARN, "${accountID}", awsConfig.AccountID)
		if !fp.Includes(policyARN, policyARNs) {
			h.logger.Error("policy is not there", zap.String("policyARN", policyARN), zap.Strings("attachedPolicies", policyARNs))
			spendAttached = false
		}
	}
	cred.SpendDiscovery = &spendAttached

	return true, nil
}
