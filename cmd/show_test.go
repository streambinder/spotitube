package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bogem/id3v2/v2"
	"github.com/streambinder/spotitube/entity/id3"
	"github.com/streambinder/spotitube/sys"
	"github.com/stretchr/testify/assert"
)

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
