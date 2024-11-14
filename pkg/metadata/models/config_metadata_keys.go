package models

import (
	"github.com/opengovern/og-util/pkg/api"
	"strings"

	"github.com/opengovern/opengovernance/pkg/metadata/errors"
)

type ConfigMetadataType string

const (
	ConfigMetadataTypeString ConfigMetadataType = "string"
	ConfigMetadataTypeInt    ConfigMetadataType = "int"
	ConfigMetadataTypeBool   ConfigMetadataType = "bool"
	ConfigMetadataTypeJSON   ConfigMetadataType = "json"
)

type MetadataKey string

const (
	MetadataKeyWorkspaceOwnership       MetadataKey = "workspace_ownership"
	MetadataKeyWorkspaceID              MetadataKey = "workspace_id"
	MetadataKeyWorkspaceName            MetadataKey = "workspace_name"
	MetadataKeyWorkspacePlan            MetadataKey = "workspace_plan"
	MetadataKeyWorkspaceCreationTime    MetadataKey = "workspace_creation_time"
	MetadataKeyWorkspaceDateTimeFormat  MetadataKey = "workspace_date_time_format"
	MetadataKeyWorkspaceDebugMode       MetadataKey = "workspace_debug_mode"
	MetadataKeyWorkspaceTimeWindow      MetadataKey = "workspace_time_window"
	MetadataKeyAssetManagementEnabled   MetadataKey = "asset_management_enabled"
	MetadataKeyComplianceEnabled        MetadataKey = "compliance_enabled"
	MetadataKeyProductManagementEnabled MetadataKey = "product_management_enabled"
	MetadataKeyCustomIDP                MetadataKey = "custom_idp"
	MetadataKeyResourceLimit            MetadataKey = "resource_limit"
	MetadataKeyConnectionLimit          MetadataKey = "connection_limit"
	MetadataKeyUserLimit                MetadataKey = "user_limit"
	MetadataKeyAllowInvite              MetadataKey = "allow_invite"
	MetadataKeyWorkspaceKeySupport      MetadataKey = "workspace_key_support"
	MetadataKeyWorkspaceMaxKeys         MetadataKey = "workspace_max_keys"
	MetadataKeyAllowedEmailDomains      MetadataKey = "allowed_email_domains"
	MetadataKeyAutoDiscoveryMethod      MetadataKey = "auto_discovery_method"
	// MetadataKeyDescribeJobInterval is the interval in minutes for describe job
	MetadataKeyDescribeJobInterval MetadataKey = "describe_job_interval"
	// MetadataKeyCostDiscoveryJobInterval is the interval in minutes for cost describe job
	MetadataKeyCostDiscoveryJobInterval MetadataKey = "cost_discovery_job_interval"
	// MetadataKeyHealthCheckJobInterval is the interval in minutes for health check job
	MetadataKeyHealthCheckJobInterval MetadataKey = "health_check_job_interval"
	// MetadataKeyMetricsJobInterval is the interval in minutes for metrics job
	MetadataKeyMetricsJobInterval    MetadataKey = "metrics_job_interval"
	MetadataKeyComplianceJobInterval MetadataKey = "compliance_job_interval"
	// MetadataKeyDataRetention retention period in days
	MetadataKeyDataRetention               MetadataKey = "data_retention_duration"
	MetadataKeyAnalyticsGitURL             MetadataKey = "analytics_git_url"
	DemoDataS3URL                          MetadataKey = "demo_data_s3_url"
	MetadataKeyAssetDiscoveryAWSPolicyARNs MetadataKey = "asset_discovery_aws_policy_arns"
	MetadataKeySpendDiscoveryAWSPolicyARNs MetadataKey = "spend_discovery_aws_policy_arns"
	MetadataKeyAssetDiscoveryAzureRoleIDs  MetadataKey = "asset_discovery_azure_role_ids"
	MetadataKeySpendDiscoveryAzureRoleIDs  MetadataKey = "spend_discovery_azure_role_ids"
	MetadataKeyCustomizationEnabled        MetadataKey = "customization_enabled"
	MetadataKeyAWSDiscoveryRequiredOnly    MetadataKey = "aws_discovery_required_only"
	MetadataKeyAzureDiscoveryRequiredOnly  MetadataKey = "azure_discovery_required_only"
	MetadataKeyAssetDiscoveryEnabled       MetadataKey = "asset_discovery_enabled"
	MetadataKeySpendDiscoveryEnabled       MetadataKey = "spend_discovery_enabled"
)

