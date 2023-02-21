package util

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUtilErrorWrapTrue(t *testing.T) {
	assert.True(t, ErrWrap(true)(func() (bool, error) { return false, errors.New("test") }()))
}

func TestUtilErrorWrapFalse(t *testing.T) {
	assert.True(t, !ErrWrap(true)(func() (bool, error) { return false, nil }()))
}
