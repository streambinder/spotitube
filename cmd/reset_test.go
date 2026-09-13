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

func mockWalkDir(cacheDir string, entries []struct {
	name  string
	isDir bool
},
) func(string, fs.WalkDirFunc) error {
	return func(_ string, f fs.WalkDirFunc) error {
		// first entry is always the root (cacheDirectory == path)
		if err := f(cacheDir, DirEntry{name: cacheDir, isDir: true}, nil); err != nil {
			return err
		}
		for _, e := range entries {
			err := f(filepath.Join(cacheDir, e.name), DirEntry{name: e.name, isDir: e.isDir}, nil)
			if err == filepath.SkipDir {
				continue // replicate real WalkDir behavior: SkipDir skips subtree, not an error
			}
			if err != nil {
				return err
			}
		}
		return nil
	}
}

func TestCmdReset(t *testing.T) {
	root := t.TempDir()

	defer mockey.UnPatchAll()
	mockey.Mock(sys.CacheDirectory).Return(root).Build()
	mockey.Mock(filepath.WalkDir).To(mockWalkDir(root, []struct {
		name  string
		isDir bool
	}{
		{spotify.TokenBasename, false}, // should be preserved (no --session)
		{"fname.txt", false},           // should be removed
	})).Build()
	mockey.Mock(rootRemoveAll).Return(nil).Build()

	assert.Nil(t, sys.ErrOnly(testExecute(cmdReset())))
}

func TestCmdResetDirectory(t *testing.T) {
	root := t.TempDir()

	defer mockey.UnPatchAll()
	mockey.Mock(sys.CacheDirectory).Return(root).Build()
	mockey.Mock(filepath.WalkDir).To(mockWalkDir(root, []struct {
		name  string
		isDir bool
	}{
		{"subdir", true}, // directory entry should trigger SkipDir
	})).Build()
	mockey.Mock(rootRemoveAll).Return(nil).Build()

	assert.Nil(t, sys.ErrOnly(testExecute(cmdReset())))
}

func TestCmdResetOpenRootFailure(t *testing.T) {
	defer mockey.UnPatchAll()
	mockey.Mock(os.OpenRoot).Return(nil, errors.New("ko")).Build()

	assert.EqualError(t, sys.ErrOnly(testExecute(cmdReset())), "ko")
}

func TestCmdResetSession(t *testing.T) {
	root := t.TempDir()

	defer mockey.UnPatchAll()
	mockey.Mock(sys.CacheDirectory).Return(root).Build()
	mockey.Mock(filepath.WalkDir).To(mockWalkDir(root, []struct {
		name  string
		isDir bool
	}{
		{spotify.TokenBasename, false}, // with --session, token should also be removed
	})).Build()
	mockey.Mock(rootRemoveAll).Return(nil).Build()

	assert.Nil(t, sys.ErrOnly(testExecute(cmdReset(), "--session")))
}

func TestCmdResetWalkDirError(t *testing.T) {
	root := t.TempDir()

	defer mockey.UnPatchAll()
	mockey.Mock(sys.CacheDirectory).Return(root).Build()
	mockey.Mock(filepath.WalkDir).To(func(_ string, f fs.WalkDirFunc) error {
		return f(root, DirEntry{name: root, isDir: true}, errors.New("ko"))
	}).Build()

	assert.EqualError(t, sys.ErrOnly(testExecute(cmdReset())), "ko")
}

func TestCmdResetRemoveFailure(t *testing.T) {
	root := t.TempDir()

	defer mockey.UnPatchAll()
	mockey.Mock(sys.CacheDirectory).Return(root).Build()
	mockey.Mock(filepath.WalkDir).To(mockWalkDir(root, []struct {
		name  string
		isDir bool
	}{
		{"fname.txt", false},
	})).Build()
	mockey.Mock(rootRemoveAll).Return(errors.New("ko")).Build()

	assert.EqualError(t, sys.ErrOnly(testExecute(cmdReset())), "ko")
}
