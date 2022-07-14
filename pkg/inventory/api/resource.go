package api

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"gitlab.com/keibiengine/keibi-engine/pkg/source"

	"gitlab.com/keibiengine/keibi-engine/pkg/cloudservice"

	"github.com/turbot/steampipe-plugin-sdk/grpc/proto"
	awsmodel "gitlab.com/keibiengine/keibi-engine/pkg/aws/model"
	azuremodel "gitlab.com/keibiengine/keibi-engine/pkg/azure/model"
	"gitlab.com/keibiengine/keibi-engine/pkg/describe"
	"gitlab.com/keibiengine/keibi-engine/pkg/keibi-es-sdk"
	"gitlab.com/keibiengine/keibi-engine/pkg/steampipe"
)

type ResourceObj struct {
	Metadata    interface{} `json:"metadata"`
	Description interface{} `json:"description"`
	SourceType  string      `json:"source_type"`
	ID          string      `json:"id"`
}
type ResourceQueryResponse struct {
	Hits ResourceQueryHits `json:"hits"`
}
type ResourceQueryHits struct {
	Total keibi.SearchTotal  `json:"total"`
	Hits  []ResourceQueryHit `json:"hits"`
}
type ResourceQueryHit struct {
	ID      string        `json:"_id"`
	Score   float64       `json:"_score"`
	Index   string        `json:"_index"`
	Type    string        `json:"_type"`
	Version int64         `json:"_version,omitempty"`
	Source  ResourceObj   `json:"_source"`
	Sort    []interface{} `json:"sort"`
}

type SteampipeResource struct {
	Name          string
	Provider      SourceType
	ResourceType  string
	ResourceGroup string
	Location      string
	ResourceID    string
	SourceID      string

	SteampipeColumns map[string]string
}

func QueryResourcesWithSteampipeColumns(
	ctx context.Context, client keibi.Client, req *GetResourcesRequest, provider *SourceType, commonFilter *bool,
) (*GetResourcesResult, error) {
	if req.Filters.ResourceType == nil || len(req.Filters.ResourceType) == 0 {
		return nil, nil
	}

	idx := (req.Page.No - 1) * req.Page.Size

	result := GetResourcesResult{
		TotalCount: 0,
	}
	for _, resourceType := range req.Filters.ResourceType {
		if commonFilter != nil {
			isCommon := cloudservice.IsCommonByResourceType(resourceType)
			if (!isCommon && *commonFilter) || (isCommon && !*commonFilter) {
				continue
			}
		}

		var response ResourceQueryResponse
		indexName := describe.ResourceTypeToESIndex(resourceType)

		sourceType := steampipe.SourceTypeByResourceType(resourceType)

		terms := make(map[string][]string)
		if !FilterIsEmpty(req.Filters.Location) {
			if sourceType == source.CloudAWS {
				terms["metadata.region"] = req.Filters.Location
			} else {
				terms["metadata.location"] = req.Filters.Location
			}
		}

		if !FilterIsEmpty(req.Filters.ResourceType) {
			terms["resource_type"] = req.Filters.ResourceType
		}

		if !FilterIsEmpty(req.Filters.SourceID) {
			terms["source_id"] = req.Filters.SourceID
		}

		if provider != nil {
			terms["source_type"] = []string{string(*provider)}
		}

		query, err := BuildResourceQuery(req.Query, terms, req.Page.Size, idx, req.Sorts, SourceType(sourceType))
		if err != nil {
			return nil, err
		}

		err = client.SearchWithTrackTotalHits(ctx,
			indexName,
			query,
			&response,
			true,
		)
		if err != nil {
			return nil, err
		}

		result.TotalCount += response.Hits.Total.Value

		for _, hit := range response.Hits.Hits {
			pluginProvider := steampipe.ExtractPlugin(resourceType)
			pluginTableName := steampipe.ExtractTableName(resourceType)

			if pluginProvider == steampipe.SteampipePluginAWS {
				b, err := json.Marshal(hit.Source.Metadata)
				if err != nil {
					return nil, err
				}
				var metadata awsmodel.Metadata
				err = json.Unmarshal(b, &metadata)
				if err != nil {
					return nil, err
				}

				resource := AWSResource{
					ResourceName:         metadata.Name,
					ResourceType:         resourceType,
					ResourceTypeName:     cloudservice.ServiceNameByResourceType(resourceType),
					ResourceID:           hit.Source.ID,
					Location:             metadata.Region,
					ProviderConnectionID: metadata.AccountID,
					Attributes:           make(map[string]string),
				}

				desc, err := steampipe.ConvertToDescription(resourceType, hit.Source)
				if err != nil {
					return nil, err
				}

				cells, err := steampipe.AWSDescriptionToRecord(desc, pluginTableName)
				if err != nil {
					return nil, err
				}
				for colName, cell := range cells {
					resource.Attributes[colName] = cell.String()
				}
				result.AWSResources = append(result.AWSResources, resource)
			} else if pluginProvider == steampipe.SteampipePluginAzure || pluginProvider == steampipe.SteampipePluginAzureAD {
				b, err := json.Marshal(hit.Source.Metadata)
				if err != nil {
					return nil, err
				}
				var metadata azuremodel.Metadata
				err = json.Unmarshal(b, &metadata)
				if err != nil {
					return nil, err
				}

				var resourceGroup string
				arr := strings.Split(metadata.ID, "/")
				if len(arr) > 4 {
					resourceGroup = arr[4]
				}

				resource := AzureResource{
					ResourceName:         metadata.Name,
					ResourceType:         resourceType,
					ResourceTypeName:     cloudservice.ServiceNameByResourceType(resourceType),
					ResourceID:           hit.Source.ID,
					ResourceGroup:        resourceGroup,
					Location:             metadata.Location,
					ProviderConnectionID: metadata.SubscriptionID,
					Attributes:           make(map[string]string),
				}

				desc, err := steampipe.ConvertToDescription(resourceType, hit.Source)
				if err != nil {
					return nil, err
				}

				var cells map[string]*proto.Column
				if pluginProvider == steampipe.SteampipePluginAzure {
					cells, err = steampipe.AzureDescriptionToRecord(desc, pluginTableName)
					if err != nil {
						return nil, err
					}
					for colName, cell := range cells {
						resource.Attributes[colName] = cell.String()
					}
				} else {
					cells, err = steampipe.AzureADDescriptionToRecord(desc, pluginTableName)
					if err != nil {
						return nil, err
					}
					for colName, cell := range cells {
						resource.Attributes[colName] = cell.String()
					}
				}
				result.AzureResources = append(result.AzureResources, resource)
			} else {
				return nil, errors.New("invalid provider")
			}
		}
	}

	if provider == nil {
		for _, aws := range result.AWSResources {
			result.AllResources = append(result.AllResources, AllResource{
				ResourceName:         aws.ResourceName,
				Provider:             SourceCloudAWS,
				ResourceType:         aws.ResourceType,
				ResourceTypeName:     cloudservice.ServiceNameByResourceType(aws.ResourceType),
				Location:             aws.Location,
				ResourceID:           aws.ResourceID,
				ProviderConnectionID: aws.ProviderConnectionID,
				Attributes:           aws.Attributes,
			})
		}

		for _, azure := range result.AzureResources {
			result.AllResources = append(result.AllResources, AllResource{
				ResourceName:         azure.ResourceName,
				Provider:             SourceCloudAzure,
				ResourceType:         azure.ResourceType,
				ResourceTypeName:     cloudservice.ServiceNameByResourceType(azure.ResourceType),
				Location:             azure.Location,
				ResourceID:           azure.ResourceID,
				ProviderConnectionID: azure.ProviderConnectionID,
				Attributes:           azure.Attributes,
			})
		}
	}

	return &result, nil
}

