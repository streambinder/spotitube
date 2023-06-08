package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func BenchmarkSlice(b *testing.B) {
	for i := 0; i < b.N; i++ {
		TestFirst(&testing.T{})
	}
}

func TestFirst(t *testing.T) {
	assert.Equal(t, "hello", First([]string{"hello", "world"}, "fallback"))
	assert.Equal(t, "fallback", First([]string{}, "fallback"))
}
