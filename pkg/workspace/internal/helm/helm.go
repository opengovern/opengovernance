package helm

import (
	"context"
	"encoding/json"
	"fmt"
	types2 "github.com/aws/aws-sdk-go-v2/service/iam/types"
	helmv2 "github.com/fluxcd/helm-controller/api/v2beta1"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/config"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/db"
	types3 "github.com/kaytu-io/kaytu-engine/pkg/workspace/types"
	"github.com/kaytu-io/kaytu-util/pkg/vault"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
	"time"
)

func FindHelmRelease(ctx context.Context, cfg config.Config, kubeClient k8sclient.Client, workspace db.Workspace) (*helmv2.HelmRelease, error) {
	key := types.NamespacedName{
		Name:      workspace.ID,
		Namespace: cfg.FluxSystemNamespace,
	}
	var helmRelease helmv2.HelmRelease
	if err := kubeClient.Get(ctx, key, &helmRelease); err != nil {
		if apierrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return &helmRelease, nil
}

func CreateHelmRelease(ctx context.Context, cfg config.Config, kubeClient k8sclient.Client, workspace db.Workspace, valuesJson []byte) error {
	helmRelease := helmv2.HelmRelease{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "helm.toolkit.fluxcd.io/v2beta1",
			Kind:       "HelmRelease",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      workspace.ID,
			Namespace: cfg.FluxSystemNamespace,
		},
		Spec: helmv2.HelmReleaseSpec{
			Interval: metav1.Duration{
				Duration: 5 + time.Minute,
			},
			TargetNamespace: workspace.ID,
			ReleaseName:     workspace.ID,
			Chart: helmv2.HelmChartTemplate{
				Spec: helmv2.HelmChartTemplateSpec{
					Chart: cfg.KaytuHelmChartLocation,
					SourceRef: helmv2.CrossNamespaceObjectReference{
						Kind:      "GitRepository",
						Name:      "flux-system",
						Namespace: cfg.FluxSystemNamespace,
					},
					Interval: &metav1.Duration{
						Duration: time.Minute,
					},
					ReconcileStrategy: "Revision",
				},
			},
			Values: &apiextensionsv1.JSON{
				Raw: valuesJson,
			},
			Install: &helmv2.Install{
				CreateNamespace: true,
			},
		},
	}
	return kubeClient.Create(ctx, &helmRelease)
}

func UpdateHelmRelease(ctx context.Context, cfg config.Config, kubeClient k8sclient.Client, workspace db.Workspace, valuesJson []byte) error {
	helmRelease, err := FindHelmRelease(ctx, cfg, kubeClient, workspace)
	if err != nil {
		return fmt.Errorf("find helm release: %w", err)
	}

	helmRelease.Spec.Values.Raw = valuesJson
	err = kubeClient.Update(ctx, helmRelease)
	if err != nil {
		return fmt.Errorf("updating replica count: %w", err)
	}

	var res corev1.PodList
	err = kubeClient.List(ctx, &res)
	if err != nil {
		return fmt.Errorf("listing pods: %w", err)
	}
	for _, pod := range res.Items {
		if strings.HasPrefix(pod.Name, "describe-scheduler") {
			err = kubeClient.Delete(ctx, &pod)
			if err != nil {
				return fmt.Errorf("deleting pods: %w", err)
			}
		}
	}

	return nil
}

func DeleteHelmRelease(ctx context.Context, cfg config.Config, kubeClient k8sclient.Client, workspace db.Workspace) error {
	helmRelease := helmv2.HelmRelease{
		ObjectMeta: metav1.ObjectMeta{
			Name:      workspace.ID,
			Namespace: cfg.FluxSystemNamespace,
		},
	}
	return kubeClient.Delete(ctx, &helmRelease)
}

func GetWorkspaceHelmValues(ctx context.Context, cfg config.Config, kubeClient k8sclient.Client, workspace db.Workspace) (*types3.KaytuWorkspaceSettings, error) {
	helmRelease, err := FindHelmRelease(ctx, cfg, kubeClient, workspace)
	if err != nil {
		return nil, fmt.Errorf("find helm release: %w", err)
	}

	var settings types3.KaytuWorkspaceSettings
	values := helmRelease.GetValues()
	valuesJSON, err := json.Marshal(values)
	if err != nil {
		return &settings, err
	}

	err = json.Unmarshal(valuesJSON, &settings)
	if err != nil {
		return &settings, err
	}
	return &settings, nil
}

