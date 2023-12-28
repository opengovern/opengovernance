package resources

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"strings"
)

type iamRole struct {
	iam       *iam.Client
	accountID string

	roleName       string
	policyDocument string
	attachedPolicy []string
}

func IAMRole(iam *iam.Client, accountID string) *iamRole {
	return &iamRole{
		iam:       iam,
		accountID: accountID,
	}
}

func (role *iamRole) WithName(name string) *iamRole {
	role.roleName = name
	return role
}

func (role *iamRole) WithTrustRelationship(policyDocument string) *iamRole {
	role.policyDocument = policyDocument
	return role
}

func (role *iamRole) WithPolicy(policyARN string) *iamRole {
	role.attachedPolicy = append(role.attachedPolicy, policyARN)
	return role
}

func (role *iamRole) ARN() string {
	return fmt.Sprintf("arn:aws:iam::%s:role/%s", role.accountID, role.roleName)
}

func (role *iamRole) CreateIdempotent() error {
	_, err := role.iam.CreateRole(context.Background(), &iam.CreateRoleInput{
		AssumeRolePolicyDocument: aws.String(role.policyDocument),
		RoleName:                 aws.String(role.roleName),
		Description:              nil,
		MaxSessionDuration:       nil,
		Path:                     nil,
		PermissionsBoundary:      nil,
		Tags:                     nil,
	})
	if err != nil {
		if !strings.Contains(err.Error(), "EntityAlreadyExists") {
			return err
		}
	}

	for _, policyARN := range role.attachedPolicy {
		_, err = role.iam.AttachRolePolicy(context.Background(), &iam.AttachRolePolicyInput{
			PolicyArn: aws.String(policyARN),
			RoleName:  aws.String(role.roleName),
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func (role *iamRole) DeleteIdempotent() error {
	for _, policyARN := range role.attachedPolicy {
		_, err := role.iam.DetachRolePolicy(context.Background(), &iam.DetachRolePolicyInput{
			PolicyArn: aws.String(policyARN),
			RoleName:  aws.String(role.roleName),
		})
		if err != nil {
			if !strings.Contains(err.Error(), "NoSuchEntity") {
				return err
			}
		}
	}

	_, err := role.iam.DeleteRole(context.Background(), &iam.DeleteRoleInput{
		RoleName: aws.String(role.roleName),
	})
	if err != nil {
		if !strings.Contains(err.Error(), "NoSuchEntity") {
			return err
		}
	}
	return nil
}
