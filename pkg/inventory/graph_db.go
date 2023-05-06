package inventory

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/kaytu-io/kaytu-aws-describer/aws"
	"github.com/kaytu-io/kaytu-azure-describer/azure"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"gitlab.com/keibiengine/keibi-engine/pkg/source"
	"gitlab.com/keibiengine/keibi-engine/pkg/utils"
)

type CategoryRootType string

const (
	DefaultTemplateRootName = "default"

	RootTypeTemplateRoot  CategoryRootType = "TemplateRoot"
	RootTypeConnectorRoot CategoryRootType = "ConnectorRoot"
)

type GraphDatabase struct {
	Driver neo4j.DriverWithContext
}

func NewGraphDatabase(driver neo4j.DriverWithContext) (GraphDatabase, error) {
	ctx := context.Background()
	session := driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close(ctx)

	// Create constraints (unique constraint also automatically creates an index)
	_, err := session.Run(ctx, "CREATE CONSTRAINT template_root_unique_name_constraint IF NOT EXISTS FOR (c:TemplateRoot) REQUIRE c.name IS UNIQUE", nil)
	if err != nil {
		return GraphDatabase{}, err
	}
	_, err = session.Run(ctx, "CREATE CONSTRAINT connector_root_unique_name_constraint IF NOT EXISTS FOR (c:ConnectorRoot) REQUIRE c.name IS UNIQUE", nil)
	if err != nil {
		return GraphDatabase{}, err
	}
	_, err = session.Run(ctx, "CREATE CONSTRAINT cloud_service_category_unique_service_id IF NOT EXISTS FOR (c:CloudServiceCategory) REQUIRE c.service_id IS UNIQUE", nil)
	if err != nil {
		return GraphDatabase{}, err
	}
	_, err = session.Run(ctx, "CREATE CONSTRAINT cloud_resource_type_unique_resource_type IF NOT EXISTS FOR (c:FilterCloudResourceType) REQUIRE c.resource_type IS UNIQUE", nil)
	if err != nil {
		return GraphDatabase{}, err
	}

	// Create resource type nodes
	awsResourceTypes := aws.GetResourceTypesMap()
	for _, resourceType := range awsResourceTypes {
		_, err = session.Run(ctx, "MERGE (resource:Filter:FilterCloudResourceType{resource_label: $resource_label, connector: $connector, resource_type: $resourceType, service_name: $serviceName});", map[string]any{
			"connector":      resourceType.Connector,
			"serviceName":    strings.ToLower(resourceType.ServiceName),
			"resource_label": resourceType.ResourceLabel,
			"resourceType":   resourceType.ResourceName,
		})
		if err != nil {
			return GraphDatabase{}, err
		}

		_, err = session.Run(ctx, "MATCH (service:Category:CloudServiceCategory{connector:$connector, service_name:$serviceName}) MATCH (resource:Filter:FilterCloudResourceType{resource_label: $resource_label, connector: $connector, resource_type: $resourceType, service_name: $serviceName}) MERGE (service)-[:USES]->(resource);", map[string]any{
			"connector":      resourceType.Connector,
			"serviceName":    strings.ToLower(resourceType.ServiceName),
			"resource_label": resourceType.ResourceLabel,
			"resourceType":   resourceType.ResourceName,
		})
		if err != nil {
			return GraphDatabase{}, err
		}
	}

	azureResourceTypes := azure.GetResourceTypesMap()
	for _, resourceType := range azureResourceTypes {
		_, err = session.Run(ctx, "MERGE (resource:Filter:FilterCloudResourceType{resource_label: $resource_label, connector: $connector, resource_type: $resourceType, service_name: $serviceName});", map[string]any{
			"connector":      resourceType.Connector,
			"serviceName":    strings.ToLower(resourceType.ServiceName),
			"resource_label": resourceType.ResourceLabel,
			"resourceType":   resourceType.ResourceName,
		})
		if err != nil {
			return GraphDatabase{}, err
		}

		_, err = session.Run(ctx, "MATCH (service:Category:CloudServiceCategory{connector:$connector, service_name:$serviceName}) MATCH (resource:Filter:FilterCloudResourceType{resource_label: $resource_label, connector: $connector, resource_type: $resourceType, service_name: $serviceName}) MERGE (service)-[:USES]->(resource);", map[string]any{
			"connector":      resourceType.Connector,
			"serviceName":    strings.ToLower(resourceType.ServiceName),
			"resource_label": resourceType.ResourceLabel,
			"resourceType":   resourceType.ResourceName,
		})
		if err != nil {
			return GraphDatabase{}, err
		}
	}

	return GraphDatabase{
		Driver: driver,
	}, nil
}

type Node struct {
	Node      neo4j.Node
	ElementID string
}

type CategoryNode struct {
	Node
	Name           string         `json:"name"`
	LogoURI        *string        `json:"logo_uri,omitempty"`
	Subcategories  []CategoryNode `json:"subcategories,omitempty"`
	Filters        []Filter       `json:"filters,omitempty"` // Filters that are directly associated with this category
	SubTreeFilters []Filter       `json:"-"`                 // SubTreeFilters List of all filters that are in the subtree of this category
}

type ServiceNode struct {
	CategoryNode
	ServiceName string      `json:"service_name"`
	Connector   source.Type `json:"connector"`
	ServiceID   string      `json:"service_id"`
	LogoURI     *string     `json:"logo_uri,omitempty"`
}

// GetParentService Returns the parent service name of this service node if it exists
func (s ServiceNode) GetParentService() *string {
	parent, ok := s.Node.Node.Props["parent_service"]
	if !ok {
		return nil
	}
	parentService, ok := parent.(string)
	if !ok {
		return nil
	}
	return &parentService
}

type Filter interface {
	GetFilterType() FilterType
}

type FilterType string

const (
	FilterTypeCloudResourceType FilterType = "FilterCloudResourceType"
	FilterTypeCost              FilterType = "FilterCost"
	FilterTypeInsightMetric     FilterType = "FilterInsightMetric"
)

