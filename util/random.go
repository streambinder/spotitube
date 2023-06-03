package util

import (
	crand "crypto/rand"
	"encoding/binary"
	mrand "math/rand"
)

func init() {
	seed()
}

func seed() {
	var b [8]byte
	_, err := crand.Read(b[:])
	if err != nil {
		panic("cannot seed math/rand package with cryptographically secure random number generator")
	}
	mrand.Seed(int64(binary.LittleEndian.Uint64(b[:])))
}
