package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func BenchmarkBool(b *testing.B) {
	for i := 0; i < b.N; i++ {
		TestUtilTernary(&testing.T{})
	}
}

func TestUtilTernary(t *testing.T) {
	assert.Equal(t, "true", Ternary(true, "true", "false"))
	assert.Equal(t, "false", Ternary(false, "true", "false"))
}
