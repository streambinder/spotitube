package sys

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func BenchmarkInt(b *testing.B) {
	for i := 0; i < b.N; i++ {
		TestUtilRandomInt(&testing.T{})
	}
}

func TestUtilRandomInt(t *testing.T) {
	assert.LessOrEqual(t, RandomInt(10), 10)
	assert.GreaterOrEqual(t, RandomInt(10, 5), 5)
}
