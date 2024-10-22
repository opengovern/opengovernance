package types

import (
	"github.com/opengovern/og-util/pkg/config"
	"github.com/opengovern/og-util/pkg/vault"
	"github.com/opengovern/opengovernance/pkg/workspace/api"
)

type KaytuWorkspaceSettings struct {
	Kaytu KaytuConfig `json:"opengovernance"`
	Vault VaultConfig `json:"vault"`
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
	OpenSearch   OpenSearchConfig `json:"opensearch"`
	Octopus      OctopusConfig    `json:"octopus"`
	Domain       DomainConfig     `json:"domain"`
}

type OpenSearchConfig struct {
	Enabled                   bool   `json:"enabled"`
	Endpoint                  string `json:"endpoint"`
	IngestionPipelineEndpoint string `json:"ingestionPipelineEndpoint"`
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

type VaultConfig struct {
	Provider vault.Provider `json:"provider"`
	AWS      struct {
		Region    string `json:"region"`
		RoleArn   string `json:"roleArn"`
		AccessKey string `json:"accessKey"`
		SecretKey string `json:"secretKey"`
	} `json:"aws"`
	Azure struct {
		BaseUrl      string `json:"baseUrl"`
		TenantId     string `json:"tenantId"`
		ClientId     string `json:"clientId"`
		ClientSecret string `json:"clientSecret"`
	} `json:"azure"`
	KeyID string `json:"keyID"`
}
