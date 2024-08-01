package transactions

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	types2 "github.com/aws/aws-sdk-go-v2/service/iam/types"
	"github.com/fluxcd/helm-controller/api/v2beta1"
	helmv2 "github.com/fluxcd/helm-controller/api/v2beta1"
	apimeta "github.com/fluxcd/pkg/apis/meta"
	api6 "github.com/hashicorp/vault/api"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/api"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/config"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/db"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/internal/helm"
	types3 "github.com/kaytu-io/kaytu-engine/pkg/workspace/types"
	"github.com/kaytu-io/kaytu-util/pkg/vault"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
)

type CreateHelmRelease struct {
	kubeClient         k8sclient.Client // the kubernetes client
	vault              vault.VaultSourceConfig
	cfg                config.Config
	db                 *db.Database
	logger             *zap.Logger
	vaultSecretHandler vault.VaultSecretHandler
}

func NewCreateHelmRelease(kubeClient k8sclient.Client, vault vault.VaultSourceConfig, handler vault.VaultSecretHandler, cfg config.Config, db *db.Database, logger *zap.Logger) *CreateHelmRelease {
	return &CreateHelmRelease{
		kubeClient:         kubeClient,
		vaultSecretHandler: handler,
		vault:              vault,
		cfg:                cfg,
		db:                 db,
		logger:             logger,
	}
}

func (t *CreateHelmRelease) Requirements() []api.TransactionID {
	return []api.TransactionID{
		//api.Transaction_CreateInsightBucket,
		//api.Transaction_CreateOpenSearch,
		//api.Transaction_CreateIngestionPipeline,
		api.Transaction_CreateServiceAccountRoles,
		api.Transaction_CreateWorkspaceKeyId,
	}
}

func (t *CreateHelmRelease) ApplyIdempotent(ctx context.Context, workspace db.Workspace) error {
	helmRelease, err := helm.FindHelmRelease(ctx, t.cfg, t.kubeClient, workspace)
	if err != nil {
		return fmt.Errorf("findHelmRelease: %w", err)
	}

	if helmRelease == nil {
		err := t.createHelmRelease(ctx, workspace)
		if err != nil {
			return fmt.Errorf("createHelmRelease: %w", err)
		}
		return ErrTransactionNeedsTime
	}

	err = t.ensureSettingsSynced(ctx, workspace, helmRelease)
	if err != nil {
		return err
	}

	if meta.IsStatusConditionTrue(helmRelease.Status.Conditions, apimeta.ReadyCondition) {
		return t.db.SetWorkspaceCreated(workspace.ID)
	} else if meta.IsStatusConditionFalse(helmRelease.Status.Conditions, apimeta.ReadyCondition) ||
		meta.IsStatusConditionTrue(helmRelease.Status.Conditions, apimeta.StalledCondition) {
		if !helmRelease.Spec.Suspend {
			helmRelease.Spec.Suspend = true
			err = t.kubeClient.Update(ctx, helmRelease)
			if err != nil {
				return fmt.Errorf("suspend helmrelease: %v", err)
			}
		} else {
			helmRelease.Spec.Suspend = false
			err = t.kubeClient.Update(ctx, helmRelease)
			if err != nil {
				return fmt.Errorf("suspend helmrelease: %v", err)
			}
		}
		return ErrTransactionNeedsTime
	}

	return ErrTransactionNeedsTime
}

func (t *CreateHelmRelease) RollbackIdempotent(ctx context.Context, workspace db.Workspace) error {
	helmRelease, err := helm.FindHelmRelease(ctx, t.cfg, t.kubeClient, workspace)
	if err != nil {
		return fmt.Errorf("find helm release: %w", err)
	}

	if helmRelease != nil {
		if err := helm.DeleteHelmRelease(ctx, t.cfg, t.kubeClient, workspace); err != nil {
			return fmt.Errorf("delete helm release: %w", err)
		}
		return ErrTransactionNeedsTime
	}

	namespace, err := t.findTargetNamespace(ctx, workspace.ID)
	if err != nil {
		return fmt.Errorf("find target namespace: %w", err)
	}
	if namespace != nil {
		if err := t.deleteTargetNamespace(ctx, workspace.ID); err != nil {
			return fmt.Errorf("delete target namespace: %w", err)
		}
		return ErrTransactionNeedsTime
	}

	return nil
}

func (t *CreateHelmRelease) ensureSettingsSynced(ctx context.Context, workspace db.Workspace, release *helmv2.HelmRelease) error {
	needsUpdate, settings, err := helm.GetUpToDateWorkspaceHelmValues(ctx, t.cfg, t.kubeClient, t.db, t.vault, workspace)
	if err != nil {
		return fmt.Errorf("get up to date workspace helm values: %w", err)
	}

	if needsUpdate {
		valuesJson, err := json.Marshal(settings)
		if err != nil {
			return err
		}

		err = helm.UpdateHelmRelease(ctx, t.cfg, t.kubeClient, workspace, valuesJson)
		if err != nil {
			return fmt.Errorf("update helm release: %w", err)
		}
	}

	return nil
}

