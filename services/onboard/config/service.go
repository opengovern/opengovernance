package config

import "github.com/kaytu-io/kaytu-util/pkg/config"

type OnboardConfig struct {
	Postgres  config.Postgres
	Http      config.HttpServer
	RabbitMQ  config.RabbitMQ
	Metadata  config.KaytuService
	Inventory config.KaytuService
	Describe  config.KaytuService

	MasterAccessKey string `yaml:"master_access_key"`
	MasterSecretKey string `yaml:"master_secret_key"`
}
