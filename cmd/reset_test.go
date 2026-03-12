package cmd

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/bytedance/mockey"
	"github.com/streambinder/spotitube/spotify"
	"github.com/streambinder/spotitube/sys"
	"github.com/stretchr/testify/assert"
)

type DirEntry struct {
	name  string
	isDir bool
}

func (e DirEntry) Name() string {
	return e.name
}

func (e DirEntry) IsDir() bool {
	return e.isDir
}

func (e DirEntry) Type() fs.FileMode {
	return 0
}

func (e DirEntry) Info() (fs.FileInfo, error) {
	return nil, nil
}

func BenchmarkReset(b *testing.B) {
	for i := 0; i < b.N; i++ {
		TestCmdReset(&testing.T{})
	}
}

func TestCmdReset(t *testing.T) {
	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(filepath.WalkDir).To(func(_ string, f fs.WalkDirFunc) error {
		// skip root entry (cacheDirectory == path)
		if err := f("", DirEntry{name: "", isDir: false}, nil); err != nil {
			return err
		}
		// token should be preserved (no --session)
		if err := f(spotify.TokenBasename, DirEntry{name: spotify.TokenBasename, isDir: false}, nil); err != nil {
			return err
		}
		// other files should be removed
		return f("fname.txt", DirEntry{name: "fname.txt", isDir: false}, nil)
	}).Build()
	mockey.Mock(os.RemoveAll).Return(nil).Build()

	// testing
	assert.Nil(t, sys.ErrOnly(testExecute(cmdReset())))
}

func TestCmdResetSession(t *testing.T) {
	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(filepath.WalkDir).To(func(_ string, f fs.WalkDirFunc) error {
		if err := f("", DirEntry{name: "", isDir: false}, nil); err != nil {
			return err
		}
		// with --session, token should also be removed
		return f(spotify.TokenBasename, DirEntry{name: spotify.TokenBasename, isDir: false}, nil)
	}).Build()
	mockey.Mock(os.RemoveAll).Return(nil).Build()

	// testing
	assert.Nil(t, sys.ErrOnly(testExecute(cmdReset(), "--session")))
}

func TestCmdResetWalkDirError(t *testing.T) {
	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(filepath.WalkDir).To(func(_ string, f fs.WalkDirFunc) error {
		return f("", DirEntry{name: "", isDir: false}, errors.New("ko"))
	}).Build()

	// testing
	assert.EqualError(t, sys.ErrOnly(testExecute(cmdReset())), "ko")
}

func TestCmdResetRemoveFailure(t *testing.T) {
	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(filepath.WalkDir).To(func(_ string, f fs.WalkDirFunc) error {
		if err := f("", DirEntry{name: "", isDir: false}, nil); err != nil {
			return err
		}
		return f("fname.txt", DirEntry{name: "fname.txt", isDir: false}, nil)
	}).Build()
	mockey.Mock(os.RemoveAll).Return(errors.New("ko")).Build()

	// testing
	assert.EqualError(t, sys.ErrOnly(testExecute(cmdReset())), "ko")
}
