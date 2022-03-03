package inventory

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
	Provider     []string `json:"provider"`
	ResourceType []string `json:"resourceType"`
	Location     []string `json:"location"`
	KeibiSource  []string `json:"keibi_source"`
}

type GetResourceResponse struct {
	Resources []Resource `json:"resources"`
	Page      Page       `json:"page"`
}

type Resource struct {
	ID            string `json:"id"`
	ResourceType  string `json:"resource_type"`
	Location      string `json:"location"`
	KeibiSourceID string `json:"keibi_source"`
}
