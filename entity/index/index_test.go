package index

import (
	"errors"
	"io/fs"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/streambinder/id3v2-sylt"
	"github.com/streambinder/spotitube/entity"
	"github.com/streambinder/spotitube/entity/id3"
	"github.com/streambinder/spotitube/util"
	"github.com/stretchr/testify/assert"
)

type DirEntry struct {
	fs.DirEntry

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

func BenchmarkIndex(b *testing.B) {
	for i := 0; i < b.N; i++ {
		TestBuild(&testing.T{})
	}
}

func TestBuild(t *testing.T) {
	// monkey patching
	defer gomonkey.NewPatches().
		ApplyFunc(filepath.WalkDir, func(_ string, f func(string, fs.DirEntry, error) error) error {
			util.ErrSuppress(f("", nil, errors.New("ko")))
			util.ErrSuppress(f("", DirEntry{name: "dir", isDir: true}, nil))
			util.ErrSuppress(f("fname.txt", DirEntry{name: "", isDir: false}, nil))
			return f("Artist - Title.mp3", DirEntry{name: "", isDir: false}, nil)
		}).
		ApplyFunc(id3.Open, func() (*id3.Tag, error) {
			return &id3.Tag{}, nil
		}).
		ApplyPrivateMethod(reflect.TypeOf(&id3.Tag{}), "userDefinedText", func() string {
			return "id"
		}).
		ApplyPrivateMethod(reflect.TypeOf(&id3v2.Tag{}), "Close", func() error {
			return nil
		}).
		Reset()

	// testing
	index := New()
	index.Set(&entity.Track{Title: "Title", Artists: []string{"Artist"}}, Offline)
	assert.Nil(t, index.Build("path", 0))
	status, ok := index.Get(&entity.Track{ID: "id", Artists: []string{"Artist"}, Title: "Title"})
	assert.True(t, ok)
	assert.Equal(t, 0, status)
	assert.Equal(t, 1, index.Size())
	assert.Equal(t, 1, index.Size(Offline))
}

func TestBuildOpenFailure(t *testing.T) {
	// monkey patching
	defer gomonkey.NewPatches().
		ApplyFunc(filepath.WalkDir, func(_ string, f func(string, fs.DirEntry, error) error) error {
			return f("fname.mp3", DirEntry{name: "", isDir: false}, nil)
		}).
		ApplyFunc(id3.Open, func() (*id3.Tag, error) {
			return nil, errors.New("ko")
		}).
		Reset()

	// testing
	assert.EqualError(t, New().Build("path"), "ko")
}
