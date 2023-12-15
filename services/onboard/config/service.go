package config

import "github.com/kaytu-io/kaytu-util/pkg/koanf"

type OnboardConfig struct {
	Postgres  koanf.Postgres
	Steampipe koanf.Postgres
	Http      koanf.HttpServer
	RabbitMQ  koanf.RabbitMQ
	Metadata  koanf.KaytuService
	Inventory koanf.KaytuService
	Describe  koanf.KaytuService
}
