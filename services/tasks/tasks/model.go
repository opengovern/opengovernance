package tasks

type WorkloadType string

const (
	WorkloadTypeJob        WorkloadType = "job"
	WorkloadTypeDeployment WorkloadType = "deployment"
)

type NatsConfig struct {
	Stream         string `yaml:"Stream"`
	Topic          string `yaml:"Topic"`
	Consumer       string `yaml:"Consumer"`
	ResultTopic    string `yaml:"ResultTopic"`
	ResultConsumer string `yaml:"ResultConsumer"`
}

type ScaleConfig struct {
	Stream                       string
	Consumer                     string
	NatsServerMonitoringEndpoint string
	LagThreshold                 string
	MinReplica                   int32
	MaxReplica                   int32
}

type Task struct {
	Name         string            `yaml:"Name"`
	Description  string            `yaml:"Description"`
	ImageURL     string            `yaml:"ImageURL"`
	Command      string            `yaml:"Command"`
	WorkloadType WorkloadType      `yaml:"WorkloadType"`
	EnvVars      map[string]string `yaml:"EnvVars"`
	Interval     uint64            `yaml:"Interval"`
	NatsConfig   NatsConfig        `yaml:"NatsConfig"`
	ScaleConfig  ScaleConfig       `yaml:"ScaleConfig"`
}
