package api

type BenchmarkTimeResponse struct {
	Aggregations BenchmarkTimeAggregation `json:"aggregations"`
}
type BenchmarkTimeAggregation struct {
	ReportTime ReportTimeAggregate `json:"reportTime"`
}
type ReportTimeAggregate struct {
	Buckets []Bucket `json:"buckets"`
}
type Bucket struct {
	Key      int64 `json:"key"`
	DocCount int64 `json:"doc_count"`
}

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
