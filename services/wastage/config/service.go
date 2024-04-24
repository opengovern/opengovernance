package config

import "github.com/kaytu-io/kaytu-util/pkg/koanf"

type WastageConfig struct {
	Postgres    koanf.Postgres     `json:"postgres,omitempty" koanf:"postgres"`
	Http        koanf.HttpServer   `json:"http,omitempty" koanf:"http"`
	Pennywise   koanf.KaytuService `json:"pennywise" koanf:"pennywise"`
	OpenAIToken string             `json:"openAIToken" koanf:"openai_token"`
}
