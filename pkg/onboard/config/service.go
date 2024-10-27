package config

import (
	"github.com/opengovern/og-util/pkg/koanf"
	"github.com/opengovern/og-util/pkg/vault"
)

type OnboardConfig struct {
	Postgres        koanf.Postgres              `json:"postgres,omitempty" koanf:"postgres"`
	Steampipe       koanf.Postgres              `json:"steampipe,omitempty" koanf:"steampipe"`
	Http            koanf.HttpServer            `json:"http,omitempty" koanf:"http"`
	Metadata        koanf.OpenGovernanceService `json:"metadata,omitempty" koanf:"metadata"`
	Inventory       koanf.OpenGovernanceService `json:"inventory,omitempty" koanf:"inventory"`
	Describe        koanf.OpenGovernanceService `json:"describe,omitempty" koanf:"describe"`
	Vault           vault.Config                `json:"vault,omitempty" koanf:"vault"`
	MasterAccessKey string                      `json:"master_access_key,omitempty" koanf:"master_access_key"`
	MasterSecretKey string                      `json:"master_secret_key,omitempty" koanf:"master_secret_key"`
}
