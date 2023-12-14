package config

import "github.com/kaytu-io/kaytu-util/pkg/config"

type OnboradConfig struct {
	Postgres config.Postgres
	Http     config.HttpServer
	RabbitMQ config.RabbitMQ
}
