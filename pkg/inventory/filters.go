package inventory

import (
	"encoding/base64"
	"fmt"
	"strconv"

	"gitlab.com/keibiengine/keibi-engine/pkg/inventory/api"
)

func FilterIsEmpty(filter []string) bool {
	return len(filter) == 0
}

func BuildSort(sorts []api.ResourceSortItem) []map[string]interface{} {
	var result []map[string]interface{}
	for _, item := range sorts {
		result = append(result, map[string]interface{}{string(item.Field) + ".keyword": string(item.Direction)})
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
