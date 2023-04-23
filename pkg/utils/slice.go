package utils

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
