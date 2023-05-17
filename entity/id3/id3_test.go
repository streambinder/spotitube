package id3

import (
	"errors"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/bogem/id3v2/v2"
	"github.com/streambinder/spotitube/util"
	"github.com/stretchr/testify/assert"
)

func BenchmarkID3(b *testing.B) {
	for i := 0; i < b.N; i++ {
		TestOpen(&testing.T{})
	}
}

func TestOpen(t *testing.T) {
	// monkey patching
	defer gomonkey.ApplyFunc(id3v2.Open, func() (*id3v2.Tag, error) {
		return id3v2.NewEmptyTag(), nil
	}).Reset()

	// testing
	tag, err := Open("", id3v2.Options{})
	assert.Nil(t, err)
	assert.NotNil(t, tag)

	mimeType, image := tag.AttachedPicture()
	assert.Empty(t, mimeType)
	assert.Empty(t, image)
	assert.Empty(t, tag.UnsynchronizedLyrics())

	tag.SetAttachedPicture([]byte("picture"))
	tag.SetUnsynchronizedLyrics("title", "lyrics")
	tag.SetTrackNumber("1")
	tag.SetSpotifyID("Spotify ID")
	tag.SetArtworkURL("Artwork URL")
	tag.SetDuration("60")
	tag.SetUpstreamURL("Upstream URL")

	mimeType, image = tag.AttachedPicture()
	assert.Equal(t, "image/jpeg", mimeType)
	assert.Equal(t, []byte("picture"), image)
	assert.Equal(t, "lyrics", tag.UnsynchronizedLyrics())
	assert.Equal(t, "1", tag.TrackNumber())
	assert.Equal(t, "Spotify ID", tag.SpotifyID())
	assert.Equal(t, "Artwork URL", tag.ArtworkURL())
	assert.Equal(t, "60", tag.Duration())
	assert.Equal(t, "Upstream URL", tag.UpstreamURL())
	assert.Equal(t, "Upstream URL", tag.UpstreamURL()) // served from cache
	assert.Equal(t, "", tag.userDefinedText("not existing"))
}

func TestOpenFailure(t *testing.T) {
	// monkey patching
	defer gomonkey.ApplyFunc(id3v2.Open, func() (*id3v2.Tag, error) {
		return nil, errors.New("ko")
	}).Reset()

	// testing
	assert.Error(t, util.ErrOnly(Open("", id3v2.Options{})), "ko")
}
