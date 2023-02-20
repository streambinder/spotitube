package spotify

import (
	"testing"

	"github.com/matryer/is"
)

func TestID(t *testing.T) {
	is.New(t).True(ID("id") == "id")
}