func GetUpToDateWorkspaceHelmValues(ctx context.Context, cfg config.Config, kubeClient k8sclient.Client,
	db *db.Database, vault vault.VaultSourceConfig,
	workspace db.Workspace) (bool, *types3.KaytuWorkspaceSettings, error) {
	settings, err := GetWorkspaceHelmValues(ctx, cfg, kubeClient, workspace)
	if err != nil {
		return false, nil, err
	}

	needsUpdate := false

	if settings.Kaytu.EnvType != cfg.EnvType {
		settings.Kaytu.EnvType = cfg.EnvType
		needsUpdate = true
	}

	if settings.Kaytu.Octopus.Namespace != cfg.KaytuOctopusNamespace {
		settings.Kaytu.Octopus.Namespace = cfg.KaytuOctopusNamespace
		needsUpdate = true
	}

	if settings.Kaytu.Domain.App != cfg.AppDomain {
		settings.Kaytu.Domain.App = cfg.AppDomain
		needsUpdate = true
	}

	if settings.Kaytu.Domain.Grpc != cfg.GrpcDomain {
		settings.Kaytu.Domain.Grpc = cfg.GrpcDomain
		needsUpdate = true
	}

	if settings.Kaytu.Domain.GrpcExternal != cfg.GrpcExternalDomain {
		settings.Kaytu.Domain.GrpcExternal = cfg.GrpcExternalDomain
		needsUpdate = true
	}

	if settings.Kaytu.OpenSearch.IngestionPipelineEndpoint != workspace.PipelineEndpoint {
		settings.Kaytu.OpenSearch.IngestionPipelineEndpoint = workspace.PipelineEndpoint
		needsUpdate = true
	}

	if settings.Kaytu.Workspace.Name != workspace.Name {
		settings.Kaytu.Workspace.Name = workspace.Name
		needsUpdate = true
	}

	if settings.Vault.Provider != cfg.Vault.Provider {
		settings.Vault.Provider = cfg.Vault.Provider
		needsUpdate = true
	}
	if settings.Vault.AWS.RoleArn != cfg.Vault.Aws.RoleArn {
		settings.Vault.AWS.RoleArn = cfg.Vault.Aws.RoleArn
		needsUpdate = true
	}
	if settings.Vault.AWS.AccessKey != cfg.Vault.Aws.AccessKey {
		settings.Vault.AWS.AccessKey = cfg.Vault.Aws.AccessKey
		needsUpdate = true
	}
	if settings.Vault.AWS.SecretKey != cfg.Vault.Aws.SecretKey {
		settings.Vault.AWS.SecretKey = cfg.Vault.Aws.SecretKey
		needsUpdate = true
	}
	if settings.Vault.AWS.Region != cfg.Vault.Aws.Region {
		settings.Vault.AWS.Region = cfg.Vault.Aws.Region
		needsUpdate = true
	}

	if settings.Vault.Azure.BaseUrl != cfg.Vault.Azure.BaseUrl {
		settings.Vault.Azure.BaseUrl = cfg.Vault.Azure.BaseUrl
		needsUpdate = true
	}
	if settings.Vault.Azure.ClientId != cfg.Vault.Azure.ClientId {
		settings.Vault.Azure.ClientId = cfg.Vault.Azure.ClientId
		needsUpdate = true
	}
	if settings.Vault.Azure.TenantId != cfg.Vault.Azure.TenantId {
		settings.Vault.Azure.TenantId = cfg.Vault.Azure.TenantId
		needsUpdate = true
	}
	if settings.Vault.Azure.ClientSecret != cfg.Vault.Azure.ClientSecret {
		settings.Vault.Azure.ClientSecret = cfg.Vault.Azure.ClientSecret
		needsUpdate = true
	}
	if settings.Vault.KeyID != workspace.VaultKeyId {
		settings.Vault.KeyID = workspace.VaultKeyId
		needsUpdate = true
	}

	if workspace.AWSUserARN != nil && settings.Kaytu.Workspace.UserARN != *workspace.AWSUserARN {
		settings.Kaytu.Workspace.UserARN = *workspace.AWSUserARN
		needsUpdate = true
	}

	if workspace.AWSUniqueId != nil {
		masterCred, err := db.GetMasterCredentialByWorkspaceUID(*workspace.AWSUniqueId)
		if err != nil {
			return false, nil, err
		}

		result, err := vault.Decrypt(ctx, masterCred.Credential)
		if err != nil {
			return false, nil, fmt.Errorf("failed to encrypt ciphertext: %v", err)
		}
		jsonResult, err := json.Marshal(result)
		if err != nil {
			return false, nil, err
		}
		var accessKey types2.AccessKey
		err = json.Unmarshal(jsonResult, &accessKey)
		if err != nil {
			return false, nil, err
		}

		if settings.Kaytu.Workspace.MasterAccessKey != *accessKey.AccessKeyId {
			settings.Kaytu.Workspace.MasterAccessKey = *accessKey.AccessKeyId
			settings.Kaytu.Workspace.MasterSecretKey = *accessKey.SecretAccessKey
			needsUpdate = true
		}
	}

	return needsUpdate, settings, nil
}
