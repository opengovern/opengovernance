package config

import (
	"github.com/kaytu-io/kaytu-util/pkg/koanf"
)

type SubscriptionConfig struct {
	Auth       koanf.KaytuService `json:"auth,omitempty" koanf:"auth"`
	Workspace  koanf.KaytuService `json:"workspace,omitempty" koanf:"workspace"`
	Scheduler  koanf.KaytuService `json:"scheduler,omitempty" koanf:"scheduler"`
	Alerting   koanf.KaytuService `json:"alerting,omitempty" koanf:"alerting"`
	Compliance koanf.KaytuService `json:"compliance,omitempty" koanf:"compliance"`
	Inventory  koanf.KaytuService `json:"inventory,omitempty" koanf:"inventory"`

	Postgres koanf.Postgres   `json:"postgres,omitempty" koanf:"postgres"`
	Http     koanf.HttpServer `json:"http,omitempty" koanf:"http"`

	UsageMetersFirehoseStreamName string `json:"usage_meters_firehose_stream_name,omitempty" koanf:"usage_meters_firehose_stream_name"`
}
