package downloader

import (
	"context"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

// fakeYouTubeDl installs a fake yt-dlp binary on PATH. It creates the
// output file the real tool would produce from the --output template,
// or fails when fail is set.
func fakeYouTubeDl(t *testing.T, fail bool) {
	t.Helper()
	dir := t.TempDir()
	script := `#!/bin/sh
out=""
fmt=""
prev=""
for arg in "$@"; do
  if [ "$prev" = "--output" ]; then out="$arg"; fi
  if [ "$prev" = "--audio-format" ]; then fmt="$arg"; fi
  prev="$arg"
done
`
	if fail {
		script += "echo \"yt-dlp blew up\" >&2\nexit 1\n"
	} else {
		script += "path=$(printf '%s' \"$out\" | sed \"s/%(ext)s/$fmt/\")\nprintf 'audio' > \"$path\"\nexit 0\n"
	}
	assert.Nil(t, os.WriteFile(filepath.Join(dir, "yt-dlp"), []byte(script), 0o755))
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

// unreachableTransport makes every blob support probe fail, so the
// download dispatch moves past the blob downloader.
func unreachableTransport(t *testing.T) {
	t.Helper()
	stubTransport(t, func(*http.Request) (*http.Response, error) {
		return nil, errors.New("ko")
	})
}

func BenchmarkDownload(b *testing.B) {
	for i := 0; i < b.N; i++ {
		TestDownload(&testing.T{})
	}
}

func TestDownload(t *testing.T) {
	unreachableTransport(t)
	fakeYouTubeDl(t, false)

	// testing: a missing target file is downloaded via yt-dlp
	path := filepath.Join(t.TempDir(), "fname.txt")
	ch := make(chan []byte, 1)
	defer close(ch)
	assert.Nil(t, Download(context.TODO(), "http://youtu.be", path, nil, ch))
}

func TestDownloadEmpty(t *testing.T) {
	assert.Nil(t, Download(context.TODO(), "", "fname.txt", nil))
}

func TestDownloadAlreadyExists(t *testing.T) {
	// testing: an existing target file is served from the cache and
	// its content is fed to the channels
	path := filepath.Join(t.TempDir(), "fname.txt")
	assert.Nil(t, os.WriteFile(path, []byte("cached"), 0o644))
	ch := make(chan []byte, 1)
	defer close(ch)
	assert.Nil(t, Download(context.TODO(), "http://youtu.be", path, nil, ch))
	assert.Equal(t, []byte("cached"), <-ch)
}

func TestDownloadMakeDirFailure(t *testing.T) {
	unreachableTransport(t)

	// the target's parent path is an existing file: MkdirAll fails
	blocker := filepath.Join(t.TempDir(), "blocker")
	assert.Nil(t, os.WriteFile(blocker, []byte{}, 0o644))
	path := filepath.Join(blocker, "sub", "fname.txt")

	// testing
	assert.NotNil(t, Download(context.TODO(), "http://youtu.be", path, nil))
}

func TestDownloadUnsupported(t *testing.T) {
	// the blob probe answers a plain text page: no downloader supports
	// the URL
	stubTransport(t, func(*http.Request) (*http.Response, error) {
		return blobResponse(200, "text/plain", ""), nil
	})

	// testing
	path := filepath.Join(t.TempDir(), "fname.txt")
	assert.ErrorContains(t, Download(context.TODO(), "http://davidepucci.it", path, nil), "unsupported url")
}

func TestDownloadYouTubeDlFailure(t *testing.T) {
	unreachableTransport(t)
	fakeYouTubeDl(t, true)

	// testing
	path := filepath.Join(t.TempDir(), "fname.txt")
	assert.ErrorContains(t, Download(context.TODO(), "http://youtu.be", path, nil), "yt-dlp blew up")
}

func TestDownloadEmptyCacheFile(t *testing.T) {
	unreachableTransport(t)
	fakeYouTubeDl(t, false)

	// testing: an empty cache file is dropped and downloaded again
	path := filepath.Join(t.TempDir(), "cached.mp3")
	assert.Nil(t, os.WriteFile(path, []byte{}, 0o600))
	ch := make(chan []byte, 1)
	defer close(ch)
	assert.Nil(t, Download(context.TODO(), "http://youtu.be", path, nil, ch))
}
