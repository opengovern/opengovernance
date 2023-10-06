package api

import "github.com/kaytu-io/kaytu-util/pkg/kaytu-es-sdk"

type ResourceCollection struct {
	ID      string                           `json:"id"`
	Name    string                           `json:"name"`
	Tags    map[string][]string              `json:"tags"`
	Filters []kaytu.ResourceCollectionFilter `json:"filters"`
}
