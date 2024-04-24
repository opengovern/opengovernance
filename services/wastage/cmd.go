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
				cnf.Postgres.Password = "mysecretpassword"
				cnf.Postgres.DB = "postgres"
			}
			db, err := connector.New(cnf.Postgres)
			if err != nil {
				return err
			}
			err = db.Conn().AutoMigrate(&model.EC2InstanceType{}, &model.EBSVolumeType{}, &model.DataAge{}, &model.Usage{})
			if err != nil {
				return err
			}
			ec2InstanceRepo := repo.NewEC2InstanceTypeRepo(db)
			ebsVolumeRepo := repo.NewEBSVolumeTypeRepo(db)
			dataAgeRepo := repo.NewDataAgeRepo(db)
			usageRepo := repo.NewUsageRepo(db)
			recomSvc := recommendation.New(ec2InstanceRepo, ebsVolumeRepo, cnf.OpenAIToken)
			costSvc := cost.New(cnf.Pennywise.BaseURL)
			ingestionSvc := ingestion.New(ec2InstanceRepo, ebsVolumeRepo, dataAgeRepo)
			go func() {
				err = ingestionSvc.Start()
				panic(err)
			}()

			return httpserver.RegisterAndStart(
				cmd.Context(),
				logger,
				cnf.Http.Address,
				api.New(costSvc, recomSvc, usageRepo, logger),
			)
		},
	}

	return cmd
}
