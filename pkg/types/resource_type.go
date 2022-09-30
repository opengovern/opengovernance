package types

import (
	"regexp"
	"strings"
)

type FullResourceType struct {
	ID   string `json:"ID"`
	Name string `json:"name"`
}

var stopWordsRe = regexp.MustCompile(`\W+`)

func ResourceTypeToESIndex(t string) string {
	t = stopWordsRe.ReplaceAllString(t, "_")
	return strings.ToLower(t)
}
