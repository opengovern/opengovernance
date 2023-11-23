package onboard

import (
	"context"
	"encoding/json"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	kaytuAws "github.com/kaytu-io/kaytu-aws-describer/aws"
	kaytuAzure "github.com/kaytu-io/kaytu-azure-describer/azure"
	"github.com/kaytu-io/kaytu-engine/pkg/describe"
	apiv2 "github.com/kaytu-io/kaytu-engine/pkg/onboard/api/v2"
	"github.com/kaytu-io/kaytu-engine/pkg/utils"
	"github.com/kaytu-io/kaytu-util/pkg/source"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

func (h HttpHandler) checkConnectionHealth(ctx context.Context, connection Source, updateMetadata bool) (Source, error) {
	var cnf map[string]any
	cnf, err := h.kms.Decrypt(connection.Credential.Secret, h.keyARN)
	if err != nil {
		h.logger.Error("failed to decrypt credential", zap.Error(err), zap.String("sourceId", connection.SourceId))
		return connection, err
	}

	var isAttached bool
	switch connection.Type {
	case source.CloudAWS:
		if connection.Credential.Version == 2 {
			awsCnf, err := apiv2.AWSCredentialV2ConfigFromMap(cnf)
			if err != nil {
				h.logger.Error("failed to get aws config", zap.Error(err), zap.String("sourceId", connection.SourceId))
				return connection, err
			}
			assumeRoleArn := kaytuAws.GetRoleArnFromName(connection.SourceId, awsCnf.AssumeRoleName)
			sdkCnf, err := kaytuAws.GetConfig(ctx, h.masterAccessKey, h.masterSecretKey, "", assumeRoleArn, awsCnf.ExternalId)
			if err != nil {
				h.logger.Error("failed to get aws config", zap.Error(err), zap.String("sourceId", connection.SourceId))
				return connection, err
			}

			iamClient := iam.NewFromConfig(sdkCnf)
			paginator := iam.NewListAttachedRolePoliciesPaginator(iamClient, &iam.ListAttachedRolePoliciesInput{
				RoleName: &awsCnf.AssumeRoleName,
			})
			var policyNames []string
			for paginator.HasMorePages() {
				page, err := paginator.NextPage(ctx)
				if err != nil {
					return connection, err
				}
				for _, policy := range page.AttachedPolicies {
					policyNames = append(policyNames, *policy.PolicyName)
				}
			}

			isAttached = true
			if len(awsCnf.HealthCheckPolicies) == 0 {
				awsCnf.HealthCheckPolicies = []string{
					"AWSAccountActivityAccess",
					"AWSBillingReadOnlyAccess",
					"AWSBudgetsReadOnlyAccess",
					"AWSOrganizationsReadOnlyAccess",
					"KaytuAdditionalResourceReadOnly",
					"ReadOnlyAccess",
					"SecurityAudit",
				}
			}
			for _, policyName := range awsCnf.HealthCheckPolicies {
				if !utils.Includes(policyNames, policyName) {
					h.logger.Error("policy is not there", zap.String("policyName", policyName), zap.Strings("attachedPolicies", policyNames))
					isAttached = false
				}
			}

			//TODO

		} else {
			var awsCnf describe.AWSAccountConfig
			awsCnf, err = describe.AWSAccountConfigFromMap(cnf)
			if err != nil {
				h.logger.Error("failed to get aws config", zap.Error(err), zap.String("sourceId", connection.SourceId))
				return connection, err
			}
			assumeRoleArn := kaytuAws.GetRoleArnFromName(connection.SourceId, awsCnf.AssumeRoleName)
			var sdkCnf aws.Config
			if awsCnf.AccountID != connection.SourceId {
				sdkCnf, err = kaytuAws.GetConfig(ctx, awsCnf.AccessKey, awsCnf.SecretKey, "", assumeRoleArn, awsCnf.ExternalID)
			} else {
				sdkCnf, err = kaytuAws.GetConfig(ctx, awsCnf.AccessKey, awsCnf.SecretKey, "", "", nil)
			}
			if err != nil {
				h.logger.Error("failed to get aws config", zap.Error(err), zap.String("sourceId", connection.SourceId))
				return connection, err
			}
			if awsCnf.AccountID != connection.SourceId {
				isAttached, err = kaytuAws.CheckAttachedPolicy(h.logger, sdkCnf, awsCnf.AssumeRoleName, kaytuAws.GetPolicyArnFromName(connection.SourceId, awsCnf.AssumeRolePolicyName))
			} else {
				isAttached, err = kaytuAws.CheckAttachedPolicy(h.logger, sdkCnf, "", kaytuAws.GetPolicyArnFromName(connection.SourceId, awsCnf.AssumeRolePolicyName))
			}
			if err == nil && isAttached && updateMetadata {
				if sdkCnf.Region == "" {
					sdkCnf.Region = "us-east-1"
				}
				var awsAccount *awsAccount
				awsAccount, err = currentAwsAccount(ctx, h.logger, sdkCnf)
				if err != nil {
					h.logger.Error("failed to get current aws account", zap.Error(err), zap.String("sourceId", connection.SourceId))
					return connection, err
				}
				metadata, err2 := NewAWSConnectionMetadata(h.logger, awsCnf, connection, *awsAccount)
				if err2 != nil {
					h.logger.Error("failed to get aws connection metadata", zap.Error(err2), zap.String("sourceId", connection.SourceId))
				}
				jsonMetadata, err2 := json.Marshal(metadata)
				if err2 != nil {
					return connection, err
				}
				connection.Metadata = jsonMetadata
			}
		}
	case source.CloudAzure:
		var azureCnf describe.AzureSubscriptionConfig
		azureCnf, err = describe.AzureSubscriptionConfigFromMap(cnf)
		if err != nil {
			h.logger.Error("failed to get azure config", zap.Error(err), zap.String("sourceId", connection.SourceId))
			return connection, err
		}
		authCnf := kaytuAzure.AuthConfig{
			TenantID:            azureCnf.TenantID,
			ClientID:            azureCnf.ClientID,
			ObjectID:            azureCnf.ObjectID,
			SecretID:            azureCnf.SecretID,
			ClientSecret:        azureCnf.ClientSecret,
			CertificatePath:     azureCnf.CertificatePath,
			CertificatePassword: azureCnf.CertificatePass,
			Username:            azureCnf.Username,
			Password:            azureCnf.Password,
		}
		isAttached, err = kaytuAzure.CheckRole(authCnf, connection.SourceId, kaytuAzure.DefaultReaderRoleDefinitionIDTemplate)

		if err == nil && isAttached && updateMetadata {
			var azSub *azureSubscription
			azSub, err = currentAzureSubscription(ctx, h.logger, connection.SourceId, authCnf)
			if err != nil {
				h.logger.Error("failed to get current azure subscription", zap.Error(err), zap.String("sourceId", connection.SourceId))
				return connection, err
			}
			metadata := NewAzureConnectionMetadata(*azSub)
			var jsonMetadata []byte
			jsonMetadata, err = json.Marshal(metadata)
			if err != nil {
				h.logger.Error("failed to marshal azure metadata", zap.Error(err), zap.String("sourceId", connection.SourceId))
				return connection, err
			}
			connection.Metadata = jsonMetadata
		}
	}
	if err != nil {
		h.logger.Warn("failed to check read permission", zap.Error(err), zap.String("sourceId", connection.SourceId))
	}
	// tracer :
	outputS, span := tracer.Start(ctx, "new_CreateSource(loop)", trace.WithSpanKind(trace.SpanKindServer))
	span.SetName("new_CreateSource(loop)")

	if !isAttached {
		var healthMessage string
		if err == nil {
			healthMessage = "Failed to find read permission"
		} else {
			healthMessage = err.Error()
		}
		connection, err = h.updateConnectionHealth(outputS, connection, source.HealthStatusUnhealthy, &healthMessage)
		if err != nil {
			h.logger.Warn("failed to update source health", zap.Error(err), zap.String("sourceId", connection.SourceId))
			return connection, err
		}
	} else {
		connection, err = h.updateConnectionHealth(outputS, connection, source.HealthStatusHealthy, utils.GetPointer(""))
		if err != nil {
			h.logger.Warn("failed to update source health", zap.Error(err), zap.String("sourceId", connection.SourceId))
			return connection, err
		}
	}
	span.End()
	return connection, nil
}
