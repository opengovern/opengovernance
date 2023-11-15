package config

import (
	"github.com/kaytu-io/kaytu-util/pkg/config"
)

type Config struct {
	Postgres   config.Postgres
	Redis      config.Redis
	Http       config.HttpServer
	Auth       config.KaytuService
	Onboard    config.KaytuService
	Scheduler  config.KaytuService
	Compliance config.KaytuService
	Inventory  config.KaytuService

	DomainSuffix               string
	KaytuHelmChartLocation     string
	KaytuOctopusNamespace      string
	FluxSystemNamespace        string
	AutoSuspendDurationMinutes int64
	S3AccessKey                string
	S3SecretKey                string
	KMSAccountRegion           string
}
