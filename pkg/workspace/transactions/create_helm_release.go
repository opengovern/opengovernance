package transactions

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	types2 "github.com/aws/aws-sdk-go-v2/service/iam/types"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	kms2 "github.com/aws/aws-sdk-go/service/kms"
	"github.com/fluxcd/helm-controller/api/v2beta1"
	helmv2 "github.com/fluxcd/helm-controller/api/v2beta1"
	apimeta "github.com/fluxcd/pkg/apis/meta"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/config"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/db"
	types3 "github.com/kaytu-io/kaytu-engine/pkg/workspace/types"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
	"time"
)

type CreateHelmRelease struct {
	kubeClient k8sclient.Client // the kubernetes client
	kmsClient  *kms.Client
	cfg        config.Config
	db         *db.Database
	reserving  bool
}

func NewCreateHelmRelease(
	kubeClient k8sclient.Client,
	kmsClient *kms.Client,
	cfg config.Config,
	db *db.Database,
) *CreateHelmRelease {
	return &CreateHelmRelease{
		kubeClient: kubeClient,
		kmsClient:  kmsClient,
		cfg:        cfg,
		db:         db,
	}
}

func (t *CreateHelmRelease) Requirements() []TransactionID {
	return []TransactionID{Transaction_CreateInsightBucket, Transaction_CreateOpenSearch, Transaction_CreateServiceAccountRoles}
}

func (t *CreateHelmRelease) Apply(workspace db.Workspace) error {
	helmRelease, err := t.findHelmRelease(context.Background(), workspace)
	if err != nil {
		return fmt.Errorf("findHelmRelease: %w", err)
	}

	if helmRelease == nil {
		err := t.createHelmRelease(workspace)
		if err != nil {
			return fmt.Errorf("createHelmRelease: %w", err)
		}
		return ErrTransactionNeedsTime
	}

	err = t.ensureSettingsSynced(workspace, helmRelease)
	if err != nil {
		return err
	}

	if meta.IsStatusConditionTrue(helmRelease.Status.Conditions, apimeta.ReadyCondition) {
		return t.db.SetWorkspaceCreated(workspace.ID)
	} else if meta.IsStatusConditionFalse(helmRelease.Status.Conditions, apimeta.ReadyCondition) ||
		meta.IsStatusConditionTrue(helmRelease.Status.Conditions, apimeta.StalledCondition) {
		if !helmRelease.Spec.Suspend {
			helmRelease.Spec.Suspend = true
			err = t.kubeClient.Update(context.Background(), helmRelease)
			if err != nil {
				return fmt.Errorf("suspend helmrelease: %v", err)
			}
		} else {
			helmRelease.Spec.Suspend = false
			err = t.kubeClient.Update(context.Background(), helmRelease)
			if err != nil {
				return fmt.Errorf("suspend helmrelease: %v", err)
			}
		}
		return ErrTransactionNeedsTime
	}

	return ErrTransactionNeedsTime
}

func (t *CreateHelmRelease) Rollback(workspace db.Workspace) error {
	helmRelease, err := t.findHelmRelease(context.Background(), workspace)
	if err != nil {
		return fmt.Errorf("find helm release: %w", err)
	}

	if helmRelease != nil {
		if err := t.deleteHelmRelease(context.Background(), workspace); err != nil {
			return fmt.Errorf("delete helm release: %w", err)
		}
		return ErrTransactionNeedsTime
	}

	namespace, err := t.findTargetNamespace(context.Background(), workspace.ID)
	if err != nil {
		return fmt.Errorf("find target namespace: %w", err)
	}
	if namespace != nil {
		if err := t.deleteTargetNamespace(context.Background(), workspace.ID); err != nil {
			return fmt.Errorf("delete target namespace: %w", err)
		}
		return ErrTransactionNeedsTime
	}

	return nil
}

