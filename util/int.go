package util

import "math/rand"

func RandomInt(max int, mins ...int) int {
	min := First(mins, 0)
	return rand.Intn(max-min) + min
}
