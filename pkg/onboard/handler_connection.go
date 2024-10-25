package onboard

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	kaytuAws "github.com/opengovern/og-aws-describer/aws"
	kaytuAzure "github.com/opengovern/og-azure-describer/azure"
	"github.com/opengovern/og-util/pkg/api"
	"github.com/opengovern/og-util/pkg/httpclient"
	"github.com/opengovern/og-util/pkg/source"
	"github.com/opengovern/opengovernance/pkg/describe/connectors"
	"github.com/opengovern/opengovernance/pkg/metadata/models"
	apiv2 "github.com/opengovern/opengovernance/pkg/onboard/api/v2"
	"github.com/opengovern/opengovernance/pkg/utils"
	"github.com/opengovern/opengovernance/services/integration/model"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"strings"
)

func (h HttpHandler) checkConnectionHealth(ctx context.Context, connection model.Connection, updateMetadata bool) (model.Connection, error) {
	var cnf map[string]any
	cnf, err := h.vaultSc.Decrypt(ctx, connection.Credential.Secret)
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

			aKey := h.masterAccessKey
			sKey := h.masterSecretKey
			if awsCnf.AccessKey != nil {
				aKey = *awsCnf.AccessKey
			}
			if awsCnf.SecretKey != nil {
				sKey = *awsCnf.SecretKey
			}

			assumeRoleArn := kaytuAws.GetRoleArnFromName(connection.SourceId, awsCnf.AssumeRoleName)
			sdkCnf, err := kaytuAws.GetConfig(ctx, aKey, sKey, "", assumeRoleArn, awsCnf.ExternalId)
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
			spendAttached = connection.Credential.SpendDiscovery != nil && *connection.Credential.SpendDiscovery
		} else {
			var awsCnf connectors.AWSAccountConfig
			awsCnf, err = connectors.AWSAccountConfigFromMap(cnf)
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
			assetDiscoveryAttached = true
			spendAttached = connection.Credential.SpendDiscovery != nil && *connection.Credential.SpendDiscovery
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
		var azureCnf connectors.AzureSubscriptionConfig
		azureCnf, err = connectors.AzureSubscriptionConfigFromMap(cnf)
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

		azureAssetDiscovery, err := h.metadataClient.GetConfigMetadata(&httpclient.Context{UserRole: api.AdminRole}, models.MetadataKeyAssetDiscoveryAzureRoleIDs)
		if err != nil {
			return connection, err
		}

		assetDiscoveryAttached = true
		for _, rawRuleID := range strings.Split(azureAssetDiscovery.GetValue().(string), ",") {
			ruleID := fmt.Sprintf(rawRuleID, azureCnf.TenantID)
			isAttached, err := kaytuAzure.CheckRole(authCnf, connection.SourceId, ruleID)
			if err != nil {
				return connection, err
			}

			if !isAttached {
				h.logger.Error("assets rule is not there", zap.String("ruleID", ruleID))
				assetDiscoveryAttached = false
			}
		}

		azureSpendDiscovery, err := h.metadataClient.GetConfigMetadata(&httpclient.Context{UserRole: api.AdminRole}, models.MetadataKeySpendDiscoveryAzureRoleIDs)
		if err != nil {
			return connection, err
		}
		spendAttached = true
		for _, rawRule := range strings.Split(azureSpendDiscovery.GetValue().(string), ",") {
			ruleID := fmt.Sprintf(rawRule, azureCnf.SubscriptionID)
			isAttached, err := kaytuAzure.CheckRole(authCnf, connection.SourceId, ruleID)
			if err != nil {
				return connection, err
			}

			if !isAttached {
				h.logger.Error("spend rule is not there", zap.String("ruleID", ruleID))
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

	assetDiscoveryAttached = true
	spendAttached = connection.Credential.SpendDiscovery != nil && *connection.Credential.SpendDiscovery
	if !assetDiscoveryAttached && !spendAttached {
		var healthMessage string
		if err == nil {
			healthMessage = "failed to find read permission"
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
