package inventory

type SourceType string

const (
	SourceCloudAWS   SourceType = "AWS"
	SourceCloudAzure SourceType = "Azure"
)

type LocationByProviderResponse struct {
	Name string `json:"name"`
}

type GetResourceRequest struct {
	Filters Filters `json:"filters" validate:"required"`
	Page    Page    `json:"page"`
}

type Page struct {
	NextMarker []byte `json:"next_marker"`
	Size       int    `json:"size"`
}

type Filters struct {
	ResourceType []string `json:"resourceType"`
	Location     []string `json:"location"`
	KeibiSource  []string `json:"keibi_source"`
}

type GetResourceResponse struct {
	Resources []AllResource `json:"resources"`
	Page      Page          `json:"page"`
}

type AllResource struct {
	Name          string              `json:"name"`
	Provider      SourceType `json:"provider"`
	ResourceType  string              `json:"resource_type"`
	Location      string              `json:"location"`
	ResourceID    string              `json:"resource_id"`
	KeibiSourceID string              `json:"keibi_source_id"`
}

type GetAzureResourceResponse struct {
	Resources []AzureResource `json:"resources"`
	Page      Page          `json:"page"`
}

type AzureResource struct {
	Name           string `json:"name"`
	Type           string `json:"type"`
	ResourceGroup  string `json:"resource_group"`
	Location       string `json:"location"`
	ResourceID     string `json:"resource_id"`
	SubscriptionID string `json:"subscription_id"`
}

type GetAWSResourceResponse struct {
	Resources []AWSResource `json:"resources"`
	Page      Page          `json:"page"`
}

type AWSResource struct {
	Name         string `json:"name"`
	ResourceType string `json:"resource_type"`
	ResourceID   string `json:"resource_id"`
	Region       string `json:"location"`
	AccountID    string `json:"account_id"`
}
