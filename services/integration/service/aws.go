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
	"github.com/aws/aws-sdk-go-v2/service/sts"
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
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

var ErrAWSReadAccessPolicy = errors.New("failed to find read access policy")

// NewAWS create a credential instance for AWS Organization.
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

func AWSCurrentAccount(ctx context.Context, cfg awsOfficial.Config) (*model.AWSAccount, error) {
	stsClient := sts.NewFromConfig(cfg)
	account, err := stsClient.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return nil, err
	}

	orgs, err := describer.OrganizationOrganization(ctx, cfg)
	if err != nil {
		var ae smithy.APIError
		if !errors.As(err, &ae) ||
			(ae.ErrorCode() != (&types.AWSOrganizationsNotInUseException{}).ErrorCode() &&
				ae.ErrorCode() != (&types.AccessDeniedException{}).ErrorCode()) {
			return nil, err
		}
	}

	acc, err := describer.OrganizationAccount(ctx, cfg, *account.Account)
	if err != nil {
		var ae smithy.APIError
		if !errors.As(err, &ae) ||
			(ae.ErrorCode() != (&types.AWSOrganizationsNotInUseException{}).ErrorCode() &&
				ae.ErrorCode() != (&types.AccessDeniedException{}).ErrorCode()) {
			return nil, err
		}
	}
	accountName := account.UserId
	if acc != nil {
		accountName = acc.Name
	}

	return &model.AWSAccount{
		AccountID:    *account.Account,
		AccountName:  accountName,
		Organization: orgs,
		Account:      acc,
	}, nil
}

// AWSHealthCheck checks the AWS credential health
func (h Credential) AWSHealthCheck(
	ctx context.Context,
	cred *model.Credential,
	update bool,
) (healthy bool, err error) {
	// defer function is called to update the credential health.
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

		if update == true {
			if dbErr := h.repo.Update(ctx, cred); dbErr != nil {
				err = dbErr
			}
		}
	}()

	config, err := h.kms.Decrypt(cred.Secret, h.keyARN)
	if err != nil {
		return false, err
	}

	awsCnf, err := fp.FromMap[model.AWSCredentialConfig](config)
	if err != nil {
		return false, err
	}

	sdkCnf, err := h.AWSSDKConfig(
		ctx,
		aws.GetRoleArnFromName(awsCnf.AccountID, awsCnf.AssumeRoleName),
		awsCnf.ExternalId,
	)

	org, accounts, err := h.AWSOrgAccounts(ctx, sdkCnf)
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

