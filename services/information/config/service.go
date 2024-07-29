package config

import "github.com/kaytu-io/kaytu-util/pkg/koanf"

type InformationConfig struct {
	Postgres koanf.Postgres   `json:"postgres,omitempty" koanf:"postgres"`
	Http     koanf.HttpServer `json:"http,omitempty" koanf:"http"`
	Grpc     koanf.GrpcServer `json:"grpc,omitempty" koanf:"grpc"`
}
