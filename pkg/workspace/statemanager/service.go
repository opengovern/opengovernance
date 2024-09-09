package statemanager

import (
	"context"
	"errors"
	"fmt"
	authclient "github.com/kaytu-io/kaytu-engine/pkg/auth/client"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/api"
	workspaceConfig "github.com/kaytu-io/kaytu-engine/pkg/workspace/config"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/db"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/transactions"
	api2 "github.com/kaytu-io/kaytu-util/pkg/api"
	"github.com/kaytu-io/kaytu-util/pkg/httpclient"
	"github.com/kaytu-io/kaytu-util/pkg/vault"
	"github.com/sony/sonyflake"
	"go.uber.org/zap"
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
	vault      vault.VaultSourceConfig
	authClient authclient.AuthServiceClient
	kubeClient k8sclient.Client // the kubernetes client

	vaultSecretHandler vault.VaultSecretHandler
}

func New(ctx context.Context, cfg workspaceConfig.Config,
	vaultClient vault.VaultSourceConfig,
	vaultSecretHandler vault.VaultSecretHandler,
	dbs *db.Database,
	kubeClient k8sclient.Client,
) (*Service, error) {
	logger, err := zap.NewProduction()
	if err != nil {
		return nil, fmt.Errorf("new zap logger: %s", err)
	}

	authClient := authclient.NewAuthServiceClient(cfg.Auth.BaseURL)

	//awsConfig, err := aws2.GetConfig(ctx, cfg.S3AccessKey, cfg.S3SecretKey, "", "", nil)
	//if err != nil {
	//	return nil, fmt.Errorf("failed to load aws config: %v", err)
	//}
	//awsConfig.Region = "us-east-1"
	//s3Client := s3.NewFromConfig(awsConfig)

	return &Service{
		logger:             logger,
		cfg:                cfg,
		db:                 dbs,
		authClient:         authClient,
		kubeClient:         kubeClient,
		vault:              vaultClient,
		vaultSecretHandler: vaultSecretHandler,
	}, nil
}

func (s *Service) CreateWorkspace(ctx context.Context) error {
	sf := sonyflake.NewSonyflake(sonyflake.Settings{})
	id, err := sf.NextID()
	if err != nil {
		return err
	}

	ownerAll := "kaytu|owner|all"
	workspace := &db.Workspace{
		ID:                       fmt.Sprintf("ws-%d", id),
		Name:                     "main",
		OwnerId:                  &ownerAll,
		Status:                   api.StateID_Provisioning,
		Size:                     api.SizeXS,
		Tier:                     api.Tier_Free,
		OrganizationID:           nil,
		IsCreated:                false,
		IsBootstrapInputFinished: true,
		AnalyticsJobID:           0,
		ComplianceTriggered:      false,
	}

	if err := s.db.CreateWorkspace(workspace); err != nil {
		return err
	}

	for _, tr := range []api.TransactionID{
		api.Transaction_CreateWorkspaceKeyId,
		api.Transaction_EnsureWorkspacePodsRunning,
		api.Transaction_EnsureDiscoveryFinished,
		api.Transaction_EnsureJobsRunning, api.Transaction_EnsureJobsFinished,
		api.Transaction_CreateRoleBinding} {
		err := s.db.CreateWorkspaceTransaction(&db.WorkspaceTransaction{
			WorkspaceID:   workspace.ID,
			TransactionID: tr,
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
			Done:          true,
		})
		if err != nil {
			return err
		}
	}

	err = s.authClient.UpdateWorkspaceMap(&httpclient.Context{UserRole: api2.InternalRole})
	if err != nil {
		return err
	}

	return nil
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
				if err := s.handleTransition(ctx, workspace); err != nil {
					if !errors.Is(err, transactions.ErrTransactionNeedsTime) {
						s.logger.Error(fmt.Sprintf("handle workspace %s: %v", workspace.ID, err))
					}
				}
			}

			if len(workspaces) == 0 {
				if err := s.CreateWorkspace(ctx); err != nil {
					s.logger.Error(fmt.Sprintf("creating workspace if empty: %v", err))
				}
			}
		}

		// reset the time ticker
		ticker.Reset(reconcilerInterval)
	}
}

func NewKubeClient() (client.Client, error) {
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		return nil, err
	}
	kubeClient, err := client.New(ctrl.GetConfigOrDie(), client.Options{Scheme: scheme})
	if err != nil {
		return nil, err
	}
	return kubeClient, nil
}
