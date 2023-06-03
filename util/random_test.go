package util

import (
	crand "crypto/rand"
	"errors"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"
)

func BenchmarkRandom(b *testing.B) {
	for i := 0; i < b.N; i++ {
		TestSeed(&testing.T{})
	}
}

func TestSeed(t *testing.T) {
	// monkey patching
	defer gomonkey.ApplyFunc(crand.Read, func() (int, error) {
		return -1, errors.New("ko")
	}).Reset()

	// testing
	assert.Panics(t, func() { seed() })
}
