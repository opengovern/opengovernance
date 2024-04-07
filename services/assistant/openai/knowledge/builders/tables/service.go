package tables

import (
	"context"
	_ "embed"
	"encoding/json"
	"github.com/goccy/go-yaml"
	"github.com/hashicorp/go-hclog"
	"github.com/invopop/jsonschema"
	steampipeAws "github.com/kaytu-io/kaytu-aws-describer/pkg/steampipe"
	"github.com/kaytu-io/kaytu-aws-describer/steampipe-plugin-aws/aws"
	steampipeAzure "github.com/kaytu-io/kaytu-azure-describer/pkg/steampipe"
	"github.com/kaytu-io/kaytu-azure-describer/steampipe-plugin-azure/azure"
	"github.com/kaytu-io/kaytu-azure-describer/steampipe-plugin-azuread/azuread"
	"github.com/kaytu-io/kaytu-engine/pkg/steampipe-plugin-kaytu/kaytu"
	"github.com/turbot/steampipe-plugin-sdk/v5/grpc/proto"
	"github.com/turbot/steampipe-plugin-sdk/v5/plugin"
	"github.com/turbot/steampipe-plugin-sdk/v5/plugin/context_key"
	"github.com/turbot/steampipe-plugin-sdk/v5/plugin/transform"
	"go.uber.org/zap"
	"strings"
)

//go:embed categories.txt
var categoriesStr string

type Column struct {
	Name          string `yaml:"Name"`
	Type          string `yaml:"Type"`
	Description   string `yaml:"Description"`
	FromJsonField string `yaml:"FromJsonField"`

	JsonSchema any `yaml:"JsonSchema,omitempty"`
}

type Table struct {
	Name          string   `yaml:"Name"`
	Description   string   `yaml:"Description"`
	Documentation string   `yaml:"Documentation"`
	Categories    []string `yaml:"Categories"`
	Columns       []Column `yaml:"Columns"`
}

type Def struct {
	Tables []Table `yaml:"Tables"`
}

func ExtractTableFiles(ctx context.Context, logger *zap.Logger) (map[string]string, error) {
	tableCategories := map[string][]string{}
	catArr := strings.Split(categoriesStr, "\n")
	for _, i := range catArr {
		if len(strings.TrimSpace(i)) == 0 {
			continue
		}

		j := strings.Split(i, ":")
		if len(j) < 2 {
			continue
		}

		tableName := strings.TrimSpace(j[0])
		categoryName := strings.TrimSpace(j[1])
		tableCategories[tableName] = append(tableCategories[tableName], categoryName)
	}

	files := map[string]string{}
	var tableNames []string

	tables := extractTables(ctx, logger, kaytu.Plugin(ctx).TableMap, nil, nil, tableCategories)
	t, err := yaml.Marshal(tables)
	if err != nil {
		return nil, err
	}
	files["kaytu_tables.yaml"] = string(t)
	for _, tb := range tables.Tables {
		tableNames = append(tableNames, tb.Name)
	}

	tables = extractTables(ctx, logger, aws.Plugin(ctx).TableMap, steampipeAws.AWSReverseMap, steampipeAws.AWSDescriptionMap, tableCategories)
	t, err = yaml.Marshal(tables)
	if err != nil {
		return nil, err
	}
	files["aws_tables.yaml"] = string(t)
	for _, tb := range tables.Tables {
		tableNames = append(tableNames, tb.Name)
	}

	tables = extractTables(ctx, logger, azure.Plugin(ctx).TableMap, steampipeAzure.AzureReverseMap, steampipeAzure.AzureDescriptionMap, tableCategories)
	t, err = yaml.Marshal(tables)
	if err != nil {
		return nil, err
	}
	files["azure_tables.yaml"] = string(t)
	for _, tb := range tables.Tables {
		tableNames = append(tableNames, tb.Name)
	}

	tables = extractTables(ctx, logger, azuread.Plugin(ctx).TableMap, steampipeAzure.AzureReverseMap, steampipeAzure.AzureDescriptionMap, tableCategories)
	t, err = yaml.Marshal(tables)
	if err != nil {
		return nil, err
	}
	files["azuread_tables.yaml"] = string(t)
	for _, tb := range tables.Tables {
		tableNames = append(tableNames, tb.Name)
	}

	yt, err := yaml.Marshal(tableNames)
	if err != nil {
		return nil, err
	}

	files["table_list.txt"] = string(yt)
	return files, nil
}

