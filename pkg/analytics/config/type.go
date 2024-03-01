package config

import "github.com/kaytu-io/kaytu-util/pkg/config"

type WorkerConfig struct {
	NATS             config.NATS
	PostgreSQL       config.Postgres
	ElasticSearch    config.ElasticSearch
	Steampipe        config.Postgres
	Onboard          config.KaytuService
	Scheduler        config.KaytuService
	Inventory        config.KaytuService
	PennywiseBaseURL string `yaml:"pennywise_base_url"`
}
