package inventory

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"gitlab.com/keibiengine/keibi-engine/pkg/source"
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
  (f1:Filter:FilterCloudResourceType{name:"EC2 Instance", cloud_provider: "AWS", cloud_service: "AWS::EC2::Instance"}),
  (f2:Filter:FilterCloudResourceType{name: "EKS Cluster", cloud_provider: "AWS", cloud_service: "AWS::EKS::Cluster"}),
  (f3:Filter:FilterCloudResourceType{name: "S3 Bucket", cloud_provider: "AWS", cloud_service: "AWS::S3::Bucket"}),
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
)

type FilterCloudResourceTypeNode struct {
	Node
	CloudProvider source.Type `json:"cloud_provider"`
	ResourceType  string      `json:"resource_type"`
	ResourceName  string      `json:"resource_name"`
}

func (f FilterCloudResourceTypeNode) GetFilterType() FilterType {
	return FilterTypeCloudResourceType
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
  WITH c MATCH (c)-[:INCLUDES*]->(:Category)-[:USES]->(f:Filter) RETURN DISTINCT f, false as isDirectChild
  UNION 
  WITH c MATCH (c)-[:USES]->(f:Filter) RETURN DISTINCT f, true as isDirectChild }
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

			return &FilterCloudResourceTypeNode{
				Node: Node{
					ElementID: node.ElementId,
				},
				CloudProvider: source.Type(cloudProvider.(string)),
				ResourceType:  strings.ToLower(resourceType.(string)),
				ResourceName:  resourceName.(string),
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

func (gdb *GraphDatabase) GetTemplateRoots(ctx context.Context) (map[string]*CategoryNode, error) {
	session := gdb.Driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close(ctx)

	var categories = make(map[string]*CategoryNode)

	// Get all categories that have no parent
	result, err := session.Run(ctx, "MATCH (c:Category:TemplateRoot) RETURN c", nil)
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
	result, err = session.Run(ctx, fmt.Sprintf(subTreeFiltersQuery, ":TemplateRoot", "true"), nil)
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
	result, err = session.Run(ctx, "MATCH (c:Category)-[:INCLUDES]->(sub:Category) WHERE NOT (:Category)-[:INCLUDES]->(c) RETURN DISTINCT c, sub", nil)
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

func (gdb *GraphDatabase) GetTemplateRootByName(ctx context.Context, name string) (*CategoryNode, error) {
	session := gdb.Driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close(ctx)

	var category *CategoryNode

	// Get the category
	result, err := session.Run(ctx, "MATCH (c:Category:TemplateRoot{name: $name}) RETURN c", map[string]interface{}{
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
	result, err = session.Run(ctx, fmt.Sprintf(subTreeFiltersQuery, "TemplateRoot{name: $name}", "true"), map[string]interface{}{
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
	result, err = session.Run(ctx, "MATCH (c:Category:TemplateRoot{name: $name})-[:INCLUDES]->(sub:Category) RETURN DISTINCT c, sub", map[string]interface{}{
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
