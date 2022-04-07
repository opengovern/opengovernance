package api

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/turbot/go-kit/helpers"
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
	ctx context.Context, client keibi.Client, req *GetResourcesRequest, provider *SourceType,
) (*GetResourcesResult, error) {
	if req.Filters.ResourceType == nil || len(req.Filters.ResourceType) == 0 {
		return nil, nil
	}

	idx, err := req.Page.GetIndex()
	if err != nil {
		return nil, err
	}

	page, err := req.Page.NextPage()
	if err != nil {
		return nil, err
	}

	result := GetResourcesResult{
		Page: page,
	}
	for _, resourceType := range req.Filters.ResourceType {
		var response ResourceQueryResponse
		indexName := describe.ResourceTypeToESIndex(resourceType)

		sourceType := SourceTypeByResourceType(resourceType)

		terms := make(map[string][]string)
		if !FilterIsEmpty(req.Filters.Location) {
			if sourceType == SourceCloudAWS {
				terms["metadata.region.keyword"] = req.Filters.Location
			} else {
				terms["metadata.location.keyword"] = req.Filters.Location
			}
		}

		if !FilterIsEmpty(req.Filters.ResourceType) {
			terms["resource_type.keyword"] = req.Filters.ResourceType
		}

		if !FilterIsEmpty(req.Filters.SourceID) {
			if sourceType == SourceCloudAWS {
				terms["account_id.keyword"] = req.Filters.SourceID
			} else {
				terms["subscription_id.keyword"] = req.Filters.SourceID
			}
		}

		if provider != nil {
			terms["source_type.keyword"] = []string{string(*provider)}
		}

		query, err := BuildResourceQuery(req.Query, terms, req.Page.Size, idx, req.Sorts, sourceType)
		if err != nil {
			return nil, err
		}

		err = client.Search(ctx,
			indexName,
			query,
			&response,
		)
		if err != nil {
			return nil, err
		}

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
					Name:         metadata.Name,
					ResourceType: resourceType,
					ResourceID:   hit.Source.ID,
					Region:       metadata.Region,
					AccountID:    metadata.AccountID,
					Attributes:   make(map[string]string),
				}

				desc, err := convertToDescription(resourceType, hit.Source)
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
					Name:           metadata.Name,
					ResourceType:   resourceType,
					ResourceID:     hit.Source.ID,
					ResourceGroup:  resourceGroup,
					Location:       metadata.Location,
					SubscriptionID: metadata.SubscriptionID,
					Attributes:     make(map[string]string),
				}

				desc, err := convertToDescription(resourceType, hit.Source)
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
				Name:         aws.Name,
				Provider:     SourceCloudAWS,
				ResourceType: aws.ResourceType,
				Location:     aws.Region,
				ResourceID:   aws.ResourceID,
				SourceID:     aws.AccountID,
				Attributes:   aws.Attributes,
			})
		}

		for _, azure := range result.AzureResources {
			result.AllResources = append(result.AllResources, AllResource{
				Name:         azure.Name,
				Provider:     SourceCloudAzure,
				ResourceType: azure.ResourceType,
				Location:     azure.Location,
				ResourceID:   azure.ResourceID,
				SourceID:     azure.SubscriptionID,
				Attributes:   azure.Attributes,
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
			field = "id.keyword"
		case SortFieldName:
			field = "metadata.name.keyword"
		case SortFieldSourceType:
			field = "source_type.keyword"
		case SortFieldResourceType:
			field = "resource_type.keyword"
		case SortFieldResourceGroup:
			field = "description.ResourceGroup.keyword"
		case SortFieldLocation:
			if provider == SourceCloudAWS {
				field = "metadata.region.keyword"
			} else {
				field = "metadata.location.keyword"
			}

		case SortFieldSourceID:
			if provider == SourceCloudAWS {
				field = "metadata.account_id.keyword"
			} else {
				field = "metadata.subscription_id.keyword"
			}
		}

		dir := string(item.Direction)
		result = append(result, map[string]interface{}{field: dir})
	}
	return result
}

func SourceTypeByResourceType(resourceType string) SourceType {
	if strings.HasPrefix(strings.ToLower(resourceType), "aws") {
		return SourceCloudAWS
	} else {
		return SourceCloudAzure
	}
}

func convertToDescription(resourceType string, data interface{}) (interface{}, error) {
	b, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	sourceType := SourceTypeByResourceType(resourceType)
	if sourceType == SourceCloudAWS {
		d := steampipe.AWSDescriptionMap[resourceType]
		err = json.Unmarshal(b, d)
		if err != nil {
			return nil, err
		}
		d = helpers.DereferencePointer(d)
		return d, nil
	} else {
		d := steampipe.AWSDescriptionMap[resourceType]
		err = json.Unmarshal(b, &d)
		if err != nil {
			return nil, err
		}
		d = helpers.DereferencePointer(d)
		return d, nil
	}
}
