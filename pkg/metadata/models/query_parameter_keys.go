package models

import (
	"github.com/kaytu-io/kaytu-engine/pkg/metadata/errors"
	"strings"
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
