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
