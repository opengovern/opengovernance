package api

import (
	"github.com/kaytu-io/kaytu-util/pkg/kaytu-es-sdk"
	"time"
)

type ResourceCollectionStatus string

const (
	ResourceCollectionStatusUnknown  ResourceCollectionStatus = ""
	ResourceCollectionStatusActive   ResourceCollectionStatus = "active"
	ResourceCollectionStatusInactive ResourceCollectionStatus = "inactive"
)

type ResourceCollection struct {
	ID          string                           `json:"id"`
	Name        string                           `json:"name"`
	Tags        map[string][]string              `json:"tags"`
	Description string                           `json:"description"`
	CreatedAt   time.Time                        `json:"created_at"`
	Status      ResourceCollectionStatus         `json:"status"`
	Filters     []kaytu.ResourceCollectionFilter `json:"filters"`
}

type ResourceCollectionLandscapeItem struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	LogoURI     string `json:"logo_uri"`
}

type ResourceCollectionLandscapeSubcategory struct {
	ID          string                            `json:"id"`
	Name        string                            `json:"name"`
	Description string                            `json:"description"`
	Items       []ResourceCollectionLandscapeItem `json:"items"`
}

type ResourceCollectionLandscapeCategory struct {
	ID            string                                   `json:"id"`
	Name          string                                   `json:"name"`
	Description   string                                   `json:"description"`
	Subcategories []ResourceCollectionLandscapeSubcategory `json:"subcategories"`
}

type ResourceCollectionLandscape struct {
	Categories []ResourceCollectionLandscapeCategory `json:"categories"`
}
