package cmd

import (
	"errors"
	"io/fs"
	"os"
	"testing"

	"github.com/bytedance/mockey"
	"github.com/streambinder/spotitube/spotify"
	"github.com/streambinder/spotitube/sys"
	"github.com/stretchr/testify/assert"
)

func BenchmarkAuth(b *testing.B) {
	for i := 0; i < b.N; i++ {
		TestCmdAuth(&testing.T{})
	}
}

func TestCmdAuth(t *testing.T) {
	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(os.Remove).Return(nil).Build()
	mockey.Mock(spotify.Authenticate).Return(&spotify.Client{}, nil).Build()

	// testing
	assert.Nil(t, sys.ErrOnly(testExecute(cmdAuth(), "--logout")))
}

func TestCmdAuthFailure(t *testing.T) {
	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(spotify.Authenticate).Return(nil, errors.New("ko")).Build()

	// testing
	assert.EqualError(t, sys.ErrOnly(testExecute(cmdAuth())), "ko")
}

func TestCmdAuthLogoutFailure(t *testing.T) {
	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(os.Remove).Return(errors.New("ko")).Build()

	// testing
	assert.EqualError(t, sys.ErrOnly(testExecute(cmdAuth(), "--logout")), "ko")
}

func TestCmdAuthLogoutNotExists(t *testing.T) {
	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(os.Remove).Return(fs.ErrNotExist).Build()
	mockey.Mock(spotify.Authenticate).Return(&spotify.Client{}, nil).Build()

	// testing
	assert.Nil(t, sys.ErrOnly(testExecute(cmdAuth(), "--logout")))
}
