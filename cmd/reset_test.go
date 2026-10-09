package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/adrg/xdg"
	"github.com/streambinder/spotitube/spotify"
	"github.com/streambinder/spotitube/sys"
	"github.com/stretchr/testify/assert"
)

// withCacheHome points the XDG cache home at a temp directory, so
// sys.CacheDirectory resolves inside it, and returns the cache directory.
// The cleanup order matters: the environment is restored first, then the
// xdg state is reloaded from it.
func withCacheHome(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Cleanup(func() { xdg.Reload() })
	t.Setenv("XDG_CACHE_HOME", dir)
	xdg.Reload()
	cacheDir := sys.CacheDirectory()
	assert.Nil(t, os.MkdirAll(cacheDir, 0o755))
	return cacheDir
}

func BenchmarkReset(b *testing.B) {
	for i := 0; i < b.N; i++ {
		TestCmdReset(&testing.T{})
	}
}

func TestCmdReset(t *testing.T) {
	cacheDir := withCacheHome(t)
	assert.Nil(t, os.WriteFile(filepath.Join(cacheDir, spotify.TokenBasename), []byte("token"), 0o644))
	assert.Nil(t, os.WriteFile(filepath.Join(cacheDir, "fname.txt"), []byte("data"), 0o644))
	subdir := filepath.Join(cacheDir, "subdir")
	assert.Nil(t, os.MkdirAll(subdir, 0o755))
	assert.Nil(t, os.WriteFile(filepath.Join(subdir, "nested.txt"), []byte("data"), 0o644))

	// testing: files and directories are removed, the session is kept
	assert.Nil(t, sys.ErrOnly(testExecute(cmdReset())))
	_, err := os.Stat(filepath.Join(cacheDir, "fname.txt"))
	assert.True(t, os.IsNotExist(err))
	_, err = os.Stat(subdir)
	assert.True(t, os.IsNotExist(err))
	_, err = os.Stat(filepath.Join(cacheDir, spotify.TokenBasename))
	assert.Nil(t, err)
}

func TestCmdResetDirectory(t *testing.T) {
	cacheDir := withCacheHome(t)
	subdir := filepath.Join(cacheDir, "subdir")
	assert.Nil(t, os.MkdirAll(subdir, 0o755))
	assert.Nil(t, os.WriteFile(filepath.Join(subdir, "nested.txt"), []byte("data"), 0o644))

	// testing: a directory entry is removed as a whole
	assert.Nil(t, sys.ErrOnly(testExecute(cmdReset())))
	_, err := os.Stat(subdir)
	assert.True(t, os.IsNotExist(err))
}

func TestCmdResetOpenRootFailure(t *testing.T) {
	cacheDir := withCacheHome(t)
	// the cache path exists as a regular file: it cannot be opened as
	// a root directory
	assert.Nil(t, os.RemoveAll(cacheDir))
	assert.Nil(t, os.WriteFile(cacheDir, []byte("not a directory"), 0o644))

	// testing
	assert.NotNil(t, sys.ErrOnly(testExecute(cmdReset())))
}

func TestCmdResetSession(t *testing.T) {
	cacheDir := withCacheHome(t)
	assert.Nil(t, os.WriteFile(filepath.Join(cacheDir, spotify.TokenBasename), []byte("token"), 0o644))

	// testing: with --session the token file is removed as well
	assert.Nil(t, sys.ErrOnly(testExecute(cmdReset(), "--session")))
	_, err := os.Stat(filepath.Join(cacheDir, spotify.TokenBasename))
	assert.True(t, os.IsNotExist(err))
}
