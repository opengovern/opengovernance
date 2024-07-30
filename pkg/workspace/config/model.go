package config

import (
	"github.com/kaytu-io/kaytu-util/pkg/config"
	"github.com/kaytu-io/kaytu-util/pkg/koanf"
	"github.com/kaytu-io/kaytu-util/pkg/vault"
)

type Config struct {
	EnvType    config.EnvType     `yaml:"env_type" koanf:"env_type"`
	Postgres   koanf.Postgres     `yaml:"postgres" koanf:"postgres"`
	Http       koanf.HttpServer   `yaml:"http" koanf:"http"`
	Auth       koanf.KaytuService `yaml:"auth" koanf:"auth"`
	Onboard    koanf.KaytuService `yaml:"onboard" koanf:"onboard"`
	Scheduler  koanf.KaytuService `yaml:"scheduler" koanf:"scheduler"`
	Compliance koanf.KaytuService `yaml:"compliance" koanf:"compliance"`
	Inventory  koanf.KaytuService `yaml:"inventory" koanf:"inventory"`

	Vault vault.Config `yaml:"vault" koanf:"vault"`

	DomainSuffix               string `yaml:"domain_suffix" koanf:"domain_suffix"`
	AppDomain                  string `yaml:"app_domain" koanf:"app_domain"`
	GrpcDomain                 string `yaml:"grpc_domain" koanf:"grpc_domain"`
	GrpcExternalDomain         string `yaml:"grpc_external_domain" koanf:"grpc_external_domain"`
	KaytuOctopusNamespace      string `yaml:"kaytu_octopus_namespace" koanf:"kaytu_octopus_namespace"`
	FluxSystemNamespace        string `yaml:"flux_system_namespace" koanf:"flux_system_namespace"`
	AutoSuspendDurationMinutes int64  `yaml:"auto_suspend_duration_minutes" koanf:"auto_suspend_duration_minutes"`
	S3AccessKey                string `yaml:"s3_access_key" koanf:"s3_access_key"`
	S3SecretKey                string `yaml:"s3_secret_key" koanf:"s3_secret_key"`
	AWSMasterAccessKey         string `yaml:"aws_master_access_key" koanf:"aws_master_access_key"`
	AWSMasterSecretKey         string `yaml:"aws_master_secret_key" koanf:"aws_master_secret_key"`
	AWSMasterPolicyARN         string `yaml:"aws_master_policy_arn" koanf:"aws_master_policy_arn"`
	AWSAccountID               string `yaml:"aws_account_id" koanf:"aws_account_id"`
	OIDCProvider               string `yaml:"oidc_provider" koanf:"oidc_provider"`
	SecurityGroupID            string `yaml:"security_group_id" koanf:"security_group_id"`
	SubnetID                   string `yaml:"subnet_id" koanf:"subnet_id"`
	DoReserve                  bool   `yaml:"do_reserve" koanf:"do_reserve"`

	OpenSearchRegion      string `yaml:"open_search_region" koanf:"open_search_region"`
	KaytuWorkspaceVersion string `yaml:"kaytu_workspace_version" koanf:"kaytu_workspace_version"`
}
