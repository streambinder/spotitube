package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFlatten(t *testing.T) {
	assert.Equal(t, UniqueFields("word word1"), "word word1")
}

func TestUniqueFields(t *testing.T) {
	assert.Equal(t, UniqueFields("word word1 word"), "word word1")
	assert.Equal(t, UniqueFields("word word1"), "word word1")
}

func TestExcerpt(t *testing.T) {
	assert.Equal(t, Excerpt("long sentence to be cut"), "long sente")
	assert.Equal(t, Excerpt("shorter"), "shorter")
	assert.Equal(t, Excerpt("shorter", true), "shorter   ")
	assert.Empty(t, Excerpt(""))
}
