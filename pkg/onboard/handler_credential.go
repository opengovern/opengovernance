package onboard

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsOrgTypes "github.com/aws/aws-sdk-go-v2/service/organizations/types"
	kaytuAws "github.com/kaytu-io/kaytu-aws-describer/aws"
	"github.com/kaytu-io/kaytu-aws-describer/aws/describer"
	kaytuAzure "github.com/kaytu-io/kaytu-azure-describer/azure"
	"github.com/kaytu-io/kaytu-engine/pkg/describe"
	"github.com/kaytu-io/kaytu-engine/pkg/onboard/api"
	apiv2 "github.com/kaytu-io/kaytu-engine/pkg/onboard/api/v2"
	"github.com/kaytu-io/kaytu-engine/pkg/onboard/db/model"
	"github.com/kaytu-io/kaytu-engine/pkg/utils"
	"github.com/kaytu-io/kaytu-util/pkg/source"
	"github.com/labstack/echo/v4"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"net/http"
	"time"
)

func generateRoleARN(accountID, roleName string) string {
	return fmt.Sprintf("arn:aws:iam::%s:role/%s", accountID, roleName)
}

func (h HttpHandler) GetAWSSDKConfig(roleARN string, externalID *string) (aws.Config, error) {
	awsConfig, err := kaytuAws.GetConfig(
		context.Background(),
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

func (h HttpHandler) GetOrgAccounts(sdkConfig aws.Config) (*awsOrgTypes.Organization, []awsOrgTypes.Account, error) {
	org, err := describer.OrganizationOrganization(context.Background(), sdkConfig)
	if err != nil {
		if !ignoreAwsOrgError(err) {
			return nil, nil, err
		}
	}

	accounts, err := describer.OrganizationAccounts(context.Background(), sdkConfig)
	if err != nil {
		if !ignoreAwsOrgError(err) {
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

func (h HttpHandler) createAWSCredential(req apiv2.CreateCredentialV2Request) (*apiv2.CreateCredentialV2Response, error) {
	awsConfig, err := h.GetAWSSDKConfig(generateRoleARN(req.AWSConfig.AccountID, req.AWSConfig.AssumeRoleName), req.AWSConfig.ExternalId)
	if err != nil {
		return nil, err
	}

	org, accounts, err := h.GetOrgAccounts(awsConfig)
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

	secretBytes, err := h.kms.Encrypt(req.AWSConfig.AsMap(), h.keyARN)
	if err != nil {
		return nil, err
	}
	cred.Secret = string(secretBytes)
	if err := h.db.CreateCredential(cred); err != nil {
		return nil, err
	}
	_, err = h.checkCredentialHealth(context.Background(), *cred)
	if err != nil {
		return nil, err
	}

	return &apiv2.CreateCredentialV2Response{ID: cred.ID.String()}, nil
}

func (h HttpHandler) autoOnboardAWSAccountsV2(ctx context.Context, credential model.Credential, maxConnections int64) ([]api.Connection, error) {
	onboardedSources := make([]api.Connection, 0)
	cnf, err := h.kms.Decrypt(credential.Secret, h.keyARN)
	if err != nil {
		return nil, err
	}

	awsCnf, err := apiv2.AWSCredentialV2ConfigFromMap(cnf)
	if err != nil {
		return nil, err
	}

	awsConfig, err := kaytuAws.GetConfig(
		context.Background(),
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
	org, err := describer.OrganizationOrganization(context.Background(), awsConfig)
	if err != nil {
		if !ignoreAwsOrgError(err) {
			return nil, err
		}
	}
	accounts, err := describer.OrganizationAccounts(context.Background(), awsConfig)
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

func (h HttpHandler) checkCredentialHealthV2(cred model.Credential) (healthy bool, err error) {
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

	config, err := h.kms.Decrypt(cred.Secret, h.keyARN)
	if err != nil {
		return false, err
	}

	switch cred.ConnectorType {
	case source.CloudAWS:
		awsConfig, err := apiv2.AWSCredentialV2ConfigFromMap(config)
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
	default:
		return false, errors.New("not implemented")
	}
	return true, nil
}

func (h HttpHandler) checkCredentialHealth(ctx context.Context, cred model.Credential) (bool, error) {
	if cred.Version == 2 {
		return h.checkCredentialHealthV2(cred)
	}

	config, err := h.kms.Decrypt(cred.Secret, h.keyARN)
	if err != nil {
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
		sdkCnf, err = kaytuAws.GetConfig(context.Background(), awsConfig.AccessKey, awsConfig.SecretKey, "", "", nil)
		if err != nil {
			return false, echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
		err = kaytuAws.CheckGetUserPermission(h.logger, sdkCnf)
		if err == nil {
			metadata, err := getAWSCredentialsMetadata(context.Background(), h.logger, awsConfig)
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
			metadata, err := getAzureCredentialsMetadata(context.Background(), azureConfig)
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
