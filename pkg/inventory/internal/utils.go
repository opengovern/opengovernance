package internal

import "gitlab.com/keibiengine/keibi-engine/pkg/inventory/api"

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
