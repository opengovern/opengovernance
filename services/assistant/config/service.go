package config

import "github.com/kaytu-io/kaytu-util/pkg/koanf"

type OpenAI struct {
	Token     string `json:"token,omitempty" koanf:"token"`
	BaseURL   string `json:"base_url,omitempty" koanf:"base_url"`
	ModelName string `json:"model_name,omitempty" koanf:"model_name"`
}

type AssistantConfig struct {
	Postgres  koanf.Postgres     `json:"postgres,omitempty" koanf:"postgres"`
	OpenAI    OpenAI             `json:"openai,omitempty" koanf:"openai"`
	Inventory koanf.KaytuService `json:"inventory,omitempty" koanf:"inventory"`
	Http      koanf.HttpServer   `json:"http,omitempty" koanf:"http"`
}
