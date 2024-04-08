package config

import (
	"github.com/kaytu-io/kaytu-util/pkg/koanf"
	"github.com/kaytu-io/kaytu-util/pkg/vault"
)

type OnboardConfig struct {
	Postgres        koanf.Postgres     `json:"postgres,omitempty" koanf:"postgres"`
	Http            koanf.HttpServer   `json:"http,omitempty" koanf:"http"`
	Metadata        koanf.KaytuService `json:"metadata,omitempty" koanf:"metadata"`
	Inventory       koanf.KaytuService `json:"inventory,omitempty" koanf:"inventory"`
	Describe        koanf.KaytuService `json:"describe,omitempty" koanf:"describe"`
	Vault           vault.Config       `json:"vault,omitempty" koanf:"vault"`
	MasterAccessKey string             `json:"master_access_key,omitempty" koanf:"master_access_key"`
	MasterSecretKey string             `json:"master_secret_key,omitempty" koanf:"master_secret_key"`
}
