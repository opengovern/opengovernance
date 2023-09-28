package api

import "github.com/kaytu-io/kaytu-util/pkg/source"

type DescribeSingleResourceRequest struct {
	Provider         source.Type `json:"provider"`
	ResourceType     string
	AccountID        string
	AccessKey        string
	SecretKey        string
	AdditionalFields map[string]string
}

type DescribeStatus struct {
	ConnectionID string
	Connector    string
	Status       DescribeResourceJobStatus
}

type ConnectionDescribeStatus struct {
	ResourceType string
	Status       DescribeResourceJobStatus
}
