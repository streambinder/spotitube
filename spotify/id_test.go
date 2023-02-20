package spotify

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestID(t *testing.T) {
	assert.True(t, ID("id") == "id")
}
