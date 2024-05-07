package wastage

import (
	"github.com/kaytu-io/kaytu-engine/pkg/httpserver"
	"github.com/kaytu-io/kaytu-engine/services/wastage/api"
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
			db, err := connector.New(cnf.Postgres, logger2.Info)
			if err != nil {
				return err
			}
			usageDb, err := connector.New(cnf.Postgres, logger2.Warn)
			if err != nil {
				return err
			}
			// create citext extension if not exists
			err = db.Conn().Exec("CREATE EXTENSION IF NOT EXISTS citext").Error
			if err != nil {
				logger.Error("failed to create citext extension", zap.Error(err))
				return err
			}
			err = db.Conn().AutoMigrate(&model.EC2InstanceType{}, &model.EBSVolumeType{}, &model.DataAge{}, &model.Usage{},
				&model.RDSDBInstance{}, &model.RDSDBStorage{}, &model.RDSProduct{})
			if err != nil {
				logger.Error("failed to auto migrate", zap.Error(err))
				return err
			}
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
			dataAgeRepo := repo.NewDataAgeRepo(db)
			usageV2Repo := repo.NewUsageV2Repo(usageDb)
			usageV1Repo := repo.NewUsageRepo(usageDb)
			costSvc := cost.New(cnf.Pennywise.BaseURL)
			recomSvc := recommendation.New(logger, ec2InstanceRepo, ebsVolumeRepo, rdsInstanceRepo, rdsStorageRepo, cnf.OpenAIToken, costSvc)
			ingestionSvc := ingestion.New(logger, db, ec2InstanceRepo, rdsRepo, rdsInstanceRepo, rdsStorageRepo, ebsVolumeRepo, dataAgeRepo)
			go ingestionSvc.Start(ctx)

			return httpserver.RegisterAndStart(
				ctx,
				logger,
				cnf.Http.Address,
				api.New(costSvc, recomSvc, ingestionSvc, usageV1Repo, usageV2Repo, logger),
			)
		},
	}

	return cmd
}
