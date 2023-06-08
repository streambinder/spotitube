package util

func Ternary[T any](expression bool, pass, otherwise T) T {
	if expression {
		return pass
	}
	return otherwise
}
