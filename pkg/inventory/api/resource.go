package api

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	awsSteampipe "github.com/kaytu-io/kaytu-aws-describer/pkg/steampipe"
	azureSteampipe "github.com/kaytu-io/kaytu-azure-describer/pkg/steampipe"
	"github.com/kaytu-io/kaytu-util/pkg/steampipe"

	"github.com/kaytu-io/kaytu-engine/pkg/types"

	"github.com/kaytu-io/kaytu-util/pkg/source"

	awsmodel "github.com/kaytu-io/kaytu-aws-describer/aws/model"
	azuremodel "github.com/kaytu-io/kaytu-azure-describer/azure/model"
	"github.com/kaytu-io/kaytu-util/pkg/keibi-es-sdk"
	"github.com/turbot/steampipe-plugin-sdk/v4/grpc/proto"
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
	Provider      source.Type
	ResourceType  string
	ResourceGroup string
	Location      string
	ResourceID    string
	SourceID      string

	SteampipeColumns map[string]string
}

func QueryResourcesWithSteampipeColumns(
	ctx context.Context, client keibi.Client, req *GetResourcesRequest, connector []source.Type) (*GetResourcesResult, error) {
	if req.Filters.ResourceType == nil || len(req.Filters.ResourceType) == 0 {
		return nil, nil
	}

	idx := (req.Page.No - 1) * req.Page.Size

	result := GetResourcesResult{
		TotalCount: 0,
	}
	for _, resourceType := range req.Filters.ResourceType {
		var response ResourceQueryResponse
		indexName := types.ResourceTypeToESIndex(resourceType)

		var sourceType source.Type
		if strings.HasPrefix(strings.ToLower(resourceType), "aws") {
			sourceType = source.CloudAWS
		} else if strings.HasPrefix(strings.ToLower(resourceType), "microsoft") {
			sourceType = source.CloudAzure
		}

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

		if !FilterIsEmpty(req.Filters.ConnectionID) {
			terms["source_id"] = req.Filters.ConnectionID
		}

		if len(connector) > 0 {
			connectorStr := make([]string, 0, len(connector))
			for _, c := range connector {
				connectorStr = append(connectorStr, c.String())
			}
			terms["source_type"] = connectorStr
		}

		query, err := BuildResourceQuery(req.Query, terms, req.Page.Size, idx, req.Sorts, sourceType)
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

		pluginProvider := steampipe.ExtractPlugin(resourceType)
		for _, hit := range response.Hits.Hits {
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
					ResourceID:           hit.Source.ID,
					Location:             metadata.Region,
					ProviderConnectionID: metadata.SourceID,
					Attributes:           make(map[string]string),
				}

				desc, err := steampipe.ConvertToDescription(resourceType, hit.Source, awsSteampipe.AWSDescriptionMap)
				if err != nil {
					return nil, err
				}

				pluginTableName := awsSteampipe.ExtractTableName(resourceType)
				cells, err := awsSteampipe.AWSDescriptionToRecord(desc, pluginTableName)
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
					ResourceID:           hit.Source.ID,
					ResourceGroup:        resourceGroup,
					Location:             metadata.Location,
					ProviderConnectionID: metadata.SourceID,
					Attributes:           make(map[string]string),
				}

				desc, err := steampipe.ConvertToDescription(resourceType, hit.Source, azureSteampipe.AzureDescriptionMap)
				if err != nil {
					return nil, err
				}
				pluginTableName := azureSteampipe.ExtractTableName(resourceType)

				var cells map[string]*proto.Column
				if pluginProvider == steampipe.SteampipePluginAzure {
					cells, err = azureSteampipe.AzureDescriptionToRecord(desc, pluginTableName)
					if err != nil {
						return nil, err
					}
					for colName, cell := range cells {
						resource.Attributes[colName] = cell.String()
					}
				} else {
					cells, err = azureSteampipe.AzureADDescriptionToRecord(desc, pluginTableName)
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

	for _, aws := range result.AWSResources {
		result.AllResources = append(result.AllResources, AllResource{
			ResourceName:         aws.ResourceName,
			Connector:            source.CloudAWS,
			ResourceType:         aws.ResourceType,
			Location:             aws.Location,
			ResourceID:           aws.ResourceID,
			ProviderConnectionID: aws.ProviderConnectionID,
			Attributes:           aws.Attributes,
		})
	}

	for _, azure := range result.AzureResources {
		result.AllResources = append(result.AllResources, AllResource{
			ResourceName:         azure.ResourceName,
			Connector:            source.CloudAzure,
			ResourceType:         azure.ResourceType,
			Location:             azure.Location,
			ResourceID:           azure.ResourceID,
			ProviderConnectionID: azure.ProviderConnectionID,
			Attributes:           azure.Attributes,
		})
	}

	return &result, nil
}

func BuildResourceQuery(query string, terms map[string][]string, size, lastIdx int, sorts []ResourceSortItem, provider source.Type) (string, error) {
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

func BuildSortResource(sorts []ResourceSortItem, connector source.Type) []map[string]interface{} {
	var result []map[string]interface{}
	for _, item := range sorts {
		field := ""
		switch item.Field {
		case SortFieldResourceID:
			field = "id"
		case SortFieldConnector:
			field = "source_type"
		case SortFieldResourceType:
			field = "resource_type"
		case SortFieldResourceGroup:
			field = "description.ResourceGroup"
		case SortFieldLocation:
			switch connector {
			case source.CloudAWS:
				field = "metadata.region"
			case source.CloudAzure:
				field = "metadata.location"
			}
		case SortFieldConnectionID:
			field = "source_id"
		}

		dir := string(item.Direction)
		result = append(result, map[string]interface{}{field: dir})
	}
	return result
}
