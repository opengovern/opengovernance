package onboard

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	awsOrgTypes "github.com/aws/aws-sdk-go-v2/service/organizations/types"
	kaytuAws "github.com/kaytu-io/kaytu-aws-describer/aws"
	"github.com/kaytu-io/kaytu-aws-describer/aws/describer"
	kaytuAzure "github.com/kaytu-io/kaytu-azure-describer/azure"
	"github.com/kaytu-io/kaytu-engine/pkg/describe"
	"github.com/kaytu-io/kaytu-engine/pkg/metadata/models"
	"github.com/kaytu-io/kaytu-engine/pkg/onboard/api"
	apiv2 "github.com/kaytu-io/kaytu-engine/pkg/onboard/api/v2"
	"github.com/kaytu-io/kaytu-engine/pkg/utils"
	"github.com/kaytu-io/kaytu-engine/services/integration/model"
	api2 "github.com/kaytu-io/kaytu-util/pkg/api"
	"github.com/kaytu-io/kaytu-util/pkg/httpclient"
	"github.com/kaytu-io/kaytu-util/pkg/source"
	"github.com/labstack/echo/v4"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"net/http"
	"strings"
	"time"
)

func generateRoleARN(accountID, roleName string) string {
	return fmt.Sprintf("arn:aws:iam::%s:role/%s", accountID, roleName)
}

func (h HttpHandler) GetAWSSDKConfig(ctx context.Context, roleARN string, externalID *string) (aws.Config, error) {
	awsConfig, err := kaytuAws.GetConfig(
		ctx,
		h.masterAccessKey,
		h.masterSecretKey,
		"",
		roleARN,
		externalID,
	)
	if err != nil {
		return aws.Config{}, err
	}
	if awsConfig.Region == "" {
		awsConfig.Region = "us-east-1"
	}
	return awsConfig, nil
}

func (h HttpHandler) GetOrgAccounts(ctx context.Context, sdkConfig aws.Config) (*awsOrgTypes.Organization, []awsOrgTypes.Account, error) {
	org, err := describer.OrganizationOrganization(ctx, sdkConfig)
	if err != nil {
		if !ignoreAwsOrgError(err) {
			h.logger.Error("failed to get organization", zap.Error(err))
			return nil, nil, err
		}
	}

	accounts, err := describer.OrganizationAccounts(ctx, sdkConfig)
	if err != nil {
		if !ignoreAwsOrgError(err) {
			h.logger.Error("failed to get organization accounts", zap.Error(err))
			return nil, nil, err
		}
	}

	return org, accounts, nil
}

func (h HttpHandler) ExtractCredentialMetadata(accountID string, org *awsOrgTypes.Organization, childAccounts []awsOrgTypes.Account) (*AWSCredentialMetadata, error) {
	metadata := AWSCredentialMetadata{
		AccountID:             accountID,
		IamUserName:           nil,
		IamApiKeyCreationDate: time.Time{},
		AttachedPolicies:      nil,
	}

	if org != nil {
		metadata.OrganizationID = org.Id
		metadata.OrganizationMasterAccountEmail = org.MasterAccountEmail
		metadata.OrganizationMasterAccountId = org.MasterAccountId
		metadata.OrganizationDiscoveredAccountCount = utils.GetPointer(len(childAccounts))
	}
	return &metadata, nil
}

