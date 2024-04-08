package config

import (
	"github.com/kaytu-io/kaytu-util/pkg/config"
	"github.com/kaytu-io/kaytu-util/pkg/vault"
)

type Config struct {
	EnvType    config.EnvType `yaml:"env_type"`
	Postgres   config.Postgres
	Http       config.HttpServer
	Auth       config.KaytuService
	Onboard    config.KaytuService
	Scheduler  config.KaytuService
	Compliance config.KaytuService
	Inventory  config.KaytuService

	Vault vault.Config `yaml:"vault"`

	DomainSuffix               string
	AppDomain                  string `yaml:"app_domain"`
	GrpcDomain                 string `yaml:"grpc_domain"`
	GrpcExternalDomain         string `yaml:"grpc_external_domain"`
	KaytuHelmChartLocation     string
	KaytuOctopusNamespace      string
	FluxSystemNamespace        string
	AutoSuspendDurationMinutes int64
	S3AccessKey                string
	S3SecretKey                string
	AWSMasterAccessKey         string `yaml:"aws_master_access_key"`
	AWSMasterSecretKey         string `yaml:"aws_master_secret_key"`
	AWSMasterPolicyARN         string `yaml:"aws_master_policy_arn"`
	AWSAccountID               string `yaml:"aws_account_id"`
	OIDCProvider               string `yaml:"oidc_provider"`
	SecurityGroupID            string `yaml:"security_group_id"`
	SubnetID                   string `yaml:"subnet_id"`
	DoReserve                  bool   `yaml:"do_reserve"`

	OpenSearchRegion string `yaml:"open_search_region"`
}
