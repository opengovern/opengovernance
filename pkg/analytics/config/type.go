package config

import "github.com/kaytu-io/kaytu-util/pkg/config"

type WorkerConfig struct {
	RabbitMQ      config.RabbitMQ
	Kafka         config.Kafka
	PostgreSQL    config.Postgres
	ElasticSearch config.ElasticSearch
	Steampipe     config.Postgres
	Onboard       config.KaytuService
	Scheduler     config.KaytuService
	Inventory     config.KaytuService
}