func (t *CreateHelmRelease) createHelmRelease(ctx context.Context, workspace db.Workspace) error {
	var userARN string
	if workspace.AWSUserARN != nil {
		userARN = *workspace.AWSUserARN
	}

	settings := types3.KaytuWorkspaceSettings{
		Kaytu: types3.KaytuConfig{
			ReplicaCount: 1,
			EnvType:      t.cfg.EnvType,
			Octopus: types3.OctopusConfig{
				Namespace: t.cfg.KaytuOctopusNamespace,
			},
			Domain: types3.DomainConfig{
				App:          t.cfg.AppDomain,
				Grpc:         t.cfg.GrpcDomain,
				GrpcExternal: t.cfg.GrpcExternalDomain,
			},
			Workspace: types3.WorkspaceConfig{
				Name:    workspace.Name,
				Size:    workspace.Size,
				UserARN: userARN,
			},
			Insights: types3.InsightsConfig{
				S3: types3.S3Config{
					AccessKey: t.cfg.S3AccessKey,
					SecretKey: t.cfg.S3SecretKey,
				},
			},
			OpenSearch: types3.OpenSearchConfig{
				Enabled:                   true,
				Endpoint:                  workspace.OpenSearchEndpoint,
				IngestionPipelineEndpoint: workspace.PipelineEndpoint,
			},
		},
	}
	if workspace.AWSUniqueId != nil {
		masterCred, err := t.db.GetMasterCredentialByWorkspaceUID(*workspace.AWSUniqueId)
		if err != nil {
			return err
		}

		if masterCred != nil {
			result, err := t.vault.Decrypt(ctx, masterCred.Credential)
			if err != nil {
				return fmt.Errorf("failed to encrypt ciphertext: %v", err)
			}
			jsonResult, err := json.Marshal(result)
			if err != nil {
				return err
			}
			var accessKey types2.AccessKey
			err = json.Unmarshal(jsonResult, &accessKey)
			if err != nil {
				return err
			}

			settings.Kaytu.Workspace.MasterAccessKey = *accessKey.AccessKeyId
			settings.Kaytu.Workspace.MasterSecretKey = *accessKey.SecretAccessKey
		}
	}

	valuesJson, err := json.Marshal(settings)
	if err != nil {
		return err
	}

	if err := helm.CreateHelmRelease(ctx, t.cfg, t.kubeClient, workspace, valuesJson); err != nil {
		return fmt.Errorf("create helm release: %w", err)
	}

	if t.cfg.Vault.Provider == vault.HashiCorpVault {
		_, err := vault.NewHashiCorpVaultClient(ctx, t.logger, t.cfg.Vault.HashiCorp, settings.Vault.KeyID)
		if err != nil {
			if strings.Contains(err.Error(), api6.ErrSecretNotFound.Error()) || strings.Contains(err.Error(), "secret value is nil") {
				b := make([]byte, 32)
				_, err := rand.Read(b)
				if err != nil {
					return err
				}

				_, err = t.vaultSecretHandler.SetSecret(ctx, settings.Vault.KeyID, b)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func (t *CreateHelmRelease) deleteTargetNamespace(ctx context.Context, name string) error {
	ns := corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
	return t.kubeClient.Delete(ctx, &ns)
}

func (t *CreateHelmRelease) findTargetNamespace(ctx context.Context, name string) (*corev1.Namespace, error) {
	key := client.ObjectKey{
		Name: name,
	}
	var ns corev1.Namespace
	if err := t.kubeClient.Get(ctx, key, &ns); err != nil {
		if apierrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("find target namespace: %w", err)
	}
	return &ns, nil
}

func (t *CreateHelmRelease) deleteHelmRelease(ctx context.Context, workspace db.Workspace) error {
	helmRelease := helmv2.HelmRelease{
		ObjectMeta: metav1.ObjectMeta{
			Name:      workspace.ID,
			Namespace: t.cfg.FluxSystemNamespace,
		},
	}
	return t.kubeClient.Delete(ctx, &helmRelease)
}

func (t *CreateHelmRelease) UpdateWorkspaceSettings(ctx context.Context, helmRelease *v2beta1.HelmRelease, settings types3.KaytuWorkspaceSettings) error {
	b, err := json.Marshal(settings)
	if err != nil {
		return fmt.Errorf("marshalling values: %w", err)
	}
	helmRelease.Spec.Values.Raw = b
	err = t.kubeClient.Update(ctx, helmRelease)
	if err != nil {
		return fmt.Errorf("updating replica count: %w", err)
	}
	return nil
}

func GetWorkspaceHelmValues(helmRelease *v2beta1.HelmRelease) (types3.KaytuWorkspaceSettings, error) {
	var settings types3.KaytuWorkspaceSettings

	values := helmRelease.GetValues()
	valuesJSON, err := json.Marshal(values)
	if err != nil {
		return settings, err
	}

	err = json.Unmarshal(valuesJSON, &settings)
	if err != nil {
		return settings, err
	}

	return settings, nil
}
