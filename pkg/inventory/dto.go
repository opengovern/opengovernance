package inventory

import (
	"gitlab.com/keibiengine/keibi-engine/pkg/describe"
	"gitlab.com/keibiengine/keibi-engine/pkg/keibi-es-sdk"
)

type SourceType string

const (
	SourceCloudAWS   SourceType = "AWS"
	SourceCloudAzure SourceType = "Azure"
)

type GetResourceRequest struct {
	ResourceType string `json:"resourceType" validate:"required"`
	ID           string `json:"ID" validate:"required"`
}

type LocationByProviderResponse struct {
	Name string `json:"name"`
}

type GetResourcesRequest struct {
	Filters Filters `json:"filters" validate:"required"`
	Page    Page    `json:"page"`
}

type Page struct {
	NextMarker []byte `json:"nextMarker"`
	Size       int    `json:"size"`
}

type Filters struct {
	ResourceType []string `json:"resourceType"`
	Location     []string `json:"location"`
	KeibiSource  []string `json:"keibiSource"`
}

type GetResourcesResponse struct {
	Resources []AllResource `json:"resourcesces"`
	Page      Page          `json:"page"`
}

type AllResource struct {
	Name         string     `json:"name"`
	Provider     SourceType `json:"provider"`
	ResourceType string     `json:"resourceType"`
	Location     string     `json:"location"`
	ResourceID   string     `json:"resourceID"`
	SourceID     string     `json:"sourceID"`
}

type GetAzureResourceResponse struct {
	Resources []AzureResource `json:"resources"`
	Page      Page            `json:"page"`
}

type AzureResource struct {
	Name           string `json:"name"`
	ResourceType   string `json:"resourceType"`
	ResourceGroup  string `json:"resourceGroup"`
	Location       string `json:"location"`
	ResourceID     string `json:"resourceID"`
	SubscriptionID string `json:"subscriptionID"`
}

type GetAWSResourceResponse struct {
	Resources []AWSResource `json:"resources"`
	Page      Page          `json:"page"`
}

type AWSResource struct {
	Name         string `json:"name"`
	ResourceType string `json:"resourceType"`
	ResourceID   string `json:"resourceID"`
	Region       string `json:"location"`
	AccountID    string `json:"accountID"`
}

type SummaryQueryResponse struct {
	Hits SummaryQueryHits `json:"hits"`
}
type SummaryQueryHits struct {
	Total keibi.SearchTotal `json:"total"`
	Hits  []SummaryQueryHit `json:"hits"`
}
type SummaryQueryHit struct {
	ID      string                       `json:"_id"`
	Score   float64                      `json:"_score"`
	Index   string                       `json:"_index"`
	Type    string                       `json:"_type"`
	Version int64                        `json:"_version,omitempty"`
	Source  describe.KafkaLookupResource `json:"_source"`
	Sort    []interface{}                `json:"sort"`
}

type GenericQueryResponse struct {
	Hits GenericQueryHits `json:"hits"`
}
type GenericQueryHits struct {
	Total keibi.SearchTotal `json:"total"`
	Hits  []GenericQueryHit `json:"hits"`
}
type GenericQueryHit struct {
	ID      string                 `json:"_id"`
	Score   float64                `json:"_score"`
	Index   string                 `json:"_index"`
	Type    string                 `json:"_type"`
	Version int64                  `json:"_version,omitempty"`
	Source  map[string]interface{} `json:"_source"`
	Sort    []interface{}          `json:"sort"`
}
