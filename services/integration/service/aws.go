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
	config entity.AWSCredentialConfig,
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
		Version:        2,
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

func (h Credential) AWSSDKConfig(ctx context.Context, roleARN string, externalID *string) (awsOfficial.Config, error) {
	awsConfig, err := aws.GetConfig(
		ctx,
		h.masterAccessKey,
		h.masterSecretKey,
		"",
		roleARN,
		externalID,
	)
	if err != nil {
		return awsOfficial.Config{}, err
	}

	if awsConfig.Region == "" {
		awsConfig.Region = "us-east-1"
	}

	return awsConfig, nil
}

// AWSHealthCheck checks the aws credential health
func (h Credential) AWSHealthCheck(
	ctx context.Context,
	cred *model.Credential,
) (healthy bool, err error) {
	defer func() {
		if err != nil {
			h.logger.Error("credential is not healthy", zap.Error(err))
		}

		if !healthy || err != nil {
			cred.HealthReason = fp.Optional(err.Error())
			cred.HealthStatus = source.HealthStatusUnhealthy
		} else {
			cred.HealthReason = fp.Optional("")
			cred.HealthStatus = source.HealthStatusHealthy
		}

		cred.LastHealthCheckTime = time.Now()

		if err := h.repo.Update(ctx, cred); err != nil {
			err = err
		}
	}()

	config, err := h.kms.Decrypt(cred.Secret, h.keyARN)
	if err != nil {
		return false, err
	}

	awsCnf, err := model.AWSCredentialConfigFromMap(config)
	if err != nil {
		return false, err
	}

	sdkCnf, err := h.AWSSDKConfig(
		ctx,
		fmt.Sprintf("arn:aws:iam::%s:role/%s", awsCnf.AccountID, awsCnf.AssumeRoleName),
		awsCnf.ExternalId,
	)

	org, accounts, err := h.OrgAccounts(ctx, sdkCnf)
	if err != nil {
		return false, err
	}

	metadata, err := model.ExtractCredentialMetadata(awsCnf.AccountID, org, accounts)
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
		RoleName: &awsCnf.AssumeRoleName,
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
		policyARN = strings.ReplaceAll(policyARN, "${accountID}", awsCnf.AccountID)
		if !fp.Includes(policyARN, policyARNs) {
			h.logger.Error("policy is not there", zap.String("policyARN", policyARN), zap.Strings("attachedPolicies", policyARNs))
			spendAttached = false
		}
	}

	cred.SpendDiscovery = &spendAttached

	return true, nil
}

func (h Credential) OrgAccounts(ctx context.Context, cfg awsOfficial.Config) (*types.Organization, []types.Account, error) {
	orgs, err := describer.OrganizationOrganization(ctx, cfg)
	if err != nil {
		var ae smithy.APIError
		if !errors.As(err, &ae) ||
			(ae.ErrorCode() != (&types.AWSOrganizationsNotInUseException{}).ErrorCode() &&
				ae.ErrorCode() != (&types.AccessDeniedException{}).ErrorCode()) {
			return nil, nil, err
		}
	}

	accounts, err := describer.OrganizationAccounts(ctx, cfg)
	if err != nil {
		var ae smithy.APIError
		if !errors.As(err, &ae) ||
			(ae.ErrorCode() != (&types.AWSOrganizationsNotInUseException{}).ErrorCode() &&
				ae.ErrorCode() != (&types.AccessDeniedException{}).ErrorCode()) {
			return nil, nil, err
		}
	}

	return orgs, accounts, nil
}
