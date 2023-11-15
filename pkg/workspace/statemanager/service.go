package statemanager

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/go-redis/cache/v8"
	"github.com/go-redis/redis/v8"
	aws2 "github.com/kaytu-io/kaytu-aws-describer/aws"
	authclient "github.com/kaytu-io/kaytu-engine/pkg/auth/client"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/config"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/db"
	"github.com/kaytu-io/kaytu-util/pkg/vault"
	contourv1 "github.com/projectcontour/contour/apis/projectcontour/v1"
	"go.uber.org/zap"
	v1 "k8s.io/api/apps/v1"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

const (
	reconcilerInterval = 30 * time.Second
)

type Service struct {
	cfg        config.Config
	logger     *zap.Logger
	db         *db.Database
	kms        *vault.KMSVaultSourceConfig
	authClient authclient.AuthServiceClient
	kubeClient k8sclient.Client // the kubernetes client
	rdb        *redis.Client
	cache      *cache.Cache
	awsConfig  aws.Config
}

func New(cfg config.Config) (*Service, error) {
	logger, err := zap.NewProduction()
	if err != nil {
		return nil, fmt.Errorf("new zap logger: %s", err)
	}

	dbs, err := db.NewDatabase(cfg, logger)
	if err != nil {
		return nil, fmt.Errorf("new database: %w", err)
	}

	authClient := authclient.NewAuthServiceClient(cfg.Auth.BaseURL)

	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Address,
		Password: "", // no password set
		DB:       0,  // use default DB
	})

	cache := cache.New(&cache.Options{
		Redis:      rdb,
		LocalCache: cache.NewTinyLFU(2000, 1*time.Minute),
	})

	kms, err := vault.NewKMSVaultSourceConfig(context.Background(), "", "", cfg.KMSAccountRegion)
	if err != nil {
		return nil, err
	}

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

	awsConfig, err := aws2.GetConfig(context.Background(), cfg.S3AccessKey, cfg.S3SecretKey, "", "", nil)
	if err != nil {
		return nil, err
	}

	return &Service{
		logger:     logger,
		cfg:        cfg,
		db:         dbs,
		kms:        kms,
		authClient: authClient,
		kubeClient: kubeClient,
		rdb:        rdb,
		cache:      cache,
		awsConfig:  awsConfig,
	}, nil
}

func (s *Service) StartReconciler() {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("reconciler crashed: %v, restarting ...\n", r)
			go s.StartReconciler()
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
				if err := s.handleWorkspace(workspace); err != nil {
					s.logger.Error(fmt.Sprintf("handle workspace %s: %v", workspace.ID, err))
				}

				if err := s.handleAutoSuspend(workspace); err != nil {
					s.logger.Error(fmt.Sprintf("handleAutoSuspend: %v", err))
				}
			}

			if err := s.syncHTTPProxy(workspaces); err != nil {
				s.logger.Error(fmt.Sprintf("syncing http proxy: %v", err))
			}
		}

		err = s.handleReservation()
		if err != nil {
			s.logger.Error(fmt.Sprintf("reservation: %v", err))
		}
		// reset the time ticker
		ticker.Reset(reconcilerInterval)
	}
}