func (h HttpHandler) createAWSCredential(ctx context.Context, req apiv2.CreateCredentialV2Request) (*apiv2.CreateCredentialV2Response, error) {
	awsConfig, err := h.GetAWSSDKConfig(ctx, generateRoleARN(req.AWSConfig.AccountID, req.AWSConfig.AssumeRoleName), req.AWSConfig.ExternalId)
	if err != nil {
		return nil, err
	}

	org, accounts, err := h.GetOrgAccounts(ctx, awsConfig)
	if err != nil {
		return nil, err
	}

	metadata, err := h.ExtractCredentialMetadata(req.AWSConfig.AccountID, org, accounts)
	if err != nil {
		return nil, err
	}

	name := metadata.AccountID
	if metadata.OrganizationID != nil {
		name = *metadata.OrganizationID
	}

	cred, err := NewAWSCredential(name, metadata, model.CredentialTypeManualAwsOrganization, 2)
	if err != nil {
		return nil, err
	}
	cred.HealthStatus = source.HealthStatusHealthy

	secretBytes, err := h.vaultSc.Encrypt(ctx, req.AWSConfig.AsMap())
	if err != nil {
		return nil, err
	}
	cred.Secret = string(secretBytes)
	if err := h.db.CreateCredential(cred); err != nil {
		return nil, err
	}
	_, err = h.checkCredentialHealth(ctx, *cred)
	if err != nil {
		return nil, err
	}

	return &apiv2.CreateCredentialV2Response{ID: cred.ID.String()}, nil
}