func (t *CreateHelmRelease) ensureSettingsSynced(workspace db.Workspace, release *helmv2.HelmRelease) error {
	settings, err := GetWorkspaceHelmValues(release)
	if err != nil {
		return err
	}

	needsUpdate := false

	if settings.Kaytu.Workspace.Name != workspace.Name {
		settings.Kaytu.Workspace.Name = workspace.Name
		needsUpdate = true
	}

	if workspace.AWSUserARN != nil && settings.Kaytu.Workspace.UserARN != *workspace.AWSUserARN {
		settings.Kaytu.Workspace.UserARN = *workspace.AWSUserARN
		needsUpdate = true
	}

	if workspace.AWSUniqueId != nil {
		masterCred, err := t.db.GetMasterCredentialByWorkspaceUID(*workspace.AWSUniqueId)
		if err != nil {
			return err
		}

		decoded, err := base64.StdEncoding.DecodeString(masterCred.Credential)
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

		var accessKey types2.AccessKey
		err = json.Unmarshal(result.Plaintext, &accessKey)
		if err != nil {
			return err
		}

		if settings.Kaytu.Workspace.MasterAccessKey != *accessKey.AccessKeyId {
			settings.Kaytu.Workspace.MasterAccessKey = *accessKey.AccessKeyId
			settings.Kaytu.Workspace.MasterSecretKey = *accessKey.SecretAccessKey
			needsUpdate = true
		}
	}

	if needsUpdate {
		err = t.UpdateWorkspaceSettings(release, settings)
		if err != nil {
			return err
		}

		var res corev1.PodList
		err = t.kubeClient.List(context.Background(), &res)
		if err != nil {
			return fmt.Errorf("listing pods: %w", err)
		}
		for _, pod := range res.Items {
			if strings.HasPrefix(pod.Name, "describe-scheduler") {
				err = t.kubeClient.Delete(context.Background(), &pod)
				if err != nil {
					return fmt.Errorf("deleting pods: %w", err)
				}
			}
		}
	}

	return nil
}

func (t *CreateHelmRelease) createHelmRelease(workspace db.Workspace) error {
	var userARN string
	if workspace.AWSUserARN != nil {
		userARN = *workspace.AWSUserARN
	}

	settings := types3.KaytuWorkspaceSettings{
		Kaytu: types3.KaytuConfig{
			ReplicaCount: 1,
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
				Enabled:  true,
				Endpoint: workspace.OpenSearchEndpoint,
			},
		},
	}
	if workspace.AWSUniqueId != nil {
		masterCred, err := t.db.GetMasterCredentialByWorkspaceUID(*workspace.AWSUniqueId)
		if err != nil {
			return err
		}

		decoded, err := base64.StdEncoding.DecodeString(masterCred.Credential)
		if err != nil {
			return err
		}

		result, err := t.kmsClient.Decrypt(context.TODO(), &kms.DecryptInput{
			CiphertextBlob:      decoded,
			EncryptionAlgorithm: kms2.EncryptionAlgorithmSpecSymmetricDefault,
			KeyId:               &t.cfg.KMSKeyARN,
			EncryptionContext:   nil,
		})
		if err != nil {
			return fmt.Errorf("failed to encrypt ciphertext: %v", err)
		}

		var accessKey types2.AccessKey
		err = json.Unmarshal(result.Plaintext, &accessKey)
		if err != nil {
			return err
		}

		settings.Kaytu.Workspace.MasterAccessKey = *accessKey.AccessKeyId
		settings.Kaytu.Workspace.MasterSecretKey = *accessKey.SecretAccessKey
	}

	settingsJSON, err := json.Marshal(settings)
	if err != nil {
		return err
	}

	helmRelease := helmv2.HelmRelease{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "helm.toolkit.fluxcd.io/v2beta1",
			Kind:       "HelmRelease",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      workspace.ID,
			Namespace: t.cfg.FluxSystemNamespace,
		},
		Spec: helmv2.HelmReleaseSpec{
			Interval: metav1.Duration{
				Duration: 5 + time.Minute,
			},
			TargetNamespace: workspace.ID,
			ReleaseName:     workspace.ID,
			Chart: helmv2.HelmChartTemplate{
				Spec: helmv2.HelmChartTemplateSpec{
					Chart: t.cfg.KaytuHelmChartLocation,
					SourceRef: helmv2.CrossNamespaceObjectReference{
						Kind:      "GitRepository",
						Name:      "flux-system",
						Namespace: t.cfg.FluxSystemNamespace,
					},
					Interval: &metav1.Duration{
						Duration: time.Minute,
					},
					ReconcileStrategy: "Revision",
				},
			},
			Values: &apiextensionsv1.JSON{
				Raw: settingsJSON,
			},
			Install: &helmv2.Install{
				CreateNamespace: true,
			},
		},
	}
	if err := t.kubeClient.Create(context.Background(), &helmRelease); err != nil {
		return fmt.Errorf("create helm release: %w", err)
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

func (t *CreateHelmRelease) findHelmRelease(ctx context.Context, workspace db.Workspace) (*helmv2.HelmRelease, error) {
	key := types.NamespacedName{
		Name:      workspace.ID,
		Namespace: t.cfg.FluxSystemNamespace,
	}
	var helmRelease helmv2.HelmRelease
	if err := t.kubeClient.Get(ctx, key, &helmRelease); err != nil {
		if apierrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return &helmRelease, nil
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

func (t *CreateHelmRelease) UpdateWorkspaceSettings(helmRelease *v2beta1.HelmRelease, settings types3.KaytuWorkspaceSettings) error {
	ctx := context.Background()
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
