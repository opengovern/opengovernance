package internal

import "strings"

func TagStringsToTagMap(tags []string) map[string][]string {
	tagUniqueMap := make(map[string]map[string]bool)
	for _, tag := range tags {
		key, value, ok := strings.Cut(tag, "=")
		if !ok {
			continue
		}
		if v, ok := tagUniqueMap[key]; !ok {
			tagUniqueMap[key] = map[string]bool{value: true}
		} else {
			v[value] = true
		}
	}

	tagMap := make(map[string][]string)
	for key, values := range tagUniqueMap {
		for value := range values {
			tagMap[key] = append(tagMap[key], value)
		}
	}

	return tagMap
}
