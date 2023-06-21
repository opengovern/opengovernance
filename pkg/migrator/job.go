package migrator

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"os"

	elasticsearchv7 "github.com/elastic/go-elasticsearch/v7"
	"github.com/kaytu-io/kaytu-util/pkg/postgres"
	"github.com/prometheus/client_golang/prometheus/push"
	"gitlab.com/keibiengine/keibi-engine/pkg/migrator/compliance"
	"gitlab.com/keibiengine/keibi-engine/pkg/migrator/db"
	"gitlab.com/keibiengine/keibi-engine/pkg/migrator/elasticsearch"
	"gitlab.com/keibiengine/keibi-engine/pkg/migrator/insight"
	"gitlab.com/keibiengine/keibi-engine/pkg/migrator/internal"
	"gitlab.com/keibiengine/keibi-engine/pkg/migrator/inventory"
	"gitlab.com/keibiengine/keibi-engine/pkg/migrator/workspace"
	"go.uber.org/zap"
)

type Job struct {
	db                    db.Database
	elastic               elasticsearchv7.Config
	logger                *zap.Logger
	pusher                *push.Pusher
	AWSComplianceGitURL   string
	AzureComplianceGitURL string
	InsightGitURL         string
	QueryGitURL           string
	githubToken           string
	conf                  JobConfig
}

func InitializeJob(
	conf JobConfig,
	logger *zap.Logger,
	prometheusPushAddress string,
) (w *Job, err error) {

	w = &Job{
		logger: logger,
	}
	defer func() {
		if err != nil && w != nil {
			w.Stop()
		}
	}()

	cfg := postgres.Config{
		Host:    conf.PostgreSQL.Host,
		Port:    conf.PostgreSQL.Port,
		User:    conf.PostgreSQL.Username,
		Passwd:  conf.PostgreSQL.Password,
		DB:      conf.PostgreSQL.DB,
		SSLMode: conf.PostgreSQL.SSLMode,
	}
	orm, err := postgres.NewClient(&cfg, logger)
	if err != nil {
		return nil, fmt.Errorf("new postgres client: %w", err)
	}

	w.db = db.Database{ORM: orm}
	fmt.Println("Connected to the postgres database: ", conf.PostgreSQL.DB)

	w.pusher = push.New(prometheusPushAddress, "migrator")
	w.AWSComplianceGitURL = conf.AWSComplianceGitURL
	w.InsightGitURL = conf.InsightGitURL
	w.AzureComplianceGitURL = conf.AzureComplianceGitURL
	w.QueryGitURL = conf.QueryGitURL
	w.githubToken = conf.GithubToken

	w.elastic = elasticsearchv7.Config{
		Addresses: []string{conf.ElasticSearch.Address},
		Username:  conf.ElasticSearch.Username,
		Password:  conf.ElasticSearch.Password,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}
	if err != nil {
		return nil, err
	}

	w.conf = conf
	return w, nil
}

func (w *Job) Run() error {
	defer func() {
		if r := recover(); r != nil {
			w.logger.Error("paniced with error", zap.Error(fmt.Errorf("%v", r)))
		}
	}()

	// compliance=# truncate benchmark_assignments, benchmark_children, benchmark_policies, benchmark_tag_rels, benchmark_tags, benchmarks, policies, policy_tags, policy_tag_rels, queries cascade;
	w.logger.Info("Starting migrator job")

	w.logger.Info("Starting AWS compliance migration")
	if err := compliance.Run(w.db, w.AWSComplianceGitURL, w.QueryGitURL, w.githubToken); err != nil {
		w.logger.Error(fmt.Sprintf("Failure while running aws compliance migration: %v", err))
	}

	w.logger.Info("Starting Azure compliance migration")
	if err := compliance.Run(w.db, w.AzureComplianceGitURL, w.QueryGitURL, w.githubToken); err != nil {
		w.logger.Error(fmt.Sprintf("Failure while running azure compliance migration: %v", err))
	}

	// run elasticsearch
	w.logger.Info("Starting elasticsearch migration")
	if err := elasticsearch.Run(w.elastic, w.logger, "/elasticsearch-index-config"); err != nil {
		w.logger.Error(fmt.Sprintf("Failure while running elasticsearch migration: %v", err))
	}

	w.logger.Info("Starting insight migration")
	if err := insight.Run(w.logger, w.db, w.InsightGitURL, w.githubToken); err != nil {
		w.logger.Error(fmt.Sprintf("Failure while running insight migration: %v", err))
	}

	w.logger.Info("Starting inventory migration")
	if err := inventory.Run(w.db, w.logger, "/inventory-data-config"); err != nil {
		w.logger.Error(fmt.Sprintf("Failure while running inventory migration: %v", err))
	}

	cfg := postgres.Config{
		Host:    w.conf.PostgreSQL.Host,
		Port:    w.conf.PostgreSQL.Port,
		User:    w.conf.PostgreSQL.Username,
		Passwd:  w.conf.PostgreSQL.Password,
		DB:      w.conf.PostgreSQL.DB,
		SSLMode: w.conf.PostgreSQL.SSLMode,
	}

	w.logger.Info("Starting workspace migration")
	if err := workspace.Run(cfg, w.logger, "/workspace-migration"); err != nil {
		w.logger.Error(fmt.Sprintf("Failure while running workspace migration: %v", err))
	}

	return nil
}

func (w *Job) Stop() {
	os.RemoveAll(internal.ComplianceGitPath)
	os.RemoveAll(internal.QueriesGitPath)
	os.RemoveAll(internal.InsightsGitPath)
}
