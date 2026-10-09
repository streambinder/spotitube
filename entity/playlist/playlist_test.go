package playlist

import (
	"testing"

	"github.com/streambinder/spotitube/entity"
	"github.com/streambinder/spotitube/sys"
	"github.com/stretchr/testify/assert"
)

var (
	testTrack = &entity.Track{
		Title:   "Title",
		Artists: []string{"Artist"},
	}
	testPlaylist = &Playlist{
		Name:   "Playlist",
		Tracks: []*entity.Track{testTrack},
	}
)

func BenchmarkPlaylist(b *testing.B) {
	for i := 0; i < b.N; i++ {
		TestEncoderM3U(&testing.T{})
		TestEncoderPLS(&testing.T{})
	}
}

func TestEncoderM3U(t *testing.T) {
	assert.Nil(t, sys.ErrOnly(testPlaylist.Encoder("m3u")))
}

func TestEncoderPLS(t *testing.T) {
	assert.Nil(t, sys.ErrOnly(testPlaylist.Encoder("pls")))
}

func TestEncoderUnknown(t *testing.T) {
	assert.Error(t, sys.ErrOnly(testPlaylist.Encoder("wut")), "unsupported encoding")
}
