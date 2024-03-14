package jsonmodels

import (
	_ "embed"
)

//go:embed schema.yaml
var schema string

func ExtractJSONModels() map[string]string {
	return map[string]string{
		"json_schema.yaml": schema,
	}
}
