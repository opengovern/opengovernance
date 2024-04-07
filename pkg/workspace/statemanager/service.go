package statemanager

import (
	"context"
	"errors"
	"fmt"
	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	"github.com/aws/aws-sdk-go-v2/service/opensearch"
	"github.com/aws/aws-sdk-go-v2/service/osis"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	helmv2 "github.com/fluxcd/helm-controller/api/v2beta1"
	aws2 "github.com/kaytu-io/kaytu-aws-describer/aws"
	authclient "github.com/kaytu-io/kaytu-engine/pkg/auth/client"
	workspaceConfig "github.com/kaytu-io/kaytu-engine/pkg/workspace/config"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/db"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/transactions"
	"github.com/kaytu-io/kaytu-util/pkg/config"
	contourv1 "github.com/projectcontour/contour/apis/projectcontour/v1"
	"go.uber.org/zap"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"runtime/debug"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

const (
	reconcilerInterval = 30 * time.Second
)

type Service struct {
	cfg        workspaceConfig.Config
	logger     *zap.Logger
	db         *db.Database
	kmsClient  *kms.Client
	authClient authclient.AuthServiceClient
	kubeClient k8sclient.Client // the kubernetes client
	opensearch *opensearch.Client
	osis       *osis.Client
	iam        *iam.Client
	iamMaster  *iam.Client
	s3Client   *s3.Client
}

func New(ctx context.Context, cfg workspaceConfig.Config) (*Service, error) {
	logger, err := zap.NewProduction()
	if err != nil {
		return nil, fmt.Errorf("new zap logger: %s", err)
	}

	dbs, err := db.NewDatabase(cfg, logger)
	if err != nil {
		return nil, fmt.Errorf("new database: %w", err)
	}

	authClient := authclient.NewAuthServiceClient(cfg.Auth.BaseURL)

	kubeClient, err := NewKubeClient()
	if err != nil {
		return nil, fmt.Errorf("new kube client: %w", err)
	}

	err = contourv1.AddToScheme(kubeClient.Scheme())
	if err != nil {
		return nil, fmt.Errorf("add contourv1 to scheme: %w", err)
	}

	err = v1.AddToScheme(kubeClient.Scheme())
	if err != nil {
		return nil, fmt.Errorf("add v1 to scheme: %w", err)
	}

	awsConfigMaster, err := aws2.GetConfig(ctx, cfg.AWSMasterAccessKey, cfg.AWSMasterSecretKey, "", "", nil)
	if err != nil {
		return nil, err
	}
	iamClientMaster := iam.NewFromConfig(awsConfigMaster)

	awsCfg, err := awsConfig.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load SDK configuration: %v", err)
	}

	awsCfg.Region = cfg.KMSAccountRegion
	kmsClient := kms.NewFromConfig(awsCfg)

	awsCfg.Region = cfg.OpenSearchRegion
	openSearchClient := opensearch.NewFromConfig(awsCfg)
	osisClient := osis.NewFromConfig(awsCfg)

	iamClient := iam.NewFromConfig(awsCfg)

	awsConfig, err := aws2.GetConfig(ctx, cfg.S3AccessKey, cfg.S3SecretKey, "", "", nil)
	if err != nil {
		return nil, err
	}
	awsConfig.Region = "us-east-1"
	s3Client := s3.NewFromConfig(awsConfig)

	return &Service{
		logger:     logger,
		cfg:        cfg,
		db:         dbs,
		authClient: authClient,
		kubeClient: kubeClient,
		kmsClient:  kmsClient,
		iam:        iamClient,
		iamMaster:  iamClientMaster,
		s3Client:   s3Client,
		opensearch: openSearchClient,
		osis:       osisClient,
	}, nil
}

func (s *Service) StartReconciler(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("%s", string(debug.Stack()))
			fmt.Printf("reconciler crashed: %v, restarting ...\n", r)
			go s.StartReconciler(ctx)
		}
	}()

	ticker := time.NewTimer(reconcilerInterval)
	defer ticker.Stop()

	for range ticker.C {
		fmt.Printf("reconsiler started\n")

		workspaces, err := s.db.ListWorkspaces()
		if err != nil {
			s.logger.Error(fmt.Sprintf("list workspaces: %v", err))
		} else {
			for _, workspace := range workspaces {
				if err := s.handleTransition(workspace); err != nil {
					if !errors.Is(err, transactions.ErrTransactionNeedsTime) {
						s.logger.Error(fmt.Sprintf("handle workspace %s: %v", workspace.ID, err))
					}
				}
			}

			if err := s.syncHTTPProxy(ctx, workspaces); err != nil {
				s.logger.Error(fmt.Sprintf("syncing http proxy: %v", err))
			}

			if err := s.syncHelmValues(ctx, workspaces); err != nil {
				s.logger.Error(fmt.Sprintf("syncing helm values: %v", err))
			}
		}
		if s.cfg.EnvType == config.EnvTypeProd && s.cfg.DoReserve {
			err = s.handleReservation()
			if err != nil {
				s.logger.Error(fmt.Sprintf("reservation: %v", err))
			}
		}

		// reset the time ticker
		ticker.Reset(reconcilerInterval)
	}
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
