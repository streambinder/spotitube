package spotify

import (
	"errors"
	"syscall"
	"testing"

	"github.com/bytedance/mockey"
	"github.com/streambinder/spotitube/sys"
	"github.com/stretchr/testify/assert"
	"github.com/zmb3/spotify/v2"
)

var fullAlbum = &spotify.FullAlbum{
	SimpleAlbum: spotify.SimpleAlbum{
		Name:    "Album",
		ID:      "123",
		Artists: []spotify.SimpleArtist{{Name: "Artist"}},
	},
	Tracks: spotify.SimpleTrackPage{
		Tracks: []spotify.SimpleTrack{
			fullTrack.SimpleTrack,
		},
	},
}

func BenchmarkAlbum(b *testing.B) {
	for i := 0; i < b.N; i++ {
		TestAlbum(&testing.T{})
	}
}

func TestAlbum(t *testing.T) {
	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(mockey.GetMethod(&spotify.Client{}, "GetAlbum")).Return(fullAlbum, nil).Build()

	// testing
	album, err := testClient().Album(fullAlbum.ID.String())
	assert.Nil(t, err)
	assert.Equal(t, fullAlbum.ID.String(), album.ID)
	assert.Equal(t, fullAlbum.Name, album.Name)
	assert.Equal(t, len(fullAlbum.Artists), len(album.Artists))
	assert.Equal(t, len(fullAlbum.Tracks.Tracks), len(album.Tracks))
}

func TestAlbumChannel(t *testing.T) {
	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(mockey.GetMethod(&spotify.Client{}, "GetAlbum")).Return(fullAlbum, nil).Build()

	// testing
	channel := make(chan interface{}, 1)
	defer close(channel)
	album, err := testClient().Album(fullAlbum.ID.String(), channel)
	assert.Nil(t, err)
	assert.Equal(t, album.Tracks[0], <-channel)
}

func TestAlbumGetAlbumFailure(t *testing.T) {
	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(mockey.GetMethod(&spotify.Client{}, "GetAlbum")).Return(nil, errors.New("ko")).Build()

	// testing
	assert.EqualError(t, sys.ErrOnly(testClient().Album(fullPlaylist.ID.String())), "ko")
}

func TestAlbumNextPageFailure(t *testing.T) {
	client := testClient()
	// shallow copy to avoid mutating package-level fixture
	albumCopy := *fullAlbum
	albumCopy.Tracks.Next = "http://0.0.0.0"
	defer func() { albumCopy.Tracks.Next = "" }()

	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(mockey.GetMethod(&spotify.Client{}, "GetAlbum")).Return(&albumCopy, nil).Build()

	// testing
	assert.True(t, errors.Is(sys.ErrOnly(client.Album(fullAlbum.ID.String())), syscall.ECONNREFUSED))
}
