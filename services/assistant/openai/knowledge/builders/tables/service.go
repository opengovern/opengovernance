package tables

import (
	"context"
	_ "embed"
	"github.com/goccy/go-yaml"
	"github.com/kaytu-io/kaytu-aws-describer/steampipe-plugin-aws/aws"
	"github.com/kaytu-io/kaytu-azure-describer/steampipe-plugin-azure/azure"
	"github.com/kaytu-io/kaytu-azure-describer/steampipe-plugin-azuread/azuread"
	"github.com/kaytu-io/kaytu-engine/pkg/steampipe-plugin-kaytu/kaytu"
	"github.com/turbot/steampipe-plugin-sdk/v5/grpc/proto"
	"github.com/turbot/steampipe-plugin-sdk/v5/plugin"
	"github.com/turbot/steampipe-plugin-sdk/v5/plugin/transform"
	"strings"
)

//go:embed categories.txt
var categoriesStr string

type Column struct {
	Name          string `yaml:"Name"`
	Type          string `yaml:"Type"`
	Description   string `yaml:"Description"`
	FromJsonField string `yaml:"FromJsonField"`
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

func ExtractTableFiles() (map[string]string, error) {
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

	t, err := yaml.Marshal(extractTables(kaytu.Plugin(context.Background()).TableMap, tableCategories))
	if err != nil {
		return nil, err
	}
	files["kaytu_tables.yaml"] = string(t)

	t, err = yaml.Marshal(extractTables(aws.Plugin(context.Background()).TableMap, tableCategories))
	if err != nil {
		return nil, err
	}
	files["aws_tables.yaml"] = string(t)

	t, err = yaml.Marshal(extractTables(azure.Plugin(context.Background()).TableMap, tableCategories))
	if err != nil {
		return nil, err
	}
	files["azure_tables.yaml"] = string(t)

	t, err = yaml.Marshal(extractTables(azuread.Plugin(context.Background()).TableMap, tableCategories))
	if err != nil {
		return nil, err
	}
	files["azuread_tables.yaml"] = string(t)
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
		return "ip address"
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

func extractFromJsonField(transforms []*transform.TransformCall) string {
	var res []string
	for _, t := range transforms {
		if arr, ok := t.Param.([]string); ok {
			res = append(res, arr...)
		}
	}
	return strings.Join(res, ",")
}

func extractTables(tableMap map[string]*plugin.Table, categories map[string][]string) Def {
	var tables []Table
	for _, def := range tableMap {

		var columns []Column
		for _, col := range def.Columns {
			columns = append(columns, Column{
				Name:          col.Name,
				Type:          columnType(col.Type),
				Description:   col.Description,
				FromJsonField: extractFromJsonField(col.Transform.Transforms),
			})
		}

		tables = append(tables, Table{
			Name:          def.Name,
			Description:   def.Description,
			Documentation: "",                   //TODO-Saleh
			Categories:    categories[def.Name], //TODO-Saleh
			Columns:       columns,
		})
	}

	return Def{Tables: tables}
}