type FilterCloudResourceTypeNode struct {
	Node
	Connector     source.Type `json:"connector"`
	ResourceType  string      `json:"resource_type"`
	ResourceLabel string      `json:"resource_name"`
	ServiceName   string      `json:"service_name"`
	LogoURI       *string     `json:"logo_uri,omitempty"`
}

func (f FilterCloudResourceTypeNode) GetFilterType() FilterType {
	return FilterTypeCloudResourceType
}

type FilterCostNode struct {
	Node
	Connector       source.Type `json:"connector"`
	CostServiceName string      `json:"cost_service_name"`
	ServiceLabel    string      `json:"service_label"`
}

func (f FilterCostNode) GetFilterType() FilterType {
	return FilterTypeCost
}

type FilterInsightMetricNode struct {
	Node
	Connector source.Type `json:"connector"`
	MetricID  string      `json:"metric_id"`
	InsightID int64       `json:"insight_id"`
	Name      string      `json:"name"`
}

func (f FilterInsightMetricNode) GetFilterType() FilterType {
	return FilterTypeInsightMetric
}

var (
	ErrKeyColumnNotFound = errors.New("key column not found")
	ErrPropertyNotFound  = errors.New("property not found")
	ErrInvalidFilter     = errors.New("invalid filter")
	ErrColumnConversion  = errors.New("could not convert column to appropriate type")
	ErrNotFound          = errors.New("not found")
)

const (
	subTreeFiltersQuery = `
MATCH (c:Category%s) WHERE %s CALL {
  WITH c MATCH (c)-[:INCLUDES*]->(:Category)-[:USES]->(f:Filter)
  RETURN DISTINCT f, false as isDirectChild
  UNION 
  WITH c MATCH (c)-[:USES]->(f:Filter)
  RETURN DISTINCT f, true as isDirectChild }
RETURN DISTINCT c, f, MAX(isDirectChild) AS isDirectChild
`
	subTreePrimaryFiltersQuery = `
MATCH (c:Category%s) WHERE %s CALL {
  WITH c MATCH (c)-[rel:INCLUDES*]->(child:Category)-[:USES]->(f:Filter)
  UNWIND rel as relation
  	WITH c,child,f,relation MATCH () WHERE (NOT 'CloudServiceCategory' IN LABELS(child) OR (NOT relation.isPrimary IS NULL AND relation.isPrimary = true))
  	RETURN DISTINCT f, false as isDirectChild
  UNION 
  WITH c MATCH (c)-[:USES]->(f:Filter)
  RETURN DISTINCT f, true as isDirectChild }
RETURN DISTINCT c, f, MAX(isDirectChild) AS isDirectChild
`
)

func getFilterFromNode(node neo4j.Node) (Filter, error) {
	for _, label := range node.Labels {
		switch label {
		case string(FilterTypeCloudResourceType):
			connector, ok := node.Props["connector"]
			if !ok {
				return nil, ErrPropertyNotFound
			}
			resourceType, ok := node.Props["resource_type"]
			if !ok {
				return nil, ErrPropertyNotFound
			}
			resourceLabel, ok := node.Props["resource_label"]
			if !ok {
				return nil, ErrPropertyNotFound
			}
			serviceName, ok := node.Props["service_name"]
			if !ok {
				return nil, ErrPropertyNotFound
			}

			logoURI, ok := node.Props["logo_uri"]
			if !ok {
				logoURI = ""
			}

			return &FilterCloudResourceTypeNode{
				Node: Node{
					Node:      node,
					ElementID: node.ElementId,
				},
				Connector:     source.Type(connector.(string)),
				ResourceType:  resourceType.(string),
				ResourceLabel: resourceLabel.(string),
				ServiceName:   strings.ToLower(serviceName.(string)),
				LogoURI:       utils.GetPointerOrNil(logoURI.(string)),
			}, nil
		case string(FilterTypeCost):
			connector, ok := node.Props["connector"]
			if !ok {
				return nil, ErrPropertyNotFound
			}
			costServiceName, ok := node.Props["cost_service_name"]
			if !ok {
				return nil, ErrPropertyNotFound
			}
			serviceLabel, ok := node.Props["service_label"]
			if !ok {
				return nil, ErrPropertyNotFound
			}
			return &FilterCostNode{
				Node: Node{
					Node:      node,
					ElementID: node.ElementId,
				},
				Connector:       source.Type(connector.(string)),
				CostServiceName: costServiceName.(string),
				ServiceLabel:    serviceLabel.(string),
			}, nil
		case string(FilterTypeInsightMetric):
			connector, ok := node.Props["connector"]
			if !ok {
				return nil, ErrPropertyNotFound
			}
			metricID, ok := node.Props["metric_id"]
			if !ok {
				return nil, ErrPropertyNotFound
			}
			insightID, ok := node.Props["insight_id"]
			if !ok {
				return nil, ErrPropertyNotFound
			}
			name, ok := node.Props["name"]
			if !ok {
				return nil, ErrPropertyNotFound
			}
			return &FilterInsightMetricNode{
				Node: Node{
					Node:      node,
					ElementID: node.ElementId,
				},
				Connector: source.Type(connector.(string)),
				MetricID:  metricID.(string),
				InsightID: insightID.(int64),
				Name:      name.(string),
			}, nil
		}
	}

	return nil, ErrInvalidFilter
}

func getCategoryFromNode(node neo4j.Node) (*CategoryNode, error) {
	name, ok := node.Props["name"]
	if !ok {
		return nil, ErrPropertyNotFound
	}

	logoURI, ok := node.Props["logo_uri"]
	if !ok {
		logoURI = ""
	}

	return &CategoryNode{
		Node: Node{
			Node:      node,
			ElementID: node.ElementId,
		},
		Name:           name.(string),
		LogoURI:        utils.GetPointerOrNil(logoURI.(string)),
		Filters:        []Filter{},
		SubTreeFilters: []Filter{},
		Subcategories:  []CategoryNode{},
	}, nil
}

