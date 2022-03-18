package inventory

func FilterIsEmpty(filter []string) bool {
	return filter == nil || len(filter) == 0
}

func BuildSort(sort Sort) []map[string]interface{} {
	var result []map[string]interface{}
	for _, item := range sort.SortBy {
		dir := string(item.Direction)
		result = append(result, map[string]interface{}{string(item.Field) + ".keyword": dir})
	}
	return result
}
