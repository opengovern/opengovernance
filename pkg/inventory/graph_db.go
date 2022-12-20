package inventory

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"gitlab.com/keibiengine/keibi-engine/pkg/source"
)

type CategoryRootType string

const (
	DefaultTemplateRootName = "default"

	RootTypeTemplateRoot      CategoryRootType = "TemplateRoot"
	RootTypeCloudProviderRoot CategoryRootType = "CloudProviderRoot"
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
	_, err = session.Run(ctx, "CREATE CONSTRAINT cloud_provider_root_unique_name_constraint IF NOT EXISTS FOR (c:CloudProviderRoot) REQUIRE c.name IS UNIQUE", nil)
	if err != nil {
		return GraphDatabase{}, err
	}
	_, err = session.Run(ctx, "CREATE CONSTRAINT cloud_service_category_unique_service_code IF NOT EXISTS FOR (c:CloudServiceCategory) REQUIRE c.service_code IS UNIQUE", nil)
	if err != nil {
		return GraphDatabase{}, err
	}
	_, err = session.Run(ctx, "CREATE CONSTRAINT cloud_resource_type_unique_resource_type IF NOT EXISTS FOR (c:FilterCloudResourceType) REQUIRE c.resource_type IS UNIQUE", nil)
	if err != nil {
		return GraphDatabase{}, err
	}

	return GraphDatabase{
		Driver: driver,
	}, nil
}

/* Example graph
CREATE (c1:Category:TemplateRoot{name:"cat1"}),
  (c2:Category{name:"cat2"}),
  (c3:Category{name:"cat3"}),
  (c4:Category{name:"cat4"}),
  (c1)-[:INCLUDES]->(c4),
  (c2)-[:INCLUDES]->(c3),
  (c1)-[:INCLUDES]->(c2),
  (f1:Filter:FilterCloudResourceType{resource_name:"EC2 Instance", cloud_provider: "AWS", resource_type: "AWS::EC2::Instance", service_code: "ec2"}),
  (f2:Filter:FilterCloudResourceType{resource_name: "EKS Cluster", cloud_provider: "AWS", resource_type: "AWS::EKS::Cluster", service_code: "eks"}),
  (f3:Filter:FilterCloudResourceType{resource_name: "S3 Bucket", cloud_provider: "AWS", resource_type: "AWS::S3::Bucket", "service_code: "s3"}),
  (c3)-[:USES]->(f1),
  (c4)-[:USES]->(f1),
  (c1)-[:USES]->(f2);

Note 1: The graph is not a tree, but a DAG.
Note 2: Filters have multiple labels, one for identifying all the filters and one to specify the type of filter
*/

type Node struct {
	ElementID string
}

type CategoryNode struct {
	Node
	Name           string         `json:"name"`
	Subcategories  []CategoryNode `json:"subcategories,omitempty"`
	Filters        []Filter       `json:"filters,omitempty"` // Filters that are directly associated with this category
	SubTreeFilters []Filter       `json:"-"`                 // SubTreeFilters List of all filters that are in the subtree of this category
}

type Filter interface {
	GetFilterType() FilterType
}

type FilterType string

const (
	FilterTypeCloudResourceType FilterType = "FilterCloudResourceType"
	FilterTypeCost              FilterType = "FilterCost"
)

type FilterCloudResourceTypeNode struct {
	Node
	CloudProvider source.Type `json:"cloud_provider"`
	ResourceType  string      `json:"resource_type"`
	ResourceName  string      `json:"resource_name"`
	ServiceCode   string      `json:"service_code"`
	Importance    string      `json:"importance"`
}

func (f FilterCloudResourceTypeNode) GetFilterType() FilterType {
	return FilterTypeCloudResourceType
}

type FilterCostNode struct {
	Node
	CloudProvider source.Type `json:"cloud_provider"`
	ServiceName   string      `json:"service_name"`
}

func (f FilterCostNode) GetFilterType() FilterType {
	return FilterTypeCost
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
  WHERE (f.importance IS NULL OR 'all' IN $importance OR f.importance IN $importance)
  RETURN DISTINCT f, false as isDirectChild
  UNION 
  WITH c MATCH (c)-[:USES]->(f:Filter)
  WHERE (f.importance IS NULL OR 'all' IN $importance OR f.importance IN $importance)
  RETURN DISTINCT f, true as isDirectChild }
RETURN DISTINCT c, f, MAX(isDirectChild) AS isDirectChild
`
)

func getFilterFromNode(node neo4j.Node) (Filter, error) {
	for _, label := range node.Labels {
		switch label {
		case string(FilterTypeCloudResourceType):
			cloudProvider, ok := node.Props["cloud_provider"]
			if !ok {
				return nil, ErrPropertyNotFound
			}
			resourceType, ok := node.Props["resource_type"]
			if !ok {
				return nil, ErrPropertyNotFound
			}
			resourceName, ok := node.Props["resource_name"]
			if !ok {
				return nil, ErrPropertyNotFound
			}
			serviceCode, ok := node.Props["service_code"]
			if !ok {
				return nil, ErrPropertyNotFound
			}
			importance, ok := node.Props["importance"]
			if !ok {
				return nil, ErrPropertyNotFound
			}

			return &FilterCloudResourceTypeNode{
				Node: Node{
					ElementID: node.ElementId,
				},
				CloudProvider: source.Type(cloudProvider.(string)),
				ResourceType:  strings.ToLower(resourceType.(string)),
				ResourceName:  resourceName.(string),
				ServiceCode:   strings.ToLower(serviceCode.(string)),
				Importance:    strings.ToLower(importance.(string)),
			}, nil
		case string(FilterTypeCost):
			cloudProvider, ok := node.Props["cloud_provider"]
			if !ok {
				return nil, ErrPropertyNotFound
			}
			serviceName, ok := node.Props["service_name"]
			if !ok {
				return nil, ErrPropertyNotFound
			}
			return &FilterCostNode{
				Node: Node{
					ElementID: node.ElementId,
				},
				CloudProvider: source.Type(cloudProvider.(string)),
				ServiceName:   serviceName.(string),
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

	return &CategoryNode{
		Node: Node{
			ElementID: node.ElementId,
		},
		Name:           name.(string),
		Filters:        []Filter{},
		SubTreeFilters: []Filter{},
		Subcategories:  []CategoryNode{},
	}, nil
}

func (gdb *GraphDatabase) GetCategoryRoots(ctx context.Context, rootType CategoryRootType, importance []string) (map[string]*CategoryNode, error) {
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
	result, err = session.Run(ctx, fmt.Sprintf(subTreeFiltersQuery, fmt.Sprintf(":%s", rootType), "true"), map[string]any{
		"importance": importance,
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

func (gdb *GraphDatabase) GetCategoryRootByName(ctx context.Context, rootType CategoryRootType, name string, importance []string) (*CategoryNode, error) {
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
		"name":       name,
		"importance": importance,
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

func (gdb *GraphDatabase) GetCategory(ctx context.Context, elementID string, importance []string) (*CategoryNode, error) {
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
		"elementID":  elementID,
		"importance": importance,
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
