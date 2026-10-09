package cmd

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestOpen(t *testing.T) {
	dir := installFakeBinaries(t, map[string]string{
		"xdg-open": "printf '%s\\n' \"$*\" >> \"{{DIR}}/xdg-open.called\"\n",
		"open":     "printf '%s\\n' \"$*\" >> \"{{DIR}}/open.called\"\n",
		"rundll32": "printf '%s\\n' \"$*\" >> \"{{DIR}}/rundll32.called\"\n",
	})

	assert.Nil(t, Open("https://davidepucci.it", "linux"))
	assert.Nil(t, Open("https://davidepucci.it", "darwin"))
	assert.Nil(t, Open("https://davidepucci.it", "windows"))
	assert.EqualError(t, Open("https://davidepucci.it", "unknown"), "unsupported platform")

	// Start returns as soon as the opener is launched: wait for the fakes
	// to record their arguments before asserting on them
	assert.Equal(t, "https://davidepucci.it", captureContent(t, filepath.Join(dir, "xdg-open.called")))
	assert.Equal(t, "https://davidepucci.it", captureContent(t, filepath.Join(dir, "open.called")))
	assert.Equal(t, "url.dll,FileProtocolHandler https://davidepucci.it",
		captureContent(t, filepath.Join(dir, "rundll32.called")))
}

// captureContent polls for a capture file written by a fake opener and
// returns its trimmed content.
func captureContent(t *testing.T, path string) string {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if content, err := os.ReadFile(path); err == nil && len(content) > 0 {
			return string(content[:len(content)-1])
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("no invocation recorded in %s", path)
	return ""
}

func BenchmarkOpen(b *testing.B) {
	for i := 0; i < b.N; i++ {
		TestOpen(&testing.T{})
	}
}

func TestOpenStartFailure(t *testing.T) {
	// no opener anywhere on PATH: Start fails for real
	installEmptyPath(t)

	assert.ErrorContains(t, Open("https://davidepucci.it", "linux"), "executable file not found")
}
