package integration

import (
	"errors"
	"fmt"
	helmv2 "github.com/fluxcd/helm-controller/api/v2beta1"
	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	integration_type "github.com/opengovern/opengovernance/services/integration/integration-type"
	"github.com/opengovern/opengovernance/services/integration/models"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	api3 "github.com/opengovern/og-util/pkg/api"
	"github.com/opengovern/og-util/pkg/httpclient"
	"github.com/opengovern/og-util/pkg/httpserver"
	"github.com/opengovern/og-util/pkg/koanf"
	"github.com/opengovern/og-util/pkg/postgres"
	"github.com/opengovern/og-util/pkg/steampipe"
	"github.com/opengovern/og-util/pkg/vault"
	"github.com/opengovern/opengovernance/services/integration/api"
	"github.com/opengovern/opengovernance/services/integration/config"
	"github.com/opengovern/opengovernance/services/integration/db"
	metadata "github.com/opengovern/opengovernance/services/metadata/client"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

func Command() *cobra.Command {
	cnf := koanf.Provide("integration", config.IntegrationConfig{})

	cmd := &cobra.Command{
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()

			logger, err := zap.NewProduction()
			if err != nil {
				return err
			}

			logger = logger.Named("integration")
			cfg := postgres.Config{
				Host:    cnf.Postgres.Host,
				Port:    cnf.Postgres.Port,
				User:    cnf.Postgres.Username,
				Passwd:  cnf.Postgres.Password,
				DB:      cnf.Postgres.DB,
				SSLMode: cnf.Postgres.SSLMode,
			}
			gorm, err := postgres.NewClient(&cfg, logger.Named("postgres"))
			db := db.NewDatabase(gorm)
			if err != nil {
				return err
			}

			err = db.Initialize()
			if err != nil {
				return err
			}

			mClient := metadata.NewMetadataServiceClient(cnf.Metadata.BaseURL)

			_, err = mClient.VaultConfigured(&httpclient.Context{UserRole: api3.AdminRole})
			if err != nil && errors.Is(err, metadata.ErrConfigNotFound) {
				return err
			}

			var vaultSc vault.VaultSourceConfig
			switch cnf.Vault.Provider {
			case vault.AwsKMS:
				vaultSc, err = vault.NewKMSVaultSourceConfig(ctx, cnf.Vault.Aws, cnf.Vault.KeyId)
				if err != nil {
					logger.Error("failed to create vault source config", zap.Error(err))
					return err
				}
			case vault.AzureKeyVault:
				vaultSc, err = vault.NewAzureVaultClient(ctx, logger, cnf.Vault.Azure, cnf.Vault.KeyId)
				if err != nil {
					logger.Error("failed to create vault source config", zap.Error(err))
					return err
				}
			case vault.HashiCorpVault:
				vaultSc, err = vault.NewHashiCorpVaultClient(ctx, logger, cnf.Vault.HashiCorp, cnf.Vault.KeyId)
				if err != nil {
					logger.Error("failed to create vault source config", zap.Error(err))
					return err
				}
			}

			cmd.SilenceUsage = true

			steampipeConn, err := steampipe.NewSteampipeDatabase(steampipe.Option{
				Host: cnf.Steampipe.Host,
				Port: cnf.Steampipe.Port,
				User: cnf.Steampipe.Username,
				Pass: cnf.Steampipe.Password,
				Db:   cnf.Steampipe.DB,
			})
			if err != nil {
				return fmt.Errorf("new steampipe client: %w", err)
			}
			logger.Info("Connected to the steampipe database", zap.String("database", cnf.Steampipe.DB))
			kubeClient, err := NewKubeClient()
			if err != nil {
				return err
			}

			for name, _ := range integration_type.IntegrationTypes {
				enabled := false
				if name == integration_type.IntegrationTypeAWSAccount || name == integration_type.IntegrationTypeGithubAccount {
					enabled = true
				}
				err := db.CreateIntegrationTypeSetup(&models.IntegrationTypeSetup{
					IntegrationType: name,
					Enabled:         enabled,
				})
				if err != nil {
					return err
				}
			}

			return httpserver.RegisterAndStart(
				cmd.Context(),
				logger,
				cnf.Http.Address,
				api.New(logger, db, vaultSc, steampipeConn, kubeClient),
			)
		},
	}

	return cmd
}

func NewKubeClient() (client.Client, error) {
	scheme := runtime.NewScheme()
	if err := helmv2.AddToScheme(scheme); err != nil {
		return nil, err
	}
	if err := corev1.AddToScheme(scheme); err != nil {
		return nil, err
	}
	if err := v1.AddToScheme(scheme); err != nil {
		return nil, err
	}
	if err := kedav1alpha1.AddToScheme(scheme); err != nil {
		return nil, err
	}
	if err := appsv1.AddToScheme(scheme); err != nil {
		return nil, err
	}

	kubeClient, err := client.New(ctrl.GetConfigOrDie(), client.Options{Scheme: scheme})
	if err != nil {
		return nil, err
	}
	return kubeClient, nil
}
