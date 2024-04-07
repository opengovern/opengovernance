package onboard

import (
	"context"
	"encoding/json"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	kaytuAws "github.com/kaytu-io/kaytu-aws-describer/aws"
	kaytuAzure "github.com/kaytu-io/kaytu-azure-describer/azure"
	"github.com/kaytu-io/kaytu-engine/pkg/auth/api"
	"github.com/kaytu-io/kaytu-engine/pkg/describe"
	"github.com/kaytu-io/kaytu-engine/pkg/httpclient"
	"github.com/kaytu-io/kaytu-engine/pkg/metadata/models"
	apiv2 "github.com/kaytu-io/kaytu-engine/pkg/onboard/api/v2"
	"github.com/kaytu-io/kaytu-engine/pkg/utils"
	"github.com/kaytu-io/kaytu-engine/services/integration/model"
	"github.com/kaytu-io/kaytu-util/pkg/source"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"strings"
)

func (h HttpHandler) checkConnectionHealth(ctx context.Context, connection model.Connection, updateMetadata bool) (model.Connection, error) {
	var cnf map[string]any
	cnf, err := h.kms.Decrypt(ctx, connection.Credential.Secret, connection.Credential.CredentialStoreKeyID, connection.Credential.CredentialStoreKeyVersion)
	if err != nil {
		h.logger.Error("failed to decrypt credential", zap.Error(err), zap.String("sourceId", connection.SourceId))
		return connection, err
	}

	var assetDiscoveryAttached, spendAttached bool
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

			assetDiscoveryAttached = true
			awsAssetDiscovery, err := h.metadataClient.GetConfigMetadata(&httpclient.Context{UserRole: api.InternalRole}, models.MetadataKeyAssetDiscoveryAWSPolicyARNs)
			if err != nil {
				return connection, err
			}

			for _, policyARN := range strings.Split(awsAssetDiscovery.GetValue().(string), ",") {
				policyARN = strings.ReplaceAll(policyARN, "${accountID}", connection.SourceId)
				if !utils.Includes(policyARNs, policyARN) {
					h.logger.Error("policy is not there", zap.String("policyARN", policyARN), zap.Strings("attachedPolicies", policyARNs))
					assetDiscoveryAttached = false
				}
			}

			spendAttached = connection.Credential.SpendDiscovery != nil && *connection.Credential.SpendDiscovery

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
				assetDiscoveryAttached, err = kaytuAws.CheckAttachedPolicy(h.logger, sdkCnf, awsCnf.AssumeRoleName, kaytuAws.GetPolicyArnFromName(connection.SourceId, awsCnf.AssumeRolePolicyName))
			} else {
				assetDiscoveryAttached, err = kaytuAws.CheckAttachedPolicy(h.logger, sdkCnf, "", kaytuAws.GetPolicyArnFromName(connection.SourceId, awsCnf.AssumeRolePolicyName))
			}
			spendAttached = assetDiscoveryAttached // backward compatibility
			if err == nil && assetDiscoveryAttached && updateMetadata {
				if sdkCnf.Region == "" {
					sdkCnf.Region = "us-east-1"
				}
				var awsAccount *awsAccount
				awsAccount, err = currentAwsAccount(ctx, h.logger, sdkCnf)
				if err != nil {
					h.logger.Error("failed to get current aws account", zap.Error(err), zap.String("sourceId", connection.SourceId))
					return connection, err
				}
				metadata, err2 := NewAWSConnectionMetadata(ctx, h.logger, awsCnf, connection, *awsAccount)
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

		azureAssetDiscovery, err := h.metadataClient.GetConfigMetadata(&httpclient.Context{UserRole: api.InternalRole}, models.MetadataKeyAssetDiscoveryAzureRoleIDs)
		if err != nil {
			return connection, err
		}

		assetDiscoveryAttached = true
		for _, ruleID := range strings.Split(azureAssetDiscovery.GetValue().(string), ",") {
			isAttached, err := kaytuAzure.CheckRole(authCnf, connection.SourceId, ruleID)
			if err != nil {
				return connection, err
			}

			if !isAttached {
				h.logger.Error("rule is not there", zap.String("ruleID", ruleID))
				assetDiscoveryAttached = false
			}
		}

		azureSpendDiscovery, err := h.metadataClient.GetConfigMetadata(&httpclient.Context{UserRole: api.InternalRole}, models.MetadataKeySpendDiscoveryAzureRoleIDs)
		if err != nil {
			return connection, err
		}
		spendAttached = true
		for _, ruleID := range strings.Split(azureSpendDiscovery.GetValue().(string), ",") {
			isAttached, err := kaytuAzure.CheckRole(authCnf, connection.SourceId, ruleID)
			if err != nil {
				return connection, err
			}

			if !isAttached {
				h.logger.Error("rule is not there", zap.String("ruleID", ruleID))
				spendAttached = false
			}
		}

		if (assetDiscoveryAttached || spendAttached) && updateMetadata {
			var azSub *azureSubscription
			azSub, err = currentAzureSubscription(ctx, h.logger, connection.SourceId, authCnf)
			if err != nil {
				h.logger.Error("failed to get current azure subscription", zap.Error(err), zap.String("sourceId", connection.SourceId))
				return connection, err
			}
			metadata := NewAzureConnectionMetadata(*azSub, azureCnf.TenantID)
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

	if !assetDiscoveryAttached && !spendAttached {
		var healthMessage string
		if err == nil {
			healthMessage = "Failed to find read permission"
		} else {
			healthMessage = err.Error()
		}
		connection, err = h.updateConnectionHealth(outputS, connection, source.HealthStatusUnhealthy, &healthMessage, aws.Bool(false), aws.Bool(false))
		if err != nil {
			h.logger.Warn("failed to update source health", zap.Error(err), zap.String("sourceId", connection.SourceId))
			return connection, err
		}
	} else {
		connection, err = h.updateConnectionHealth(outputS, connection, source.HealthStatusHealthy, utils.GetPointer(""), &spendAttached, &assetDiscoveryAttached)
		if err != nil {
			h.logger.Warn("failed to update source health", zap.Error(err), zap.String("sourceId", connection.SourceId))
			return connection, err
		}
	}
	span.End()
	return connection, nil
}
