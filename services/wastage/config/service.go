package config

import "github.com/kaytu-io/kaytu-util/pkg/koanf"

type WastageConfig struct {
	Postgres    koanf.Postgres     `json:"postgres,omitempty" koanf:"postgres"`
	Http        koanf.HttpServer   `json:"http,omitempty" koanf:"http"`
	Grpc        koanf.GrpcServer   `json:"grpc,omitempty" koanf:"grpc"`
	Pennywise   koanf.KaytuService `json:"pennywise" koanf:"pennywise"`
	OpenAIToken string             `json:"openAIToken" koanf:"openai_token"`
}
