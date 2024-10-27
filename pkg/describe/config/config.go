package config

import (
	"github.com/opengovern/og-util/pkg/config"
	"github.com/opengovern/og-util/pkg/vault"
)

type ServerlessProviderType string

const (
	ServerlessProviderTypeAWSLambda      ServerlessProviderType = "aws-lambda"
	ServerlessProviderTypeAzureFunctions ServerlessProviderType = "azure-functions"
	ServerlessProviderTypeLocal          ServerlessProviderType = "local"
)

func (s ServerlessProviderType) String() string {
	return string(s)
}

type SchedulerConfig struct {
	ComplianceIntervalHours    int    `yaml:"compliance_interval_hours"`
	EventHubConnectionString   string `yaml:"event_hub_connection_string"`
	ServiceBusConnectionString string `yaml:"service_bus_connection_string"`
	ServerlessProvider         string `yaml:"serverless_provider"`
	ElasticSearch              config.ElasticSearch
	Onboard                    config.OpenGovernanceService
	NATS                       config.NATS
	Vault                      vault.Config `yaml:"vault" koanf:"vault"`
}