func (h Credential) AWSOrgAccounts(ctx context.Context, cfg awsOfficial.Config) (*types.Organization, []types.Account, error) {
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

	awsCnf, err := fp.FromMap[model.AWSCredentialConfig](cnf)
	if err != nil {
		return nil, err
	}

	awsConfig, err := aws.GetConfig(
		context.Background(),
		h.masterAccessKey,
		h.masterSecretKey,
		"",
		aws.GetRoleArnFromName(awsCnf.AccountID, awsCnf.AssumeRoleName),
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

		maxConnections, err := h.connSvc.MaxConnections(ctx)
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

func NewAWSAutoOnboardedConnection(
	org *types.Organization,
	account types.Account,
	creationMethod source.SourceCreationMethod,
	description string,
	creds model.Credential,
	awsConfig awsOfficial.Config,
) (model.Connection, error) {
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

func (h Connection) NewAWS(
	ctx context.Context,
	account model.AWSAccount,
	description string,
	req entity.AWSCredentialConfig,
) (model.Connection, error) {
	cfg := describe.AWSAccountConfig{
		AccessKey: h.masterAccessKey,
		SecretKey: h.masterSecretKey,
	}

	maxConnections, err := h.MaxConnections(ctx)
	if err != nil {
		h.logger.Error("cannot read number of the available connections", zap.Error(err))

		return model.Connection{}, err
	}

	currentConnections, err := h.Count(ctx, nil)
	if err != nil {
		h.logger.Error("cannot read number of the current connections", zap.Error(err))

		return model.Connection{}, err
	}

	if currentConnections+1 > maxConnections {
		return model.Connection{}, ErrMaxConnectionsExceeded
	}

	id := uuid.New()
	provider := source.CloudAWS

	credName := fmt.Sprintf("%s - %s - default credentials", provider, account.AccountID)
	creds := model.Credential{
		ID:             uuid.New(),
		Name:           &credName,
		ConnectorType:  provider,
		Secret:         "",
		CredentialType: model.CredentialTypeAutoAws,
	}

	accountName := account.AccountID
	if account.AccountName != nil {
		accountName = *account.AccountName
	}
	accountEmail := ""
	if account.Account != nil && account.Account.Email != nil {
		accountEmail = *account.Account.Email
	}

	s := model.Connection{
		ID:                   id,
		SourceId:             account.AccountID,
		Name:                 accountName,
		Email:                accountEmail,
		Type:                 provider,
		Description:          description,
		CredentialID:         creds.ID,
		Credential:           creds,
		LifecycleState:       model.ConnectionLifecycleStateInProgress,
		AssetDiscoveryMethod: source.AssetDiscoveryMethodTypeScheduled,
		LastHealthCheckTime:  time.Now(),
		CreationMethod:       source.SourceCreationMethodManual,
	}
	s.Credential.Version = 2

	if len(strings.TrimSpace(s.Name)) == 0 {
		s.Name = s.SourceId
	}

	metadata, err := model.NewAWSConnectionMetadata(ctx, cfg, s, account)
	if err != nil {
		h.logger.Warn("cannot create metadata for the aws connection", zap.Error(err))
	}

	marshalMetadata, err := json.Marshal(metadata)
	if err != nil {
		marshalMetadata = []byte("{}")
	}
	s.Metadata = marshalMetadata

	secretBytes, err := h.kms.Encrypt(req.AsMap(), h.keyARN)
	if err != nil {
		h.logger.Error("cannot encrypt request data into the connection", zap.Error(err))

		return model.Connection{}, err
	}
	s.Credential.Secret = string(secretBytes)

	return s, nil
}

func (h Credential) AWSUpdate(ctx context.Context, id uuid.UUID, req entity.UpdateAWSCredentialRequest) error {
	ctx, span := h.tracer.Start(ctx, "update-aws-credential")
	defer span.End()

	cred, err := h.Get(ctx, id.String())
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())

		return err
	}
	span.AddEvent("information", trace.WithAttributes(
		attribute.String("credential name", *cred.Name),
	))

	if req.Name != nil {
		cred.Name = req.Name
	}

	cnf, err := h.kms.Decrypt(cred.Secret, h.keyARN)
	if err != nil {
		return err
	}

	config, err := fp.FromMap[entity.AWSCredentialConfig](cnf)
	if err != nil {
		return err
	}

	if req.Config != nil {
		if req.Config.AssumeRoleName != "" {
			config.AssumeRoleName = req.Config.AssumeRoleName
		}

		if req.Config.AccountID != "" {
			config.AccountID = req.Config.AccountID
		}

		if req.Config.ExternalId != nil {
			config.ExternalId = req.Config.ExternalId
		}
	}

	awsConfig, err := h.AWSSDKConfig(
		ctx,
		aws.GetRoleArnFromName(config.AccountID, config.AssumeRoleName),
		req.Config.ExternalId,
	)
	if err != nil {
		h.logger.Error("reading aws sdk configuration failed", zap.Error(err))

		return err
	}

	org, accounts, err := h.AWSOrgAccounts(ctx, awsConfig)
	if err != nil {
		h.logger.Error("getting aws accounts and organizations", zap.Error(err))

		return err
	}

	metadata, err := model.ExtractCredentialMetadata(config.AccountID, org, accounts)
	if err != nil {
		return err
	}

	jsonMetadata, err := json.Marshal(metadata)
	if err != nil {
		return err
	}

	cred.Metadata = jsonMetadata

	secretBytes, err := h.kms.Encrypt(config.AsMap(), h.keyARN)
	if err != nil {
		return err
	}
	cred.Secret = string(secretBytes)

	if err := h.repo.Update(ctx, cred); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	if _, err := h.AWSHealthCheck(ctx, cred, true); err != nil {
		return err
	}

	return nil
}

// AWSCredentialConfig reads credentials configuration from aws credential secret and return it.
func (h Credential) AWSCredentialConfig(ctx context.Context, credential model.Credential) (*model.AWSCredentialConfig, error) {
	raw, err := h.kms.Decrypt(credential.Secret, h.keyARN)
	if err != nil {
		return nil, err
	}

	cnf, err := fp.FromMap[model.AWSCredentialConfig](raw)
	if err != nil {
		return nil, err
	}

	return cnf, nil
}

// AWSHealthCheck checks the connection health status and update the returned model. if the update flag is false then
// the database is not get updated.
func (h Connection) AWSHealthCheck(ctx context.Context, connection model.Connection, update bool) (model.Connection, error) {
	var cnf map[string]any

	cnf, err := h.kms.Decrypt(connection.Credential.Secret, h.keyARN)
	if err != nil {
		h.logger.Error("failed to decrypt credential", zap.Error(err), zap.String("connectionId", connection.SourceId))
		return connection, err
	}

	awsCnf, err := fp.FromMap[model.AWSCredentialConfig](cnf)
	if err != nil {
		h.logger.Error("failed to get aws config", zap.Error(err), zap.String("connectionId", connection.SourceId))
		return connection, err
	}

	assumeRoleArn := aws.GetRoleArnFromName(connection.SourceId, awsCnf.AssumeRoleName)

	sdkCnf, err := aws.GetConfig(ctx, h.masterAccessKey, h.masterSecretKey, "", assumeRoleArn, awsCnf.ExternalId)
	if err != nil {
		h.logger.Error("failed to get aws config", zap.Error(err), zap.String("connectionId", connection.SourceId))
		return connection, err
	}

	iamClient := iam.NewFromConfig(sdkCnf)
	paginator := iam.NewListAttachedRolePoliciesPaginator(iamClient, &iam.ListAttachedRolePoliciesInput{
		RoleName: &awsCnf.AssumeRoleName,
	})
	var policyARNs []string
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return connection, err
		}
		for _, policy := range page.AttachedPolicies {
			policyARNs = append(policyARNs, *policy.PolicyArn)
		}
	}

	assetDiscoveryAttached := true

	awsAssetDiscovery, err := h.meta.Client.GetConfigMetadata(&httpclient.Context{UserRole: api.InternalRole}, models.MetadataKeyAssetDiscoveryAWSPolicyARNs)
	if err != nil {
		return connection, err
	}

	for _, policyARN := range strings.Split(awsAssetDiscovery.GetValue().(string), ",") {
		policyARN = strings.ReplaceAll(policyARN, "${accountID}", connection.SourceId)
		if !fp.Includes(policyARN, policyARNs) {
			h.logger.Error("policy is not there", zap.String("policyARN", policyARN), zap.Strings("attachedPolicies", policyARNs))

			assetDiscoveryAttached = false
		}
	}

	spendAttached := connection.Credential.SpendDiscovery != nil && *connection.Credential.SpendDiscovery

	if !assetDiscoveryAttached && !spendAttached {
		var healthMessage string
		if err == nil {
			healthMessage = "Failed to find read permission"
		} else {
			healthMessage = err.Error()
		}

		connection, err = h.UpdateHealth(ctx, connection, source.HealthStatusUnhealthy, &healthMessage, fp.Optional(false), fp.Optional(false), update)
		if err != nil {
			h.logger.Warn("failed to update connection health", zap.Error(err), zap.String("connectionId", connection.SourceId))

			return connection, err
		}
	} else {
		connection, err = h.UpdateHealth(ctx, connection, source.HealthStatusHealthy, fp.Optional(""), &spendAttached, &assetDiscoveryAttached, update)
		if err != nil {
			h.logger.Warn("failed to update connection health", zap.Error(err), zap.String("connectionId", connection.SourceId))

			return connection, err
		}
	}

	return connection, nil
}
