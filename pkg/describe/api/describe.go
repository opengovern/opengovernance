package api

import "gitlab.com/keibiengine/keibi-engine/pkg/source"

type DescribeSingleResourceRequest struct {
	Provider         source.Type `json:"provider"`
	ResourceType     string
	AccountID        string
	AccessKey        string
	SecretKey        string
	AdditionalFields map[string]string
}
