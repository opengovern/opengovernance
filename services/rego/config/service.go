package config

import "github.com/opengovern/og-util/pkg/koanf"

type RegoConfig struct {
	Http          koanf.HttpServer    `json:"http,omitempty" koanf:"http"`
	ElasticSearch koanf.ElasticSearch `json:"elasticsearch,omitempty" koanf:"elasticsearch"`
	Steampipe     koanf.Postgres      `json:"steampipe,omitempty" koanf:"steampipe"`
}
