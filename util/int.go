package util

import (
	"crypto/rand"
	"math/big"
)

func RandomInt(max int, mins ...int) int {
	min := First(mins, 0)
	return int(ErrWrap(big.NewInt(0))(rand.Int(rand.Reader, big.NewInt(int64(max-min)))).Int64()) + min
}
