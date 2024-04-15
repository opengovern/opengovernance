package config

import (
	"github.com/kaytu-io/kaytu-util/pkg/koanf"
)

type EsSinkConfig struct {
	ElasticSearch koanf.ElasticSearch `json:"elasticsearch" koanf:"elasticsearch"`
	NATS          koanf.NATS          `json:"nats" koanf:"nats"`
	Http          koanf.HttpServer    `json:"http" koanf:"http"`
	Grpc          koanf.GrpcServer    `json:"grpc" koanf:"grpc"`
}
