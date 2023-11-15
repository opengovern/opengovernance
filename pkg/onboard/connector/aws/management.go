package aws

import (
	"context"
	_ "embed"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	kaytuAws "github.com/kaytu-io/kaytu-aws-describer/aws"
)

//go:embed management_policy.json
var managementPolicyStr string

func CreateManagement(workspaceID string) error {
	userName := fmt.Sprintf("jump-%s", workspaceID)
	policyName := fmt.Sprintf("policy-jump-%s", workspaceID)

	var cfg aws.Config
	cfg, err := kaytuAws.GetConfig(context.Background(), "", "", "", "", nil)
	if err != nil {
		return err
	}

	ctx := context.Background()
	iamClient := iam.NewFromConfig(cfg)

	user, err := iamClient.CreateUser(ctx, &iam.CreateUserInput{UserName: aws.String(userName)})
	if err != nil {
		return err
	}
	_ = user

	policy, err := iamClient.CreatePolicy(ctx, &iam.CreatePolicyInput{
		PolicyDocument: aws.String(managementPolicyStr),
		PolicyName:     aws.String(policyName),
	})
	if err != nil {
		return err
	}
	_ = policy

	role, err := iamClient.CreateRole(ctx, &iam.CreateRoleInput{
		AssumeRolePolicyDocument: nil,
		RoleName:                 nil,
		Description:              nil,
		MaxSessionDuration:       nil,
		Path:                     nil,
		PermissionsBoundary:      nil,
		Tags:                     nil,
	})
	if err != nil {
		return err
	}
	_ = role

	return nil
}
