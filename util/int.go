package util

import (
	"crypto/rand"
	"math/big"
)

func RandomInt(upperbound int, lowerbounds ...int) int {
	lowerbound := First(lowerbounds, 0)
	return int(ErrWrap(big.NewInt(0))(rand.Int(rand.Reader, big.NewInt(int64(upperbound-lowerbound)))).Int64()) + lowerbound
}