func columnType(t proto.ColumnType) string {
	switch t {
	case proto.ColumnType_BOOL:
		return "boolean"
	case proto.ColumnType_INT:
		return "int"
	case proto.ColumnType_DOUBLE:
		return "double"
	case proto.ColumnType_STRING:
		return "string"
	case proto.ColumnType_JSON:
		return "json"
	case proto.ColumnType_DATETIME:
		return "datetime"
	case proto.ColumnType_IPADDR:
		return "ip_address"
	case proto.ColumnType_CIDR:
		return "CIDR"
	case proto.ColumnType_TIMESTAMP:
		return "timestamp"
	case proto.ColumnType_INET:
		return "INET"
	case proto.ColumnType_LTREE:
		return "LTREE"
	default:
		return "unknown"
	}
}

func extractFromJsonField(transforms *transform.ColumnTransforms) string {
	if transforms == nil {
		return ""
	}

	var res []string
	for _, t := range transforms.Transforms {
		if t == nil {
			continue
		}
		if arr, ok := t.Param.([]string); ok {
			res = append(res, arr...)
		}
	}
	return strings.Join(res, ",")
}

func extractTables(ctx context.Context, logger *zap.Logger, tableMap map[string]*plugin.Table,
	tableToResourceTypeMap map[string]string,
	resourceTypeToTypeMap map[string]any, categories map[string][]string) Def {
	var tables []Table
	for _, def := range tableMap {
		var columns []Column
		for _, col := range def.Columns {
			if col == nil {
				continue
			}
			column := Column{
				Name:          col.Name,
				Type:          columnType(col.Type),
				Description:   col.Description,
				FromJsonField: extractFromJsonField(col.Transform),
				JsonSchema:    nil,
			}
			if col.Type == proto.ColumnType_JSON && col.Transform != nil && resourceTypeToTypeMap != nil && tableToResourceTypeMap != nil {
				resourceType, ok := tableToResourceTypeMap[def.Name]
				if !ok {
					logger.Debug("resource type not found for table", zap.String("table", def.Name))
					continue
				}
				descObj, ok := resourceTypeToTypeMap[resourceType]
				if !ok {
					logger.Debug("resource type not found in resource type map", zap.String("table", def.Name), zap.String("resourceType", resourceType))
					continue
				}
				//descObjRecursiveZeroValue := utils.GetNestedZeroValue(descObj)
				ctx2 := context.WithValue(ctx, context_key.Logger, hclog.NewNullLogger())
				colObj, err := col.Transform.Execute(ctx2, &transform.TransformData{
					HydrateItem: descObj,
					ColumnName:  column.Name,
				})
				if err != nil || colObj == nil {
					logger.Debug("skipping generating json schema for column", zap.String("table", def.Name), zap.String("column", col.Name), zap.Error(err))
					continue
				}
				jsonSchema, err := jsonschema.Reflect(colObj).MarshalJSON()
				if err != nil {
					logger.Debug("failed to generate json schema for column", zap.String("table", def.Name), zap.String("column", col.Name), zap.Error(err))
					continue
				}
				var schema any
				err = json.Unmarshal(jsonSchema, &schema)
				if err != nil {
					logger.Debug("failed to unmarshal json schema for column", zap.String("table", def.Name), zap.String("column", col.Name), zap.Error(err))
					continue
				}
				column.JsonSchema = schema
			}
			columns = append(columns, column)
		}

		tables = append(tables, Table{
			Name:          def.Name,
			Description:   def.Description,
			Documentation: "", //TODO-Saleh
			Categories:    categories[def.Name],
			Columns:       columns,
		})
	}

	return Def{Tables: tables}
}