func getCloudServiceFromNode(node neo4j.Node) (*ServiceNode, error) {
	cat, err := getCategoryFromNode(node)
	if err != nil {
		return nil, err
	}

	serviceName, ok := node.Props["service_name"]
	if !ok {
		return nil, ErrPropertyNotFound
	}
	connector, ok := node.Props["connector"]
	if !ok {
		return nil, ErrPropertyNotFound
	}
	serviceId, ok := node.Props["service_id"]
	if !ok {
		return nil, ErrPropertyNotFound
	}

	// optional property
	logoURI, ok := node.Props["logo_uri"]
	if !ok {
		logoURI = ""
	}

	return &ServiceNode{
		CategoryNode: *cat,
		ServiceName:  serviceName.(string),
		ServiceID:    serviceId.(string),
		Connector:    source.Type(connector.(string)),
		LogoURI:      utils.GetPointerOrNil(logoURI.(string)),
	}, nil
}

func (gdb *GraphDatabase) GetCategoryRoots(ctx context.Context, rootType CategoryRootType) (map[string]*CategoryNode, error) {
	session := gdb.Driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close(ctx)

	var categories = make(map[string]*CategoryNode)

	// Get all categories that have no parent
	result, err := session.Run(ctx, fmt.Sprintf("MATCH (c:Category:%s) RETURN c", rootType), nil)
	if err != nil {
		return nil, err
	}
	for result.Next(ctx) {
		rawCategory, ok := result.Record().Get("c")
		if !ok {
			return nil, ErrKeyColumnNotFound
		}
		categoryNode, ok := rawCategory.(neo4j.Node)
		if !ok {
			return nil, ErrColumnConversion
		}

		category, err := getCategoryFromNode(categoryNode)
		if err != nil {
			return nil, err
		}
		categories[category.ElementID] = category
	}

	// Get all the filters that are in the subtree of each category with no parent
	result, err = session.Run(ctx, fmt.Sprintf(subTreeFiltersQuery, fmt.Sprintf(":%s", rootType), "true"), map[string]any{})
	if err != nil {
		return nil, err
	}
	for result.Next(ctx) {
		rawCategory, ok := result.Record().Get("c")
		if !ok {
			return nil, ErrKeyColumnNotFound
		}
		rawFilter, ok := result.Record().Get("f")
		if !ok {
			return nil, ErrKeyColumnNotFound
		}
		isChildRaw, ok := result.Record().Get("isDirectChild")
		if !ok {
			return nil, ErrKeyColumnNotFound
		}
		categoryNode, ok := rawCategory.(neo4j.Node)
		if !ok {
			return nil, ErrColumnConversion
		}
		filterNode, ok := rawFilter.(neo4j.Node)
		if !ok {
			return nil, ErrColumnConversion
		}
		isChild, ok := isChildRaw.(bool)
		if !ok {
			return nil, ErrColumnConversion
		}

		category, ok := categories[categoryNode.ElementId]
		if !ok {
			category, err = getCategoryFromNode(categoryNode)
			if err != nil {
				return nil, err
			}
			categories[categoryNode.ElementId] = category
		}

		filter, err := getFilterFromNode(filterNode)
		if err != nil {
			return nil, err
		}
		category.SubTreeFilters = append(category.SubTreeFilters, filter)
		if isChild {
			category.Filters = append(category.Filters, filter)
		}
	}

	// Get all the subcategories of each category with no parent
	result, err = session.Run(ctx, fmt.Sprintf("MATCH (c:Category:%s)-[:INCLUDES]->(sub:Category) RETURN DISTINCT c, sub", rootType), nil)
	if err != nil {
		return nil, err
	}
	for result.Next(ctx) {
		rawCategory, ok := result.Record().Get("c")
		if !ok {
			return nil, ErrKeyColumnNotFound
		}
		rawSubcategory, ok := result.Record().Get("sub")
		if !ok {
			return nil, ErrKeyColumnNotFound
		}
		categoryNode, ok := rawCategory.(neo4j.Node)
		if !ok {
			return nil, ErrColumnConversion
		}
		subcategoryNode, ok := rawSubcategory.(neo4j.Node)
		if !ok {
			return nil, ErrColumnConversion
		}

		category, ok := categories[categoryNode.ElementId]
		if !ok {
			category, err = getCategoryFromNode(categoryNode)
			if err != nil {
				return nil, err
			}
			categories[categoryNode.ElementId] = category
		}

		subcategory, err := getCategoryFromNode(subcategoryNode)
		if err != nil {
			return nil, err
		}
		category.Subcategories = append(category.Subcategories, *subcategory)
	}

	return categories, nil
}

func (gdb *GraphDatabase) GetCategoryRootByName(ctx context.Context, rootType CategoryRootType, name string) (*CategoryNode, error) {
	session := gdb.Driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close(ctx)

	var category *CategoryNode

	// Get the category
	result, err := session.Run(ctx, fmt.Sprintf("MATCH (c:Category:%s{name: $name}) RETURN c", rootType), map[string]interface{}{
		"name": name,
	})
	if err != nil {
		return nil, err
	}
	record, err := result.Single(ctx)
	if err != nil {
		return nil, err
	}
	rawCategory, ok := record.Get("c")
	if !ok {
		return nil, ErrKeyColumnNotFound
	}
	categoryNode, ok := rawCategory.(neo4j.Node)
	if !ok {
		return nil, ErrColumnConversion
	}

	category, err = getCategoryFromNode(categoryNode)
	if err != nil {
		return nil, err
	}
	if category == nil {
		return nil, ErrNotFound
	}

	// Get all the filters that are in the subtree of the category
	result, err = session.Run(ctx, fmt.Sprintf(subTreeFiltersQuery, fmt.Sprintf(":%s{name: $name}", rootType), "true"), map[string]interface{}{
		"name": name,
	})
	if err != nil {
		return nil, err
	}

	for result.Next(ctx) {
		rawFilter, ok := result.Record().Get("f")
		if !ok {
			return nil, ErrKeyColumnNotFound
		}
		filterNode, ok := rawFilter.(neo4j.Node)
		if !ok {
			return nil, ErrColumnConversion
		}
		isChildRaw, ok := result.Record().Get("isDirectChild")
		if !ok {
			return nil, ErrKeyColumnNotFound
		}
		isChild, ok := isChildRaw.(bool)
		if !ok {
			return nil, ErrColumnConversion
		}

		filter, err := getFilterFromNode(filterNode)
		if err != nil {
			return nil, err
		}
		category.SubTreeFilters = append(category.SubTreeFilters, filter)
		if isChild {
			category.Filters = append(category.Filters, filter)
		}
	}

	// Get all the subcategories of the category
	result, err = session.Run(ctx, fmt.Sprintf("MATCH (c:Category:%s{name: $name})-[:INCLUDES]->(sub:Category) RETURN DISTINCT c, sub", rootType), map[string]interface{}{
		"name": name,
	})
	if err != nil {
		return nil, err
	}
	for result.Next(ctx) {
		rawSubcategory, ok := result.Record().Get("sub")
		if !ok {
			return nil, ErrKeyColumnNotFound
		}
		subcategoryNode, ok := rawSubcategory.(neo4j.Node)
		if !ok {
			return nil, ErrColumnConversion
		}

		subcategory, err := getCategoryFromNode(subcategoryNode)
		if err != nil {
			return nil, err
		}
		category.Subcategories = append(category.Subcategories, *subcategory)
	}

	return category, nil
}

