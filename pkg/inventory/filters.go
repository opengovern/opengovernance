package inventory

import (
	"encoding/base64"
	"fmt"
	"strconv"
)

func FilterIsEmpty(filter []string) bool {
	return filter == nil || len(filter) == 0
}

func BuildSort(sorts []ResourceSortItem) []map[string]interface{} {
	var result []map[string]interface{}
	for _, item := range sorts {
		dir := string(item.Direction)
		result = append(result, map[string]interface{}{string(item.Field) + ".keyword": dir})
	}
	return result
}

func BuildMarker(idx int) string {
	return base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%d", idx)))
}

func MarkerToIdx(marker string) (int, error) {
	b, err := base64.StdEncoding.DecodeString(marker)
	if err != nil {
		return 0, err
	}

	i, err := strconv.Atoi(string(b))
	if err != nil {
		return 0, err
	}

	return i, nil
}