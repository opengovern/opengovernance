package onboard

import (
	"context"
	"encoding/json"
	"fmt"
	awsOrgTypes "github.com/aws/aws-sdk-go-v2/service/organizations/types"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	kaytuAws "github.com/kaytu-io/kaytu-aws-describer/aws"
	"github.com/kaytu-io/kaytu-aws-describer/aws/describer"
	"github.com/kaytu-io/kaytu-engine/pkg/describe"
	"github.com/kaytu-io/kaytu-engine/pkg/onboard/api"
	apiv2 "github.com/kaytu-io/kaytu-engine/pkg/onboard/api/v2"
	"github.com/kaytu-io/kaytu-engine/pkg/utils"
	"github.com/kaytu-io/kaytu-util/pkg/source"
	"github.com/labstack/echo/v4"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"net/http"
)

func generateRoleARN(accountID, roleName string) string {
	return fmt.Sprintf("arn:aws:iam::%s:role/%s", accountID, roleName)
}

func (h HttpHandler) createAWSCredential(req apiv2.CreateCredentialV2Request) (*apiv2.CreateCredentialV2Response, error) {
	reqAWSCnf, err := req.GetAWSConfig()
	if err != nil {
		return nil, err
	}

	awsConfig, err := kaytuAws.GetConfig(
		context.Background(),
		h.masterAccessKey,
		h.masterSecretKey,
		"",
		generateRoleARN(reqAWSCnf.AccountID, reqAWSCnf.AssumeRoleName),
		reqAWSCnf.ExternalId,
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

	secretBytes, err := h.kms.Encrypt(reqAWSCnf.AsMap(), h.keyARN)
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
	awsCnf, err := describe.AWSAccountConfigFromMap(cnf)
	if err != nil {
		return nil, err
	}
	cfg, err := kaytuAws.GetConfig(
		ctx,
		awsCnf.AccessKey,
		awsCnf.SecretKey,
		"",
		awsCnf.AssumeAdminRoleName,
		nil)
	h.logger.Info("discovering accounts", zap.String("credentialId", credential.ID.String()))
	if cfg.Region == "" {
		cfg.Region = "us-east-1"
	}
	accounts, err := discoverAWSAccounts(ctx, cfg)
	if err != nil {
		h.logger.Error("failed to discover accounts", zap.Error(err))
		return nil, err
	}
	h.logger.Info("discovered accounts", zap.Int("count", len(accounts)))
	// tracer :
	outputS, span := tracer.Start(ctx, "new_GetSourcesOfType", trace.WithSpanKind(trace.SpanKindServer))
	span.SetName("new_GetSourcesOfType")

	existingConnections, err := h.db.GetSourcesOfType(credential.ConnectorType)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	span.AddEvent("information", trace.WithAttributes(
		attribute.String("credential connector ", string(credential.ConnectorType)),
	))
	span.End()

	existingConnectionAccountIDs := make([]string, 0, len(existingConnections))
	for _, conn := range existingConnections {
		existingConnectionAccountIDs = append(existingConnectionAccountIDs, conn.SourceId)
	}
	accountsToOnboard := make([]awsAccount, 0)
	// tracer :
	outputS1, span1 := tracer.Start(outputS, "new_UpdateSource(loop)", trace.WithSpanKind(trace.SpanKindServer))
	span1.SetName("new_UpdateSource(loop)")

	for _, account := range accounts {
		if !utils.Includes(existingConnectionAccountIDs, account.AccountID) {
			accountsToOnboard = append(accountsToOnboard, account)
		} else {
			for _, conn := range existingConnections {
				if conn.LifecycleState == ConnectionLifecycleStateArchived {
					h.logger.Info("Archived Connection",
						zap.String("accountID", conn.SourceId))
				}
				if conn.SourceId == account.AccountID {
					name := account.AccountID
					if account.AccountName != nil {
						name = *account.AccountName
					}

					if conn.CredentialID.String() != credential.ID.String() {
						h.logger.Warn("organization account is onboarded as an standalone account",
							zap.String("accountID", account.AccountID),
							zap.String("connectionID", conn.ID.String()))
					}

					localConn := conn
					if conn.Name != name {
						localConn.Name = name
					}
					if account.Account.Status != awsOrgTypes.AccountStatusActive {
						localConn.LifecycleState = ConnectionLifecycleStateArchived
					} else if localConn.LifecycleState == ConnectionLifecycleStateArchived {
						localConn.LifecycleState = ConnectionLifecycleStateDiscovered
						if credential.AutoOnboardEnabled {
							localConn.LifecycleState = ConnectionLifecycleStateOnboard
						}
					}
					if conn.Name != name || account.Account.Status != awsOrgTypes.AccountStatusActive || conn.LifecycleState != localConn.LifecycleState {
						// tracer :
						_, span2 := tracer.Start(outputS1, "new_UpdateSource", trace.WithSpanKind(trace.SpanKindServer))
						span2.SetName("new_UpdateSource")

						_, err := h.db.UpdateSource(&localConn)
						if err != nil {
							span2.RecordError(err)
							span2.SetStatus(codes.Error, err.Error())
							h.logger.Error("failed to update source", zap.Error(err))
							return nil, err
						}
						span1.AddEvent("information", trace.WithAttributes(
							attribute.String("source name", localConn.Name),
						))
						span2.End()
					}
				}
			}
		}
	}
	span1.End()
	// TODO add tag filter
	// tracer :
	outputS3, span3 := tracer.Start(outputS1, "new_CountSources(loop)", trace.WithSpanKind(trace.SpanKindServer))
	span3.SetName("new_CountSources(loop)")

	h.logger.Info("onboarding accounts", zap.Int("count", len(accountsToOnboard)))
	for _, account := range accountsToOnboard {
		//assumeRoleArn := kaytuAws.GetRoleArnFromName(account.AccountID, awsCnf.AssumeRoleName)
		//sdkCnf, err := kaytuAws.GetConfig(ctx.Request().Context(), awsCnf.AccessKey, awsCnf.SecretKey, assumeRoleArn, assumeRoleArn, awsCnf.ExternalID)
		//if err != nil {
		//	h.logger.Warn("failed to get config", zap.Error(err))
		//	return err
		//}
		//isAttached, err := kaytuAws.CheckAttachedPolicy(h.logger, sdkCnf, awsCnf.AssumeRoleName, kaytuAws.SecurityAuditPolicyARN)
		//if err != nil {
		//	h.logger.Warn("failed to check get user permission", zap.Error(err))
		//	continue
		//}
		//if !isAttached {
		//	h.logger.Warn("security audit policy not attached", zap.String("accountID", account.AccountID))
		//	continue
		//}
		h.logger.Info("onboarding account", zap.String("accountID", account.AccountID))
		_, span4 := tracer.Start(outputS3, "new_CountSources", trace.WithSpanKind(trace.SpanKindServer))
		span4.SetName("new_CountSources")

		count, err := h.db.CountSources()
		if err != nil {
			span4.RecordError(err)
			span4.SetStatus(codes.Error, err.Error())
			return nil, err
		}
		span4.End()

		if count >= maxConnections {
			return nil, echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("maximum number of connections reached: [%d/%d]", count, maxConnections))
		}

		src := NewAWSAutoOnboardedConnection(
			h.logger,
			awsCnf,
			account,
			source.SourceCreationMethodAutoOnboard,
			fmt.Sprintf("Auto onboarded account %s", account.AccountID),
			credential,
		)
		// tracer :
		outputS5, span5 := tracer.Start(outputS3, "new_Transaction", trace.WithSpanKind(trace.SpanKindServer))
		span5.SetName("new_Transaction")

		err = h.db.orm.Transaction(func(tx *gorm.DB) error {
			_, span6 := tracer.Start(outputS5, "new_CreateSource", trace.WithSpanKind(trace.SpanKindServer))
			span6.SetName("new_CreateSource")

			err := h.db.CreateSource(&src)
			if err != nil {
				span6.RecordError(err)
				span6.SetStatus(codes.Error, err.Error())
				return err
			}
			span1.AddEvent("information", trace.WithAttributes(
				attribute.String("source name", src.Name),
			))
			span6.End()

			//TODO: add enable account

			return nil
		})
		if err != nil {
			span5.RecordError(err)
			span5.SetStatus(codes.Error, err.Error())
			return nil, err
		}
		span5.End()

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
	span3.End()

	return onboardedSources, nil
}
