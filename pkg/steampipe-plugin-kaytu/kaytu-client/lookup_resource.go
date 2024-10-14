package kaytu_client

import (
	"context"
	steampipesdk "github.com/opengovern/og-util/pkg/steampipe"
	"runtime"

	es "github.com/opengovern/og-util/pkg/opengovernance-es-sdk"
	"github.com/opengovern/og-util/pkg/source"
	"github.com/opengovern/opengovernance/pkg/steampipe-plugin-kaytu/kaytu-sdk/config"
	"github.com/turbot/steampipe-plugin-sdk/v5/plugin"
)

const (
	InventorySummaryIndex = "inventory_summary"
)

type Tag struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type LookupResource struct {
	// ResourceID is the globally unique ID of the resource.
	ResourceID string `json:"resource_id"`
	// Name is the name of the resource.
	Name string `json:"name"`
	// SourceType is the type of the source of the resource, i.e. AWS Cloud, Azure Cloud.
	SourceType source.Type `json:"source_type"`
	// ResourceType is the type of the resource.
	ResourceType string `json:"resource_type"`
	// ServiceName is the service of the resource.
	ServiceName string `json:"service_name"`
	// Category is the category of the resource.
	Category string `json:"category"`
	// ResourceGroup is the group of resource (Azure only)
	ResourceGroup string `json:"resource_group"`
	// Location is location/region of the resource
	Location string `json:"location"`
	// SourceID is aws account id or azure subscription id
	SourceID string `json:"source_id"`
	// ResourceJobID is the DescribeResourceJob ID that described this resource
	ResourceJobID uint `json:"resource_job_id"`
	// SourceJobID is the DescribeSourceJob ID that the ResourceJobID was created for
	SourceJobID uint `json:"source_job_id"`
	// ScheduleJobID
	ScheduleJobID uint `json:"schedule_job_id"`
	// CreatedAt is when the DescribeSourceJob is created
	CreatedAt int64 `json:"created_at"`
	// IsCommon
	IsCommon bool `json:"is_common"`
	// Tags
	Tags []Tag `json:"tags"`
}

type LookupResourceHit struct {
	ID      string         `json:"_id"`
	Score   float64        `json:"_score"`
	Index   string         `json:"_index"`
	Type    string         `json:"_type"`
	Version int64          `json:"_version,omitempty"`
	Source  LookupResource `json:"_source"`
	Sort    []any          `json:"sort"`
}

type LookupResourceHits struct {
	Total es.SearchTotal      `json:"total"`
	Hits  []LookupResourceHit `json:"hits"`
}

type LookupResourceSearchResponse struct {
	PitID string             `json:"pit_id"`
	Hits  LookupResourceHits `json:"hits"`
}

type LookupResourcePaginator struct {
	paginator *es.BaseESPaginator
}

func (k Client) NewLookupResourcePaginator(filters []es.BoolFilter, limit *int64) (LookupResourcePaginator, error) {
	paginator, err := es.NewPaginator(k.ES.ES(), InventorySummaryIndex, filters, limit)
	if err != nil {
		return LookupResourcePaginator{}, err
	}

	p := LookupResourcePaginator{
		paginator: paginator,
	}

	return p, nil
}

func (p LookupResourcePaginator) HasNext() bool {
	return !p.paginator.Done()
}

func (p LookupResourcePaginator) Close(ctx context.Context) error {
	return p.paginator.Deallocate(ctx)
}

func (p LookupResourcePaginator) NextPage(ctx context.Context) ([]LookupResource, error) {
	var response LookupResourceSearchResponse
	err := p.paginator.SearchWithLog(ctx, &response, true)
	if err != nil {
		return nil, err
	}

	var values []LookupResource
	for _, hit := range response.Hits.Hits {
		values = append(values, hit.Source)
	}

	hits := int64(len(response.Hits.Hits))
	if hits > 0 {
		p.paginator.UpdateState(hits, response.Hits.Hits[hits-1].Sort, response.PitID)
	} else {
		p.paginator.UpdateState(hits, nil, "")
	}

	return values, nil
}

var lookupMapping = map[string]string{
	"connector":     "source_type",
	"region":        "location",
	"connection_id": "source_id",
}

func ListLookupResources(ctx context.Context, d *plugin.QueryData, _ *plugin.HydrateData) (any, error) {
	plugin.Logger(ctx).Trace("ListLookupResources", d)
	runtime.GC()
	// create service
	cfg := config.GetConfig(d.Connection)
	ke, err := config.NewClientCached(cfg, d.ConnectionCache, ctx)
	if err != nil {
		plugin.Logger(ctx).Error("ListLookupResources NewClientCached", "error", err)
		return nil, err
	}
	k := Client{ES: ke}

	sc, err := steampipesdk.NewSelfClientCached(ctx, d.ConnectionCache)
	if err != nil {
		plugin.Logger(ctx).Error("ListLookupResources NewSelfClientCached", "error", err)
		return nil, err
	}
	encodedResourceCollectionFilters, err := sc.GetConfigTableValueOrNil(ctx, steampipesdk.KaytuConfigKeyResourceCollectionFilters)
	if err != nil {
		plugin.Logger(ctx).Error("ListLookupResources GetConfigTableValueOrNil for resource_collection_filters", "error", err)
		return nil, err
	}
	clientType, err := sc.GetConfigTableValueOrNil(ctx, steampipesdk.KaytuConfigKeyClientType)
	if err != nil {
		plugin.Logger(ctx).Error("ListLookupResources GetConfigTableValueOrNil for client_type", "error", err)
		return nil, err
	}

	plugin.Logger(ctx).Trace("Columns", d.FetchType)
	paginator, err := k.NewLookupResourcePaginator(
		es.BuildFilterWithDefaultFieldName(ctx, d.QueryContext, lookupMapping,
			"", nil, encodedResourceCollectionFilters, clientType, true),
		d.QueryContext.Limit)
	if err != nil {
		plugin.Logger(ctx).Error("ListLookupResources NewLookupResourcePaginator", "error", err)
		return nil, err
	}

	for paginator.HasNext() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			plugin.Logger(ctx).Error("ListLookupResources NextPage", "error", err)
			return nil, err
		}
		plugin.Logger(ctx).Trace("ListLookupResources", "next page")

		for _, v := range page {
			d.StreamListItem(ctx, v)
		}
	}

	err = paginator.Close(ctx)
	if err != nil {
		return nil, err
	}

	return nil, nil
}