var MetadataKeys = []MetadataKey{
	MetadataKeyWorkspaceOwnership,
	MetadataKeyWorkspaceID,
	MetadataKeyWorkspaceName,
	MetadataKeyWorkspacePlan,
	MetadataKeyWorkspaceCreationTime,
	MetadataKeyWorkspaceDateTimeFormat,
	MetadataKeyWorkspaceDebugMode,
	MetadataKeyWorkspaceTimeWindow,
	MetadataKeyAssetManagementEnabled,
	MetadataKeyComplianceEnabled,
	MetadataKeyProductManagementEnabled,
	MetadataKeyCustomIDP,
	MetadataKeyResourceLimit,
	MetadataKeyConnectionLimit,
	MetadataKeyUserLimit,
	MetadataKeyAllowInvite,
	MetadataKeyWorkspaceKeySupport,
	MetadataKeyWorkspaceMaxKeys,
	MetadataKeyAllowedEmailDomains,
	MetadataKeyAutoDiscoveryMethod,
	MetadataKeyDescribeJobInterval,
	MetadataKeyCostDiscoveryJobInterval,
	MetadataKeyHealthCheckJobInterval,
	MetadataKeyMetricsJobInterval,
	MetadataKeyComplianceJobInterval,
	MetadataKeyDataRetention,
	MetadataKeyAnalyticsGitURL,
	MetadataKeyAssetDiscoveryAWSPolicyARNs,
	MetadataKeySpendDiscoveryAWSPolicyARNs,
	MetadataKeyAssetDiscoveryAzureRoleIDs,
	MetadataKeySpendDiscoveryAzureRoleIDs,
	MetadataKeyCustomizationEnabled,
	MetadataKeyAWSDiscoveryRequiredOnly,
	MetadataKeyAzureDiscoveryRequiredOnly,
	MetadataKeyAssetDiscoveryEnabled,
	MetadataKeySpendDiscoveryEnabled,
}

func (k MetadataKey) String() string {
	return string(k)
}

func (k MetadataKey) GetConfigMetadataType() ConfigMetadataType {
	switch k {
	case MetadataKeyWorkspaceOwnership:
		return ConfigMetadataTypeString
	case MetadataKeyWorkspaceID:
		return ConfigMetadataTypeString
	case MetadataKeyWorkspaceName:
		return ConfigMetadataTypeString
	case MetadataKeyWorkspacePlan:
		return ConfigMetadataTypeString
	case MetadataKeyWorkspaceCreationTime:
		return ConfigMetadataTypeInt
	case MetadataKeyWorkspaceDateTimeFormat:
		return ConfigMetadataTypeString
	case MetadataKeyWorkspaceDebugMode:
		return ConfigMetadataTypeBool
	case MetadataKeyWorkspaceTimeWindow:
		return ConfigMetadataTypeString
	case MetadataKeyAssetManagementEnabled:
		return ConfigMetadataTypeBool
	case MetadataKeyComplianceEnabled:
		return ConfigMetadataTypeBool
	case MetadataKeyProductManagementEnabled:
		return ConfigMetadataTypeBool
	case MetadataKeyCustomIDP:
		return ConfigMetadataTypeString
	case MetadataKeyResourceLimit:
		return ConfigMetadataTypeInt
	case MetadataKeyConnectionLimit:
		return ConfigMetadataTypeInt
	case MetadataKeyUserLimit:
		return ConfigMetadataTypeInt
	case MetadataKeyAllowInvite:
		return ConfigMetadataTypeBool
	case MetadataKeyWorkspaceKeySupport:
		return ConfigMetadataTypeBool
	case MetadataKeyWorkspaceMaxKeys:
		return ConfigMetadataTypeInt
	case MetadataKeyAllowedEmailDomains:
		return ConfigMetadataTypeJSON
	case MetadataKeyAutoDiscoveryMethod:
		return ConfigMetadataTypeString
	case MetadataKeyDescribeJobInterval:
		return ConfigMetadataTypeInt
	case MetadataKeyCostDiscoveryJobInterval:
		return ConfigMetadataTypeInt
	case MetadataKeyHealthCheckJobInterval:
		return ConfigMetadataTypeInt
	case MetadataKeyMetricsJobInterval:
		return ConfigMetadataTypeInt
	case MetadataKeyComplianceJobInterval:
		return ConfigMetadataTypeInt
	case MetadataKeyDataRetention:
		return ConfigMetadataTypeInt
	case MetadataKeyAnalyticsGitURL:
		return ConfigMetadataTypeString
	case MetadataKeyAssetDiscoveryAWSPolicyARNs:
		return ConfigMetadataTypeString
	case MetadataKeySpendDiscoveryAWSPolicyARNs:
		return ConfigMetadataTypeString
	case MetadataKeyAssetDiscoveryAzureRoleIDs:
		return ConfigMetadataTypeString
	case MetadataKeySpendDiscoveryAzureRoleIDs:
		return ConfigMetadataTypeString
	case MetadataKeyCustomizationEnabled:
		return ConfigMetadataTypeBool
	case MetadataKeyAWSDiscoveryRequiredOnly:
		return ConfigMetadataTypeBool
	case MetadataKeyAzureDiscoveryRequiredOnly:
		return ConfigMetadataTypeBool
	case MetadataKeyAssetDiscoveryEnabled:
		return ConfigMetadataTypeBool
	case MetadataKeySpendDiscoveryEnabled:
		return ConfigMetadataTypeBool
	}
	return ""
}

