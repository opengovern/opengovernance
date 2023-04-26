package internal

import (
	"sort"

	"gitlab.com/keibiengine/keibi-engine/pkg/inventory/api"
)

func Paginate[T api.ServiceSummary | api.Connection | api.LocationResponse | api.FilterCloudResourceType](page, size int64, arr []T) []T {
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 1
	}
	start := (page - 1) * size
	end := start + size
	if start > int64(len(arr)) {
		start = int64(len(arr))
	}
	if end > int64(len(arr)) {
		end = int64(len(arr))
	}
	return arr[start:end]
}

func SortFilters(filters []api.Filter, by string) []api.Filter {
	sort.Slice(filters, func(i, j int) bool {
		switch by {
		case "name":
			return filters[i].GetFilterName() < filters[j].GetFilterName()
		case "count":
			switch filters[i].GetFilterType() {
			case api.FilterTypeCloudResourceType:
				fi := filters[i].(*api.FilterCloudResourceType)
				switch filters[j].GetFilterType() {
				case api.FilterTypeCloudResourceType:
					fj := filters[j].(*api.FilterCloudResourceType)
					if fi.ResourceCount != fj.ResourceCount {
						return fi.ResourceCount > fj.ResourceCount
					}
				case api.FilterTypeInsightMetric:
					fj := filters[j].(*api.FilterInsightMetric)
					if fi.ResourceCount != fj.Value {
						return fi.ResourceCount > fj.Value
					}
				}
			case api.FilterTypeInsightMetric:
				fi := filters[i].(*api.FilterInsightMetric)
				switch filters[j].GetFilterType() {
				case api.FilterTypeCloudResourceType:
					fj := filters[j].(*api.FilterCloudResourceType)
					if fi.Value != fj.ResourceCount {
						return fi.Value > fj.ResourceCount
					}
				case api.FilterTypeInsightMetric:
					fj := filters[j].(*api.FilterInsightMetric)
					if fi.Value != fj.Value {
						return fi.Value > fj.Value
					}
				}
			}
		default:
			return filters[i].GetFilterName() < filters[j].GetFilterName()
		}
		return filters[i].GetFilterName() < filters[j].GetFilterName()
	})
	return filters
}
