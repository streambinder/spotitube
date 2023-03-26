package entity

import (
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPaths(t *testing.T) {
	id := "123"
	downloadPath := (&Track{ID: id}).Path().Download()
	assert.Equal(t, id+"."+format, path.Base(downloadPath))
}
