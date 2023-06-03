package util

import "math/rand"

func RandomAlpha() rune {
	return rune('a' - 1 + rand.Intn(26))
}