func (gdb *GraphDatabase) GetCategory(ctx context.Context, elementID string) (*CategoryNode, error) {
	session := gdb.Driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close(ctx)

	var category *CategoryNode

	// Get the category
	result, err := session.Run(ctx, "MATCH (c:Category) WHERE elementId(c) = $element_id RETURN c", map[string]interface{}{
		"element_id": elementID,
	})
	if err != nil {
		return nil, err
	}
	record, err := result.Single(ctx)
	if err != nil {
		return nil, err
	}
	rawCategory, ok := record.Get("c")
	if !ok {
		return nil, ErrKeyColumnNotFound
	}
	categoryNode, ok := rawCategory.(neo4j.Node)
	if !ok {
		return nil, ErrColumnConversion
	}

	category, err = getCategoryFromNode(categoryNode)
	if err != nil {
		return nil, err
	}
	if category == nil {
		return nil, ErrNotFound
	}

	// Get all the filters that are in the subtree of the category
	result, err = session.Run(ctx, fmt.Sprintf(subTreeFiltersQuery, "", "elementId(c) = $elementID"), map[string]interface{}{
		"elementID": elementID,
	})
	if err != nil {
		return nil, err
	}

	for result.Next(ctx) {
		rawFilter, ok := result.Record().Get("f")
		if !ok {
			return nil, ErrKeyColumnNotFound
		}
		filterNode, ok := rawFilter.(neo4j.Node)
		if !ok {
			return nil, ErrColumnConversion
		}
		isChildRaw, ok := result.Record().Get("isDirectChild")
		if !ok {
			return nil, ErrKeyColumnNotFound
		}
		isChild, ok := isChildRaw.(bool)
		if !ok {
			return nil, ErrColumnConversion
		}

		filter, err := getFilterFromNode(filterNode)
		if err != nil {
			return nil, err
		}
		category.SubTreeFilters = append(category.SubTreeFilters, filter)
		if isChild {
			category.Filters = append(category.Filters, filter)
		}
	}

	// Get all the subcategories of the category
	result, err = session.Run(ctx, "MATCH (c:Category)-[:INCLUDES]->(sub:Category) WHERE elementId(c) = $element_id RETURN DISTINCT c, sub", map[string]interface{}{
		"element_id": elementID,
	})
	if err != nil {
		return nil, err
	}
	for result.Next(ctx) {
		rawSubcategory, ok := result.Record().Get("sub")
		if !ok {
			return nil, ErrKeyColumnNotFound
		}
		subcategoryNode, ok := rawSubcategory.(neo4j.Node)
		if !ok {
			return nil, ErrColumnConversion
		}

		subcategory, err := getCategoryFromNode(subcategoryNode)
		if err != nil {
			return nil, err
		}
		category.Subcategories = append(category.Subcategories, *subcategory)
	}

	return category, nil
}

func (gdb *GraphDatabase) GetPrimaryCategoryRootByName(ctx context.Context, rootType CategoryRootType, name string) (*CategoryNode, error) {
	session := gdb.Driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close(ctx)

	var category *CategoryNode

	// Get the category
	result, err := session.Run(ctx, fmt.Sprintf("MATCH (c:Category:%s{name: $name}) RETURN c", rootType), map[string]interface{}{
		"name": name,
	})
	if err != nil {
		return nil, err
	}
	record, err := result.Single(ctx)
	if err != nil {
		return nil, err
	}
	rawCategory, ok := record.Get("c")
	if !ok {
		return nil, ErrKeyColumnNotFound
	}
	categoryNode, ok := rawCategory.(neo4j.Node)
	if !ok {
		return nil, ErrColumnConversion
	}

	category, err = getCategoryFromNode(categoryNode)
	if err != nil {
		return nil, err
	}
	if category == nil {
		return nil, ErrNotFound
	}

	// Get all the filters that are in the subtree of the category
	result, err = session.Run(ctx, fmt.Sprintf(subTreePrimaryFiltersQuery, fmt.Sprintf(":%s{name: $name}", rootType), "true"), map[string]interface{}{
		"name": name,
	})
	if err != nil {
		return nil, err
	}

	for result.Next(ctx) {
		rawFilter, ok := result.Record().Get("f")
		if !ok {
			return nil, ErrKeyColumnNotFound
		}
		filterNode, ok := rawFilter.(neo4j.Node)
		if !ok {
			return nil, ErrColumnConversion
		}
		isChildRaw, ok := result.Record().Get("isDirectChild")
		if !ok {
			return nil, ErrKeyColumnNotFound
		}
		isChild, ok := isChildRaw.(bool)
		if !ok {
			return nil, ErrColumnConversion
		}

		filter, err := getFilterFromNode(filterNode)
		if err != nil {
			return nil, err
		}
		category.SubTreeFilters = append(category.SubTreeFilters, filter)
		if isChild {
			category.Filters = append(category.Filters, filter)
		}
	}

	// Get all the subcategories of the category
	result, err = session.Run(ctx, fmt.Sprintf("MATCH (c:Category:%s{name: $name})-[rel:INCLUDES]->(sub:Category) WHERE ((NOT 'CloudServiceCategory' IN LABELS(sub)) OR (rel.isPrimary is NULL) OR (rel.isPrimary = true)) RETURN DISTINCT c, sub", rootType), map[string]interface{}{
		"name": name,
	})
	if err != nil {
		return nil, err
	}
	for result.Next(ctx) {
		rawSubcategory, ok := result.Record().Get("sub")
		if !ok {
			return nil, ErrKeyColumnNotFound
		}
		subcategoryNode, ok := rawSubcategory.(neo4j.Node)
		if !ok {
			return nil, ErrColumnConversion
		}

		subcategory, err := getCategoryFromNode(subcategoryNode)
		if err != nil {
			return nil, err
		}
		category.Subcategories = append(category.Subcategories, *subcategory)
	}

	return category, nil
}

