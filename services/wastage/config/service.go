package config

import "github.com/kaytu-io/kaytu-util/pkg/koanf"

type AzBlobConfig struct {
	TenantID   string `json:"tenantId" koanf:"tenant_id"`
	AccountUrl string `json:"accountUrl" koanf:"account_url"`
	Container  string `json:"container" koanf:"container"`
}

type WastageConfig struct {
	Postgres    koanf.Postgres     `json:"postgres,omitempty" koanf:"postgres"`
	Http        koanf.HttpServer   `json:"http,omitempty" koanf:"http"`
	Grpc        koanf.GrpcServer   `json:"grpc,omitempty" koanf:"grpc"`
	Pennywise   koanf.KaytuService `json:"pennywise" koanf:"pennywise"`
	OpenAIToken string             `json:"openAIToken" koanf:"openai_token"`
	AzBlob      AzBlobConfig       `json:"azBlob" koanf:"az_blob"`
}
