package config

import "github.com/kaytu-io/kaytu-util/pkg/config"

type SubscriptionConfig struct {
	Auth       config.KaytuService `koanf:"auth"`
	Workspace  config.KaytuService `koanf:"workspace"`
	Scheduler  config.KaytuService `koanf:"scheduler"`
	Alerting   config.KaytuService `koanf:"alerting"`
	Compliance config.KaytuService `koanf:"compliance"`
	Inventory  config.KaytuService `koanf:"inventory"`

	Postgres config.Postgres   `koanf:"postgres"`
	Http     config.HttpServer `koanf:"http"`
}
