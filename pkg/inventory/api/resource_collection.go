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
