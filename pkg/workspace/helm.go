package workspace

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"reflect"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3Types "github.com/aws/aws-sdk-go-v2/service/s3/types"

	helmv2 "github.com/fluxcd/helm-controller/api/v2beta1"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type KaytuWorkspaceSettings struct {
	Kaytu KaytuConfig `json:"kaytu"`
}
type KaytuConfig struct {
	ReplicaCount int             `json:"replicaCount"`
	Workspace    WorkspaceConfig `json:"workspace"`
	Docker       DockerConfig    `json:"docker"`
	Insights     InsightsConfig  `json:"insights"`
}
type InsightsConfig struct {
	S3 S3Config `json:"s3"`
}
type S3Config struct {
	AccessKey string `json:"accessKey"`
	SecretKey string `json:"secretKey"`
}
type DockerConfig struct {
	Config string `json:"config"`
}
type WorkspaceConfig struct {
	Name string `json:"name"`
}

func (s *Server) newKubeClient() (client.Client, error) {
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

func (s *Server) createInsightBucket(ctx context.Context, workspace *Workspace) error {
	cli := s3.NewFromConfig(s.awsConfig)
	_, err := cli.CreateBucket(ctx, &s3.CreateBucketInput{
		Bucket: aws.String(fmt.Sprintf("insights-%s", workspace.ID)),
	})
	var bucketAlreadyExists *s3Types.BucketAlreadyExists
	if errors.As(err, &bucketAlreadyExists) {
		return nil
	}
	return err
}

func (s *Server) createHelmRelease(ctx context.Context, workspace *Workspace) error {
	id := workspace.ID

	if err := s.createInsightBucket(ctx, workspace); err != nil {
		return err
	}

	settings := KaytuWorkspaceSettings{
		Kaytu: KaytuConfig{
			ReplicaCount: 1,
			Workspace: WorkspaceConfig{
				Name: workspace.Name,
			},
			Insights: InsightsConfig{
				S3: S3Config{
					AccessKey: s.cfg.S3AccessKey,
					SecretKey: s.cfg.S3SecretKey,
				},
			},
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
			Name:      id,
			Namespace: s.cfg.FluxSystemNamespace,
		},
		Spec: helmv2.HelmReleaseSpec{
			Interval: metav1.Duration{
				Duration: 5 + time.Minute,
			},
			TargetNamespace: id,
			ReleaseName:     id,
			Chart: helmv2.HelmChartTemplate{
				Spec: helmv2.HelmChartTemplateSpec{
					Chart: s.cfg.KaytuHelmChartLocation,
					SourceRef: helmv2.CrossNamespaceObjectReference{
						Kind:      "GitRepository",
						Name:      "flux-system",
						Namespace: s.cfg.FluxSystemNamespace,
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

func getReplicaCount(values map[string]interface{}) (int, error) {
	if v, ok := values["kaytu"]; ok {
		if vm, ok := v.(map[string]interface{}); ok {
			if v, ok := vm["replicaCount"]; ok {
				if c, ok := v.(float64); ok {
					return int(c), nil
				} else {
					return 0, fmt.Errorf("invalid replicaCount type: %v", reflect.TypeOf(v))
				}
			} else {
				return 1, nil // default
			}
		} else {
			return 0, fmt.Errorf("invalid kaytu type: %v", reflect.TypeOf(v))
		}
	} else {
		return 0, fmt.Errorf("kaytu not found")
	}
}

func updateValuesSetReplicaCount(values map[string]interface{}, replicaCount int) (map[string]interface{}, error) {
	if v, ok := values["kaytu"]; ok {
		if vm, ok := v.(map[string]interface{}); ok {
			vm["replicaCount"] = replicaCount
			values["kaytu"] = vm
			return values, nil
		} else {
			return nil, fmt.Errorf("invalid kaytu type: %v", reflect.TypeOf(v))
		}
	} else {
		return nil, fmt.Errorf("kaytu not found")
	}
}

func (s *Server) deleteTargetNamespace(ctx context.Context, name string) error {
	ns := corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
	return s.kubeClient.Delete(ctx, &ns)
}

func (s *Server) createTargetNamespace(ctx context.Context, name string) error {
	ns := corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
	return s.kubeClient.Create(ctx, &ns)
}

func (s *Server) findTargetNamespace(ctx context.Context, name string) (*corev1.Namespace, error) {
	key := client.ObjectKey{
		Name: name,
	}
	var ns corev1.Namespace
	if err := s.kubeClient.Get(ctx, key, &ns); err != nil {
		if apierrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("find target namespace: %w", err)
	}
	return &ns, nil
}

func (s *Server) findHelmRelease(ctx context.Context, workspace *Workspace) (*helmv2.HelmRelease, error) {
	key := types.NamespacedName{
		Name:      workspace.ID,
		Namespace: s.cfg.FluxSystemNamespace,
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
			Name:      workspace.ID,
			Namespace: s.cfg.FluxSystemNamespace,
		},
	}
	return s.kubeClient.Delete(ctx, &helmRelease)
}
