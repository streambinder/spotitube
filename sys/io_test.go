package sys

import (
	"io/fs"
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/adrg/xdg"
	"github.com/stretchr/testify/assert"
)

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

// crossDeviceSource returns a source file living on a different filesystem
// than the destination directory, so that os.Rename between the two fails
// for real (EXDEV) and FileMoveOrCopy must fall back to copying. It skips
// the test when no second filesystem is available.
func crossDeviceSource(t *testing.T, content string) (src, dst string) {
	t.Helper()
	dstDir := t.TempDir()
	srcDir, err := os.MkdirTemp("/dev/shm", "spotitube-test-*")
	if err != nil {
		t.Skip("no second filesystem available for a cross-device rename")
	}
	t.Cleanup(func() { os.RemoveAll(srcDir) })

	srcInfo, err := os.Stat(srcDir)
	assert.Nil(t, err)
	dstInfo, err := os.Stat(dstDir)
	assert.Nil(t, err)
	if srcInfo.Sys().(*syscall.Stat_t).Dev == dstInfo.Sys().(*syscall.Stat_t).Dev {
		t.Skip("no second filesystem available for a cross-device rename")
	}
	src = filepath.Join(srcDir, "src.txt")
	dst = filepath.Join(dstDir, "dst.txt")
	assert.Nil(t, os.WriteFile(src, []byte(content), 0o600))
	return src, dst
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
	// the source sits on another filesystem: the rename fails for real
	// and the copy fallback has to take over
	src, dst := crossDeviceSource(t, "data")
	assert.Nil(t, FileMoveOrCopy(src, dst))
	content, err := os.ReadFile(dst)
	assert.Nil(t, err)
	assert.Equal(t, "data", string(content))
	_, err = os.Stat(src)
	assert.ErrorIs(t, err, fs.ErrNotExist)
	assert.Equal(t, []string{"dst.txt"}, dirEntries(t, filepath.Dir(dst)))
}

func TestFileAlreadyExists(t *testing.T) {
	_, src, dst := tempSource(t, "data")
	assert.Nil(t, os.WriteFile(dst, []byte("existing"), 0o600))

	assert.EqualError(t, FileMoveOrCopy(src, dst), "destination already exists: "+dst)
	content, err := os.ReadFile(dst)
	assert.Nil(t, err)
	assert.Equal(t, "existing", string(content))
}

func TestFileAlreadyExistsOverwrite(t *testing.T) {
	_, src, dst := tempSource(t, "data")
	assert.Nil(t, os.WriteFile(dst, []byte("existing"), 0o600))

	assert.Nil(t, FileMoveOrCopy(src, dst, true))
	content, err := os.ReadFile(dst)
	assert.Nil(t, err)
	assert.Equal(t, "data", string(content))
	_, err = os.Stat(src)
	assert.ErrorIs(t, err, fs.ErrNotExist)
}

func TestFileCopyReadFailure(t *testing.T) {
	// a missing source fails the rename first and the read right after
	dir := t.TempDir()
	err := FileMoveOrCopy(filepath.Join(dir, "missing.txt"), filepath.Join(dir, "dst.txt"))
	assert.ErrorIs(t, err, fs.ErrNotExist)
}

func TestFileCopyTempFailure(t *testing.T) {
	// the destination directory does not exist: staging the copy on a
	// temp file next to the destination fails for real
	_, src, _ := tempSource(t, "data")
	dst := filepath.Join(t.TempDir(), "missing", "dst.txt")

	err := FileMoveOrCopy(src, dst)
	assert.ErrorIs(t, err, fs.ErrNotExist)
	assert.Equal(t, []string{"src.txt"}, dirEntries(t, filepath.Dir(src)))
}

func TestFileCopyRenameFailure(t *testing.T) {
	// the destination is an existing directory: both the direct rename
	// and the rename of the staged copy fail for real
	dir, src, dst := tempSource(t, "data")
	assert.Nil(t, os.Mkdir(dst, 0o755))

	err := FileMoveOrCopy(src, dst, true)
	assert.ErrorContains(t, err, "rename")
	assert.ElementsMatch(t, []string{"src.txt", "dst.txt"}, dirEntries(t, dir))
	info, statErr := os.Stat(dst)
	assert.Nil(t, statErr)
	assert.True(t, info.IsDir())
}

func TestFileBaseStem(t *testing.T) {
	assert.Equal(t, "hello", FileBaseStem("hello.txt"))
}

// reloadXDG re-reads the XDG base directories from the environment and
// arranges for them to be re-read again from the ambient environment once
// the test is over. It must be called before t.Setenv.
func reloadXDG(t *testing.T) {
	t.Helper()
	t.Cleanup(func() { xdg.Reload() })
}

func TestCacheDirectory(t *testing.T) {
	reloadXDG(t)
	cacheHome := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", cacheHome)
	xdg.Reload()

	assert.Equal(t, filepath.Join(cacheHome, "spotitube"), CacheDirectory())
}

func TestCacheDirectoryFallback(t *testing.T) {
	reloadXDG(t)
	// a regular file in the way makes the cache home impossible to create
	dir := t.TempDir()
	blocker := filepath.Join(dir, "blocker")
	assert.Nil(t, os.WriteFile(blocker, []byte("file"), 0o600))
	t.Setenv("XDG_CACHE_HOME", filepath.Join(blocker, "sub"))
	xdg.Reload()

	assert.Equal(t, "/tmp/spotitube", CacheDirectory())
}

func TestCacheFile(t *testing.T) {
	reloadXDG(t)
	cacheHome := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", cacheHome)
	xdg.Reload()

	assert.Equal(t, filepath.Join(cacheHome, "spotitube", "fname.txt"), CacheFile("fname.txt"))
}

func TestCacheFileFallback(t *testing.T) {
	reloadXDG(t)
	dir := t.TempDir()
	blocker := filepath.Join(dir, "blocker")
	assert.Nil(t, os.WriteFile(blocker, []byte("file"), 0o600))
	t.Setenv("XDG_CACHE_HOME", filepath.Join(blocker, "sub"))
	xdg.Reload()

	assert.Equal(t, "/tmp/spotitube/fname.txt", CacheFile("fname.txt"))
}
