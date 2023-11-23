package onboard

import (
	"context"
	"encoding/json"
	"fmt"
	awsOrgTypes "github.com/aws/aws-sdk-go-v2/service/organizations/types"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	kaytuAws "github.com/kaytu-io/kaytu-aws-describer/aws"
	"github.com/kaytu-io/kaytu-aws-describer/aws/describer"
	"github.com/kaytu-io/kaytu-engine/pkg/onboard/api"
	apiv2 "github.com/kaytu-io/kaytu-engine/pkg/onboard/api/v2"
	"github.com/kaytu-io/kaytu-engine/pkg/utils"
	"github.com/kaytu-io/kaytu-util/pkg/source"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
	"net/http"
)

func generateRoleARN(accountID, roleName string) string {
	return fmt.Sprintf("arn:aws:iam::%s:role/%s", accountID, roleName)
}

func (h HttpHandler) createAWSCredential(req apiv2.CreateCredentialV2Request) (*apiv2.CreateCredentialV2Response, error) {
	awsConfig, err := kaytuAws.GetConfig(
		context.Background(),
		h.masterAccessKey,
		h.masterSecretKey,
		"",
		generateRoleARN(req.AWSConfig.AccountID, req.AWSConfig.AssumeRoleName),
		req.AWSConfig.ExternalId,
	)
	if err != nil {
		return nil, err
	}
	if awsConfig.Region == "" {
		awsConfig.Region = "us-east-1"
	}

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

	awsAccounts := make([]awsAccount, 0)
	for _, account := range accounts {
		if account.Id == nil {
			continue
		}
		localAccount := account
		awsAccounts = append(awsAccounts, awsAccount{
			AccountID:    *localAccount.Id,
			AccountName:  localAccount.Name,
			Organization: org,
			Account:      &localAccount,
		})
	}

	stsClient := sts.NewFromConfig(awsConfig)
	caller, err := stsClient.GetCallerIdentity(context.Background(), &sts.GetCallerIdentityInput{})
	if err != nil {
		return nil, err
	}
	metadata := AWSCredentialMetadata{
		AccountID: *caller.Account,
	}
	if org != nil {
		metadata.OrganizationID = org.Id
		metadata.OrganizationMasterAccountEmail = org.MasterAccountEmail
		metadata.OrganizationMasterAccountId = org.MasterAccountId
		metadata.OrganizationDiscoveredAccountCount = utils.GetPointer(len(accounts))
	}

	name := metadata.AccountID
	if metadata.OrganizationID != nil {
		name = *metadata.OrganizationID
	}

	cred, err := NewAWSCredential(name, &metadata, CredentialTypeManualAwsOrganization, 2)
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

	return &apiv2.CreateCredentialV2Response{ID: cred.ID.String()}, nil
}

func (h HttpHandler) autoOnboardAWSAccountsV2(ctx context.Context, credential Credential, maxConnections int64) ([]api.Connection, error) {
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
						localConn.LifecycleState = ConnectionLifecycleStateArchived
					} else if localConn.LifecycleState == ConnectionLifecycleStateArchived {
						localConn.LifecycleState = ConnectionLifecycleStateDiscovered
						if credential.AutoOnboardEnabled {
							localConn.LifecycleState = ConnectionLifecycleStateOnboard
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
