package playlist

import (
	"errors"
	"testing"

	"github.com/bytedance/mockey"
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

func TestEncoderInitFailure(t *testing.T) {
	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(mockey.GetMethod(&M3UEncoder{}, "init")).Return(errors.New("ko")).Build()

	// testing
	assert.EqualError(t, sys.ErrOnly(testPlaylist.Encoder("m3u")), "ko")
}

func TestEncoderUnknown(t *testing.T) {
	assert.Error(t, sys.ErrOnly(testPlaylist.Encoder("wut")), "unsupported encoding")
}
