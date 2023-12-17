package transactions

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	kms2 "github.com/aws/aws-sdk-go/service/kms"
	authapi "github.com/kaytu-io/kaytu-engine/pkg/auth/api"
	"github.com/kaytu-io/kaytu-engine/pkg/httpclient"
	api2 "github.com/kaytu-io/kaytu-engine/pkg/onboard/api"
	apiv2 "github.com/kaytu-io/kaytu-engine/pkg/onboard/api/v2"
	"github.com/kaytu-io/kaytu-engine/pkg/onboard/client"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/api"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/config"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/db"
	"github.com/kaytu-io/kaytu-util/pkg/source"
	"strings"
)

type EnsureCredentialOnboarded struct {
	kmsClient *kms.Client
	cfg       config.Config
	db        *db.Database
}

func NewEnsureCredentialOnboarded(
	kmsClient *kms.Client,
	cfg config.Config,
	db *db.Database,
) *EnsureCredentialOnboarded {
	return &EnsureCredentialOnboarded{
		kmsClient: kmsClient,
		cfg:       cfg,
		db:        db,
	}
}

func (t *EnsureCredentialOnboarded) Requirements() []TransactionID {
	return []TransactionID{Transaction_CreateMasterCredential}
}

func (t *EnsureCredentialOnboarded) Apply(workspace db.Workspace) error {
	creds, err := t.db.ListCredentialsByWorkspaceID(workspace.ID)
	if err != nil {
		return err
	}

	if len(creds) == 0 {
		return ErrTransactionNeedsTime
	}

	for _, cred := range creds {
		if !cred.IsCreated {
			err := t.addCredentialToWorkspace(workspace, cred)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (t *EnsureCredentialOnboarded) Rollback(workspace db.Workspace) error {
	creds, err := t.db.ListCredentialsByWorkspaceID(workspace.ID)
	if err != nil {
		return fmt.Errorf("listing credentials: %w", err)
	}
	for _, cred := range creds {
		err = t.db.DeleteCredential(cred.ID)
		if err != nil {
			return fmt.Errorf("deleting credentials: %w", err)
		}
	}
	return nil
}

func (t *EnsureCredentialOnboarded) addCredentialToWorkspace(workspace db.Workspace, cred db.Credential) error {
	onboardURL := strings.ReplaceAll(t.cfg.Onboard.BaseURL, "%NAMESPACE%", workspace.ID)
	onboardClient := client.NewOnboardServiceClient(onboardURL, nil)

	var request api.AddCredentialRequest
	decoded, err := base64.StdEncoding.DecodeString(cred.Metadata)
	if err != nil {
		return err
	}

	result, err := t.kmsClient.Decrypt(context.TODO(), &kms.DecryptInput{
		CiphertextBlob:      decoded,
		EncryptionAlgorithm: kms2.EncryptionAlgorithmSpecSymmetricDefault,
		KeyId:               &t.cfg.KMSKeyARN,
		EncryptionContext:   nil, //TODO-Saleh use workspaceID
	})
	if err != nil {
		return fmt.Errorf("failed to encrypt ciphertext: %v", err)
	}

	err = json.Unmarshal(result.Plaintext, &request)
	if err != nil {
		return err
	}

	if cred.ConnectorType == source.CloudAWS {
		if cred.SingleConnection {
			_, err := onboardClient.PostConnectionAws(&httpclient.Context{UserRole: authapi.InternalRole}, api2.CreateAwsConnectionRequest{
				Name:      "",
				AWSConfig: request.AWSConfig,
			})
			if err != nil {
				return err
			}
		} else {
			credential, err := onboardClient.CreateCredentialV2(&httpclient.Context{UserRole: authapi.InternalRole}, apiv2.CreateCredentialV2Request{
				Connector: cred.ConnectorType,
				AWSConfig: request.AWSConfig,
			})
			if err != nil {
				return err
			}

			_, err = onboardClient.AutoOnboard(&httpclient.Context{UserRole: authapi.InternalRole}, credential.ID)
			if err != nil {
				return err
			}
		}
	} else {
		credential, err := onboardClient.PostCredentials(&httpclient.Context{UserRole: authapi.InternalRole}, api2.CreateCredentialRequest{
			SourceType: cred.ConnectorType,
			Config:     request.AzureConfig,
		})
		if err != nil {
			return err
		}

		_, err = onboardClient.AutoOnboard(&httpclient.Context{UserRole: authapi.InternalRole}, credential.ID)
		if err != nil {
			return err
		}
	}

	err = t.db.SetIsCreated(cred.ID)
	if err != nil {
		return err
	}

	return nil
}
