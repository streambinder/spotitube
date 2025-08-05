package cmd

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
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
	defer gomonkey.NewPatches().
		ApplyFunc(filepath.WalkDir, func(_ string, f func(string, fs.DirEntry, error) error) error {
			sys.ErrSuppress(f("", DirEntry{name: "", isDir: false}, errors.New("some error")))
			sys.ErrSuppress(f(spotify.TokenBasename, DirEntry{name: spotify.TokenBasename, isDir: false}, nil))
			sys.ErrSuppress(f("fname.txt", DirEntry{name: "fname.txt", isDir: false}, nil))
			return nil
		}).
		ApplyFunc(os.RemoveAll, func() error {
			return nil
		}).
		Reset()

	// testing
	assert.Nil(t, sys.ErrOnly(testExecute(cmdReset())))
}
