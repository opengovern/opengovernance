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
			return err
		}
	}
	return nil
}

func (t *CreateServiceAccountRoles) createRole(workspace db.Workspace, serviceName string) error {
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
		RoleName: aws.String(fmt.Sprintf("kaytu-service-%s-%s", workspace.ID, serviceName)),
	})
	if strings.Contains(err.Error(), "EntityAlreadyExists") {
		return nil
	}
	return err
}
