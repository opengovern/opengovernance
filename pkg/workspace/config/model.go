package config

import (
	"github.com/kaytu-io/kaytu-util/pkg/config"
	"time"
)

type Config struct {
	Postgres  config.Postgres
	Redis     config.Redis
	Http      config.HttpServer
	Auth      config.KaytuService
	Onboard   config.KaytuService
	Inventory config.KaytuService

	DomainSuffix           string
	KaytuHelmChartLocation string
	KaytuOctopusNamespace  string
	FluxSystemNamespace    string
	AutoSuspendDuration    time.Duration
	S3AccessKey            string
	S3SecretKey            string
	KMSAccountRegion       string
}
