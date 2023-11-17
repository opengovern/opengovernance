package resources

import (
	"fmt"
	azure "github.com/kaytu-io/kaytu-engine/pkg/workspace/costestimator/azure_terracost/resource_types"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/costestimator/query"
	"go.uber.org/zap"
)

type ResourceRequest struct {
	Address string
	Request any
}

func GetResource(logger *zap.Logger, provider string, resourceType string, request ResourceRequest) (*query.Resource, error) {
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
		fmt.Println("READING COMPONENTS", request)
		components, err := provider.ResourceComponents(logger, resourceType, request)
		if err != nil {
			return nil, err
		}
		resource.Components = components
	}
	logger.Info("Components", zap.Any("Components", resource.Components))
	return &resource, nil
}
