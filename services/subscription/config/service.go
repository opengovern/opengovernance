package config

import "github.com/kaytu-io/kaytu-util/pkg/config"

type SubscriptionConfig struct {
	Postgres config.Postgres
	Http     config.HttpServer
}
