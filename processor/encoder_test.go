package processor

import (
	"os"
	"testing"

	"github.com/streambinder/spotitube/entity/id3"
	"github.com/stretchr/testify/assert"
)

func BenchmarkEncoder(b *testing.B) {
	for i := 0; i < b.N; i++ {
		TestEncoderDo(&testing.T{})
	}
}

func TestEncoderDo(t *testing.T) {
	download := seedDownload(t, []byte{})

	assert.Nil(t, encoder{}.Do(track))

	// the written tag round-trips: the Spotify ID is readable back
	tag, err := id3.OpenSpotifyID(download)
	assert.Nil(t, err)
	defer tag.Close()
	assert.Equal(t, track.ID, tag.SpotifyID())
}

func TestEncoderDoUnsupported(t *testing.T) {
	assert.NotNil(t, encoder{}.Do("hello"))
}

func TestEncoderDoOpenFailure(t *testing.T) {
	useTempCache(t)

	assert.ErrorContains(t, encoder{}.Do(track), "no such file or directory")
}

func TestEncoderDoSaveFailure(t *testing.T) {
	download := seedDownload(t, []byte{})

	// saving writes a sibling temporary file first: block that path with
	// a directory, so the write fails even with write permissions
	if err := os.Mkdir(download+"-id3v2", 0o755); err != nil {
		t.Fatal(err)
	}

	assert.Error(t, encoder{}.Do(track))
}
