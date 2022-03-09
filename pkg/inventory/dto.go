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
	NextMarker []byte `json:"nextMarker"`
	Size       int    `json:"size"`
}

type Filters struct {
	ResourceType []string `json:"resourceType"`
	Location     []string `json:"location"`
	KeibiSource  []string `json:"keibiSource"`
}

type GetResourceResponse struct {
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
