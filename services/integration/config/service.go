package config

import "github.com/kaytu-io/kaytu-util/pkg/koanf"

type OnboardConfig struct {
	Postgres        koanf.Postgres     `json:"postgres,omitempty"`
	Steampipe       koanf.Postgres     `json:"steampipe,omitempty"`
	Http            koanf.HttpServer   `json:"http,omitempty"`
	RabbitMQ        koanf.RabbitMQ     `json:"rabbit_mq,omitempty"`
	KMS             koanf.KMS          `json:"kms,omitempty"`
	Metadata        koanf.KaytuService `json:"metadata,omitempty"`
	Inventory       koanf.KaytuService `json:"inventory,omitempty"`
	Describe        koanf.KaytuService `json:"describe,omitempty"`
	MasterAccessKey string             `json:"master_access_key,omitempty"`
	MasterSecretKey string             `json:"master_secret_key,omitempty"`
}
