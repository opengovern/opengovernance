package transactions

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/opensearch"
	types3 "github.com/aws/aws-sdk-go-v2/service/opensearch/types"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/api"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/db"
	"strings"
)

type CreateOpenSearch struct {
	securityGroupID string
	subnetID        string
	vmType          types3.OpenSearchPartitionInstanceType
	instanceCount   int32
	db              *db.Database

	iam        *iam.Client
	opensearch *opensearch.Client
}

func NewCreateOpenSearch(
	securityGroupID string,
	subnetID string,
	vmType types3.OpenSearchPartitionInstanceType,
	instanceCount int32,
	db *db.Database,
	iam *iam.Client,
	opensearch *opensearch.Client,
) *CreateOpenSearch {
	return &CreateOpenSearch{
		securityGroupID: securityGroupID,
		subnetID:        subnetID,
		vmType:          vmType,
		instanceCount:   instanceCount,
		db:              db,
		iam:             iam,
		opensearch:      opensearch,
	}
}

func (t *CreateOpenSearch) Requirements() []api.TransactionID {
	return []api.TransactionID{api.Transaction_CreateServiceAccountRoles}
}

func (t *CreateOpenSearch) Apply(workspace db.Workspace) error {
	processing, endpoint, err := t.isOpenSearchCreationFinished(workspace)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			if err := t.createOpenSearch(workspace); err != nil {
				return err
			}
			return ErrTransactionNeedsTime
		}
		return err
	}

	if processing {
		return ErrTransactionNeedsTime
	}

	if endpoint == "" {
		return ErrTransactionNeedsTime
	}

	err = t.db.UpdateWorkspaceOpenSearchEndpoint(workspace.ID, endpoint)
	if err != nil {
		return err
	}

	return nil
}

func (t *CreateOpenSearch) Rollback(workspace db.Workspace) error {
	domainName := workspace.ID

	domain, err := t.opensearch.DescribeDomain(context.Background(), &opensearch.DescribeDomainInput{
		DomainName: aws.String(domainName),
	})
	if err != nil {
		if strings.Contains(err.Error(), "ResourceNotFoundException") {
			return nil
		}
		return err
	}

	deleted := false
	if domain.DomainStatus.Deleted != nil {
		deleted = *domain.DomainStatus.Deleted
	}

	if !deleted {
		_, err := t.opensearch.DeleteDomain(context.Background(), &opensearch.DeleteDomainInput{DomainName: aws.String(domainName)})
		if err != nil {
			return err
		}
	} else {
		processing := false
		if domain.DomainStatus.Processing != nil {
			processing = *domain.DomainStatus.Processing
		}

		if processing {
			return ErrTransactionNeedsTime
		}
		return nil
	}

	return ErrTransactionNeedsTime
}

func (t *CreateOpenSearch) isOpenSearchCreationFinished(workspace db.Workspace) (bool, string, error) {
	domainName := workspace.ID

	domain, err := t.opensearch.DescribeDomain(context.Background(), &opensearch.DescribeDomainInput{
		DomainName: aws.String(domainName),
	})
	if err != nil {
		return false, "", err
	}

	processing := true
	if domain.DomainStatus.Processing != nil {
		processing = *domain.DomainStatus.Processing
	}

	var endpoint string
	for k, v := range domain.DomainStatus.Endpoints {
		if k == "vpc" {
			endpoint = "https://" + v
		} else {
			endpoint = v
		}
	}
	return processing, endpoint, nil
}

