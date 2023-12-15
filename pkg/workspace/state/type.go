package state

import (
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/api"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/transactions"
)

type StateID string

const (
	StateID_Reserving     StateID = "RESERVING"
	StateID_Reserved      StateID = "RESERVED"
	StateID_Bootstrapping StateID = "BOOTSTRAPPING"
	StateID_Provisioned   StateID = "PROVISIONED"
	StateID_Deleting      StateID = "DELETING"
	StateID_Deleted       StateID = "DELETED"
)

type State interface {
	Requirements() []transactions.TransactionID
	ProcessingStateID() StateID
	FinishedStateID() StateID
}

var AllStates = []State{
	Bootstrapping{},
	Deleting{},
	Reserved{},
}

type KaytuWorkspaceSettings struct {
	Kaytu KaytuConfig `json:"kaytu"`
}
type KaytuConfig struct {
	ReplicaCount int              `json:"replicaCount"`
	Workspace    WorkspaceConfig  `json:"workspace"`
	Docker       DockerConfig     `json:"docker"`
	Insights     InsightsConfig   `json:"insights"`
	OpenSearch   OpenSearchConfig `json:"opensearch"`
}
type OpenSearchConfig struct {
	Enabled  bool   `json:"enabled"`
	Endpoint string `json:"endpoint"`
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
