package migrator

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"os"

	"github.com/kaytu-io/kaytu-engine/pkg/auth/api"
	"github.com/kaytu-io/kaytu-engine/pkg/internal/httpclient"
	"github.com/kaytu-io/kaytu-engine/pkg/metadata/client"
	"github.com/kaytu-io/kaytu-engine/pkg/metadata/models"
	"github.com/kaytu-io/kaytu-engine/pkg/migrator/insight"

	elasticsearchv7 "github.com/elastic/go-elasticsearch/v7"
	"github.com/kaytu-io/kaytu-engine/pkg/migrator/compliance"
	"github.com/kaytu-io/kaytu-engine/pkg/migrator/db"
	"github.com/kaytu-io/kaytu-engine/pkg/migrator/elasticsearch"
	"github.com/kaytu-io/kaytu-engine/pkg/migrator/internal"
	"github.com/kaytu-io/kaytu-engine/pkg/migrator/inventory"
	"github.com/kaytu-io/kaytu-engine/pkg/migrator/workspace"
	"github.com/kaytu-io/kaytu-util/pkg/postgres"
	"github.com/prometheus/client_golang/prometheus/push"
	"go.uber.org/zap"
)

type GitConfig struct {
	AWSComplianceGitURL   string
	AzureComplianceGitURL string
	InsightGitURL         string
	QueryGitURL           string
	githubToken           string
}

type Job struct {
	db             db.Database
	elastic        elasticsearchv7.Config
	logger         *zap.Logger
	pusher         *push.Pusher
	metadataClient client.MetadataServiceClient
	conf           JobConfig
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

	w.metadataClient = client.NewMetadataServiceClient(conf.Metadata.BaseURL)

	w.conf = conf
	return w, nil
}

func NewJob(
	database db.Database,
	elastic elasticsearchv7.Config,
	logger *zap.Logger,
	pusher *push.Pusher,
	metadataClient client.MetadataServiceClient,
	conf JobConfig,
) *Job {
	return &Job{
		db:             database,
		elastic:        elastic,
		logger:         logger,
		pusher:         pusher,
		metadataClient: metadataClient,
		conf:           conf,
	}
}

func (w *Job) Run() error {
	defer func() {
		if r := recover(); r != nil {
			w.logger.Error("paniced with error", zap.Error(fmt.Errorf("%v", r)))
		}
	}()

	// compliance=# truncate benchmark_assignments, benchmark_children, benchmark_policies, benchmark_tag_rels, benchmark_tags, benchmarks, policies, policy_tags, policy_tag_rels, queries cascade;
	w.logger.Info("Starting migrator job")

	gitConfig := GitConfig{
		AWSComplianceGitURL:   w.conf.AWSComplianceGitURL,
		AzureComplianceGitURL: w.conf.AzureComplianceGitURL,
		InsightGitURL:         w.conf.InsightGitURL,
		QueryGitURL:           w.conf.QueryGitURL,
		githubToken:           w.conf.GithubToken,
	}

	if value, err := w.metadataClient.GetConfigMetadata(&httpclient.Context{
		UserRole: api.AdminRole,
	}, models.MetadataKeyAWSComplianceGitURL); err == nil && len(value.GetValue().(string)) > 0 {
		gitConfig.AWSComplianceGitURL = value.GetValue().(string)
	}
	if value, err := w.metadataClient.GetConfigMetadata(&httpclient.Context{
		UserRole: api.AdminRole,
	}, models.MetadataKeyAzureComplianceGitURL); err == nil && len(value.GetValue().(string)) > 0 {
		gitConfig.AzureComplianceGitURL = value.GetValue().(string)
	}
	if value, err := w.metadataClient.GetConfigMetadata(&httpclient.Context{
		UserRole: api.AdminRole,
	}, models.MetadataKeyInsightsGitURL); err == nil && len(value.GetValue().(string)) > 0 {
		gitConfig.InsightGitURL = value.GetValue().(string)
	}
	if value, err := w.metadataClient.GetConfigMetadata(&httpclient.Context{
		UserRole: api.AdminRole,
	}, models.MetadataKeyQueriesGitURL); err == nil && len(value.GetValue().(string)) > 0 {
		gitConfig.QueryGitURL = value.GetValue().(string)
	}

	w.logger.Info("Starting compliance migration")
	if err := compliance.Run(w.db, []string{gitConfig.AWSComplianceGitURL, gitConfig.AzureComplianceGitURL}, gitConfig.QueryGitURL, gitConfig.githubToken); err != nil {
		w.logger.Error(fmt.Sprintf("Failure while running aws compliance migration: %v", err))
	}

	w.logger.Info("Starting insight migration")
	if err := insight.Run(w.logger, w.db, gitConfig.InsightGitURL, gitConfig.githubToken); err != nil {
		w.logger.Error(fmt.Sprintf("Failure while running insight migration: %v", err))
	}

	// run elasticsearch
	w.logger.Info("Starting elasticsearch migration")
	if err := elasticsearch.Run(w.elastic, w.logger, "/elasticsearch-index-config"); err != nil {
		w.logger.Error(fmt.Sprintf("Failure while running elasticsearch migration: %v", err))
	}

	cfg := postgres.Config{
		Host:    w.conf.PostgreSQL.Host,
		Port:    w.conf.PostgreSQL.Port,
		User:    w.conf.PostgreSQL.Username,
		Passwd:  w.conf.PostgreSQL.Password,
		SSLMode: w.conf.PostgreSQL.SSLMode,
	}

	w.logger.Info("Starting inventory migration")
	if err := inventory.Run(cfg, w.logger, "/inventory-data-config"); err != nil {
		w.logger.Error(fmt.Sprintf("Failure while running inventory migration: %v", err))
	}

	w.logger.Info("Starting workspace migration")
	if err := workspace.Run(cfg, w.logger, "/workspace-migration"); err != nil {
		w.logger.Error(fmt.Sprintf("Failure while running workspace migration: %v", err))
	}

	err := os.RemoveAll(internal.ComplianceGitPath)
	if err != nil {
		w.logger.Error("Failure while removing compliance git path", zap.Error(err))
	}
	err = os.RemoveAll(internal.QueriesGitPath)
	if err != nil {
		w.logger.Error("Failure while removing queries git path", zap.Error(err))
	}
	err = os.RemoveAll(internal.InsightsGitPath)
	if err != nil {
		w.logger.Error("Failure while removing insights git path", zap.Error(err))
	}

	return nil
}

func (w *Job) Stop() {
	os.RemoveAll(internal.ComplianceGitPath)
	os.RemoveAll(internal.QueriesGitPath)
	os.RemoveAll(internal.InsightsGitPath)
}
