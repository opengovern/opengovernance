package transactions

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	kms2 "github.com/aws/aws-sdk-go/service/kms"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/api"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/config"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/db"
	"github.com/kaytu-io/kaytu-util/pkg/source"
	"strings"
)

type CreateMasterCredential struct {
	iam *iam.Client
	kms *kms.Client
	cfg config.Config
	db  *db.Database
}

func NewCreateMasterCredential(
	iam *iam.Client,
	kms *kms.Client,
	cfg config.Config,
	db *db.Database,
) *CreateMasterCredential {
	return &CreateMasterCredential{
		iam: iam,
		kms: kms,
		cfg: cfg,
		db:  db,
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

	result, err := t.kms.Encrypt(context.TODO(), &kms.EncryptInput{
		KeyId:               &t.cfg.KMSKeyARN,
		Plaintext:           js,
		EncryptionAlgorithm: kms2.EncryptionAlgorithmSpecSymmetricDefault,
		EncryptionContext:   nil, //TODO-Saleh use workspaceID
		GrantTokens:         nil,
	})
	if err != nil {
		return fmt.Errorf("failed to encrypt ciphertext: %v", err)
	}
	encoded := base64.StdEncoding.EncodeToString(result.CiphertextBlob)

	err = t.db.CreateMasterCredential(&db.MasterCredential{
		WorkspaceID:   *workspace.AWSUniqueId,
		ConnectorType: source.CloudAWS,
		Credential:    encoded,
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
