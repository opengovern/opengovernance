package models

import (
	"strings"

	"github.com/opengovern/opencomply/services/metadata/errors"
)

type QueryParameterKey string

func (k QueryParameterKey) String() string {
	return string(k)
}

var QueryParameterKeys = []QueryParameterKey{
	"awsEksClusterOldestVersionSupported",
	"awsApiGatewayValidEndpointConfigurationTypes",
	"awsClassicLoadBalancerPredefinedPolicyName",
	"awsOpensearchLatestVersion",
	"awsOpensearchAllowedDataInstanceTypes",
	"awsOpensearchAllowedDedicatedMasterTypes",
	"awsAllowedInstanceTypes",
	"awsOpensearchClusterNodesLimit",
	"awsEc2NamingPattern",
	"awsTrustedEndpoints",
	"awsTrustedAccounts",
	"awsWebTierTags",
	"awsAppTierTags",
	"awsApprovedIPs",
	"awsSafelistedIPs",
	"azureAksLatestVersion",
	"awsLambdaFunctionAllowedRuntimes",
	"awsLambdaFunctionAllowedRoles",
	"awsLambdaFunctionAllowedTimeouts",
	"awsLambdaFunctionAllowedMemorySizes",
	"awsIamBlacklistedPolicies",
	"awsEc2InstanceValidInstanceTypes",
	"awsEbsSnapshotAgeMaxDays",
	"azureComputeSnapshotAgeMaxDays",
	"awsRdsBaselineRestorableTimeInHrs",
	"awsRdsBaselineRetentionPeriodDays",
	"awsEbsInstancesBackupPeriod",
}

func ParseQueryParameterKey(key string) (QueryParameterKey, error) {
	lowerKey := strings.ToLower(key)
	for _, k := range QueryParameterKeys {
		if lowerKey == strings.ToLower(k.String()) {
			return k, nil
		}
	}
	return "", errors.ErrQueryParameterKeyNotFound
}
