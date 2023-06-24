package migrator

import (
	"github.com/kaytu-io/kaytu-engine/pkg/config"
	config2 "github.com/kaytu-io/kaytu-util/pkg/config"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

type JobConfig struct {
	PostgreSQL            config.Postgres
	ElasticSearch         config.ElasticSearch
	QueryGitURL           string `yaml:"query_git_url"`
	GithubToken           string `yaml:"github_token"`
	AWSComplianceGitURL   string `yaml:"aws_compliance_git_url"`
	InsightGitURL         string `yaml:"insight_git_url"`
	AzureComplianceGitURL string `yaml:"azure_compliance_git_url"`
	PrometheusPushAddress string `yaml:"prometheus_push_address"`
}

func JobCommand() *cobra.Command {
	var (
		cnf JobConfig
	)
	config2.ReadFromEnv(&cnf, nil)

	cmd := &cobra.Command{
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true
			logger, err := zap.NewProduction()
			if err != nil {
				return err
			}

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
		},
	}

	return cmd
}
