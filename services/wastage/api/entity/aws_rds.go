package entity

type AwsRdsClusterType string

const (
	AwsRdsClusterTypeSingleInstance     AwsRdsClusterType = "Single-AZ"
	AwsRdsClusterTypeMultiAzOneInstance AwsRdsClusterType = "Multi-AZ"
	AwsRdsClusterTypeMultiAzTwoInstance AwsRdsClusterType = "Multi-AZ (readable standbys)"
)

type AwsRds struct {
	InstanceType  string            `json:"instanceType"`
	Engine        string            `json:"engine"`
	EngineVersion string            `json:"engineVersion"`
	ClusterType   AwsRdsClusterType `json:"clusterType"`

	StorageType       *string `json:"storageType"`
	StorageSize       *int32  `json:"storageSize"`
	StorageIops       *int32  `json:"storageIops"`
	StorageThroughput *int32  `json:"storageThroughput"`
}

type RightsizingAwsRds struct {
	Region        string            `json:"region"`
	InstanceType  string            `json:"instanceType"`
	Engine        string            `json:"engine"`
	EngineVersion string            `json:"engineVersion"`
	ClusterType   AwsRdsClusterType `json:"clusterType"`

	VCPU     int64 `json:"vCPU"`
	MemoryGb int64 `json:"memoryGb"`

	StorageType       *string `json:"storageType"`
	StorageSize       *int32  `json:"storageSize"`
	StorageIops       *int32  `json:"storageIops"`
	StorageThroughput *int32  `json:"storageThroughput"`

	Cost float64 `json:"cost"`
}

type AwsRdsRightsizingRecommendation struct {
	Current     RightsizingAwsRds  `json:"current"`
	Recommended *RightsizingAwsRds `json:"recommended"`

	Description string `json:"description"`
}
