package api

import (
	"context"

	"github.com/kaytu-io/kaytu-util/pkg/source"

	"github.com/kaytu-io/kaytu-util/pkg/keibi-es-sdk"
)

type GetResourcesResult struct {
	AllResources   []AllResource
	AzureResources []AzureResource
	AWSResources   []AWSResource
	TotalCount     int64 `json:"totalCount"`
}

func QueryResources(ctx context.Context, client keibi.Client, req *GetResourcesRequest, connector []source.Type) (*GetResourcesResult, error) {
	if len(req.Filters.ResourceType) == 1 {
		return QueryResourcesWithSteampipeColumns(ctx, client, req, connector)
	} else {
		return QueryResourcesFromInventorySummary(ctx, client, req, connector)
	}
}

func QueryResourcesFromInventorySummary(ctx context.Context, client keibi.Client, req *GetResourcesRequest, connectors []source.Type) (*GetResourcesResult, error) {
	lastIdx := (req.Page.No - 1) * req.Page.Size

	resources, resultCount, err := QuerySummaryResources(ctx, client, req.Query, req.Filters, connectors, req.Page.Size, lastIdx, req.Sorts)
	if err != nil {
		return nil, err
	}

	var awsResources []AWSResource
	var azureResources []AzureResource
	var allResources []AllResource
	for _, resource := range resources {
		allResources = append(allResources, AllResource{
			ResourceName:         resource.Name,
			Connector:            resource.SourceType,
			ResourceType:         resource.ResourceType,
			ConnectionID:         resource.SourceID,
			Location:             resource.Location,
			ResourceID:           resource.ResourceID,
			ProviderConnectionID: resource.SourceID,
		})
		switch resource.SourceType {
		case source.CloudAWS:
			awsResources = append(awsResources, AWSResource{
				ResourceName: resource.Name,
				ResourceType: resource.ResourceType,
				ResourceID:   resource.ResourceID,
				Location:     resource.Location,
				ConnectionID: resource.SourceID,
			})
		case source.CloudAzure:
			azureResources = append(azureResources, AzureResource{
				ResourceName:  resource.Name,
				ResourceType:  resource.ResourceType,
				ResourceGroup: resource.ResourceGroup,
				Location:      resource.Location,
				ResourceID:    resource.ResourceID,
				ConnectionID:  resource.SourceID,
			})
		}
	}
	return &GetResourcesResult{
		AllResources: allResources,
		//AWSResources:   awsResources,
		//AzureResources: azureResources,
		TotalCount: resultCount.Value,
	}, nil
}