func (gdb *GraphDatabase) GetPrimaryCategory(ctx context.Context, elementID string) (*CategoryNode, error) {
	session := gdb.Driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close(ctx)

	var category *CategoryNode

	// Get the category
	result, err := session.Run(ctx, "MATCH (c:Category) WHERE elementId(c) = $element_id RETURN c", map[string]interface{}{
		"element_id": elementID,
	})
	if err != nil {
		return nil, err
	}
	record, err := result.Single(ctx)
	if err != nil {
		return nil, err
	}
	rawCategory, ok := record.Get("c")
	if !ok {
		return nil, ErrKeyColumnNotFound
	}
	categoryNode, ok := rawCategory.(neo4j.Node)
	if !ok {
		return nil, ErrColumnConversion
	}

	category, err = getCategoryFromNode(categoryNode)
	if err != nil {
		return nil, err
	}
	if category == nil {
		return nil, ErrNotFound
	}

	// Get all the filters that are in the subtree of the category
	result, err = session.Run(ctx, fmt.Sprintf(subTreePrimaryFiltersQuery, "", "elementId(c) = $elementID"), map[string]interface{}{
		"elementID": elementID,
	})
	if err != nil {
		return nil, err
	}

	for result.Next(ctx) {
		rawFilter, ok := result.Record().Get("f")
		if !ok {
			return nil, ErrKeyColumnNotFound
		}
		filterNode, ok := rawFilter.(neo4j.Node)
		if !ok {
			return nil, ErrColumnConversion
		}
		isChildRaw, ok := result.Record().Get("isDirectChild")
		if !ok {
			return nil, ErrKeyColumnNotFound
		}
		isChild, ok := isChildRaw.(bool)
		if !ok {
			return nil, ErrColumnConversion
		}

		filter, err := getFilterFromNode(filterNode)
		if err != nil {
			return nil, err
		}
		category.SubTreeFilters = append(category.SubTreeFilters, filter)
		if isChild {
			category.Filters = append(category.Filters, filter)
		}
	}

	// Get all the subcategories of the category
	result, err = session.Run(ctx, "MATCH (c:Category)-[rel:INCLUDES]->(sub:Category) WHERE (elementId(c) = $element_id AND (NOT 'CloudServiceCategory' IN LABELS(sub) OR rel.isPrimary is NULL OR rel.isPrimary = true)) RETURN DISTINCT c, sub", map[string]interface{}{
		"element_id": elementID,
	})
	if err != nil {
		return nil, err
	}
	for result.Next(ctx) {
		rawSubcategory, ok := result.Record().Get("sub")
		if !ok {
			return nil, ErrKeyColumnNotFound
		}
		subcategoryNode, ok := rawSubcategory.(neo4j.Node)
		if !ok {
			return nil, ErrColumnConversion
		}

		subcategory, err := getCategoryFromNode(subcategoryNode)
		if err != nil {
			return nil, err
		}
		category.Subcategories = append(category.Subcategories, *subcategory)
	}

	return category, nil
}

func (gdb *GraphDatabase) GetSubcategories(ctx context.Context, elementID string) (*CategoryNode, error) {
	session := gdb.Driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close(ctx)

	var category *CategoryNode

	// Get all the subcategories of the category
	result, err := session.Run(ctx, "MATCH (c:Category)-[:INCLUDES]->(sub:Category) WHERE elementId(c) = $element_id RETURN DISTINCT c,sub", map[string]interface{}{
		"element_id": elementID,
	})
	if err != nil {
		return nil, err
	}
	for result.Next(ctx) {
		if category == nil {
			rawCategory, ok := result.Record().Get("c")
			if !ok {
				return nil, ErrKeyColumnNotFound
			}
			categoryNode, ok := rawCategory.(neo4j.Node)
			if !ok {
				return nil, ErrColumnConversion
			}

			category, err = getCategoryFromNode(categoryNode)
			if err != nil {
				return nil, err
			}
		}
		rawSubcategory, ok := result.Record().Get("sub")
		if !ok {
			return nil, ErrKeyColumnNotFound
		}
		subcategoryNode, ok := rawSubcategory.(neo4j.Node)
		if !ok {
			return nil, ErrColumnConversion
		}

		subcategory, err := getCategoryFromNode(subcategoryNode)
		if err != nil {
			return nil, err
		}
		category.Subcategories = append(category.Subcategories, *subcategory)
	}

	return category, nil
}

