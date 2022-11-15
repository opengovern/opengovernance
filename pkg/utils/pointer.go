package utils

func PAdd[T int | int64 | int32](a, b *T) *T {
	if a == nil && b == nil {
		return nil
	} else if a == nil {
		return b
	} else if b == nil {
		return a
	} else {
		v := *a + *b
		return &v
	}
}
