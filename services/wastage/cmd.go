package wastage

import (
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/alitto/pond"
	"github.com/kaytu-io/kaytu-engine/pkg/httpserver"
	"github.com/kaytu-io/kaytu-engine/services/wastage/api"
	"github.com/kaytu-io/kaytu-engine/services/wastage/api/wastage"
	"github.com/kaytu-io/kaytu-engine/services/wastage/config"
	"github.com/kaytu-io/kaytu-engine/services/wastage/cost"
	"github.com/kaytu-io/kaytu-engine/services/wastage/db/connector"
	"github.com/kaytu-io/kaytu-engine/services/wastage/db/model"
	"github.com/kaytu-io/kaytu-engine/services/wastage/db/repo"
	"github.com/kaytu-io/kaytu-engine/services/wastage/ingestion"
	"github.com/kaytu-io/kaytu-engine/services/wastage/recommendation"
	"github.com/kaytu-io/kaytu-util/pkg/koanf"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	logger2 "gorm.io/gorm/logger"
	"os"
	"time"
)

var (
	AuthGRPCURI    = os.Getenv("AUTH_GRPC_URI")
	GCPProjectID   = os.Getenv("GCP_PROJECT_ID")
	GCPPrivateKey  = os.Getenv("GCP_PRIVATE_KEY")
	GCPClientEmail = os.Getenv("GCP_CLIENT_EMAIL")
)

func Command() *cobra.Command {
	cnf := koanf.Provide("wastage", config.WastageConfig{})

	cmd := &cobra.Command{
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			logger, err := zap.NewProduction()
			if err != nil {
				return err
			}

			logger = logger.Named("wastage")

			cmd.SilenceUsage = true

			if cnf.Http.Address == "" {
				cnf.Http.Address = "localhost:8000"
				cnf.Pennywise.BaseURL = "http://localhost:8080"
				cnf.Postgres.Host = "localhost"
				cnf.Postgres.Port = "5432"
				cnf.Postgres.Username = "postgres"
				cnf.Postgres.Password = ""
				cnf.Postgres.DB = ""
			}
			db, err := connector.New(cnf.Postgres, logger, logger2.Info)
			if err != nil {
				return err
			}
			usageDb, err := connector.New(cnf.Postgres, logger, logger2.Warn)
			if err != nil {
				return err
			}
			// create citext extension if not exists
			err = db.Conn().Exec("CREATE EXTENSION IF NOT EXISTS citext").Error
			if err != nil {
				logger.Error("failed to create citext extension", zap.Error(err))
				return err
			}
			err = db.Conn().AutoMigrate(&model.DataAge{}, &model.Usage{}, &model.User{}, &model.Organization{})

			err = usageDb.Conn().AutoMigrate(&model.Usage{}, &model.UsageV2{})
			if err != nil {
				logger.Error("failed to auto migrate", zap.Error(err))
				return err
			}
			ec2InstanceRepo := repo.NewEC2InstanceTypeRepo(db)
			rdsRepo := repo.NewRDSProductRepo(db)
			rdsInstanceRepo := repo.NewRDSDBInstanceRepo(db)
			rdsStorageRepo := repo.NewRDSDBStorageRepo(logger, db)
			ebsVolumeRepo := repo.NewEBSVolumeTypeRepo(db)
			computeMachineTypeRepo := repo.NewGCPComputeMachineTypeRepo(db)
			computeDiskTypeRepo := repo.NewGCPComputeDiskTypeRepo(db)
			computeSKURepo := repo.NewGCPComputeSKURepo(db)
			dataAgeRepo := repo.NewDataAgeRepo(db)
			usageV2Repo := repo.NewUsageV2Repo(usageDb)
			usageV1Repo := repo.NewUsageRepo(usageDb)
			userRepo := repo.NewUserRepo(db)
			orgRepo := repo.NewOrganizationRepo(db)
			costSvc := cost.New(cnf.Pennywise.BaseURL)

			cred, err := azidentity.NewDefaultAzureCredential(&azidentity.DefaultAzureCredentialOptions{TenantID: cnf.AzBlob.TenantID})
			if err != nil {
				logger.Error("failed to create azure credential", zap.Error(err))
				return err
			}

			blobClient, err := azblob.NewClient(cnf.AzBlob.AccountUrl, cred, nil)
			if err != nil {
				logger.Error("failed to create blob client", zap.Error(err))
				return err
			}

			recomSvc := recommendation.New(logger, ec2InstanceRepo, ebsVolumeRepo, rdsInstanceRepo, rdsStorageRepo, computeMachineTypeRepo, computeDiskTypeRepo, computeSKURepo, cnf.OpenAIToken, costSvc)
			ingestionSvc := ingestion.New(logger, db, ec2InstanceRepo, rdsRepo, rdsInstanceRepo, rdsStorageRepo, ebsVolumeRepo, dataAgeRepo)

			gcpCredentials := map[string]string{
				"type":         "service_account",
				"project_id":   GCPProjectID,
				"private_key":  GCPPrivateKey,
				"client_email": GCPClientEmail,
			}
			gcpIngestionSvc, err := ingestion.NewGcpService(ctx, logger, dataAgeRepo, computeMachineTypeRepo, computeDiskTypeRepo, computeSKURepo, db, gcpCredentials, GCPProjectID)
			go ingestionSvc.Start(ctx)
			go gcpIngestionSvc.Start(ctx)

			blobWorkerPool := pond.New(50, 1000000,
				pond.Strategy(pond.Eager()),
				pond.Context(ctx),
				pond.IdleTimeout(10*time.Second),
				pond.MinWorkers(1))

			grpcServer := wastage.NewServer(logger, cnf, blobClient, blobWorkerPool, usageV2Repo, recomSvc)
			err = wastage.StartGrpcServer(grpcServer, cnf.Grpc.Address, AuthGRPCURI)
			if err != nil {
				return err
			}

			return httpserver.RegisterAndStart(
				ctx,
				logger,
				cnf.Http.Address,
				api.New(cnf, blobClient, blobWorkerPool, costSvc, recomSvc, ingestionSvc, usageV1Repo, usageV2Repo, userRepo, orgRepo, logger),
			)
		},
	}

	return cmd
}
