package transactions

import (
	"context"
	"encoding/json"
	"fmt"
	authapi "github.com/kaytu-io/kaytu-engine/pkg/auth/api"
	"github.com/kaytu-io/kaytu-engine/pkg/httpclient"
	api2 "github.com/kaytu-io/kaytu-engine/pkg/onboard/api"
	apiv2 "github.com/kaytu-io/kaytu-engine/pkg/onboard/api/v2"
	"github.com/kaytu-io/kaytu-engine/pkg/onboard/client"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/api"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/config"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/db"
	"github.com/kaytu-io/kaytu-util/pkg/source"
	"github.com/kaytu-io/kaytu-util/pkg/vault"
	"strings"
)

type EnsureCredentialOnboarded struct {
	vault vault.VaultSourceConfig
	cfg   config.Config
	db    *db.Database
}

func NewEnsureCredentialOnboarded(
	vault vault.VaultSourceConfig,
	cfg config.Config,
	db *db.Database,
) *EnsureCredentialOnboarded {
	return &EnsureCredentialOnboarded{
		vault: vault,
		cfg:   cfg,
		db:    db,
	}
}

func (t *EnsureCredentialOnboarded) Requirements() []api.TransactionID {
	return []api.TransactionID{api.Transaction_CreateMasterCredential, api.Transaction_CreateHelmRelease}
}

func (t *EnsureCredentialOnboarded) ApplyIdempotent(ctx context.Context, workspace db.Workspace) error {
	creds, err := t.db.ListCredentialsByWorkspaceID(workspace.ID)
	if err != nil {
		return err
	}

	if len(creds) == 0 {
		return ErrTransactionNeedsTime
	}

	for _, cred := range creds {
		if !cred.IsCreated {
			err := t.addCredentialToWorkspace(ctx, workspace, cred)
			if err != nil {
				return err
			}
		}
	}

	if !workspace.IsBootstrapInputFinished {
		return ErrTransactionNeedsTime
	}

	return nil
}

func (t *EnsureCredentialOnboarded) RollbackIdempotent(ctx context.Context, workspace db.Workspace) error {
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

func (t *EnsureCredentialOnboarded) addCredentialToWorkspace(ctx context.Context, workspace db.Workspace, cred db.Credential) error {
	onboardURL := strings.ReplaceAll(t.cfg.Onboard.BaseURL, "%NAMESPACE%", workspace.ID)
	onboardClient := client.NewOnboardServiceClient(onboardURL)

	var request api.AddCredentialRequest

	result, err := t.vault.Decrypt(ctx, cred.Metadata)
	if err != nil {
		return fmt.Errorf("failed to encrypt ciphertext: %v", err)
	}

	jsonResult, err := json.Marshal(result)
	if err != nil {
		return err
	}

	err = json.Unmarshal(jsonResult, &request)
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
