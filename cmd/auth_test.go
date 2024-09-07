package cmd

import (
	"errors"
	"os"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/streambinder/spotitube/spotify"
	"github.com/streambinder/spotitube/util"
	"github.com/stretchr/testify/assert"
)

func BenchmarkAuth(b *testing.B) {
	for i := 0; i < b.N; i++ {
		TestCmdAuth(&testing.T{})
	}
}

func TestCmdAuth(t *testing.T) {
	// monkey patching
	defer gomonkey.NewPatches().
		ApplyFunc(os.Remove, func() error {
			return nil
		}).
		ApplyFunc(spotify.Authenticate, func() (*spotify.Client, error) {
			util.ErrSuppress(printProcessor(""))
			return &spotify.Client{}, nil
		}).
		Reset()

	// testing
	assert.Nil(t, util.ErrOnly(testExecute(cmdAuth(), "--remote", "--logout")))
}

func TestCmdAuthFailure(t *testing.T) {
	// monkey patching
	defer gomonkey.ApplyFunc(spotify.Authenticate, func() (*spotify.Client, error) {
		return nil, errors.New("ko")
	}).Reset()

	// testing
	assert.EqualError(t, util.ErrOnly(testExecute(cmdAuth())), "ko")
}

func TestCmdAuthLogoutFailure(t *testing.T) {
	// monkey patching
	defer gomonkey.ApplyFunc(os.Remove, func() error {
		return errors.New("ko")
	}).Reset()

	// testing
	assert.EqualError(t, util.ErrOnly(testExecute(cmdAuth(), "--logout")), "ko")
}
