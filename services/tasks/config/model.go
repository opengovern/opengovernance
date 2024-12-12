package config

import (
	"github.com/opengovern/og-util/pkg/koanf"
	"github.com/opengovern/og-util/pkg/vault"
)

type Config struct {
	Postgres koanf.Postgres   `yaml:"postgres" koanf:"postgres"`
	Http     koanf.HttpServer `yaml:"http" koanf:"http"`

	Vault vault.Config `yaml:"vault" koanf:"vault"`
}
