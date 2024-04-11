package config

import "github.com/kaytu-io/kaytu-util/pkg/config"

type MigratorConfig struct {
	IsManual bool `yaml:"is_manual"`

	PostgreSQL              config.Postgres
	Steampipe               config.Postgres
	ElasticSearch           config.ElasticSearch
	Metadata                config.KaytuService
	AnalyticsGitURL         string `yaml:"analytics_git_url"`
	ControlEnrichmentGitURL string `yaml:"control_enrichment_git_url"`
	GithubToken             string `yaml:"github_token"`
	SupersetBaseURL         string `yaml:"superset_base_url"`
	SupersetAdminPassword   string `yaml:"superset_admin_password"`
	PrometheusPushAddress   string `yaml:"prometheus_push_address"`
}
