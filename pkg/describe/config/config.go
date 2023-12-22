package config

import "github.com/kaytu-io/kaytu-util/pkg/config"

type SchedulerConfig struct {
	ComplianceIntervalHours int `yaml:"compliance_interval_hours"`
	Kafka                   config.Kafka
	ElasticSearch           config.ElasticSearch
}
