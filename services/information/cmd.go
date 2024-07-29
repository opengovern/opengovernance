package information

import (
	"github.com/kaytu-io/kaytu-engine/services/information/api"
	"github.com/kaytu-io/kaytu-engine/services/information/config"
	"github.com/kaytu-io/kaytu-engine/services/information/db/model"
	"github.com/kaytu-io/kaytu-engine/services/information/db/repo"
	"github.com/kaytu-io/kaytu-engine/services/information/service"
	"github.com/kaytu-io/kaytu-util/pkg/httpserver"
	"github.com/kaytu-io/kaytu-util/pkg/koanf"
	"github.com/kaytu-io/kaytu-util/pkg/postgres"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

func Command() *cobra.Command {
	cnf := koanf.Provide("information", config.InformationConfig{
		Postgres: koanf.Postgres{
			Host:     "localhost",
			Port:     "5432",
			Username: "postgres",
		},
		Http: koanf.HttpServer{
			Address: "localhost:8000",
		},
	})

	cmd := &cobra.Command{
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			logger, err := zap.NewProduction()
			if err != nil {
				return err
			}

			logger = logger.Named("wastage")

			cmd.SilenceUsage = true

			db, err := postgres.NewClient(&postgres.Config{
				Host:   cnf.Postgres.Host,
				Port:   cnf.Postgres.Port,
				User:   cnf.Postgres.Username,
				Passwd: cnf.Postgres.Password,
				DB:     cnf.Postgres.DB,
			}, logger)
			if err != nil {
				return err
			}
			// create citext extension if not exists
			err = db.Exec("CREATE EXTENSION IF NOT EXISTS citext").Error
			if err != nil {
				logger.Error("failed to create citext extension", zap.Error(err))
				return err
			}
			err = db.AutoMigrate(&model.CspmUsage{})

			cspmUsageRepo := repo.NewCspmUsageRepo(db)

			informationService := service.NewInformationService(cnf, logger, cspmUsageRepo)
			return httpserver.RegisterAndStart(
				ctx,
				logger,
				cnf.Http.Address,
				api.New(cnf, logger, informationService),
			)
		},
	}

	return cmd
}
