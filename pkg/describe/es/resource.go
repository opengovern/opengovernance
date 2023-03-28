package es

import (
	"context"

	"gitlab.com/keibiengine/keibi-engine/pkg/keibi-es-sdk"
)

type FetchResourceByIDResponse struct {
	Resource Resource `json:"_source"`
}

func FetchResourceByID(client keibi.Client, index string, resourceID string) (*Resource, error) {
	resource := FetchResourceByIDResponse{}
	err := client.GetByID(context.TODO(), index, resourceID, &resource)
	if err != nil {
		return nil, err
	}

	if resource.Resource.ID == "" {
		return nil, nil
	}

	return &resource.Resource, nil
}

type FetchLookupResourceByIDResponse struct {
	LookupResource LookupResource `json:"_source"`
}

func FetchLookupResourceByID(client keibi.Client, index string, resourceID string) (*LookupResource, error) {
	lookupResource := FetchLookupResourceByIDResponse{}
	err := client.GetByID(context.TODO(), index, resourceID, &lookupResource)
	if err != nil {
		return nil, err
	}

	if lookupResource.LookupResource.ResourceID == "" {
		return nil, nil
	}

	return &lookupResource.LookupResource, nil
}
