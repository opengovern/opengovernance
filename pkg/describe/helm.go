package describe

import (
	"encoding/json"
	"fmt"
	"time"

	helmv2 "github.com/fluxcd/helm-controller/api/v2beta1"
	"github.com/kaytu-io/kaytu-engine/pkg/internal/httpserver"
	"github.com/labstack/echo/v4"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
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

func (s *HttpServer) createStackHelmRelease(ctx echo.Context, stack Stack) error {
	settings := StackReleaseConfig{
		KafkaTopics: KafkaTopics{
			Resources: stack.StackID,
		},
	}
	settingsJSON, err := json.Marshal(settings)
	if err != nil {
		return err
	}
	workspaceId := httpserver.GetWorkspaceID(ctx)
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
	if err := s.kubeClient.Create(ctx.Request().Context(), &helmRelease); err != nil {
		return fmt.Errorf("create helm release: %w", err)
	}
	return nil
}

func (s *HttpServer) deleteStackHelmRelease(ctx echo.Context, stack Stack) error {
	helmRelease := helmv2.HelmRelease{
		ObjectMeta: metav1.ObjectMeta{
			Name:      stack.StackID,
			Namespace: s.helmConfig.FluxSystemNamespace,
		},
	}
	return s.kubeClient.Delete(ctx.Request().Context(), &helmRelease)
}
