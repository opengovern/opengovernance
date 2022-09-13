package api

import (
	"context"

	"gitlab.com/keibiengine/keibi-engine/pkg/cloudservice"

	"gitlab.com/keibiengine/keibi-engine/pkg/keibi-es-sdk"
)

type GetResourcesResult struct {
	AllResources   []AllResource
	AzureResources []AzureResource
	AWSResources   []AWSResource
	TotalCount     int64 `json:"totalCount,omitempty"`
}

func QueryResources(ctx context.Context, client keibi.Client, req *GetResourcesRequest, provider *SourceType, commonFilter *bool) (*GetResourcesResult, error) {
	if req.Filters.ResourceType == nil || len(req.Filters.ResourceType) == 0 {
		return QueryResourcesFromInventorySummary(ctx, client, req, provider, commonFilter)
	} else {
		return QueryResourcesWithSteampipeColumns(ctx, client, req, provider, commonFilter)
	}
}

func QueryResourcesFromInventorySummary(ctx context.Context, client keibi.Client, req *GetResourcesRequest, provider *SourceType, commonFilter *bool) (*GetResourcesResult, error) {
	lastIdx := (req.Page.No - 1) * req.Page.Size

	resources, resultCount, err := QuerySummaryResources(ctx, client, req.Query, req.Filters, provider, req.Page.Size, lastIdx, req.Sorts, commonFilter)
	if err != nil {
		return nil, err
	}

	if provider != nil && *provider == SourceCloudAWS {
		var awsResources []AWSResource
		for _, resource := range resources {
			awsResources = append(awsResources, AWSResource{
				ResourceName:         resource.Name,
				ResourceType:         resource.ResourceType,
				ResourceTypeName:     cloudservice.ResourceTypeName(resource.ResourceType),
				ResourceCategory:     cloudservice.CategoryByResourceType(resource.ResourceType),
				ResourceID:           resource.ResourceID,
				Location:             resource.Location,
				ProviderConnectionID: resource.SourceID,
			})
		}
		return &GetResourcesResult{
			AWSResources: awsResources,
			TotalCount:   resultCount.Value,
		}, nil
	}

	if provider != nil && *provider == SourceCloudAzure {
		var azureResources []AzureResource
		for _, resource := range resources {
			azureResources = append(azureResources, AzureResource{
				ResourceName:         resource.Name,
				ResourceType:         resource.ResourceType,
				ResourceTypeName:     cloudservice.ResourceTypeName(resource.ResourceType),
				ResourceCategory:     cloudservice.CategoryByResourceType(resource.ResourceType),
				ResourceGroup:        resource.ResourceGroup,
				Location:             resource.Location,
				ResourceID:           resource.ResourceID,
				ProviderConnectionID: resource.SourceID,
			})
		}
		return &GetResourcesResult{
			AzureResources: azureResources,
			TotalCount:     resultCount.Value,
		}, nil
	}

	var allResources []AllResource
	for _, resource := range resources {
		allResources = append(allResources, AllResource{
			ResourceName:         resource.Name,
			Provider:             SourceType(resource.SourceType),
			ResourceType:         resource.ResourceType,
			ResourceTypeName:     cloudservice.ResourceTypeName(resource.ResourceType),
			ResourceCategory:     cloudservice.CategoryByResourceType(resource.ResourceType),
			Location:             resource.Location,
			ResourceID:           resource.ResourceID,
			ProviderConnectionID: resource.SourceID,
		})
	}
	return &GetResourcesResult{
		AllResources: allResources,
		TotalCount:   resultCount.Value,
	}, nil
}
