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
	"github.com/aws/aws-sdk-go-v2/service/organizations"
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

		if dberr := h.repo.Update(ctx, cred); dberr != nil {
			err = dberr
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

func (h Credential) AWSOnboard(ctx context.Context, credential model.Credential) ([]model.Connection, error) {
	onboardedSources := make([]model.Connection, 0)
	cnf, err := h.kms.Decrypt(credential.Secret, h.keyARN)
	if err != nil {
		return nil, err
	}

	awsCnf, err := model.AWSCredentialConfigFromMap(cnf)
	if err != nil {
		return nil, err
	}

	awsConfig, err := aws.GetConfig(
		context.Background(),
		h.masterAccessKey,
		h.masterSecretKey,
		"",
		fmt.Sprintf("arn:aws:iam::%s:role/%s", awsCnf.AccountID, awsCnf.AssumeRoleName),
		awsCnf.ExternalId,
	)
	if err != nil {
		return nil, err
	}

	if awsConfig.Region == "" {
		awsConfig.Region = "us-east-1"
	}

	h.logger.Info("discovering accounts", zap.String("credentialId", credential.ID.String()))

	org, err := describer.OrganizationOrganization(context.Background(), awsConfig)
	if err != nil {
		var ae smithy.APIError
		if !errors.As(err, &ae) ||
			(ae.ErrorCode() != (&types.AWSOrganizationsNotInUseException{}).ErrorCode() &&
				ae.ErrorCode() != (&types.AccessDeniedException{}).ErrorCode()) {
			return nil, err
		}
	}

	accounts, err := describer.OrganizationAccounts(context.Background(), awsConfig)
	if err != nil {
		var ae smithy.APIError
		if !errors.As(err, &ae) ||
			(ae.ErrorCode() != (&types.AWSOrganizationsNotInUseException{}).ErrorCode() &&
				ae.ErrorCode() != (&types.AccessDeniedException{}).ErrorCode()) {
			return nil, err
		}
	}

	h.logger.Info("discovered accounts", zap.Int("count", len(accounts)))

	existingConnections, err := h.connSvc.List(ctx, []source.Type{credential.ConnectorType})
	if err != nil {
		return nil, err
	}

	existingConnectionAccountIDs := make([]string, 0, len(existingConnections))
	for _, conn := range existingConnections {
		existingConnectionAccountIDs = append(existingConnectionAccountIDs, conn.SourceId)
	}
	accountsToOnboard := make([]types.Account, 0)

	for _, account := range accounts {
		if !fp.Includes(*account.Id, existingConnectionAccountIDs) {
			accountsToOnboard = append(accountsToOnboard, account)
		} else {
			for _, conn := range existingConnections {
				if conn.SourceId == *account.Id {
					name := *account.Id
					if account.Name != nil {
						name = *account.Name
					}

					if conn.CredentialID.String() != credential.ID.String() {
						h.logger.Warn("organization account is onboarded as an standalone account",
							zap.String("accountID", *account.Id),
							zap.String("connectionID", conn.ID.String()))
					}

					localConn := conn
					if conn.Name != name {
						localConn.Name = name
					}
					if account.Status != types.AccountStatusActive {
						localConn.LifecycleState = model.ConnectionLifecycleStateArchived
					} else if localConn.LifecycleState == model.ConnectionLifecycleStateArchived {
						localConn.LifecycleState = model.ConnectionLifecycleStateDiscovered
						if credential.AutoOnboardEnabled {
							localConn.LifecycleState = model.ConnectionLifecycleStateOnboard
						}
					}
					if conn.Name != name || account.Status != types.AccountStatusActive || conn.LifecycleState != localConn.LifecycleState {
						if err := h.connSvc.Update(ctx, localConn); err != nil {
							h.logger.Error("failed to update source", zap.Error(err))

							return nil, err
						}
					}
				}
			}
		}
	}

	h.logger.Info("onboarding accounts", zap.Int("count", len(accountsToOnboard)))
	for _, account := range accountsToOnboard {
		h.logger.Info("onboarding account", zap.String("accountID", *account.Id))
		count, err := h.connSvc.Count(ctx, nil)
		if err != nil {
			return nil, err
		}

		maxConnections, err := h.connSvc.MaxConnections()
		if err != nil {
			return nil, err
		}

		if count >= maxConnections {
			return nil, ErrMaxConnectionsExceeded
		}

		src, err := NewAWSAutoOnboardedConnection(
			org,
			account,
			source.SourceCreationMethodAutoOnboard,
			fmt.Sprintf("Auto onboarded account %s", *account.Id),
			credential,
			awsConfig,
		)
		if err != nil {
			return nil, err
		}

		if err := h.connSvc.Create(ctx, src); err != nil {
			return nil, err
		}

		metadata := make(map[string]any)
		if src.Metadata.String() != "" {
			err := json.Unmarshal(src.Metadata, &metadata)
			if err != nil {
				return nil, err
			}
		}

		onboardedSources = append(onboardedSources, src)
	}

	return onboardedSources, nil
}

func NewAWSAutoOnboardedConnection(org *types.Organization, account types.Account, creationMethod source.SourceCreationMethod, description string, creds model.Credential, awsConfig awsOfficial.Config) (model.Connection, error) {
	id := uuid.New()

	name := *account.Id
	if account.Name != nil {
		name = *account.Name
	}

	lifecycleState := model.ConnectionLifecycleStateDiscovered
	if creds.AutoOnboardEnabled {
		lifecycleState = model.ConnectionLifecycleStateInProgress
	}

	if account.Status != types.AccountStatusActive {
		lifecycleState = model.ConnectionLifecycleStateArchived
	}

	s := model.Connection{
		ID:                   id,
		SourceId:             *account.Id,
		Name:                 name,
		Description:          description,
		Type:                 source.CloudAWS,
		CredentialID:         creds.ID,
		Credential:           creds,
		LifecycleState:       lifecycleState,
		AssetDiscoveryMethod: source.AssetDiscoveryMethodTypeScheduled,
		LastHealthCheckTime:  time.Now(),
		CreationMethod:       creationMethod,
	}
	metadata := model.AWSConnectionMetadata{
		AccountID:           *account.Id,
		AccountName:         name,
		Organization:        nil,
		OrganizationAccount: &account,
		OrganizationTags:    nil,
	}
	if creds.CredentialType == model.CredentialTypeAutoAws {
		metadata.AccountType = model.AWSAccountTypeStandalone
	} else {
		metadata.AccountType = model.AWSAccountTypeOrganizationMember
	}

	metadata.Organization = org
	if org != nil {
		if org.MasterAccountId != nil &&
			*metadata.Organization.MasterAccountId == *account.Id {
			metadata.AccountType = model.AWSAccountTypeOrganizationManager
		}

		organizationClient := organizations.NewFromConfig(awsConfig)
		tags, err := organizationClient.ListTagsForResource(context.TODO(), &organizations.ListTagsForResourceInput{
			ResourceId: &metadata.AccountID,
		})
		if err != nil {
			return model.Connection{}, err
		}
		metadata.OrganizationTags = make(map[string]string)
		for _, tag := range tags.Tags {
			if tag.Key == nil || tag.Value == nil {
				continue
			}
			metadata.OrganizationTags[*tag.Key] = *tag.Value
		}
	}

	jsonMetadata, err := json.Marshal(metadata)
	if err != nil {
		return model.Connection{}, err
	}

	s.Metadata = jsonMetadata

	return s, nil
}
