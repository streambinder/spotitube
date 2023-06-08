package util

func First[T any](slice []T, fallback T) T {
	if len(slice) > 0 {
		return slice[0]
	}
	return fallback
}
