package index

import (
	"errors"
	"io/fs"
	"path/filepath"
	"testing"

	"github.com/bytedance/mockey"
	"github.com/streambinder/id3v2-sylt"
	"github.com/streambinder/spotitube/entity"
	"github.com/streambinder/spotitube/entity/id3"
	"github.com/streambinder/spotitube/sys"
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
	defer mockey.UnPatchAll()
	mockey.Mock(filepath.WalkDir).To(func(_ string, f fs.WalkDirFunc) error {
		sys.ErrSuppress(f("", nil, errors.New("ko")))
		sys.ErrSuppress(f("", DirEntry{name: "dir", isDir: true}, nil))
		sys.ErrSuppress(f("fname.txt", DirEntry{name: "", isDir: false}, nil))
		return f("Artist - Title.mp3", DirEntry{name: "", isDir: false}, nil)
	}).Build()
	mockey.Mock(id3.Open).Return(&id3.Tag{}, nil).Build()
	mockey.Mock(mockey.GetMethod(&id3.Tag{}, "userDefinedText")).Return("id").Build()
	mockey.Mock(mockey.GetMethod(&id3v2.Tag{}, "Close")).Return(nil).Build()

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
	defer mockey.UnPatchAll()
	mockey.Mock(filepath.WalkDir).To(func(_ string, f fs.WalkDirFunc) error {
		return f("fname.mp3", DirEntry{name: "", isDir: false}, nil)
	}).Build()
	mockey.Mock(id3.Open).Return(nil, errors.New("ko")).Build()

	// testing
	assert.EqualError(t, New().Build("path"), "ko")
}

func TestBuildCloseFailure(t *testing.T) {
	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(filepath.WalkDir).To(func(_ string, f fs.WalkDirFunc) error {
		return f("fname.mp3", DirEntry{name: "", isDir: false}, nil)
	}).Build()
	mockey.Mock(id3.Open).Return(&id3.Tag{}, nil).Build()
	mockey.Mock(mockey.GetMethod(&id3.Tag{}, "userDefinedText")).Return("").Build()
	mockey.Mock(mockey.GetMethod(&id3v2.Tag{}, "Close")).Return(errors.New("ko")).Build()

	// testing
	assert.EqualError(t, New().Build("path"), "ko")
}
