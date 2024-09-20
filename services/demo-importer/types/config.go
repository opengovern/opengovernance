package types

import "github.com/kaytu-io/kaytu-util/pkg/config"

type DemoImporterConfig struct {
	IsManual bool `yaml:"is_manual"`

	PostgreSQL            config.Postgres
	ElasticSearch         config.ElasticSearch
	Metadata              config.KaytuService
	DemoDataGitURL        string `yaml:"demo_data_git_url"`
	GithubToken           string `yaml:"github_token"`
	PrometheusPushAddress string `yaml:"prometheus_push_address"`
	OpensslPassword       string `yaml:"openssl_password"`
}
