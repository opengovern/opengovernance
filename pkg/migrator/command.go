package migrator

import (
	utilConfig "github.com/kaytu-io/kaytu-util/pkg/config"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

type JobConfig struct {
	IsManual bool `yaml:"is_manual"`

	PostgreSQL            utilConfig.Postgres
	ElasticSearch         utilConfig.ElasticSearch
	Metadata              utilConfig.KaytuService
	AnalyticsGitURL       string `yaml:"analytics_git_url"`
	GithubToken           string `yaml:"github_token"`
	PrometheusPushAddress string `yaml:"prometheus_push_address"`

	RabbitMqService  string `yaml:"rabbit_mq_service"`
	RabbitMqUsername string `yaml:"rabbit_mq_username"`
	RabbitMqPassword string `yaml:"rabbit_mq_password"`
	RabbitMqQueue    string `yaml:"rabbit_mq_queue"`
}

func WorkerCommand() *cobra.Command {
	var (
		cnf JobConfig
	)
	utilConfig.ReadFromEnv(&cnf, nil)

	cmd := &cobra.Command{
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true
			logger, err := zap.NewProduction()
			if err != nil {
				return err
			}
			if cnf.IsManual {
				w, err := InitializeJob(
					cnf,
					logger,
					cnf.PrometheusPushAddress,
				)
				if err != nil {
					return err
				}

				defer w.Stop()

				return w.Run()
			}

			w, err := InitializeWorker(
				cnf,
				logger,
				cnf.PrometheusPushAddress,
			)
			if err != nil {
				return err
			}

			defer w.Stop()

			return w.Run()
		},
	}

	return cmd
}
