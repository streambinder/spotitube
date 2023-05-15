package util

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func BenchmarkError(b *testing.B) {
	for i := 0; i < b.N; i++ {
		TestUtilErrWrap(&testing.T{})
		TestUtilErrOnly(&testing.T{})
	}
}

func TestUtilErrWrap(t *testing.T) {
	assert.True(t, ErrWrap(true)(func() (bool, error) { return false, errors.New("test") }()))
	assert.True(t, !ErrWrap(true)(func() (bool, error) { return false, nil }()))
}

func TestUtilErrOnly(t *testing.T) {
	assert.Nil(t, ErrOnly())
	assert.Nil(t, ErrOnly(func() (string, error) { return "", nil }()))
	assert.EqualError(t, ErrOnly(func() (string, error) { return "", errors.New("ko") }()), "ko")
}
