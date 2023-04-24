package utils

import "strings"

func Includes[T string](arr []T, item T) bool {
	for _, i := range arr {
		if i == item {
			return true
		}
	}
	return false
}

func IncludesAll[T string](arr []T, items []T) bool {
	for _, item := range items {
		if !Includes(arr, item) {
			return false
		}
	}
	return true
}

func ToLowerStringSlice(arr []string) []string {
	res := make([]string, 0, len(arr))
	for _, item := range arr {
		res = append(res, strings.ToLower(item))
	}
	return res
}

func Paginate[T any](page, size int64, arr []T) []T {
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
