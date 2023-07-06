package describe

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	helmv2 "github.com/fluxcd/helm-controller/api/v2beta1"
	"github.com/kaytu-io/kaytu-engine/pkg/describe/api"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type StackReleaseConfig struct {
	KafkaTopics KafkaTopics `json:"kafkaTopics"`
}

type KafkaTopics struct {
	Resources string `json:"resources"`
}

type HelmConfig struct {
	KeibiHelmChartLocation string
	FluxSystemNamespace    string
}

func (s *HttpServer) newKubeClient() (client.Client, error) {
	scheme := runtime.NewScheme()
	if err := helmv2.AddToScheme(scheme); err != nil {
		return nil, err
	}
	if err := corev1.AddToScheme(scheme); err != nil {
		return nil, err
	}
	kubeClient, err := client.New(ctrl.GetConfigOrDie(), client.Options{Scheme: scheme})
	if err != nil {
		return nil, err
	}
	return kubeClient, nil
}

func (s *HttpServer) createStackHelmRelease(ctx context.Context, workspaceId string, stack api.Stack) error {

	settings := StackReleaseConfig{
		KafkaTopics: KafkaTopics{
			Resources: stack.StackID,
		},
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
			Name:      stack.StackID,
			Namespace: workspaceId,
		},
		Spec: helmv2.HelmReleaseSpec{
			Interval: metav1.Duration{
				Duration: 5 + time.Minute,
			},
			TargetNamespace: workspaceId,
			ReleaseName:     stack.StackID,
			Chart: helmv2.HelmChartTemplate{
				Spec: helmv2.HelmChartTemplateSpec{
					Chart: s.helmConfig.KeibiHelmChartLocation,
					SourceRef: helmv2.CrossNamespaceObjectReference{
						Kind:      "GitRepository",
						Name:      "flux-system",
						Namespace: s.helmConfig.FluxSystemNamespace,
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
	if err := s.kubeClient.Create(ctx, &helmRelease); err != nil {
		return fmt.Errorf("create helm release: %w", err)
	}
	return nil
}

func (s *HttpServer) findHelmRelease(ctx context.Context, stack api.Stack, workspaceId string) (*helmv2.HelmRelease, error) {
	key := types.NamespacedName{
		Name:      stack.StackID,
		Namespace: workspaceId,
	}
	var helmRelease helmv2.HelmRelease
	if err := s.kubeClient.Get(ctx, key, &helmRelease); err != nil {
		if apierrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return &helmRelease, nil
}

func (s *HttpServer) deleteStackHelmRelease(stack api.Stack, workspaceId string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	helmRelease := helmv2.HelmRelease{
		ObjectMeta: metav1.ObjectMeta{
			Name:      stack.StackID,
			Namespace: workspaceId,
		},
	}
	return s.kubeClient.Delete(ctx, &helmRelease)
}
