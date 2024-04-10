package transactions

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/api"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/config"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/db"
	"github.com/kaytu-io/kaytu-util/pkg/source"
	"github.com/kaytu-io/kaytu-util/pkg/vault"
	"strings"
)

type CreateMasterCredential struct {
	iam   *iam.Client
	vault vault.VaultSourceConfig
	cfg   config.Config
	db    *db.Database
}

func NewCreateMasterCredential(
	iam *iam.Client,
	vault vault.VaultSourceConfig,
	cfg config.Config,
	db *db.Database,
) *CreateMasterCredential {
	return &CreateMasterCredential{
		iam:   iam,
		vault: vault,
		cfg:   cfg,
		db:    db,
	}
}

func (t *CreateMasterCredential) Requirements() []api.TransactionID {
	return nil
}

func (t *CreateMasterCredential) ApplyIdempotent(ctx context.Context, workspace db.Workspace) error {
	userName := fmt.Sprintf("kaytu-user-%s", *workspace.AWSUniqueId)
	iamUser, err := t.iam.CreateUser(ctx, &iam.CreateUserInput{
		UserName:            aws.String(userName),
		Path:                nil,
		PermissionsBoundary: nil,
		Tags:                nil,
	})
	if err != nil {
		if !strings.Contains(err.Error(), "EntityAlreadyExists") {
			return err
		}
		u, err := t.iam.GetUser(ctx, &iam.GetUserInput{UserName: aws.String(userName)})
		if err != nil {
			return err
		}
		iamUser = &iam.CreateUserOutput{
			User: u.User,
		}
	}
	policy, err := t.iam.CreatePolicy(ctx, &iam.CreatePolicyInput{
		PolicyDocument: aws.String(`{
	"Version": "2012-10-17",
	"Statement": {
		"Effect": "Allow",
		"Action": "sts:AssumeRole",
		"Resource": "*"
	}
}`),
		PolicyName: aws.String(userName + "-assume-role"),
	})
	if err != nil {
		if !strings.Contains(err.Error(), "EntityAlreadyExists") {
			return err
		}
	} else {
		_, err = t.iam.AttachUserPolicy(ctx, &iam.AttachUserPolicyInput{
			PolicyArn: policy.Policy.Arn,
			UserName:  aws.String(userName),
		})
	}

	key, err := t.iam.CreateAccessKey(ctx, &iam.CreateAccessKeyInput{
		UserName: aws.String(userName),
	})
	if err != nil {
		if strings.Contains(err.Error(), "LimitExceeded") {
			accessKeys, err := t.iam.ListAccessKeys(ctx, &iam.ListAccessKeysInput{
				UserName: aws.String(userName),
			})
			if err != nil {
				if !strings.Contains(err.Error(), "NoSuchEntity") {
					return err
				}
				accessKeys = &iam.ListAccessKeysOutput{}
			}
			for _, accessKey := range accessKeys.AccessKeyMetadata {
				_, err := t.iam.DeleteAccessKey(ctx, &iam.DeleteAccessKeyInput{
					UserName:    aws.String(userName),
					AccessKeyId: accessKey.AccessKeyId,
				})
				if err != nil {
					return err
				}
			}
			return ErrTransactionNeedsTime
		}
		return err
	}

	js, err := json.Marshal(key.AccessKey)
	if err != nil {
		return err
	}

	jsMap := make(map[string]any)
	err = json.Unmarshal(js, &jsMap)

	result, err := t.vault.Encrypt(ctx, jsMap)
	if err != nil {
		return fmt.Errorf("failed to encrypt ciphertext: %v", err)
	}

	err = t.db.CreateMasterCredential(&db.MasterCredential{
		WorkspaceID:   *workspace.AWSUniqueId,
		ConnectorType: source.CloudAWS,
		Credential:    string(result),
	})
	if err != nil {
		return err
	}

	err = t.db.UpdateWorkspaceAWSUser(workspace.ID, iamUser.User.Arn)
	if err != nil {
		return err
	}

	return nil
}

func (t *CreateMasterCredential) RollbackIdempotent(ctx context.Context, workspace db.Workspace) error {
	if workspace.AWSUniqueId != nil {
		userName := fmt.Sprintf("kaytu-user-%s", *workspace.AWSUniqueId)
		accessKeys, err := t.iam.ListAccessKeys(ctx, &iam.ListAccessKeysInput{
			UserName: aws.String(userName),
		})
		if err != nil {
			if !strings.Contains(err.Error(), "NoSuchEntity") {
				return err
			}
			accessKeys = &iam.ListAccessKeysOutput{}
		}
		for _, accessKey := range accessKeys.AccessKeyMetadata {
			_, err := t.iam.DeleteAccessKey(ctx, &iam.DeleteAccessKeyInput{
				UserName:    aws.String(userName),
				AccessKeyId: accessKey.AccessKeyId,
			})
			if err != nil {
				return err
			}
		}

		policies, err := t.iam.ListAttachedUserPolicies(ctx, &iam.ListAttachedUserPoliciesInput{
			UserName: aws.String(userName),
		})
		if err != nil {
			if !strings.Contains(err.Error(), "NoSuchEntity") {
				return err
			}
			policies = &iam.ListAttachedUserPoliciesOutput{}
		}

		for _, policy := range policies.AttachedPolicies {
			_, err = t.iam.DetachUserPolicy(ctx, &iam.DetachUserPolicyInput{
				UserName:  aws.String(userName),
				PolicyArn: policy.PolicyArn,
			})
			if err != nil {
				return err
			}

			_, err = t.iam.DeleteUserPolicy(ctx, &iam.DeleteUserPolicyInput{
				PolicyName: policy.PolicyName,
				UserName:   aws.String(userName),
			})
			if err != nil {
				return err
			}
		}

		_, err = t.iam.DeleteUser(ctx, &iam.DeleteUserInput{
			UserName: aws.String(userName),
		})
		if err != nil {
			if !strings.Contains(err.Error(), "NoSuchEntity") {
				return err
			}
		}

		err = t.db.DeleteMasterCredential(*workspace.AWSUniqueId)
		if err != nil {
			return err
		}
	}
	return nil
}
