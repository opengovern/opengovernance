package models

import (
	"strings"

	"gitlab.com/keibiengine/keibi-engine/pkg/auth/api"
	"gitlab.com/keibiengine/keibi-engine/pkg/metadata/errors"
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
	// MetadataKeyHealthCheckJobInterval is the interval in minutes for health check job
	MetadataKeyHealthCheckJobInterval MetadataKey = "health_check_job_interval"
	// MetadataKeyInsightJobInterval is the interval in minutes for insight job
	MetadataKeyInsightJobInterval MetadataKey = "insight_job_interval"
	// MetadataKeyMetricsJobInterval is the interval in minutes for metrics job
	MetadataKeyMetricsJobInterval MetadataKey = "metrics_job_interval"
	// MetadataKeyDataRetention retention period in days
	MetadataKeyDataRetention MetadataKey = "data_retention_duration"
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
	MetadataKeyHealthCheckJobInterval,
	MetadataKeyInsightJobInterval,
	MetadataKeyMetricsJobInterval,
	MetadataKeyDataRetention,
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
	case MetadataKeyHealthCheckJobInterval:
		return ConfigMetadataTypeInt
	case MetadataKeyInsightJobInterval:
		return ConfigMetadataTypeInt
	case MetadataKeyMetricsJobInterval:
		return ConfigMetadataTypeInt
	case MetadataKeyDataRetention:
		return ConfigMetadataTypeInt
	}
	return ""
}

func (k MetadataKey) GetMinAuthRole() api.Role {
	switch k {
	case MetadataKeyWorkspaceOwnership:
		return api.KeibiAdminRole
	case MetadataKeyWorkspaceID:
		return api.KeibiAdminRole
	case MetadataKeyWorkspaceName:
		return api.AdminRole
	case MetadataKeyWorkspacePlan:
		return api.KeibiAdminRole
	case MetadataKeyWorkspaceCreationTime:
		return api.KeibiAdminRole
	case MetadataKeyWorkspaceDateTimeFormat:
		return api.AdminRole
	case MetadataKeyWorkspaceDebugMode:
		return api.AdminRole
	case MetadataKeyWorkspaceTimeWindow:
		return api.AdminRole
	case MetadataKeyAssetManagementEnabled:
		return api.KeibiAdminRole
	case MetadataKeyComplianceEnabled:
		return api.KeibiAdminRole
	case MetadataKeyProductManagementEnabled:
		return api.KeibiAdminRole
	case MetadataKeyCustomIDP:
		return api.KeibiAdminRole
	case MetadataKeyResourceLimit:
		return api.KeibiAdminRole
	case MetadataKeyConnectionLimit:
		return api.KeibiAdminRole
	case MetadataKeyUserLimit:
		return api.KeibiAdminRole
	case MetadataKeyAllowInvite:
		return api.KeibiAdminRole
	case MetadataKeyWorkspaceKeySupport:
		return api.KeibiAdminRole
	case MetadataKeyWorkspaceMaxKeys:
		return api.KeibiAdminRole
	case MetadataKeyAllowedEmailDomains:
		return api.KeibiAdminRole
	case MetadataKeyAutoDiscoveryMethod:
		return api.KeibiAdminRole
	case MetadataKeyDescribeJobInterval:
		return api.KeibiAdminRole
	case MetadataKeyHealthCheckJobInterval:
		return api.KeibiAdminRole
	case MetadataKeyInsightJobInterval:
		return api.KeibiAdminRole
	case MetadataKeyMetricsJobInterval:
		return api.KeibiAdminRole
	case MetadataKeyDataRetention:
		return api.KeibiAdminRole
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
