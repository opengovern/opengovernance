package types

import (
	"regexp"
	"strings"
)

type FullResourceType struct {
	ID   string `json:"ID" example:"/subscriptions/123/resourceGroups/rg-1/providers/Microsoft.Compute/virtualMachines"` // Resource type ID
	Name string `json:"name" example:"Microsoft.Compute/virtualMachines"`                                                // Resource type name
}

var stopWordsRe = regexp.MustCompile(`\W+`)

func ResourceTypeToESIndex(t string) string {
	t = stopWordsRe.ReplaceAllString(t, "_")
	return strings.ToLower(t)
}
