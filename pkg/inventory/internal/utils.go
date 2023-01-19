package internal

import (
	"sort"

	"gitlab.com/keibiengine/keibi-engine/pkg/inventory/api"
)

func Paginate[T api.ServiceSummary | api.AccountSummary](page, size int, arr []T) []T {
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 1
	}
	start := (page - 1) * size
	end := start + size
	if start > len(arr) {
		start = len(arr)
	}
	if end > len(arr) {
		end = len(arr)
	}
	return arr[start:end]
}

func SortFilters(filters []api.Filter) []api.Filter {
	sort.Slice(filters, func(i, j int) bool {
		switch filters[i].GetFilterType() {
		case api.FilterTypeCloudResourceType:
			fi := filters[i].(*api.FilterCloudResourceType)
			switch filters[j].GetFilterType() {
			case api.FilterTypeCloudResourceType:
				fj := filters[j].(*api.FilterCloudResourceType)
				if fi.Weight != fj.Weight {
					return fi.Weight < fj.Weight
				}
			case api.FilterTypeInsight:
				fj := filters[j].(*api.FilterInsight)
				if fi.Weight != fj.Weight {
					return fi.Weight < fj.Weight
				}
			}
		case api.FilterTypeInsight:
			fi := filters[i].(*api.FilterInsight)
			switch filters[j].GetFilterType() {
			case api.FilterTypeCloudResourceType:
				fj := filters[j].(*api.FilterCloudResourceType)
				if fi.Weight != fj.Weight {
					return fi.Weight < fj.Weight
				}
			case api.FilterTypeInsight:
				fj := filters[j].(*api.FilterInsight)
				if fi.Weight != fj.Weight {
					return fi.Weight < fj.Weight
				}
			}
		}
		return filters[i].GetFilterName() < filters[j].GetFilterName()
	})
	return filters
}
