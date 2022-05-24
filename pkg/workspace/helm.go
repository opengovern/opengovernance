package workspace

import (
	"context"
	"fmt"
	"time"

	helmv2 "github.com/fluxcd/helm-controller/api/v2beta1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	FluxSystemNamespace = "flux-system"
)

func (s *Server) newKubeClient() (client.Client, error) {
	scheme := runtime.NewScheme()
	if err := helmv2.AddToScheme(scheme); err != nil {
		return nil, err
	}
	kubeClient, err := client.New(ctrl.GetConfigOrDie(), client.Options{Scheme: scheme})
	if err != nil {
		return nil, err
	}
	return kubeClient, nil
}

func (s *Server) createHelmRelease(ctx context.Context, workspace *Workspace) error {
	id := workspace.ID.String()

	helmRelease := helmv2.HelmRelease{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "helm.toolkit.fluxcd.io/v2beta1",
			Kind:       "HelmRelease",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      id,
			Namespace: FluxSystemNamespace,
		},
		Spec: helmv2.HelmReleaseSpec{
			Interval: metav1.Duration{
				Duration: 5 + time.Minute,
			},
			TargetNamespace: id,
			ReleaseName:     id,
			Chart: helmv2.HelmChartTemplate{
				Spec: helmv2.HelmChartTemplateSpec{
					Chart: "./keibi",
					SourceRef: helmv2.CrossNamespaceObjectReference{
						Kind:      "GitRepository",
						Name:      "flux-system",
						Namespace: FluxSystemNamespace,
					},
					Interval: &metav1.Duration{
						Duration: time.Minute,
					},
					ReconcileStrategy: "Revision",
				},
			},
			Values: &apiextensionsv1.JSON{
				Raw: []byte(`{"domain": "` + workspace.Domain + `"}`),
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

func (s *Server) findHelmRelease(ctx context.Context, workspace *Workspace) (*helmv2.HelmRelease, error) {
	key := types.NamespacedName{
		Name:      workspace.ID.String(),
		Namespace: FluxSystemNamespace,
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

func (s *Server) deleteHelmRelease(ctx context.Context, workspace *Workspace) error {
	helmRelease := helmv2.HelmRelease{
		ObjectMeta: metav1.ObjectMeta{
			Name:      workspace.ID.String(),
			Namespace: FluxSystemNamespace,
		},
	}
	return s.kubeClient.Delete(ctx, &helmRelease)
}
