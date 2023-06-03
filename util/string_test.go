package util

import (
	"testing"

	"github.com/agnivade/levenshtein"
	"github.com/stretchr/testify/assert"
)

func BenchmarkString(b *testing.B) {
	for i := 0; i < b.N; i++ {
		TestFlatten(&testing.T{})
		TestUniqueFields(&testing.T{})
		TestLevenshteinBoundedDistance(&testing.T{})
		TestExcerpt(&testing.T{})
		TestPad(&testing.T{})
		TestHumanizeBytes(&testing.T{})
		TestFallback(&testing.T{})
		TestContainsEach(&testing.T{})
		TestLegalizeFilename(&testing.T{})
	}
}

func TestFlatten(t *testing.T) {
	assert.Equal(t, "word word1", UniqueFields("word word1"))
}

func TestUniqueFields(t *testing.T) {
	assert.Equal(t, "word word1", UniqueFields("word word1 word"))
	assert.Equal(t, "word word1", UniqueFields("word word1"))
}

func TestLevenshteinBoundedDistance(t *testing.T) {
	assert.Equal(t, levenshtein.ComputeDistance("word1", "word2"), LevenshteinBoundedDistance("wOrd1", "woRd2"))
	assert.Equal(t, levenshtein.ComputeDistance("word1", "word3"), LevenshteinBoundedDistance("word1 wOrd2", "word2 Word3"))
	assert.Equal(t, levenshtein.ComputeDistance("word1", "word2"), LevenshteinBoundedDistance("worD1 word1", "word2"))
	assert.Equal(t, levenshtein.ComputeDistance("", "word2"), LevenshteinBoundedDistance("word1 word1", "word1 word2"))
}

func TestExcerpt(t *testing.T) {
	assert.Equal(t, "very long sente", Excerpt("very long sentence"))
	assert.Equal(t, "shorter", Excerpt("shorter"))
	assert.Equal(t, "short", Excerpt("short", -1))
	assert.Equal(t, "sho", Excerpt("shorter", 3))
	assert.Empty(t, Excerpt(""))
}

func TestPad(t *testing.T) {
	assert.Equal(t, "very long sente", Pad("very long sentence"))
	assert.Equal(t, "shorter        ", Pad("shorter"))
	assert.Equal(t, "short", Pad("short", -1))
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

func TestContainsEach(t *testing.T) {
	assert.True(t, Contains("hello world", "world", "hello"))
	assert.False(t, Contains("hello", "hello", "world"))
}

func TestLegalizeFilename(t *testing.T) {
	assert.Equal(t, "file name", LegalizeFilename(`file/\?% *:|"<>name`))
}
