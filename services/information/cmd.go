package information

import (
	"github.com/opengovern/og-util/pkg/httpserver"
	"github.com/opengovern/og-util/pkg/koanf"
	"github.com/opengovern/og-util/pkg/postgres"
	"github.com/opengovern/opengovernance/services/information/api"
	"github.com/opengovern/opengovernance/services/information/config"
	"github.com/opengovern/opengovernance/services/information/db/model"
	"github.com/opengovern/opengovernance/services/information/db/repo"
	"github.com/opengovern/opengovernance/services/information/service"
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

			logger = logger.Named("information")

			cmd.SilenceUsage = true

			db, err := postgres.NewClient(&postgres.Config{
				Host:    cnf.Postgres.Host,
				Port:    cnf.Postgres.Port,
				User:    cnf.Postgres.Username,
				Passwd:  cnf.Postgres.Password,
				DB:      cnf.Postgres.DB,
				SSLMode: cnf.Postgres.SSLMode,
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