func (gdb *GraphDatabase) GetCategoryRootSubcategoriesByName(ctx context.Context, rootType CategoryRootType, name string) (*CategoryNode, error) {
	session := gdb.Driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close(ctx)

	var category *CategoryNode

	result, err := session.Run(ctx, fmt.Sprintf("MATCH (c:Category:%s{name: $name})-[:INCLUDES]->(sub:Category) RETURN c, sub", rootType), map[string]interface{}{
		"name": name,
	})
	if err != nil {
		return nil, err
	}

	for result.Next(ctx) {
		if category == nil {
			rawCategory, ok := result.Record().Get("c")
			if !ok {
				return nil, ErrKeyColumnNotFound
			}
			categoryNode, ok := rawCategory.(neo4j.Node)
			if !ok {
				return nil, ErrColumnConversion
			}

			category, err = getCategoryFromNode(categoryNode)
			if err != nil {
				return nil, err
			}
		}

		rawSubcategory, ok := result.Record().Get("sub")
		if !ok {
			return nil, ErrKeyColumnNotFound
		}
		subcategoryNode, ok := rawSubcategory.(neo4j.Node)
		if !ok {
			return nil, ErrColumnConversion
		}

		subcategory, err := getCategoryFromNode(subcategoryNode)
		if err != nil {
			return nil, err
		}
		category.Subcategories = append(category.Subcategories, *subcategory)
	}

	return category, nil
}

func (gdb *GraphDatabase) GetNormalCategoryNodes(ctx context.Context, connector source.Type) ([]CategoryNode, error) {
	session := gdb.Driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close(ctx)

	var nodes = make(map[string]*CategoryNode)

	// Get all services of the selected connector
	result, err := session.Run(ctx, "MATCH (c:Category:NormalCategory) WHERE $connector = '' OR c.connector = $connector RETURN DISTINCT c", map[string]interface{}{
		"connector": connector.String(),
	})
	if err != nil {
		return nil, err
	}
	for result.Next(ctx) {
		rawCategory, ok := result.Record().Get("c")
		if !ok {
			return nil, ErrKeyColumnNotFound
		}
		categoryNode, ok := rawCategory.(neo4j.Node)
		if !ok {
			return nil, ErrColumnConversion
		}

		node, err := getCategoryFromNode(categoryNode)
		if err != nil {
			return nil, err
		}
		nodes[node.ElementID] = node
	}

	result, err = session.Run(ctx, fmt.Sprintf(subTreeFiltersQuery, fmt.Sprintf(":NormalCategory"), "$connector = '' OR c.connector = $connector"), map[string]any{
		"connector": connector.String(),
	})
	if err != nil {
		return nil, err
	}
	for result.Next(ctx) {
		rawCategory, ok := result.Record().Get("c")
		if !ok {
			return nil, ErrKeyColumnNotFound
		}
		rawFilter, ok := result.Record().Get("f")
		if !ok {
			return nil, ErrKeyColumnNotFound
		}
		isChildRaw, ok := result.Record().Get("isDirectChild")
		if !ok {
			return nil, ErrKeyColumnNotFound
		}
		categoryNode, ok := rawCategory.(neo4j.Node)
		if !ok {
			return nil, ErrColumnConversion
		}
		filterNode, ok := rawFilter.(neo4j.Node)
		if !ok {
			return nil, ErrColumnConversion
		}
		isChild, ok := isChildRaw.(bool)
		if !ok {
			return nil, ErrColumnConversion
		}

		node, ok := nodes[categoryNode.ElementId]
		if !ok {
			node, err = getCategoryFromNode(categoryNode)
			if err != nil {
				return nil, err
			}
			nodes[categoryNode.ElementId] = node
		}

		filter, err := getFilterFromNode(filterNode)
		if err != nil {
			return nil, err
		}
		node.SubTreeFilters = append(node.SubTreeFilters, filter)
		if isChild {
			node.Filters = append(node.Filters, filter)
		}
	}

	result, err = session.Run(ctx, fmt.Sprintf("MATCH (c:Category:getCategoryFromNode)-[:INCLUDES]->(sub:Category) WHERE $connector = '' OR c.connector = $connector RETURN DISTINCT c, sub"), map[string]any{
		"connector": connector.String(),
	})
	if err != nil {
		return nil, err
	}
	for result.Next(ctx) {
		rawCategory, ok := result.Record().Get("c")
		if !ok {
			return nil, ErrKeyColumnNotFound
		}
		rawSubcategory, ok := result.Record().Get("sub")
		if !ok {
			return nil, ErrKeyColumnNotFound
		}
		categoryNode, ok := rawCategory.(neo4j.Node)
		if !ok {
			return nil, ErrColumnConversion
		}
		subcategoryNode, ok := rawSubcategory.(neo4j.Node)
		if !ok {
			return nil, ErrColumnConversion
		}

		node, ok := nodes[categoryNode.ElementId]
		if !ok {
			node, err = getCategoryFromNode(categoryNode)
			if err != nil {
				return nil, err
			}
			nodes[categoryNode.ElementId] = node
		}

		subcategory, err := getCategoryFromNode(subcategoryNode)
		if err != nil {
			return nil, err
		}
		node.Subcategories = append(node.Subcategories, *subcategory)
	}

	nodesArr := make([]CategoryNode, 0, len(nodes))
	for _, node := range nodes {
		nodesArr = append(nodesArr, *node)
	}

	return nodesArr, nil
}

