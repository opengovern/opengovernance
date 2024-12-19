package config

import "github.com/opengovern/og-util/pkg/koanf"

type RegoConfig struct {
	Http koanf.HttpServer `json:"http,omitempty" koanf:"http"`
}
