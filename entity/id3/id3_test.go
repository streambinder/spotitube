package id3

import (
	"errors"
	"testing"

	"github.com/bytedance/mockey"
	"github.com/streambinder/id3v2-sylt"
	"github.com/streambinder/spotitube/sys"
	"github.com/stretchr/testify/assert"
)

func BenchmarkID3(b *testing.B) {
	for i := 0; i < b.N; i++ {
		TestOpen(&testing.T{})
	}
}

func TestOpen(t *testing.T) {
	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(id3v2.Open).To(func(_ string, _ id3v2.Options) (*id3v2.Tag, error) {
		return id3v2.NewEmptyTag(), nil
	}).Build()

	// testing
	tag, err := Open("", id3v2.Options{})
	assert.Nil(t, err)
	assert.NotNil(t, tag)

	mimeType, image := tag.AttachedPicture()
	assert.Empty(t, mimeType)
	assert.Empty(t, image)
	assert.Empty(t, tag.UnsynchronizedLyrics())
	assert.Empty(t, tag.SynchronizedLyrics())

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
	assert.Equal(t, "[01:01.15]lyrics", tag.SynchronizedLyrics())
	assert.Equal(t, "1", tag.TrackNumber())
	assert.Equal(t, "Spotify ID", tag.SpotifyID())
	assert.Equal(t, "Artwork URL", tag.ArtworkURL())
	assert.Equal(t, "60", tag.Duration())
	assert.Equal(t, "Upstream URL", tag.UpstreamURL())
	assert.Equal(t, "Upstream URL", tag.UpstreamURL()) // served from cache
	assert.Equal(t, "", tag.userDefinedText("not existing"))
}

func TestUserDefinedTextInvalidFrame(t *testing.T) {
	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(id3v2.Open).To(func(_ string, _ id3v2.Options) (*id3v2.Tag, error) {
		tag := id3v2.NewEmptyTag()
		// add a non-UserDefinedTextFrame to the TXXX frame ID to trigger continue
		tag.AddFrame(tag.CommonID("User defined text information frame"), id3v2.TextFrame{
			Encoding: tag.DefaultEncoding(),
			Text:     "invalid",
		})
		return tag, nil
	}).Build()

	// testing
	tag, err := Open("", id3v2.Options{})
	assert.Nil(t, err)
	assert.Equal(t, "", tag.userDefinedText("nonexistent"))
}

func TestOpenFailure(t *testing.T) {
	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(id3v2.Open).Return(nil, errors.New("ko")).Build()

	// testing
	assert.EqualError(t, sys.ErrOnly(Open("", id3v2.Options{})), "ko")
}

func TestClose(t *testing.T) {
	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(Open).Return(&Tag{}, nil).Build()

	// testing
	tag, err := Open("", id3v2.Options{})
	assert.Nil(t, err)
	assert.Nil(t, tag.Close())
}

func TestCloseFailure(t *testing.T) {
	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(Open).Return(&Tag{}, nil).Build()
	mockey.Mock(mockey.GetMethod(&id3v2.Tag{}, "Close")).Return(errors.New("ko")).Build()

	// testing
	tag, err := Open("", id3v2.Options{})
	assert.Nil(t, err)
	assert.EqualError(t, tag.Close(), "ko")
}

func TestCloseErrNoFile(t *testing.T) {
	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(Open).Return(&Tag{}, nil).Build()
	mockey.Mock(mockey.GetMethod(&id3v2.Tag{}, "Close")).Return(id3v2.ErrNoFile).Build()

	// testing
	tag, err := Open("", id3v2.Options{})
	assert.Nil(t, err)
	assert.Nil(t, tag.Close())
}