func (h HttpHandler) autoOnboardAWSAccountsV2(ctx context.Context, credential model.Credential, maxConnections int64) ([]api.Connection, error) {
	onboardedSources := make([]api.Connection, 0)
	cnf, err := h.vaultSc.Decrypt(ctx, credential.Secret)
	if err != nil {
		return nil, err
	}

	awsCnf, err := apiv2.AWSCredentialV2ConfigFromMap(cnf)
	if err != nil {
		return nil, err
	}

	awsConfig, err := kaytuAws.GetConfig(
		ctx,
		h.masterAccessKey,
		h.masterSecretKey,
		"",
		generateRoleARN(awsCnf.AccountID, awsCnf.AssumeRoleName),
		awsCnf.ExternalId,
	)
	if err != nil {
		return nil, err
	}
	if awsConfig.Region == "" {
		awsConfig.Region = "us-east-1"
	}

	h.logger.Info("discovering accounts", zap.String("credentialId", credential.ID.String()))
	org, err := describer.OrganizationOrganization(ctx, awsConfig)
	if err != nil {
		if !ignoreAwsOrgError(err) {
			return nil, err
		}
	}
	accounts, err := describer.OrganizationAccounts(ctx, awsConfig)
	if err != nil {
		if !ignoreAwsOrgError(err) {
			return nil, err
		}
	}
	h.logger.Info("discovered accounts", zap.Int("count", len(accounts)))

	existingConnections, err := h.db.GetSourcesOfType(credential.ConnectorType)
	if err != nil {
		return nil, err
	}

	existingConnectionAccountIDs := make([]string, 0, len(existingConnections))
	for _, conn := range existingConnections {
		existingConnectionAccountIDs = append(existingConnectionAccountIDs, conn.SourceId)
	}
	accountsToOnboard := make([]awsOrgTypes.Account, 0)

	for _, account := range accounts {
		if !utils.Includes(existingConnectionAccountIDs, *account.Id) {
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
					if account.Status != awsOrgTypes.AccountStatusActive {
						localConn.LifecycleState = model.ConnectionLifecycleStateArchived
					} else if localConn.LifecycleState == model.ConnectionLifecycleStateArchived {
						localConn.LifecycleState = model.ConnectionLifecycleStateDiscovered
						if credential.AutoOnboardEnabled {
							localConn.LifecycleState = model.ConnectionLifecycleStateOnboard
						}
					}
					if conn.Name != name || account.Status != awsOrgTypes.AccountStatusActive || conn.LifecycleState != localConn.LifecycleState {
						_, err := h.db.UpdateSource(&localConn)
						if err != nil {
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
		count, err := h.db.CountSources()
		if err != nil {
			return nil, err
		}

		if count >= maxConnections {
			return nil, echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("maximum number of connections reached: [%d/%d]", count, maxConnections))
		}

		src, err := NewAWSAutoOnboardedConnectionV2(
			ctx,
			org,
			h.logger,
			account,
			source.SourceCreationMethodAutoOnboard,
			fmt.Sprintf("Auto onboarded account %s", *account.Id),
			credential,
			awsConfig,
		)

		err = h.db.CreateSource(src)
		if err != nil {
			return nil, err
		}

		metadata := make(map[string]any)
		if src.Metadata.String() != "" {
			err := json.Unmarshal(src.Metadata, &metadata)
			if err != nil {
				return nil, err
			}
		}

		onboardedSources = append(onboardedSources, api.Connection{
			ID:                   src.ID,
			ConnectionID:         src.SourceId,
			ConnectionName:       src.Name,
			Email:                src.Email,
			Connector:            src.Type,
			Description:          src.Description,
			CredentialID:         src.CredentialID.String(),
			CredentialName:       src.Credential.Name,
			OnboardDate:          src.CreatedAt,
			LifecycleState:       api.ConnectionLifecycleState(src.LifecycleState),
			AssetDiscoveryMethod: src.AssetDiscoveryMethod,
			LastHealthCheckTime:  src.LastHealthCheckTime,
			HealthReason:         src.HealthReason,
			Metadata:             metadata,
		})
	}

	return onboardedSources, nil
}

func (h HttpHandler) checkCredentialHealthV2(ctx context.Context, cred model.Credential) (healthy bool, err error) {
	defer func() {
		if err != nil {
			h.logger.Error("credential is not healthy", zap.Error(err))
		}
		if !healthy || err != nil {
			errStr := err.Error()
			cred.HealthReason = &errStr
			cred.HealthStatus = source.HealthStatusUnhealthy
			err = echo.NewHTTPError(http.StatusBadRequest, "credential is not healthy")
		} else {
			cred.HealthStatus = source.HealthStatusHealthy
			cred.HealthReason = utils.GetPointer("")
		}
		cred.LastHealthCheckTime = time.Now()
		_, dbErr := h.db.UpdateCredential(&cred)
		if dbErr != nil {
			err = dbErr
		}
	}()

	config, err := h.vaultSc.Decrypt(ctx, cred.Secret)
	if err != nil {
		h.logger.Error("failed to decrypt credential", zap.Error(err))
		return false, err
	}

	switch cred.ConnectorType {
	case source.CloudAWS:
		awsConfig, err := apiv2.AWSCredentialV2ConfigFromMap(config)
		if err != nil {
			h.logger.Error("failed to parse aws config", zap.Error(err))
			return false, err
		}
		sdkCnf, err := h.GetAWSSDKConfig(ctx, generateRoleARN(awsConfig.AccountID, awsConfig.AssumeRoleName), awsConfig.ExternalId)
		if err != nil {
			h.logger.Error("failed to get aws sdk config", zap.Error(err))
			return false, err
		}

		org, accounts, err := h.GetOrgAccounts(ctx, sdkCnf)
		if err != nil {
			h.logger.Error("failed to get org accounts", zap.Error(err))
			return false, err
		}

		metadata, err := h.ExtractCredentialMetadata(awsConfig.AccountID, org, accounts)
		if err != nil {
			h.logger.Error("failed to extract metadata", zap.Error(err))
			return false, err
		}
		jsonMetadata, err := json.Marshal(metadata)
		if err != nil {
			h.logger.Error("failed to marshal metadata", zap.Error(err))
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
				h.logger.Error("failed to list attached role policies", zap.Error(err))
				return false, err
			}
			for _, policy := range page.AttachedPolicies {
				policyARNs = append(policyARNs, *policy.PolicyArn)
			}
		}

		spendAttached := true
		ctx2 := &httpclient.Context{UserRole: api2.InternalRole}
		ctx2.Ctx = ctx
		awsSpendDiscovery, err := h.metadataClient.GetConfigMetadata(ctx2, models.MetadataKeySpendDiscoveryAWSPolicyARNs)
		if err != nil {
			h.logger.Error("failed to get spend discovery aws policy arns", zap.Error(err))
			return false, err
		}
		for _, policyARN := range strings.Split(awsSpendDiscovery.GetValue().(string), ",") {
			policyARN = strings.ReplaceAll(policyARN, "${accountID}", awsConfig.AccountID)
			if !utils.Includes(policyARNs, policyARN) {
				h.logger.Error("policy is not there", zap.String("policyARN", policyARN), zap.Strings("attachedPolicies", policyARNs))
				spendAttached = false
			}
		}
		cred.SpendDiscovery = &spendAttached
	default:
		return false, errors.New("not implemented")
	}
	return true, nil
}

func (h HttpHandler) checkCredentialHealth(ctx context.Context, cred model.Credential) (bool, error) {
	if cred.Version == 2 {
		return h.checkCredentialHealthV2(ctx, cred)
	}

	config, err := h.vaultSc.Decrypt(ctx, cred.Secret)
	if err != nil {
		h.logger.Error("failed to decrypt credential", zap.Error(err))
		return false, echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	switch cred.ConnectorType {
	case source.CloudAWS:
		var awsConfig describe.AWSAccountConfig
		awsConfig, err = describe.AWSAccountConfigFromMap(config)
		if err != nil {
			return false, echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		var sdkCnf aws.Config
		sdkCnf, err = kaytuAws.GetConfig(ctx, awsConfig.AccessKey, awsConfig.SecretKey, "", "", nil)
		if err != nil {
			return false, echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
		err = kaytuAws.CheckGetUserPermission(h.logger, sdkCnf)
		if err == nil {
			metadata, err := getAWSCredentialsMetadata(ctx, h.logger, awsConfig)
			if err != nil {
				return false, echo.NewHTTPError(http.StatusBadRequest, err.Error())
			}
			jsonMetadata, err := json.Marshal(metadata)
			if err != nil {
				return false, echo.NewHTTPError(http.StatusInternalServerError, err.Error())
			}
			cred.Metadata = jsonMetadata
		}
	case source.CloudAzure:
		var azureConfig describe.AzureSubscriptionConfig
		azureConfig, err = describe.AzureSubscriptionConfigFromMap(config)
		if err != nil {
			return false, echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		err = kaytuAzure.CheckSPNAccessPermission(kaytuAzure.AuthConfig{
			TenantID:            azureConfig.TenantID,
			ObjectID:            azureConfig.ObjectID,
			SecretID:            azureConfig.SecretID,
			ClientID:            azureConfig.ClientID,
			ClientSecret:        azureConfig.ClientSecret,
			CertificatePath:     azureConfig.CertificatePath,
			CertificatePassword: azureConfig.CertificatePass,
			Username:            azureConfig.Username,
			Password:            azureConfig.Password,
		})
		if err == nil {
			metadata, err := getAzureCredentialsMetadata(ctx, azureConfig)
			if err != nil {
				return false, echo.NewHTTPError(http.StatusBadRequest, err.Error())
			}
			jsonMetadata, err := json.Marshal(metadata)
			if err != nil {
				return false, echo.NewHTTPError(http.StatusInternalServerError, err.Error())
			}
			cred.Metadata = jsonMetadata
		}
	}

	if err != nil {
		errStr := err.Error()
		cred.HealthReason = &errStr
		cred.HealthStatus = source.HealthStatusUnhealthy
	} else {
		cred.HealthStatus = source.HealthStatusHealthy
		cred.HealthReason = utils.GetPointer("")
	}
	cred.LastHealthCheckTime = time.Now()
	// tracer :
	_, span := tracer.Start(ctx, "new_UpdateCredential", trace.WithSpanKind(trace.SpanKindServer))
	span.SetName("new_UpdateCredential")

	_, dbErr := h.db.UpdateCredential(&cred)
	if dbErr != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return false, echo.NewHTTPError(http.StatusInternalServerError, dbErr.Error())
	}
	span.AddEvent("information", trace.WithAttributes(
		attribute.String("credential name ", *cred.Name),
	))
	span.End()

	if err != nil {
		return false, echo.NewHTTPError(http.StatusBadRequest, "credential is not healthy")
	}

	return true, nil
}
