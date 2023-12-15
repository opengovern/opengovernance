package config

import "github.com/kaytu-io/kaytu-util/pkg/config"

type OnboardConfig struct {
	Postgres  config.Postgres
	Http      config.HttpServer
	RabbitMQ  config.RabbitMQ
	Metadata  config.KaytuService
	Inventory config.KaytuService
	Describe  config.KaytuService
}
