package resources

import (
	azure "github.com/kaytu-io/kaytu-engine/pkg/workspace/costestimator/azure_terracost/resource_types"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/costestimator/query"
)

type ResourceRequest struct {
	Address string
	Request any
}

func GetResource(provider string, resourceType string, request ResourceRequest) (*query.Resource, error) {
	var resource query.Resource
	if provider == "AWS" {
		resource = query.Resource{
			Address:    request.Address,
			Provider:   provider,
			Type:       resourceType,
			Components: nil,
		}
	} else if provider == "Azure" {
		resource = query.Resource{
			Address:    request.Address,
			Provider:   provider,
			Type:       resourceType,
			Components: nil,
		}
		provider, err := azure.NewProvider(provider)
		if err != nil {
			return nil, err
		}
		components, err := provider.ResourceComponents(resourceType, request)
		if err != nil {
			return nil, err
		}
		resource.Components = components
	}
	return &resource, nil
}
