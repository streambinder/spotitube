package processor

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/adrg/xdg"
)

// useTempCache points the XDG cache home — from which the track download
// path derives — at a fresh temporary directory, and returns the shared
// track's download path inside it. The xdg package resolves its base
// directories at init time, so it is reloaded after setting the
// environment, and reloaded back once the test is over.
func useTempCache(t *testing.T) string {
	t.Helper()
	t.Cleanup(xdg.Reload)
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	xdg.Reload()

	download := track.Path().Download()
	if err := os.MkdirAll(filepath.Dir(download), 0o755); err != nil {
		t.Fatal(err)
	}
	return download
}

// seedDownload creates the shared track's downloaded file holding the
// given content, inside a temporary cache home.
func seedDownload(t *testing.T, content []byte) string {
	t.Helper()
	download := useTempCache(t)
	if err := os.WriteFile(download, content, 0o644); err != nil {
		t.Fatal(err)
	}
	return download
}

// fakeFFmpeg installs a fake ffmpeg executable at the front of PATH. The
// loudness detection invocation (carrying print_format=json) reports a
// measurement with the given integrated loudness on stderr and exits with
// detectCode; the normalization invocation fails with normalizeCode when
// non-zero (printing "ko" on stderr), and otherwise materializes the file
// named by its -y argument, like the real tool would.
func fakeFFmpeg(t *testing.T, integrated string, detectCode, normalizeCode int) {
	t.Helper()
	measurement := fmt.Sprintf(
		`{"input_i": %q, "input_tp": "-1.50", "input_lra": "11.00", "input_thresh": "-34.00", "target_offset": "0.50"}`,
		integrated,
	)
	script := fmt.Sprintf(`#!/bin/sh
for arg in "$@"; do
	case "$arg" in
	*print_format=json*)
		echo '%s' >&2
		exit %d
		;;
	esac
done
if [ %d -ne 0 ]; then
	echo 'ko' >&2
	exit %d
fi
previous=""
for arg in "$@"; do
	if [ "$previous" = "-y" ]; then
		printf 'encoded audio' > "$arg"
	fi
	previous="$arg"
done
exit 0
`, measurement, detectCode, normalizeCode, normalizeCode)

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "ffmpeg"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}
