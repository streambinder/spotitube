package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// ffmpegName and ytdlpName are the bare names production code invokes
// its tools by.
const (
	ffmpegName = "ffmpeg"
	ytdlpName  = "yt-dlp"
)

// installFakeBinaries writes the given shell scripts as executables into a
// fresh temporary directory and points PATH at that directory alone, so the
// production code - which invokes its tools by bare name - picks the fakes
// up and finds nothing else. In script bodies, {{DIR}} expands to the
// directory itself, so scripts can drop capture files next to themselves.
// Scripts must stick to shell builtins or absolute paths: PATH contains
// nothing but the fake directory while they run.
func installFakeBinaries(t *testing.T, scripts map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, body := range scripts {
		script := "#!/bin/sh\n" + strings.ReplaceAll(body, "{{DIR}}", dir)
		assert.Nil(t, os.WriteFile(filepath.Join(dir, name), []byte(script), 0o755))
	}
	t.Setenv("PATH", dir)
	return dir
}

// installEmptyPath points PATH at a fresh directory holding no executable
// at all, so every tool lookup and launch fails.
func installEmptyPath(t *testing.T) string {
	t.Helper()
	return installFakeBinaries(t, nil)
}

// printfScript returns a script body that prints the given text to stdout
// and exits with the given code. The text must not contain single quotes.
func printfScript(text string, code int) string {
	return fmt.Sprintf("printf '%%s' '%s'\nexit %d\n", text, code)
}
