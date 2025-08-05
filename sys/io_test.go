package sys

import (
	"errors"
	"io/fs"
	"os"
	"testing"

	"github.com/adrg/xdg"
	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"
)

func BenchmarkIO(b *testing.B) {
	for i := 0; i < b.N; i++ {
		TestFileCopy(&testing.T{})
		TestFileBaseStem(&testing.T{})
		TestCacheDirectory(&testing.T{})
		TestCacheFile(&testing.T{})
	}
}

func TestFileMove(t *testing.T) {
	src, dst := "/tmp/test_a.txt", "/tmp/test_b.txt"
	file, err := os.Create(src)
	assert.Nil(t, err)
	assert.Nil(t, file.Close())
	assert.Nil(t, FileMoveOrCopy(src, dst))
	assert.Nil(t, os.Remove(dst))
}

func TestFileCopy(t *testing.T) {
	// monkey patching
	defer gomonkey.NewPatches().
		ApplyFunc(os.Rename, func() error {
			return errors.New("not renaming")
		}).
		ApplyFunc(os.ReadFile, func() ([]byte, error) {
			return []byte{}, nil
		}).
		ApplyFunc(os.WriteFile, func() error {
			return nil
		}).
		ApplyFunc(os.Remove, func() error {
			return nil
		}).
		Reset()

	// testing
	assert.Nil(t, FileMoveOrCopy("/a", "/a"))
}

func TestFileAlreadyExists(t *testing.T) {
	// monkey patching
	defer gomonkey.ApplyFunc(os.Stat, func() (fs.FileInfo, error) {
		return nil, nil
	}).Reset()

	// testing
	assert.Error(t, FileMoveOrCopy("/a", "/a"))
}

func TestFileCopyReadFailure(t *testing.T) {
	// monkey patching
	defer gomonkey.NewPatches().
		ApplyFunc(os.Rename, func() error {
			return errors.New("not renaming")
		}).
		ApplyFunc(os.ReadFile, func() ([]byte, error) {
			return nil, errors.New("ko")
		}).
		Reset()

	// testing
	assert.EqualError(t, FileMoveOrCopy("/a", "/a"), "ko")
}

func TestFileCopyWriteFailure(t *testing.T) {
	// monkey patching
	defer gomonkey.NewPatches().
		ApplyFunc(os.Rename, func() error {
			return errors.New("not renaming")
		}).
		ApplyFunc(os.ReadFile, func() ([]byte, error) {
			return []byte{}, nil
		}).
		ApplyFunc(os.WriteFile, func() error {
			return errors.New("ko")
		}).
		Reset()

	// testing
	assert.EqualError(t, FileMoveOrCopy("/a", "/a"), "ko")
}

func TestFileBaseStem(t *testing.T) {
	assert.Equal(t, "hello", FileBaseStem("hello.txt"))
}

func TestCacheDirectory(t *testing.T) {
	// monkey patching
	defer gomonkey.ApplyFunc(xdg.CacheFile, func() (string, error) {
		return "/dir/spotitube", nil
	}).Reset()

	// testing
	assert.Equal(t, "/dir/spotitube", CacheDirectory())
}

func TestCacheDirectoryFallback(t *testing.T) {
	// monkey patching
	defer gomonkey.ApplyFunc(xdg.CacheFile, func() (string, error) {
		return "", errors.New("ko")
	}).Reset()

	// testing
	assert.Equal(t, "/tmp/spotitube", CacheDirectory())
}

func TestCacheFile(t *testing.T) {
	// monkey patching
	defer gomonkey.ApplyFunc(xdg.CacheFile, func() (string, error) {
		return "/dir/spotitube", nil
	}).Reset()

	// testing
	assert.Equal(t, "/dir/spotitube/fname.txt", CacheFile("fname.txt"))
}

func TestCacheFileFallback(t *testing.T) {
	// monkey patching
	defer gomonkey.ApplyFunc(xdg.CacheFile, func() (string, error) {
		return "", errors.New("ko")
	}).Reset()

	// testing
	assert.Equal(t, "/tmp/spotitube/fname.txt", CacheFile("fname.txt"))
}
