package api

import (
	"context"

	"gitlab.com/keibiengine/keibi-engine/pkg/cloudservice"

	internal "gitlab.com/keibiengine/keibi-engine/pkg/internal/api"
	"gitlab.com/keibiengine/keibi-engine/pkg/keibi-es-sdk"
)

type GetResourcesResult struct {
	AllResources   []AllResource
	AzureResources []AzureResource
	AWSResources   []AWSResource
	Page           internal.Page
}

func QueryResources(ctx context.Context, client keibi.Client, req *GetResourcesRequest, provider *SourceType) (*GetResourcesResult, error) {
	if req.Filters.ResourceType == nil || len(req.Filters.ResourceType) == 0 {
		return QueryResourcesFromInventorySummary(ctx, client, req, provider)
	} else {
		return QueryResourcesWithSteampipeColumns(ctx, client, req, provider)
	}
}

func QueryResourcesFromInventorySummary(ctx context.Context, client keibi.Client, req *GetResourcesRequest, provider *SourceType) (*GetResourcesResult, error) {
	lastIdx, err := req.Page.GetIndex()
	if err != nil {
		return nil, err
	}

	resources, err := QuerySummaryResources(ctx, client, req.Query, req.Filters, provider, req.Page.Size, lastIdx, req.Sorts)
	if err != nil {
		return nil, err
	}

	page, err := req.Page.NextPage()
	if err != nil {
		return nil, err
	}

	if provider != nil && *provider == SourceCloudAWS {
		var awsResources []AWSResource
		for _, resource := range resources {
			awsResources = append(awsResources, AWSResource{
				Name:             resource.Name,
				ResourceType:     resource.ResourceType,
				ResourceTypeName: cloudservice.ServiceNameByResourceType(resource.ResourceType),
				ResourceID:       resource.ResourceID,
				Region:           resource.Location,
				AccountID:        resource.SourceID,
			})
		}
		return &GetResourcesResult{
			AWSResources: awsResources,
			Page:         page,
		}, nil
	}

	if provider != nil && *provider == SourceCloudAzure {
		var azureResources []AzureResource
		for _, resource := range resources {
			azureResources = append(azureResources, AzureResource{
				Name:             resource.Name,
				ResourceType:     resource.ResourceType,
				ResourceTypeName: cloudservice.ServiceNameByResourceType(resource.ResourceType),
				ResourceGroup:    resource.ResourceGroup,
				Location:         resource.Location,
				ResourceID:       resource.ResourceID,
				SubscriptionID:   resource.SourceID,
			})
		}
		return &GetResourcesResult{
			AzureResources: azureResources,
			Page:           page,
		}, nil
	}

	var allResources []AllResource
	for _, resource := range resources {
		allResources = append(allResources, AllResource{
			Name:             resource.Name,
			Provider:         SourceType(resource.SourceType),
			ResourceType:     resource.ResourceType,
			ResourceTypeName: cloudservice.ServiceNameByResourceType(resource.ResourceType),
			Location:         resource.Location,
			ResourceID:       resource.ResourceID,
			SourceID:         resource.SourceID,
		})
	}
	return &GetResourcesResult{
		AllResources: allResources,
		Page:         page,
	}, nil
}
