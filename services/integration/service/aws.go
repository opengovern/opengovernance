package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	awsOfficial "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/organizations/types"
	"github.com/aws/smithy-go"
	"github.com/google/uuid"
	"github.com/kaytu-io/kaytu-aws-describer/aws"
	"github.com/kaytu-io/kaytu-aws-describer/aws/describer"
	"github.com/kaytu-io/kaytu-engine/services/integration/api/entity"
	"github.com/kaytu-io/kaytu-engine/services/integration/model"
	"github.com/kaytu-io/kaytu-util/pkg/source"
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
