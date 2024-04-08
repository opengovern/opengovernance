package types

import (
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/api"
	"github.com/kaytu-io/kaytu-util/pkg/config"
)

type KaytuWorkspaceSettings struct {
	Kaytu KaytuConfig `json:"kaytu"`
}

type OctopusConfig struct {
	Namespace string `json:"namespace"`
}

type DomainConfig struct {
	App          string `json:"app"`
	Grpc         string `json:"grpc"`
	GrpcExternal string `json:"grpc_external"`
}

type KaytuConfig struct {
	ReplicaCount int              `json:"replicaCount"`
	EnvType      config.EnvType   `json:"envType"`
	Workspace    WorkspaceConfig  `json:"workspace"`
	Docker       DockerConfig     `json:"docker"`
	Insights     InsightsConfig   `json:"insights"`
	OpenSearch   OpenSearchConfig `json:"opensearch"`
	Octopus      OctopusConfig    `json:"octopus"`
	Domain       DomainConfig     `json:"domain"`
}

type OpenSearchConfig struct {
	Enabled                   bool   `json:"enabled"`
	Endpoint                  string `json:"endpoint"`
	IngestionPipelineEndpoint string `json:"ingestionPipelineEndpoint"`
}

type InsightsConfig struct {
	S3 S3Config `json:"s3"`
}

type S3Config struct {
	AccessKey string `json:"accessKey"`
	SecretKey string `json:"secretKey"`
}

type DockerConfig struct {
	Config string `json:"config"`
}

type WorkspaceConfig struct {
	Name            string            `json:"name"`
	Size            api.WorkspaceSize `json:"size"`
	UserARN         string            `json:"userARN"`
	MasterAccessKey string            `json:"masterAccessKey"`
	MasterSecretKey string            `json:"masterSecretKey"`
}
