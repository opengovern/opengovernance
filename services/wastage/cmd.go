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
			db, err := connector.New(cnf.Postgres)
			if err != nil {
				return err
			}
			err = db.Conn().AutoMigrate(&model.EC2InstanceType{}, &model.EBSVolumeType{}, &model.DataAge{}, &model.Usage{},
				&model.RDSDBInstance{}, &model.RDSDBStorage{}, &model.RDSProduct{})
			if err != nil {
				return err
			}
			ec2InstanceRepo := repo.NewEC2InstanceTypeRepo(db)
			rdsRepo := repo.NewRDSProductRepo(db)
			rdsInstanceRepo := repo.NewRDSDBInstanceRepo(db)
			rdsStorageRepo := repo.NewRDSDBStorageRepo(db)
			ebsVolumeRepo := repo.NewEBSVolumeTypeRepo(db)
			dataAgeRepo := repo.NewDataAgeRepo(db)
			usageRepo := repo.NewUsageRepo(db)
			costSvc := cost.New(cnf.Pennywise.BaseURL)
			recomSvc := recommendation.New(ec2InstanceRepo, ebsVolumeRepo, rdsInstanceRepo, cnf.OpenAIToken, costSvc)
			ingestionSvc := ingestion.New(logger, ec2InstanceRepo, rdsRepo, rdsInstanceRepo, rdsStorageRepo, ebsVolumeRepo, dataAgeRepo)
			go func() {
				err = ingestionSvc.Start(ctx)
				panic(err)
			}()

			return httpserver.RegisterAndStart(
				ctx,
				logger,
				cnf.Http.Address,
				api.New(costSvc, recomSvc, usageRepo, logger),
			)
		},
	}

	return cmd
}
