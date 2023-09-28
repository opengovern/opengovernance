package migrator

import (
	"crypto/tls"
	"fmt"
	"github.com/go-git/go-git/v5"
	githttp "github.com/go-git/go-git/v5/plumbing/transport/http"
	"net/http"
	"os"
	"path"

	"github.com/kaytu-io/kaytu-engine/pkg/migrator/analytics"
	"github.com/kaytu-io/kaytu-engine/pkg/migrator/onboard"

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
	AnalyticsGitURL string
	githubToken     string
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
		AnalyticsGitURL: w.conf.AnalyticsGitURL,
		githubToken:     w.conf.GithubToken,
	}

	if value, err := w.metadataClient.GetConfigMetadata(&httpclient.Context{
		UserRole: api.AdminRole,
	}, models.MetadataKeyAnalyticsGitURL); err == nil && len(value.GetValue().(string)) > 0 {
		gitConfig.AnalyticsGitURL = value.GetValue().(string)
	}

	// run elasticsearch
	w.logger.Info("Starting elasticsearch migration")
	if err := elasticsearch.Run(w.elastic, w.logger, "/elasticsearch-index-config"); err != nil {
		w.logger.Error("Failure while running elasticsearch migration", zap.Error(err))
	}

	cfg := postgres.Config{
		Host:    w.conf.PostgreSQL.Host,
		Port:    w.conf.PostgreSQL.Port,
		User:    w.conf.PostgreSQL.Username,
		Passwd:  w.conf.PostgreSQL.Password,
		SSLMode: w.conf.PostgreSQL.SSLMode,
	}

	w.logger.Info("cloning analytics git")
	os.RemoveAll(internal.AnalyticsGitPath)
	_, err := git.PlainClone(internal.AnalyticsGitPath, false, &git.CloneOptions{
		Auth: &githttp.BasicAuth{
			Username: "abc123",
			Password: gitConfig.githubToken,
		},
		URL:      gitConfig.AnalyticsGitURL,
		Progress: os.Stdout,
	})
	if err != nil {
		w.logger.Error("Failure while running analytics migration", zap.Error(err))
		return err
	}

	w.logger.Info("Starting analytics migration")
	if err := analytics.PopulateDatabase(w.logger, cfg); err != nil {
		w.logger.Error("Failure while running analytics migration", zap.Error(err))
	}

	w.logger.Info("Starting compliance migration")
	if err = compliance.PopulateDatabase(w.db.ORM); err != nil {
		w.logger.Error(fmt.Sprintf("Failure while running compliance migration: %v", err))
	}

	w.logger.Info("Starting insight migration")
	if err := insight.PopulateDatabase(w.logger, w.db.ORM); err != nil {
		w.logger.Error(fmt.Sprintf("Failure while running insight migration: %v", err))
	}

	w.logger.Info("Starting onboard migration")
	if err := onboard.Run(w.logger, cfg, path.Join(internal.AnalyticsGitPath, "connection_groups")); err != nil {
		w.logger.Error("Failure while running onboard migration", zap.Error(err))
	}

	w.logger.Info("Starting inventory migration")
	if err := inventory.Run(cfg, w.logger, "/inventory-data-config"); err != nil {
		w.logger.Error("Failure while running inventory migration", zap.Error(err))
	}

	w.logger.Info("Starting workspace migration")
	if err := workspace.Run(cfg, w.logger, "/workspace-migration"); err != nil {
		w.logger.Error("Failure while running workspace migration", zap.Error(err))
	}

	return nil
}

func (w *Job) Stop() {
	os.RemoveAll(internal.AnalyticsGitPath)
}
