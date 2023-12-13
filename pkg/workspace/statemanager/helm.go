package statemanager

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	types2 "github.com/aws/aws-sdk-go-v2/service/iam/types"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	kms2 "github.com/aws/aws-sdk-go/service/kms"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/api"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/db"
	"strings"
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
	ReplicaCount int              `json:"replicaCount"`
	Workspace    WorkspaceConfig  `json:"workspace"`
	Docker       DockerConfig     `json:"docker"`
	Insights     InsightsConfig   `json:"insights"`
	OpenSearch   OpenSearchConfig `json:"opensearch"`
}
type OpenSearchConfig struct {
	Enabled bool   `json:"enabled"`
	ARN     string `json:"arn"`
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
	Name            string            `json:"name"`
	Size            api.WorkspaceSize `json:"size"`
	UserARN         string            `json:"userARN"`
	MasterAccessKey string            `json:"masterAccessKey"`
	MasterSecretKey string            `json:"masterSecretKey"`
}

func NewKubeClient() (client.Client, error) {
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

func (s *Service) createInsightBucket(ctx context.Context, workspace *db.Workspace) error {
	s.awsConfig.Region = "us-east-1"
	cli := s3.NewFromConfig(s.awsConfig)
	_, err := cli.CreateBucket(ctx, &s3.CreateBucketInput{
		Bucket: aws.String(fmt.Sprintf("insights-%s", workspace.ID)),
		//CreateBucketConfiguration: &s3Types.CreateBucketConfiguration{
		//	LocationConstraint: s3Types.BucketLocationConstraintUsGovEast1,
		//},
	})
	var bucketAlreadyExists *s3Types.BucketAlreadyExists
	if errors.As(err, &bucketAlreadyExists) {
		return nil
	}
	return err
}

func (s *Service) deleteInsightBucket(ctx context.Context, workspace *db.Workspace) error {
	bucketName := fmt.Sprintf("insights-%s", workspace.ID)
	s.awsConfig.Region = "us-east-1"
	cli := s3.NewFromConfig(s.awsConfig)
	objects, err := cli.ListObjects(ctx, &s3.ListObjectsInput{
		Bucket: aws.String(bucketName),
	})
	if err != nil {
		var noSuchBucket *s3Types.NoSuchBucket
		if errors.As(err, &noSuchBucket) {
			return nil
		}
		return err
	}

	var objs []s3Types.ObjectIdentifier
	for _, obj := range objects.Contents {
		objs = append(objs, s3Types.ObjectIdentifier{
			Key: obj.Key,
		})
	}
	if len(objs) > 0 {
		_, err = cli.DeleteObjects(ctx, &s3.DeleteObjectsInput{
			Bucket: aws.String(bucketName),
			Delete: &s3Types.Delete{
				Objects: objs,
			},
		})
		if err != nil {
			return err
		}
	}

	_, err = cli.DeleteBucket(ctx, &s3.DeleteBucketInput{
		Bucket: aws.String(bucketName),
	})
	return err
}

func (s *Service) createHelmRelease(ctx context.Context, workspace *db.Workspace) error {
	id := workspace.ID

	if err := s.createInsightBucket(ctx, workspace); err != nil {
		return err
	}

	processing, arn, err := s.isOpenSearchCreationFinished(workspace)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			if err := s.createOpenSearch(workspace); err != nil {
				return err
			}
			return nil
		}
		return err
	}

	if processing {
		return nil
	}

	var userARN string
	if workspace.AWSUserARN != nil {
		userARN = *workspace.AWSUserARN
	}

	settings := KaytuWorkspaceSettings{
		Kaytu: KaytuConfig{
			ReplicaCount: 1,
			Workspace: WorkspaceConfig{
				Name:    workspace.Name,
				Size:    workspace.Size,
				UserARN: userARN,
			},
			Insights: InsightsConfig{
				S3: S3Config{
					AccessKey: s.cfg.S3AccessKey,
					SecretKey: s.cfg.S3SecretKey,
				},
			},
			OpenSearch: OpenSearchConfig{
				Enabled: true,
				ARN:     arn,
			},
		},
	}
	if workspace.AWSUniqueId != nil {
		masterCred, err := s.db.GetMasterCredentialByWorkspaceUID(*workspace.AWSUniqueId)
		if err != nil {
			return err
		}

		decoded, err := base64.StdEncoding.DecodeString(masterCred.Credential)
		if err != nil {
			return err
		}

		result, err := s.kmsClient.Decrypt(context.TODO(), &kms.DecryptInput{
			CiphertextBlob:      decoded,
			EncryptionAlgorithm: kms2.EncryptionAlgorithmSpecSymmetricDefault,
			KeyId:               &s.cfg.KMSKeyARN,
			EncryptionContext:   nil,
		})
		if err != nil {
			return fmt.Errorf("failed to encrypt ciphertext: %v", err)
		}

		var accessKey types2.AccessKey
		err = json.Unmarshal(result.Plaintext, &accessKey)
		//err = json.Unmarshal([]byte(masterCred.Credential), &accessKey)
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

func (s *Service) deleteTargetNamespace(ctx context.Context, name string) error {
	ns := corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
	return s.kubeClient.Delete(ctx, &ns)
}

func (s *Service) createTargetNamespace(ctx context.Context, name string) error {
	ns := corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
	return s.kubeClient.Create(ctx, &ns)
}

func (s *Service) findTargetNamespace(ctx context.Context, name string) (*corev1.Namespace, error) {
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

func (s *Service) FindHelmRelease(ctx context.Context, workspace *db.Workspace) (*helmv2.HelmRelease, error) {
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

func (s *Service) deleteHelmRelease(ctx context.Context, workspace *db.Workspace) error {
	helmRelease := helmv2.HelmRelease{
		ObjectMeta: metav1.ObjectMeta{
			Name:      workspace.ID,
			Namespace: s.cfg.FluxSystemNamespace,
		},
	}
	return s.kubeClient.Delete(ctx, &helmRelease)
}
