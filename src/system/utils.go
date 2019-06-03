package system

import (
	"math/rand"
	"time"
)

const (
	// SystemLetterBytes : random string generator characters
	SystemLetterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	// SystemLetterIdxBits : random string generator bits
	SystemLetterIdxBits = 6
	// SystemLetterIdxMask : random string generator mask
	SystemLetterIdxMask = 1<<SystemLetterIdxBits - 1
	// SystemLetterIdxMax : random string generator max
	SystemLetterIdxMax = 63 / SystemLetterIdxBits
)

// MakeRange : return a range array between input int(s) min and max
func MakeRange(min, max int) []int {
	a := make([]int, max-min+1)
	for i := range a {
		a[i] = min + i
	}
	return a
}

// RandString : return a (input int) n-long random string
func RandString(n int) string {
	var src = rand.NewSource(time.Now().UnixNano())
	b := make([]byte, n)
	for i, cache, remain := n-1, src.Int63(), SystemLetterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = src.Int63(), SystemLetterIdxMax
		}
		if idx := int(cache & SystemLetterIdxMask); idx < len(SystemLetterBytes) {
			b[i] = SystemLetterBytes[idx]
			i--
		}
		cache >>= SystemLetterIdxBits
		remain--
	}

	return string(b)
}
