package config

import "github.com/kaytu-io/kaytu-util/pkg/config"

type SubscriptionConfig struct {
	Auth       config.KaytuService
	Workspace  config.KaytuService
	Scheduler  config.KaytuService
	Alerting   config.KaytuService
	Compliance config.KaytuService
	Inventory  config.KaytuService

	Postgres config.Postgres
	Http     config.HttpServer
}