func (gdb *GraphDatabase) GetCloudServiceNodes(ctx context.Context, connector source.Type) ([]ServiceNode, error) {
	session := gdb.Driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close(ctx)

	var services = make(map[string]*ServiceNode)

	// Get all services of the selected connector
	result, err := session.Run(ctx, "MATCH (c:Category:CloudServiceCategory) WHERE $connector = '' OR c.connector = $connector RETURN DISTINCT c", map[string]interface{}{
		"connector": connector.String(),
	})
	if err != nil {
		return nil, err
	}
	for result.Next(ctx) {
		rawCategory, ok := result.Record().Get("c")
		if !ok {
			return nil, ErrKeyColumnNotFound
		}
		categoryNode, ok := rawCategory.(neo4j.Node)
		if !ok {
			return nil, ErrColumnConversion
		}

		service, err := getCloudServiceFromNode(categoryNode)
		if err != nil {
			return nil, err
		}
		services[service.ElementID] = service
	}

	result, err = session.Run(ctx, fmt.Sprintf(subTreeFiltersQuery, fmt.Sprintf(":CloudServiceCategory"), "$connector = '' OR c.connector = $connector"), map[string]any{
		"connector": connector.String(),
	})
	if err != nil {
		return nil, err
	}
	for result.Next(ctx) {
		rawCategory, ok := result.Record().Get("c")
		if !ok {
			return nil, ErrKeyColumnNotFound
		}
		rawFilter, ok := result.Record().Get("f")
		if !ok {
			return nil, ErrKeyColumnNotFound
		}
		isChildRaw, ok := result.Record().Get("isDirectChild")
		if !ok {
			return nil, ErrKeyColumnNotFound
		}
		categoryNode, ok := rawCategory.(neo4j.Node)
		if !ok {
			return nil, ErrColumnConversion
		}
		filterNode, ok := rawFilter.(neo4j.Node)
		if !ok {
			return nil, ErrColumnConversion
		}
		isChild, ok := isChildRaw.(bool)
		if !ok {
			return nil, ErrColumnConversion
		}

		service, ok := services[categoryNode.ElementId]
		if !ok {
			service, err = getCloudServiceFromNode(categoryNode)
			if err != nil {
				return nil, err
			}
			services[categoryNode.ElementId] = service
		}

		filter, err := getFilterFromNode(filterNode)
		if err != nil {
			return nil, err
		}
		service.SubTreeFilters = append(service.SubTreeFilters, filter)
		if isChild {
			service.Filters = append(service.Filters, filter)
		}
	}

	result, err = session.Run(ctx, fmt.Sprintf("MATCH (c:Category:CloudServiceCategory)-[:INCLUDES]->(sub:Category) WHERE $connector = '' OR c.connector = $connector RETURN DISTINCT c, sub"), map[string]any{
		"connector": connector.String(),
	})
	if err != nil {
		return nil, err
	}
	for result.Next(ctx) {
		rawCategory, ok := result.Record().Get("c")
		if !ok {
			return nil, ErrKeyColumnNotFound
		}
		rawSubcategory, ok := result.Record().Get("sub")
		if !ok {
			return nil, ErrKeyColumnNotFound
		}
		categoryNode, ok := rawCategory.(neo4j.Node)
		if !ok {
			return nil, ErrColumnConversion
		}
		subcategoryNode, ok := rawSubcategory.(neo4j.Node)
		if !ok {
			return nil, ErrColumnConversion
		}

		service, ok := services[categoryNode.ElementId]
		if !ok {
			service, err = getCloudServiceFromNode(categoryNode)
			if err != nil {
				return nil, err
			}
			services[categoryNode.ElementId] = service
		}

		subcategory, err := getCategoryFromNode(subcategoryNode)
		if err != nil {
			return nil, err
		}
		service.Subcategories = append(service.Subcategories, *subcategory)
	}

	servicesArr := make([]ServiceNode, 0, len(services))
	for _, service := range services {
		servicesArr = append(servicesArr, *service)
	}

	return servicesArr, nil
}

func (gdb *GraphDatabase) GetCloudServiceNodesByCategory(ctx context.Context, connector source.Type, categoryID string) ([]ServiceNode, error) {
	session := gdb.Driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close(ctx)

	var services = make(map[string]*ServiceNode)

	// Get all services of the selected connector and category
	result, err := session.Run(ctx, "MATCH (par)-[:INCLUDES*]->(c:Category:CloudServiceCategory) WHERE $connector = '' OR c.connector = $connector AND elementId(par) = $category_id RETURN DISTINCT c", map[string]interface{}{
		"connector":   connector.String(),
		"category_id": categoryID,
	})
	if err != nil {
		return nil, err
	}
	for result.Next(ctx) {
		rawCategory, ok := result.Record().Get("c")
		if !ok {
			return nil, ErrKeyColumnNotFound
		}
		categoryNode, ok := rawCategory.(neo4j.Node)
		if !ok {
			return nil, ErrColumnConversion
		}

		service, err := getCloudServiceFromNode(categoryNode)
		if err != nil {
			return nil, err
		}
		services[service.ElementID] = service
	}

	result, err = session.Run(ctx, fmt.Sprintf(subTreeFiltersQuery, fmt.Sprintf(":CloudServiceCategory"), "$connector = '' OR c.connector = $connector"), map[string]any{
		"connector": connector.String(),
	})
	if err != nil {
		return nil, err
	}
	for result.Next(ctx) {
		rawCategory, ok := result.Record().Get("c")
		if !ok {
			return nil, ErrKeyColumnNotFound
		}
		rawFilter, ok := result.Record().Get("f")
		if !ok {
			return nil, ErrKeyColumnNotFound
		}
		isChildRaw, ok := result.Record().Get("isDirectChild")
		if !ok {
			return nil, ErrKeyColumnNotFound
		}
		categoryNode, ok := rawCategory.(neo4j.Node)
		if !ok {
			return nil, ErrColumnConversion
		}
		filterNode, ok := rawFilter.(neo4j.Node)
		if !ok {
			return nil, ErrColumnConversion
		}
		isChild, ok := isChildRaw.(bool)
		if !ok {
			return nil, ErrColumnConversion
		}

		service, ok := services[categoryNode.ElementId]
		if !ok {
			continue
		}

		filter, err := getFilterFromNode(filterNode)
		if err != nil {
			return nil, err
		}
		service.SubTreeFilters = append(service.SubTreeFilters, filter)
		if isChild {
			service.Filters = append(service.Filters, filter)
		}
	}

	result, err = session.Run(ctx, fmt.Sprintf("MATCH (c:Category:CloudServiceCategory)-[:INCLUDES]->(sub:Category) WHERE $connector = '' OR c.connector = $connector RETURN DISTINCT c, sub"), map[string]any{
		"connector": connector.String(),
	})
	if err != nil {
		return nil, err
	}
	for result.Next(ctx) {
		rawCategory, ok := result.Record().Get("c")
		if !ok {
			return nil, ErrKeyColumnNotFound
		}
		rawSubcategory, ok := result.Record().Get("sub")
		if !ok {
			return nil, ErrKeyColumnNotFound
		}
		categoryNode, ok := rawCategory.(neo4j.Node)
		if !ok {
			return nil, ErrColumnConversion
		}
		subcategoryNode, ok := rawSubcategory.(neo4j.Node)
		if !ok {
			return nil, ErrColumnConversion
		}

		service, ok := services[categoryNode.ElementId]
		if !ok {
			continue
		}

		subcategory, err := getCategoryFromNode(subcategoryNode)
		if err != nil {
			return nil, err
		}
		service.Subcategories = append(service.Subcategories, *subcategory)
	}

	servicesArr := make([]ServiceNode, 0, len(services))
	for _, service := range services {
		servicesArr = append(servicesArr, *service)
	}

	return servicesArr, nil
}