func (k MetadataKey) GetMinAuthRole() api.Role {
	switch k {
	case MetadataKeyWorkspaceOwnership:
		return api.AdminRole
	case MetadataKeyWorkspaceID:
		return api.AdminRole
	case MetadataKeyWorkspaceName:
		return api.AdminRole
	case MetadataKeyWorkspacePlan:
		return api.AdminRole
	case MetadataKeyWorkspaceCreationTime:
		return api.AdminRole
	case MetadataKeyWorkspaceDateTimeFormat:
		return api.AdminRole
	case MetadataKeyWorkspaceDebugMode:
		return api.AdminRole
	case MetadataKeyWorkspaceTimeWindow:
		return api.AdminRole
	case MetadataKeyAssetManagementEnabled:
		return api.AdminRole
	case MetadataKeyComplianceEnabled:
		return api.AdminRole
	case MetadataKeyProductManagementEnabled:
		return api.AdminRole
	case MetadataKeyCustomIDP:
		return api.AdminRole
	case MetadataKeyResourceLimit:
		return api.AdminRole
	case MetadataKeyConnectionLimit:
		return api.AdminRole
	case MetadataKeyUserLimit:
		return api.AdminRole
	case MetadataKeyAllowInvite:
		return api.AdminRole
	case MetadataKeyWorkspaceKeySupport:
		return api.AdminRole
	case MetadataKeyWorkspaceMaxKeys:
		return api.AdminRole
	case MetadataKeyAllowedEmailDomains:
		return api.AdminRole
	case MetadataKeyAutoDiscoveryMethod:
		return api.AdminRole
	case MetadataKeyDescribeJobInterval:
		return api.AdminRole
	case MetadataKeyCostDiscoveryJobInterval:
		return api.AdminRole
	case MetadataKeyHealthCheckJobInterval:
		return api.AdminRole
	case MetadataKeyMetricsJobInterval:
		return api.AdminRole
	case MetadataKeyComplianceJobInterval:
		return api.AdminRole
	case MetadataKeyDataRetention:
		return api.AdminRole
	case MetadataKeyAnalyticsGitURL:
		return api.AdminRole
	case MetadataKeyAssetDiscoveryAWSPolicyARNs:
		return api.AdminRole
	case MetadataKeySpendDiscoveryAWSPolicyARNs:
		return api.AdminRole
	case MetadataKeyAssetDiscoveryAzureRoleIDs:
		return api.AdminRole
	case MetadataKeySpendDiscoveryAzureRoleIDs:
		return api.AdminRole
	case MetadataKeyCustomizationEnabled:
		return api.AdminRole
	case MetadataKeyAWSDiscoveryRequiredOnly:
		return api.AdminRole
	case MetadataKeyAzureDiscoveryRequiredOnly:
		return api.AdminRole
	case MetadataKeyAssetDiscoveryEnabled:
		return api.AdminRole
	case MetadataKeySpendDiscoveryEnabled:
		return api.AdminRole
	}
	return ""
}

func ParseMetadataKey(key string) (MetadataKey, error) {
	lowerKey := strings.ToLower(key)
	for _, k := range MetadataKeys {
		if lowerKey == strings.ToLower(k.String()) {
			return k, nil
		}
	}
	return "", errors.ErrMetadataKeyNotFound
}
