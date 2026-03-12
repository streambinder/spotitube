package sys

import (
	"errors"
	"os"
	"testing"

	"github.com/adrg/xdg"
	"github.com/bytedance/mockey"
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
	defer mockey.UnPatchAll()
	mockey.Mock(os.Rename).Return(errors.New("not renaming")).Build()
	mockey.Mock(os.ReadFile).Return([]byte{}, nil).Build()
	mockey.Mock(os.WriteFile).Return(nil).Build()
	mockey.Mock(os.Remove).Return(nil).Build()

	// testing
	assert.Nil(t, FileMoveOrCopy("/a", "/a"))
}

func TestFileAlreadyExists(t *testing.T) {
	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(os.Stat).Return(nil, nil).Build()

	// testing
	assert.Error(t, FileMoveOrCopy("/a", "/a"))
}

func TestFileAlreadyExistsOverwrite(t *testing.T) {
	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(os.Stat).Return(nil, nil).Build()
	mockey.Mock(os.Rename).Return(nil).Build()

	// testing
	assert.Nil(t, FileMoveOrCopy("/a", "/b", true))
}

func TestFileCopyRemoveFailure(t *testing.T) {
	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(os.Rename).Return(errors.New("not renaming")).Build()
	mockey.Mock(os.ReadFile).Return([]byte{}, nil).Build()
	mockey.Mock(os.WriteFile).Return(nil).Build()
	mockey.Mock(os.Remove).Return(errors.New("ko")).Build()

	// testing
	assert.EqualError(t, FileMoveOrCopy("/a", "/a"), "ko")
}

func TestFileCopyReadFailure(t *testing.T) {
	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(os.Rename).Return(errors.New("not renaming")).Build()
	mockey.Mock(os.ReadFile).Return(nil, errors.New("ko")).Build()

	// testing
	assert.EqualError(t, FileMoveOrCopy("/a", "/a"), "ko")
}

func TestFileCopyWriteFailure(t *testing.T) {
	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(os.Rename).Return(errors.New("not renaming")).Build()
	mockey.Mock(os.ReadFile).Return([]byte{}, nil).Build()
	mockey.Mock(os.WriteFile).Return(errors.New("ko")).Build()

	// testing
	assert.EqualError(t, FileMoveOrCopy("/a", "/a"), "ko")
}

func TestFileBaseStem(t *testing.T) {
	assert.Equal(t, "hello", FileBaseStem("hello.txt"))
}

func TestCacheDirectory(t *testing.T) {
	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(xdg.CacheFile).Return("/dir/spotitube", nil).Build()

	// testing
	assert.Equal(t, "/dir/spotitube", CacheDirectory())
}

func TestCacheDirectoryFallback(t *testing.T) {
	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(xdg.CacheFile).Return("", errors.New("ko")).Build()

	// testing
	assert.Equal(t, "/tmp/spotitube", CacheDirectory())
}

func TestCacheFile(t *testing.T) {
	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(xdg.CacheFile).Return("/dir/spotitube", nil).Build()

	// testing
	assert.Equal(t, "/dir/spotitube/fname.txt", CacheFile("fname.txt"))
}

func TestCacheFileFallback(t *testing.T) {
	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(xdg.CacheFile).Return("", errors.New("ko")).Build()

	// testing
	assert.Equal(t, "/tmp/spotitube/fname.txt", CacheFile("fname.txt"))
}
