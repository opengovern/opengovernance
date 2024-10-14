package api

import (
	"time"

	"github.com/opengovern/og-util/pkg/opengovernance-es-sdk"
	"github.com/opengovern/og-util/pkg/source"
)

type ResourceCollectionStatus string

const (
	ResourceCollectionStatusUnknown  ResourceCollectionStatus = ""
	ResourceCollectionStatusActive   ResourceCollectionStatus = "active"
	ResourceCollectionStatusInactive ResourceCollectionStatus = "inactive"
)

type ResourceCollection struct {
	ID          string                                    `json:"id"`
	Name        string                                    `json:"name"`
	Tags        map[string][]string                       `json:"tags"`
	Description string                                    `json:"description"`
	CreatedAt   time.Time                                 `json:"created_at"`
	Status      ResourceCollectionStatus                  `json:"status"`
	Filters     []opengovernance.ResourceCollectionFilter `json:"filters"`

	Connectors      []source.Type `json:"connectors,omitempty"`
	LastEvaluatedAt *time.Time    `json:"last_evaluated_at,omitempty"`
	ResourceCount   *int          `json:"resource_count,omitempty"`
	ConnectionCount *int          `json:"connection_count,omitempty"`
	MetricCount     *int          `json:"metric_count,omitempty"`
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
