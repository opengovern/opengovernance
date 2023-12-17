package transactions

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/db"
	"strings"
)

var serviceNames = []string{
	"alerting",
	"analytics-worker",
	"checkup-worker",
	"compliance",
	"compliance-report-worker",
	"compliance-summarizer",
	"cost-estimator",
	"insight-worker",
	"inventory",
	"metadata",
	"migrator",
	"onboard",
	"reporter",
	"scheduler",
	"steampipe",
}

var rolePolicies = map[string][]string{
	"scheduler": {"arn:aws:iam::${accountID}:policy/lambda-invoke-policy"},
}

type CreateServiceAccountRoles struct {
	iam               *iam.Client
	kaytuAWSAccountID string
	kaytuOIDCProvider string
}

func NewCreateServiceAccountRoles(
	iam *iam.Client,
	kaytuAWSAccountID string,
	kaytuOIDCProvider string,
) *CreateServiceAccountRoles {
	return &CreateServiceAccountRoles{
		iam:               iam,
		kaytuAWSAccountID: kaytuAWSAccountID,
		kaytuOIDCProvider: kaytuOIDCProvider,
	}
}

func (t *CreateServiceAccountRoles) Requirements() []TransactionID {
	return nil
}

func (t *CreateServiceAccountRoles) Apply(workspace db.Workspace) error {
	for _, serviceName := range serviceNames {
		if err := t.createRole(workspace, serviceName); err != nil {
			return err
		}
	}
	return nil
}

func (t *CreateServiceAccountRoles) Rollback(workspace db.Workspace) error {
	for _, serviceName := range serviceNames {
		_, err := t.iam.DeleteRole(context.Background(), &iam.DeleteRoleInput{
			RoleName: aws.String(fmt.Sprintf("kaytu-service-%s-%s", workspace.ID, serviceName)),
		})
		if err != nil {
			if !strings.Contains(err.Error(), "NoSuchEntity") {
				return err
			}
		}
	}
	return nil
}

func (t *CreateServiceAccountRoles) createRole(workspace db.Workspace, serviceName string) error {
	roleName := aws.String(fmt.Sprintf("kaytu-service-%s-%s", workspace.ID, serviceName))
	_, err := t.iam.CreateRole(context.Background(), &iam.CreateRoleInput{
		AssumeRolePolicyDocument: aws.String(fmt.Sprintf(`{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Principal": {
                "Federated": "arn:aws:iam::%[1]s:oidc-provider/%[2]s"
            },
            "Action": "sts:AssumeRoleWithWebIdentity",
            "Condition": {
                "StringLike": {
                    "%[2]s:sub": "system:serviceaccount:%[3]s:%[4]s",
                    "%[2]s:aud": "sts.amazonaws.com"
                }
            }
        }
    ]
}`, t.kaytuAWSAccountID, t.kaytuOIDCProvider, workspace.ID, serviceName)),
		RoleName: roleName,
	})
	if err != nil {
		if !strings.Contains(err.Error(), "EntityAlreadyExists") {
			return err
		}
	}

	if v, ok := rolePolicies[serviceName]; ok && len(v) > 0 {
		for _, policyARN := range v {
			policyARN = strings.ReplaceAll(policyARN, "${accountID}", t.kaytuAWSAccountID)

			_, err = t.iam.AttachRolePolicy(context.Background(), &iam.AttachRolePolicyInput{
				PolicyArn: aws.String(policyARN),
				RoleName:  roleName,
			})
			if err != nil {
				return err
			}
		}
	}
	return nil
}
