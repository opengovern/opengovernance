package statemanager

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/db"
	"strings"
)

func (s *Service) createRoles(workspace *db.Workspace) error {
	roles := []string{
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

	for _, role := range roles {
		if err := s.createRole(workspace, role); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) createRole(workspace *db.Workspace, serviceName string) error {
	_, err := s.iam.CreateRole(context.Background(), &iam.CreateRoleInput{
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
}`, s.kaytuAWSAccountID, s.kaytuOIDCProvider, workspace.ID, serviceName)),
		RoleName: aws.String(fmt.Sprintf("kaytu-service-%s-%s", workspace.ID, serviceName)),
	})
	if strings.Contains(err.Error(), "EntityAlreadyExists") {
		return nil
	}
	return err
}
