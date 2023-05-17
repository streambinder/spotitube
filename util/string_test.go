package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func BenchmarkString(b *testing.B) {
	for i := 0; i < b.N; i++ {
		TestFlatten(&testing.T{})
		TestUniqueFields(&testing.T{})
		TestExcerpt(&testing.T{})
		TestPad(&testing.T{})
		TestHumanizeBytes(&testing.T{})
		TestFallback(&testing.T{})
	}
}

func TestFlatten(t *testing.T) {
	assert.Equal(t, "word word1", UniqueFields("word word1"))
}

func TestUniqueFields(t *testing.T) {
	assert.Equal(t, "word word1", UniqueFields("word word1 word"))
	assert.Equal(t, "word word1", UniqueFields("word word1"))
}

func TestExcerpt(t *testing.T) {
	assert.Equal(t, "long sente", Excerpt("long sentence"))
	assert.Equal(t, "shorter", Excerpt("shorter"))
	assert.Equal(t, "sho", Excerpt("shorter", 3))
	assert.Empty(t, Excerpt(""))
}

func TestPad(t *testing.T) {
	assert.Equal(t, "long sente", Pad("long sentence"))
	assert.Equal(t, "shorter   ", Pad("shorter"))
	assert.Equal(t, "sho", Pad("shorter", 3))
	assert.Equal(t, " ", Pad("", 1))
}

func TestHumanizeBytes(t *testing.T) {
	assert.Equal(t, "1B", HumanizeBytes(1))
	assert.Equal(t, "1.0kB", HumanizeBytes(1000))
	assert.Equal(t, "1.5kB", HumanizeBytes(1500))
	assert.Equal(t, "1.0MB", HumanizeBytes(1000000))
	assert.Equal(t, "1.0GB", HumanizeBytes(1000000000))
	assert.Equal(t, "1.0TB", HumanizeBytes(1000000000000))
	assert.Equal(t, "1.0PB", HumanizeBytes(1000000000000000))
	assert.Equal(t, "1.0EB", HumanizeBytes(1000000000000000000))
}

func TestFallback(t *testing.T) {
	assert.Equal(t, "hello", Fallback("hello", "world"))
	assert.Equal(t, "world", Fallback("", "world"))
}
