package config

import "github.com/kaytu-io/kaytu-util/pkg/koanf"

type IntegrationConfig struct {
	Postgres        koanf.Postgres     `json:"postgres,omitempty" koanf:"postgres"`
	Steampipe       koanf.Postgres     `json:"steampipe,omitempty" koanf:"steampipe"`
	Http            koanf.HttpServer   `json:"http,omitempty" koanf:"http"`
	KMS             koanf.KMS          `json:"kms,omitempty" koanf:"kms"`
	Metadata        koanf.KaytuService `json:"metadata,omitempty" koanf:"metadata"`
	Inventory       koanf.KaytuService `json:"inventory,omitempty" koanf:"inventory"`
	Describe        koanf.KaytuService `json:"describe,omitempty" koanf:"describe"`
	MasterAccessKey string             `json:"master_access_key,omitempty" koanf:"master_access_key"`
	MasterSecretKey string             `json:"master_secret_key,omitempty" koanf:"master_secret_key"`
}