func (gdb *GraphDatabase) GetCloudServiceNode(ctx context.Context, connector source.Type, serviceName string) (*ServiceNode, error) {
	session := gdb.Driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close(ctx)

	// Get all services of the selected connector
	result, err := session.Run(ctx, "MATCH (c:Category:CloudServiceCategory) WHERE ($connector = '' OR c.connector = $connector) AND (c.service_name = $service_name) RETURN DISTINCT c", map[string]interface{}{
		"connector":    connector.String(),
		"service_name": serviceName,
	})
	if err != nil {
		return nil, err
	}

	singleNode, err := result.Single(ctx)
	if err != nil {
		return nil, err
	}

	rawCategory, ok := singleNode.Get("c")
	if !ok {
		return nil, ErrKeyColumnNotFound
	}
	categoryNode, ok := rawCategory.(neo4j.Node)
	if !ok {
		return nil, ErrColumnConversion
	}

	service, err := getCloudServiceFromNode(categoryNode)
	if err != nil {
		return nil, err
	}

	result, err = session.Run(ctx, fmt.Sprintf(subTreeFiltersQuery, fmt.Sprintf(":CloudServiceCategory"), "($connector = '' OR c.connector = $connector) AND (c.service_name = $service_name)"), map[string]any{
		"connector":    connector.String(),
		"service_name": serviceName,
	})
	if err != nil {
		return nil, err
	}

	for result.Next(ctx) {
		rawFilter, ok := result.Record().Get("f")
		if !ok {
			return nil, ErrKeyColumnNotFound
		}
		isChildRaw, ok := result.Record().Get("isDirectChild")
		if !ok {
			return nil, ErrKeyColumnNotFound
		}
		filterNode, ok := rawFilter.(neo4j.Node)
		if !ok {
			return nil, ErrColumnConversion
		}
		isChild, ok := isChildRaw.(bool)
		if !ok {
			return nil, ErrColumnConversion
		}

		filter, err := getFilterFromNode(filterNode)
		if err != nil {
			return nil, err
		}
		service.SubTreeFilters = append(service.SubTreeFilters, filter)
		if isChild {
			service.Filters = append(service.Filters, filter)
		}
	}

	result, err = session.Run(ctx, fmt.Sprintf("MATCH (c:Category:CloudServiceCategory)-[:INCLUDES]->(sub:Category) WHERE ($connector = '' OR c.connector = $connector) AND (c.service_name = $service_name) RETURN DISTINCT c, sub"), map[string]any{
		"connector":    connector.String(),
		"service_name": serviceName,
	})
	if err != nil {
		return nil, err
	}
	for result.Next(ctx) {
		rawSubcategory, ok := result.Record().Get("sub")
		if !ok {
			return nil, ErrKeyColumnNotFound
		}
		subcategoryNode, ok := rawSubcategory.(neo4j.Node)
		if !ok {
			return nil, ErrColumnConversion
		}

		subcategory, err := getCategoryFromNode(subcategoryNode)
		if err != nil {
			return nil, err
		}
		service.Subcategories = append(service.Subcategories, *subcategory)
	}

	return service, nil
}

func (gdb *GraphDatabase) GetResourceType(ctx context.Context, connector source.Type, resourceTypeName string) (*FilterCloudResourceTypeNode, error) {
	session := gdb.Driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close(ctx)

	// Get all services of the selected connector
	result, err := session.Run(ctx, "MATCH (f:Filter:FilterCloudResourceType{resource_type: $resourceType}) WHERE $connector = '' OR f.connector = $connector RETURN DISTINCT f", map[string]interface{}{
		"resourceType": resourceTypeName,
		"connector":    connector.String(),
	})
	if err != nil {
		return nil, err
	}

	singleNode, err := result.Single(ctx)
	if err != nil {
		return nil, err
	}

	rawFilter, ok := singleNode.Get("f")
	if !ok {
		return nil, ErrKeyColumnNotFound
	}
	filterNode, ok := rawFilter.(neo4j.Node)
	if !ok {
		return nil, ErrColumnConversion
	}

	filter, err := getFilterFromNode(filterNode)
	if err != nil {
		return nil, err
	}

	if filter.GetFilterType() != FilterTypeCloudResourceType {
		return nil, fmt.Errorf("filter is not of type FilterTypeCloudResourceType")
	}

	return filter.(*FilterCloudResourceTypeNode), nil
}

func (gdb *GraphDatabase) GetFilters(ctx context.Context, connector source.Type, serviceNames []string, filterType *FilterType) ([]Filter, error) {
	session := gdb.Driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close(ctx)

	filterTypeStr := ""
	if filterType != nil {
		filterTypeStr = string(*filterType)
	}
	if serviceNames == nil {
		serviceNames = []string{}
	}

	result, err := session.Run(ctx,
		fmt.Sprintf("MATCH (f:Filter:%s) WHERE ((f.connector IS NULL OR $connector = '' OR f.connector = $connector) AND ($service_names = [] OR (NOT f.service_name IS NULL AND f.service_name IN $service_names))) RETURN f;", filterTypeStr),
		map[string]any{
			"connector":     connector.String(),
			"service_names": serviceNames,
		},
	)
	if err != nil {
		return nil, err
	}

	var filters []Filter
	for result.Next(ctx) {
		rawFilter, ok := result.Record().Get("f")
		if !ok {
			return nil, ErrKeyColumnNotFound
		}
		filterNode, ok := rawFilter.(neo4j.Node)
		if !ok {
			return nil, ErrColumnConversion
		}

		filter, err := getFilterFromNode(filterNode)
		if err != nil {
			return nil, err
		}
		filters = append(filters, filter)
	}

	return filters, nil
}
