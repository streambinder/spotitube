package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func BenchmarkChar(b *testing.B) {
	for i := 0; i < b.N; i++ {
		TestUtilRandomAlpha(&testing.T{})
	}
}

func TestUtilRandomAlpha(t *testing.T) {
	assert.NotEmpty(t, RandomAlpha())
}
