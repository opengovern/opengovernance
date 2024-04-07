package resources

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"strings"
)

type iamPolicy struct {
	iam       *iam.Client
	accountID string

	policyName     string
	policyDocument string
}

func IAMPolicy(iam *iam.Client, accountID string) *iamPolicy {
	return &iamPolicy{
		iam:       iam,
		accountID: accountID,
	}
}

func (policy *iamPolicy) WithName(name string) *iamPolicy {
	policy.policyName = name
	return policy
}

func (policy *iamPolicy) WithPolicyDocument(policyDocument string) *iamPolicy {
	policy.policyDocument = policyDocument
	return policy
}

func (policy *iamPolicy) ARN() string {
	return fmt.Sprintf("arn:aws:iam::%s:policy/%s", policy.accountID, policy.policyName)
}

func (policy *iamPolicy) CreateIdempotent(ctx context.Context) error {
	_, err := policy.iam.CreatePolicy(ctx, &iam.CreatePolicyInput{
		PolicyDocument: aws.String(policy.policyDocument),
		PolicyName:     aws.String(policy.policyName),
		Description:    nil,
		Path:           nil,
		Tags:           nil,
	})
	if err != nil {
		if !strings.Contains(err.Error(), "EntityAlreadyExists") {
			return err
		}
	}
	return nil
}

func (policy *iamPolicy) DeleteIdempotent(ctx context.Context) error {
	_, err := policy.iam.DeletePolicy(ctx, &iam.DeletePolicyInput{
		PolicyArn: aws.String(policy.ARN()),
	})
	if err != nil {
		if !strings.Contains(err.Error(), "NotFound") {
			return err
		}
	}
	return nil
}
