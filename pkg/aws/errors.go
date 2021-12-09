package aws

import (
	"errors"
	"strings"

	"github.com/aws/smithy-go"
)

func IsUnsupportedOrInvalidError(resource, region string, err error) bool {
	var ae smithy.APIError
	if errors.As(err, &ae) {
		switch ae.ErrorCode() {
		case "InvalidAction":
			return true
		case "UnsupportedOperation":
			return true
		}
	}

	// The following resources types describe calls are not supported
	// in the corresponding regions. The error message is usually
	// not very clear about the error. For now just ignoring them.
	switch resource {
	case "AWS::Route53Resolver::ResolverDNSSECConfig",
		"AWS::Route53Resolver::ResolverQueryLoggingConfigAssociation",
		"AWS::Route53Resolver::ResolverQueryLoggingConfig":
		if isInRegion(region, "ap-northeast-3") {
			return true
		}
	case "AWS::RDS::DBProxy",
		"AWS::RDS::DBProxyTargetGroup",
		"AWS::RDS::DBProxyEndpoint",
		"AWS::Lambda::CodeSigningConfig",
		"AWS::S3::StorageLens":
		if isInRegion(region, "ap-northeast-3", "eu-north-1", "eu-west-3", "sa-east-1") {
			return true
		}
	case "AWS::WorkSpaces::ConnectionAlias",
		"AWS::WorkSpaces::Workspace":
		if isInRegion(region, "ap-northeast-3", "eu-north-1", "eu-west-3", "us-east-2", "us-west-1") {
			return true
		}
	}

	return false
}

func isInRegion(region string, regions ...string) bool {
	for _, r := range regions {
		if strings.EqualFold(region, r) {
			return true
		}
	}

	return false
}
