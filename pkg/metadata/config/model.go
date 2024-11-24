package config

import (
	"github.com/opengovern/og-util/pkg/koanf"
	"github.com/opengovern/og-util/pkg/vault"
)

type Config struct {
	Postgres   koanf.Postgres     `yaml:"postgres" koanf:"postgres"`
	Http       koanf.HttpServer   `yaml:"http" koanf:"http"`
	Onboard    koanf.KaytuService `yaml:"onboard" koanf:"onboard"`
	Scheduler  koanf.KaytuService `yaml:"scheduler" koanf:"scheduler"`
	Compliance koanf.KaytuService `yaml:"compliance" koanf:"compliance"`
	Inventory  koanf.KaytuService `yaml:"inventory" koanf:"inventory"`

	Vault vault.Config `yaml:"vault" koanf:"vault"`

	OpengovernanceNamespace      string `yaml:"opengovernance_namespace" koanf:"opengovernance_namespace"`
	PrimaryDomainURL             string `yaml:"primary_domain_url" koanf:"primary_domain_url"`
	DexGrpcAddr                  string `yaml:"dex_grpc_addr" koanf:"dex_grpc_addr"`
	SampledataIntegrationsCheck  string `yaml:"sampledata_integrations_check" koanf:"sampledata_integrations_check"`
	DexPublicClientRedirectUris  string `yaml:"dex_public_client_redirect_uris" koanf:"dex_public_client_redirect_uris"`
	DexPrivateClientRedirectUris string `yaml:"dex_private_client_redirect_uris" koanf:"dex_private_client_redirect_uris"`
}
