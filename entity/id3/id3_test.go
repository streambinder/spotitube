package id3

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bogem/id3v2/v2"
	"github.com/stretchr/testify/assert"
)

func BenchmarkID3(b *testing.B) {
	for i := 0; i < b.N; i++ {
		TestOpen(&testing.T{})
	}
}

// writeEmptyFile creates an empty file in a temporary directory and
// returns its path
func writeEmptyFile(t *testing.T, name string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	assert.Nil(t, os.WriteFile(path, []byte{}, 0o600))
	return path
}

// writeTaggedFile creates a file holding a real ID3v2 tag with the
// given user-defined text frames and returns its path
func writeTaggedFile(t *testing.T, name string, frames map[string]string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)

	tag := id3v2.NewEmptyTag()
	for key, value := range frames {
		tag.AddUserDefinedTextFrame(id3v2.UserDefinedTextFrame{
			Encoding:    tag.DefaultEncoding(),
			Description: key,
			Value:       value,
		})
	}

	file, err := os.Create(path)
	assert.Nil(t, err)
	_, err = tag.WriteTo(file)
	assert.Nil(t, err)
	assert.Nil(t, file.Close())
	return path
}

func TestOpen(t *testing.T) {
	// testing
	tag, err := Open(writeEmptyFile(t, "track.mp3"), id3v2.Options{})
	assert.Nil(t, err)
	assert.NotNil(t, tag)
	defer func() { assert.Nil(t, tag.Close()) }()

	mimeType, image := tag.AttachedPicture()
	assert.Empty(t, mimeType)
	assert.Empty(t, image)
	assert.Empty(t, tag.UnsynchronizedLyrics())

	tag.SetAttachedPicture([]byte("picture"))
	tag.SetLyrics("title", "lyrics")
	tag.SetLyrics("title", "[01:01.15]lyrics")
	tag.SetTrackNumber("1")
	tag.SetSpotifyID("Spotify ID")
	tag.SetArtworkURL("Artwork URL")
	tag.SetDuration("60")
	tag.SetUpstreamURL("Upstream URL")

	mimeType, image = tag.AttachedPicture()
	assert.Equal(t, "image/jpeg", mimeType)
	assert.Equal(t, []byte("picture"), image)
	assert.Equal(t, "[01:01.15]lyrics", tag.UnsynchronizedLyrics())
	assert.Equal(t, "1", tag.TrackNumber())
	assert.Equal(t, "Spotify ID", tag.SpotifyID())
	assert.Equal(t, "Artwork URL", tag.ArtworkURL())
	assert.Equal(t, "60", tag.Duration())
	assert.Equal(t, "Upstream URL", tag.UpstreamURL())
	assert.Equal(t, "Upstream URL", tag.UpstreamURL()) // served from cache
	assert.Equal(t, "", tag.userDefinedText("not existing"))
}

func TestOpenSpotifyID(t *testing.T) {
	// testing: only the user-defined text frame is parsed, any other
	// frame in the file is left out by the restricted parse options
	path := writeTaggedFile(t, "track.mp3", map[string]string{
		frameSpotifyID: "Spotify ID",
	})

	tag, err := OpenSpotifyID(path)
	assert.Nil(t, err)
	assert.NotNil(t, tag)
	defer func() { assert.Nil(t, tag.Close()) }()
	assert.Equal(t, "Spotify ID", tag.SpotifyID())
}

func TestUserDefinedTextInvalidFrame(t *testing.T) {
	// testing: a non-UserDefinedTextFrame stored under the user-defined
	// text frame ID is skipped while scanning the frames
	rawTag := id3v2.NewEmptyTag()
	rawTag.AddFrame(rawTag.CommonID(frameUserDefinedText), id3v2.TextFrame{
		Encoding: rawTag.DefaultEncoding(),
		Text:     "invalid",
	})
	tag := &Tag{*rawTag, make(map[string]string)}

	assert.Equal(t, "", tag.userDefinedText("nonexistent"))
}

func TestOpenFailure(t *testing.T) {
	// testing: opening a file that does not exist fails
	tag, err := Open(filepath.Join(t.TempDir(), "missing.mp3"), id3v2.Options{})
	assert.Nil(t, tag)
	assert.Error(t, err)
}

func TestClose(t *testing.T) {
	// testing
	tag, err := Open(writeEmptyFile(t, "track.mp3"), id3v2.Options{})
	assert.Nil(t, err)
	assert.Nil(t, tag.Close())
}

func TestCloseFailure(t *testing.T) {
	// testing: closing an already closed file-backed tag surfaces the
	// underlying file error
	tag, err := Open(writeEmptyFile(t, "track.mp3"), id3v2.Options{})
	assert.Nil(t, err)
	assert.Nil(t, tag.Close())
	assert.Error(t, tag.Close())
}

func TestCloseErrNoFile(t *testing.T) {
	// testing: closing a tag that was never initialized with a file is
	// not an error
	tag := &Tag{*id3v2.NewEmptyTag(), make(map[string]string)}
	assert.Nil(t, tag.Close())
}
