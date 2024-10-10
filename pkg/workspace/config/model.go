package config

import (
	"github.com/kaytu-io/kaytu-util/pkg/koanf"
	"github.com/kaytu-io/kaytu-util/pkg/vault"
)

type Config struct {
	Postgres   koanf.Postgres     `yaml:"postgres" koanf:"postgres"`
	Http       koanf.HttpServer   `yaml:"http" koanf:"http"`
	Auth       koanf.KaytuService `yaml:"auth" koanf:"auth"`
	Onboard    koanf.KaytuService `yaml:"onboard" koanf:"onboard"`
	Scheduler  koanf.KaytuService `yaml:"scheduler" koanf:"scheduler"`
	Compliance koanf.KaytuService `yaml:"compliance" koanf:"compliance"`
	Metadata   koanf.KaytuService `yaml:"metadata" koanf:"metadata"`
	Inventory  koanf.KaytuService `yaml:"inventory" koanf:"inventory"`

	Vault vault.Config `yaml:"vault" koanf:"vault"`

	KaytuOctopusNamespace       string `yaml:"kaytu_octopus_namespace" koanf:"kaytu_octopus_namespace"`
	PrimaryDomainURL            string `yaml:"primary_domain_url" koanf:"primary_domain_url"`
	KaytuWorkspaceVersion       string `yaml:"kaytu_workspace_version" koanf:"kaytu_workspace_version"`
	DexGrpcAddr                 string `yaml:"dex_grpc_addr" koanf:"dex_grpc_addr"`
	SampledataIntegrationsCheck string `yaml:"sampledata_integrations_check" koanf:"sampledata_integrations_check"`
}
