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
