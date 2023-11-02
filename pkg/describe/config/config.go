package config

import "github.com/kaytu-io/kaytu-util/pkg/config"

type SchedulerConfig struct {
	ComplianceIntervalHours int
	Kafka                   config.Kafka
}
