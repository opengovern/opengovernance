package transactions

import (
	"context"
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/opensearch"
	types3 "github.com/aws/aws-sdk-go-v2/service/opensearch/types"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/api"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/config"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/db"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/transactions/resources"
)

type CreateOpenSearch struct {
	conf          config.Config
	vmType        types3.OpenSearchPartitionInstanceType
	instanceCount int32
	db            *db.Database

	iam        *iam.Client
	opensearch *opensearch.Client
}

func NewCreateOpenSearch(
	config config.Config,
	vmType types3.OpenSearchPartitionInstanceType,
	instanceCount int32,
	db *db.Database,
	iam *iam.Client,
	opensearch *opensearch.Client,
) *CreateOpenSearch {
	return &CreateOpenSearch{
		conf:          config,
		vmType:        vmType,
		instanceCount: instanceCount,
		db:            db,
		iam:           iam,
		opensearch:    opensearch,
	}
}

func (t *CreateOpenSearch) Requirements() []api.TransactionID {
	return []api.TransactionID{api.Transaction_CreateServiceAccountRoles}
}

func (t *CreateOpenSearch) Resources(workspace db.Workspace) (rs []resources.Resource) {
	bucketName := fmt.Sprintf("dlq-%s", workspace.ID)
	iamPolicy := resources.IAMPolicy(t.iam, t.conf.AWSAccountID).
		WithName(fmt.Sprintf("kaytu-dlq-policy-%s", workspace.ID)).
		WithPolicyDocument(fmt.Sprintf(`{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "WriteToS3DLQ",
      "Effect": "Allow",
      "Action": "s3:PutObject",
      "Resource": "arn:aws:s3:::%s/*"
    }
  ]
}`, bucketName))
	rs = append(rs, iamPolicy)

	iamRole := resources.IAMRole(t.iam, t.conf.AWSAccountID).
		WithName(fmt.Sprintf("kaytu-opensearch-master-%s", workspace.ID)).
		WithTrustRelationship(fmt.Sprintf(`{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
			"Principal": {
				"AWS": [
					"arn:aws:iam::%[2]s:role/kaytu-service-%[1]s-migrator",
					"arn:aws:iam::%[2]s:role/kaytu-service-%[1]s-compliance",
					"arn:aws:iam::%[2]s:role/kaytu-service-%[1]s-compliance-report-worker",
					"arn:aws:iam::%[2]s:role/kaytu-service-%[1]s-compliance-summarizer",
					"arn:aws:iam::%[2]s:role/kaytu-service-%[1]s-cost-estimator",
					"arn:aws:iam::%[2]s:role/kaytu-service-%[1]s-insight-worker",
					"arn:aws:iam::%[2]s:role/kaytu-service-%[1]s-inventory",
					"arn:aws:iam::%[2]s:role/kaytu-service-%[1]s-reporter",
					"arn:aws:iam::%[2]s:role/kaytu-service-%[1]s-scheduler",
					"arn:aws:iam::%[2]s:role/kaytu-service-%[1]s-steampipe",
					"arn:aws:iam::%[2]s:role/kaytu-service-%[1]s-analytics-worker",
                    "arn:aws:iam::%[2]s:role/teleport-agent-access",
                    "arn:aws:iam::%[2]s:role/teleport-db-access"
				]
			},
            "Action": [
				"sts:AssumeRole",
				"sts:TagSession"
			]
        },
        {
            "Effect": "Allow",
            "Principal": {
                "Service": "osis-pipelines.amazonaws.com"
            },
            "Action": "sts:AssumeRole"
        }
    ]
}`, workspace.ID, t.conf.AWSAccountID)).
		WithPolicy(iamPolicy.ARN()).
		WithPolicy("arn:aws:iam::aws:policy/AmazonOpenSearchServiceFullAccess")
	rs = append(rs, iamRole)

	opensearchDomain := resources.OpenSearchDomain(t.opensearch, t.db).
		WithName(workspace.ID).
		WithMasterRoleARN(iamRole.ARN()).
		WithInstanceType(t.vmType).
		WithInstanceCount(t.instanceCount).
		WithSubnetID(t.conf.SubnetID).
		WithSecurityGroupID(t.conf.SecurityGroupID).
		WithWorkspaceID(workspace.ID)
	rs = append(rs, opensearchDomain)

	return rs
}

func (t *CreateOpenSearch) ApplyIdempotent(ctx context.Context, workspace db.Workspace) error {
	rs := t.Resources(workspace)
	for i := 0; i < len(rs); i++ {
		if err := rs[i].CreateIdempotent(ctx); err != nil {
			if errors.Is(err, resources.ErrResourceNeedsTime) {
				return ErrTransactionNeedsTime
			}
			return err
		}
	}
	return nil
}

func (t *CreateOpenSearch) RollbackIdempotent(ctx context.Context, workspace db.Workspace) error {
	rs := t.Resources(workspace)
	for i := len(rs) - 1; i >= 0; i-- {
		if err := rs[i].DeleteIdempotent(ctx); err != nil {
			if errors.Is(err, resources.ErrResourceNeedsTime) {
				return ErrTransactionNeedsTime
			}
			return err
		}
	}
	return nil
}
