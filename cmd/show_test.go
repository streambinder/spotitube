package cmd

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bogem/id3v2/v2"
	"github.com/streambinder/pango/entity/id3"
	"github.com/streambinder/pango/sys"
	"github.com/stretchr/testify/assert"
)

// captureStdout returns what fn writes to the standard output.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	reader, writer, err := os.Pipe()
	assert.Nil(t, err)
	previous := os.Stdout
	os.Stdout = writer
	t.Cleanup(func() { os.Stdout = previous })

	fn()

	assert.Nil(t, writer.Close())
	contents, err := io.ReadAll(reader)
	assert.Nil(t, err)
	return string(contents)
}

// writeShowTrack creates a real audio file carrying a full ID3 tag.
func writeShowTrack(t *testing.T, path string, picture bool) {
	t.Helper()
	assert.Nil(t, os.WriteFile(path, []byte{}, 0o644))

	tag, err := id3.Open(path, id3v2.Options{Parse: true})
	assert.Nil(t, err)
	tag.SetTitle("Title")
	tag.SetArtist("Artist")
	tag.SetAlbum("Album")
	tag.SetYear("1970")
	tag.SetTrackNumber("1")
	tag.SetSpotifyID("123")
	tag.SetDuration("60")
	if picture {
		tag.SetAttachedPicture([]byte("some picture data"))
	}
	assert.Nil(t, tag.Save())
	assert.Nil(t, tag.Close())
}

func BenchmarkShow(b *testing.B) {
	for i := 0; i < b.N; i++ {
		TestCmdShow(&testing.T{})
	}
}

func TestCmdShow(t *testing.T) {
	dir := t.TempDir()
	first := filepath.Join(dir, "track1.mp3")
	second := filepath.Join(dir, "track2.mp3")
	writeShowTrack(t, first, true)
	writeShowTrack(t, second, true)

	// testing: two tracks exercise the separator between printouts too
	assert.Nil(t, sys.ErrOnly(testExecute(cmdShow(), first, second)))
}

func TestCmdShowOpenFailure(t *testing.T) {
	// testing: a missing file cannot be opened
	assert.NotNil(t, sys.ErrOnly(testExecute(cmdShow(), filepath.Join(t.TempDir(), "missing.mp3"))))
}

func TestCmdShowPictureFallback(t *testing.T) {
	path := filepath.Join(t.TempDir(), "track.mp3")
	writeShowTrack(t, path, false)

	// testing: a tag without a picture falls back to a text notice
	assert.Nil(t, sys.ErrOnly(testExecute(cmdShow(), path)))
}

func TestCmdShowLyricsLeadingBlankLines(t *testing.T) {
	path := filepath.Join(t.TempDir(), "track.mp3")
	writeShowTrack(t, path, false)

	tag, err := id3.Open(path, id3v2.Options{Parse: true})
	assert.Nil(t, err)
	tag.SetLyrics("Title", "\n\nfirst lyric line\nsecond lyric line")
	assert.Nil(t, tag.Save())
	assert.Nil(t, tag.Close())

	// testing: a stored frame that starts with blank lines reports
	// its first content line on the Lyrics row, not the fallback
	output := captureStdout(t, func() {
		assert.Nil(t, sys.ErrOnly(testExecute(cmdShow(), path)))
	})
	for _, line := range strings.Split(output, "\n") {
		// match the row by its label: the temporary path embedded in
		// the Path row contains the test name, hence "Lyrics" itself
		if strings.HasPrefix(strings.TrimSpace(line), "Lyrics") {
			assert.Contains(t, line, "first lyric line")
			assert.NotContains(t, line, fallback)
			return
		}
	}
	t.Fatal("no Lyrics row in the show output")
}
