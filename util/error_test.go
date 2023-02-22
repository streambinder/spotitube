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

func TestUtilErrorOnly(t *testing.T) {
	assert.Nil(t, ErrOnly(func() (string, error) { return "", nil }()))
}
