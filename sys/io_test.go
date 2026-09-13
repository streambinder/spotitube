package sys

import (
	"errors"
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/adrg/xdg"
	"github.com/bytedance/mockey"
	"github.com/stretchr/testify/assert"
)

// renameFallback mocks os.Rename so that the first call fails
// (forcing the copy fallback) while later ones really rename
func renameFallback() {
	calls := 0
	mockey.Mock(os.Rename).To(func(oldpath, newpath string) error {
		calls++
		if calls == 1 {
			return errors.New("not renaming")
		}
		return syscall.Rename(oldpath, newpath)
	}).Build()
}

func tempSource(t *testing.T, content string) (dir, src, dst string) {
	t.Helper()
	dir = t.TempDir()
	src = filepath.Join(dir, "src.txt")
	dst = filepath.Join(dir, "dst.txt")
	assert.Nil(t, os.WriteFile(src, []byte(content), 0o600))
	return dir, src, dst
}

func dirEntries(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	assert.Nil(t, err)
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	return names
}

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
	renameFallback()

	// testing
	dir, src, dst := tempSource(t, "data")
	assert.Nil(t, FileMoveOrCopy(src, dst))
	content, err := os.ReadFile(dst)
	assert.Nil(t, err)
	assert.Equal(t, "data", string(content))
	assert.NotContains(t, dirEntries(t, dir), "src.txt")
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
	renameFallback()
	mockey.Mock(os.Remove).Return(errors.New("ko")).Build()

	// testing
	_, src, dst := tempSource(t, "data")
	assert.EqualError(t, FileMoveOrCopy(src, dst), "ko")
}

func TestFileCopyReadFailure(t *testing.T) {
	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(os.Rename).Return(errors.New("not renaming")).Build()
	mockey.Mock(os.ReadFile).Return(nil, errors.New("ko")).Build()

	// testing
	assert.EqualError(t, FileMoveOrCopy("/a", "/a"), "ko")
}

func TestFileCopyTempFailure(t *testing.T) {
	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(os.Rename).Return(errors.New("not renaming")).Build()
	mockey.Mock(os.CreateTemp).Return(nil, errors.New("ko")).Build()

	// testing
	_, src, dst := tempSource(t, "data")
	assert.EqualError(t, FileMoveOrCopy(src, dst), "ko")
}

func TestFileCopyWriteFailure(t *testing.T) {
	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(os.Rename).Return(errors.New("not renaming")).Build()

	// testing: a read-only handle makes the staged write fail
	dir, src, dst := tempSource(t, "data")
	readonly, err := os.OpenFile(filepath.Join(dir, "readonly.txt"), os.O_CREATE|os.O_RDONLY, 0o600)
	assert.Nil(t, err)
	mockey.Mock(os.CreateTemp).To(func(_, _ string) (*os.File, error) {
		return readonly, nil
	}).Build()

	assert.ErrorContains(t, FileMoveOrCopy(src, dst), "bad file descriptor")
	assert.NotContains(t, dirEntries(t, dir), "dst.txt")
	assert.Len(t, dirEntries(t, dir), 1) // only src.txt: no temp leftovers
}

func TestFileCopyRenameFailure(t *testing.T) {
	// testing
	dir, src, dst := tempSource(t, "data")

	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(os.Rename).To(func(oldpath, _ string) error {
		if oldpath == src {
			return errors.New("not renaming")
		}
		return errors.New("ko")
	}).Build()

	assert.EqualError(t, FileMoveOrCopy(src, dst), "ko")
	assert.NotContains(t, dirEntries(t, dir), "dst.txt")
	assert.Len(t, dirEntries(t, dir), 1) // only src.txt: no temp leftovers
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
