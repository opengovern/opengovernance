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

func (t *CreateMasterCredential) Apply(workspace db.Workspace) error {
	userName := fmt.Sprintf("kaytu-user-%s", *workspace.AWSUniqueId)
	iamUser, err := t.iam.CreateUser(context.Background(), &iam.CreateUserInput{
		UserName:            aws.String(userName),
		Path:                nil,
		PermissionsBoundary: nil,
		Tags:                nil,
	})
	if err != nil {
		return err
	}
	policy, err := t.iam.CreatePolicy(context.Background(), &iam.CreatePolicyInput{
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
		return err
	}

	_, err = t.iam.AttachUserPolicy(context.Background(), &iam.AttachUserPolicyInput{
		PolicyArn: policy.Policy.Arn,
		UserName:  aws.String(userName),
	})

	key, err := t.iam.CreateAccessKey(context.Background(), &iam.CreateAccessKeyInput{
		UserName: aws.String(userName),
	})
	if err != nil {
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

func (t *CreateMasterCredential) Rollback(workspace db.Workspace) error {
	if workspace.AWSUniqueId != nil {
		userName := fmt.Sprintf("kaytu-user-%s", *workspace.AWSUniqueId)
		accessKeys, err := t.iam.ListAccessKeys(context.Background(), &iam.ListAccessKeysInput{
			UserName: aws.String(userName),
		})
		if err != nil {
			if !strings.Contains(err.Error(), "NoSuchEntity") {
				return err
			}
			accessKeys = &iam.ListAccessKeysOutput{}
		}
		for _, accessKey := range accessKeys.AccessKeyMetadata {
			_, err := t.iam.DeleteAccessKey(context.Background(), &iam.DeleteAccessKeyInput{
				UserName:    aws.String(userName),
				AccessKeyId: accessKey.AccessKeyId,
			})
			if err != nil {
				return err
			}
		}

		policies, err := t.iam.ListAttachedUserPolicies(context.Background(), &iam.ListAttachedUserPoliciesInput{
			UserName: aws.String(userName),
		})
		if err != nil {
			if !strings.Contains(err.Error(), "NoSuchEntity") {
				return err
			}
			policies = &iam.ListAttachedUserPoliciesOutput{}
		}

		for _, policy := range policies.AttachedPolicies {
			_, err = t.iam.DetachUserPolicy(context.Background(), &iam.DetachUserPolicyInput{
				UserName:  aws.String(userName),
				PolicyArn: policy.PolicyArn,
			})
			if err != nil {
				return err
			}

			_, err = t.iam.DeleteUserPolicy(context.Background(), &iam.DeleteUserPolicyInput{
				PolicyName: policy.PolicyName,
				UserName:   aws.String(userName),
			})
			if err != nil {
				return err
			}
		}

		_, err = t.iam.DeleteUser(context.Background(), &iam.DeleteUserInput{
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