func BuildResourceQuery(query string, terms map[string][]string, size, lastIdx int, sorts []ResourceSortItem, provider SourceType) (string, error) {
	q := map[string]interface{}{
		"size": size,
		"from": lastIdx,
	}
	if sorts != nil && len(sorts) > 0 {
		q["sort"] = BuildSortResource(sorts, provider)
	}

	boolQuery := make(map[string]interface{})
	if terms != nil && len(terms) > 0 {
		var filters []map[string]interface{}
		for k, vs := range terms {
			filters = append(filters, map[string]interface{}{
				"terms": map[string][]string{
					k: vs,
				},
			})
		}

		boolQuery["filter"] = filters
	}
	if len(query) > 0 {
		boolQuery["must"] = map[string]interface{}{
			"multi_match": map[string]interface{}{
				"fields": []string{"resource_id", "name", "source_type", "resource_type", "resource_group",
					"location", "source_id"},
				"query":     query,
				"fuzziness": "AUTO",
			},
		}
	}
	if len(boolQuery) > 0 {
		q["query"] = map[string]interface{}{
			"bool": boolQuery,
		}
	}

	queryBytes, err := json.Marshal(q)
	if err != nil {
		return "", err
	}
	return string(queryBytes), nil
}

func BuildSortResource(sorts []ResourceSortItem, provider SourceType) []map[string]interface{} {
	var result []map[string]interface{}
	for _, item := range sorts {
		field := ""
		switch item.Field {
		case SortFieldResourceID:
			field = "id"
		case SortFieldName:
			field = "metadata.name"
		case SortFieldSourceType:
			field = "source_type"
		case SortFieldResourceType:
			field = "resource_type"
		case SortFieldResourceGroup:
			field = "description.ResourceGroup"
		case SortFieldLocation:
			if provider == SourceCloudAWS {
				field = "metadata.region"
			} else {
				field = "metadata.location"
			}

		case SortFieldSourceID:
			field = "source_id"
		}

		dir := string(item.Direction)
		result = append(result, map[string]interface{}{field: dir})
	}
	return result
}