func (t *CreateOpenSearch) createOpenSearch(workspace db.Workspace) error {
	domainName := workspace.ID

	out, err := t.iam.CreateRole(context.Background(), &iam.CreateRoleInput{
		AssumeRolePolicyDocument: aws.String(fmt.Sprintf(`{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
			"Principal": {
				"AWS": [
					"arn:aws:iam::435670955331:role/kaytu-service-%[1]s-migrator",
					"arn:aws:iam::435670955331:role/kaytu-service-%[1]s-compliance",
					"arn:aws:iam::435670955331:role/kaytu-service-%[1]s-compliance-report-worker",
					"arn:aws:iam::435670955331:role/kaytu-service-%[1]s-compliance-summarizer",
					"arn:aws:iam::435670955331:role/kaytu-service-%[1]s-cost-estimator",
					"arn:aws:iam::435670955331:role/kaytu-service-%[1]s-insight-worker",
					"arn:aws:iam::435670955331:role/kaytu-service-%[1]s-inventory",
					"arn:aws:iam::435670955331:role/kaytu-service-%[1]s-reporter",
					"arn:aws:iam::435670955331:role/kaytu-service-%[1]s-scheduler",
					"arn:aws:iam::435670955331:role/kaytu-service-%[1]s-steampipe",
					"arn:aws:iam::435670955331:role/kaytu-service-%[1]s-analytics-worker"
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
}`, workspace.ID)),
		RoleName:            aws.String(fmt.Sprintf("kaytu-opensearch-master-%s", workspace.ID)),
		Description:         nil,
		MaxSessionDuration:  nil,
		Path:                nil,
		PermissionsBoundary: nil,
		Tags:                nil,
	})
	if err != nil {
		return err
	}

	bucketName := fmt.Sprintf("dlq-%s", workspace.ID)
	policy, err := t.iam.CreatePolicy(context.Background(), &iam.CreatePolicyInput{
		PolicyDocument: aws.String(fmt.Sprintf(`{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "WriteToS3DLQ",
      "Effect": "Allow",
      "Action": "s3:PutObject",
      "Resource": "arn:aws:s3:::%s/*"
    }
  ]
}`, bucketName)),
		PolicyName:  aws.String(fmt.Sprintf("kaytu-dlq-policy-%s", workspace.ID)),
		Description: nil,
		Path:        nil,
		Tags:        nil,
	})
	if err != nil {
		return err
	}

	_, err = t.iam.AttachRolePolicy(context.Background(), &iam.AttachRolePolicyInput{
		PolicyArn: policy.Policy.Arn,
		RoleName:  out.Role.RoleName,
	})
	if err != nil {
		return err
	}

	_, err = t.iam.AttachRolePolicy(context.Background(), &iam.AttachRolePolicyInput{
		PolicyArn: aws.String("arn:aws:iam::aws:policy/AmazonOpenSearchServiceFullAccess"),
		RoleName:  out.Role.RoleName,
	})
	if err != nil {
		return err
	}

	_, err = t.opensearch.CreateDomain(context.Background(), &opensearch.CreateDomainInput{
		DomainName:     aws.String(domainName),
		AccessPolicies: nil,
		AdvancedOptions: map[string]string{
			"indices.query.bool.max_clause_count":    "1024",
			"override_main_response_version":         "false",
			"rest.action.multi.allow_explicit_index": "true",
			"indices.fielddata.cache.size":           "20",
		},
		AdvancedSecurityOptions: &types3.AdvancedSecurityOptionsInput{
			AnonymousAuthEnabled:        nil,
			Enabled:                     aws.Bool(true),
			InternalUserDatabaseEnabled: nil,
			MasterUserOptions: &types3.MasterUserOptions{
				MasterUserARN:      out.Role.Arn,
				MasterUserName:     nil,
				MasterUserPassword: nil,
			},
			SAMLOptions: nil,
		},
		AutoTuneOptions: &types3.AutoTuneOptionsInput{DesiredState: types3.AutoTuneDesiredStateDisabled},
		ClusterConfig: &types3.ClusterConfig{
			ColdStorageOptions:        &types3.ColdStorageOptions{Enabled: aws.Bool(false)},
			DedicatedMasterCount:      nil,
			DedicatedMasterEnabled:    aws.Bool(false),
			DedicatedMasterType:       "",
			InstanceCount:             aws.Int32(t.instanceCount),
			InstanceType:              t.vmType,
			MultiAZWithStandbyEnabled: aws.Bool(false),
			WarmCount:                 nil,
			WarmEnabled:               aws.Bool(false),
			WarmType:                  "",
			ZoneAwarenessConfig:       nil,
			ZoneAwarenessEnabled:      aws.Bool(false),
		},
		CognitoOptions: nil,
		DomainEndpointOptions: &types3.DomainEndpointOptions{
			CustomEndpoint:               nil,
			CustomEndpointCertificateArn: nil,
			CustomEndpointEnabled:        aws.Bool(false),
			EnforceHTTPS:                 aws.Bool(true),
			TLSSecurityPolicy:            "Policy-Min-TLS-1-0-2019-07",
		},
		EBSOptions: &types3.EBSOptions{
			EBSEnabled: aws.Bool(true),
			Iops:       aws.Int32(3000),
			Throughput: aws.Int32(125),
			VolumeSize: aws.Int32(10),
			VolumeType: "gp3",
		},
		EncryptionAtRestOptions: &types3.EncryptionAtRestOptions{
			Enabled:  aws.Bool(true),
			KmsKeyId: nil,
		},
		EngineVersion:               aws.String("OpenSearch_2.11"),
		LogPublishingOptions:        nil,
		NodeToNodeEncryptionOptions: &types3.NodeToNodeEncryptionOptions{Enabled: aws.Bool(true)},
		OffPeakWindowOptions: &types3.OffPeakWindowOptions{
			Enabled: aws.Bool(true),
			OffPeakWindow: &types3.OffPeakWindow{
				WindowStartTime: &types3.WindowStartTime{
					Hours:   0,
					Minutes: 0,
				},
			},
		},
		SoftwareUpdateOptions: nil,
		TagList:               nil,
		VPCOptions: &types3.VPCOptions{
			SecurityGroupIds: []string{t.securityGroupID},
			SubnetIds:        []string{t.subnetID},
		},
	})
	if err != nil {
		return err
	}

	return nil
}
