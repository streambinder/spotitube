package spotify

import (
	"errors"
	"syscall"
	"testing"
	"time"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/streambinder/spotitube/util"
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
	defer gomonkey.NewPatches().
		ApplyFunc(time.Sleep, func() {}).
		ApplyMethod(&spotify.Client{}, "GetAlbum", func() (*spotify.FullAlbum, error) {
			return fullAlbum, nil
		}).
		Reset()

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
	defer gomonkey.NewPatches().
		ApplyFunc(time.Sleep, func() {}).
		ApplyMethod(&spotify.Client{}, "GetAlbum", func() (*spotify.FullAlbum, error) {
			return fullAlbum, nil
		}).
		Reset()

	// testing
	channel := make(chan interface{}, 1)
	defer close(channel)
	album, err := testClient().Album(fullAlbum.ID.String(), channel)
	assert.Nil(t, err)
	assert.Equal(t, album.Tracks[0], <-channel)
}

func TestPlaylistGetAlbumFailure(t *testing.T) {
	// monkey patching
	defer gomonkey.NewPatches().
		ApplyFunc(time.Sleep, func() {}).
		ApplyMethod(&spotify.Client{}, "GetAlbum", func() (*spotify.FullAlbum, error) {
			return nil, errors.New("ko")
		}).
		Reset()

	// testing
	assert.EqualError(t, util.ErrOnly(testClient().Album(fullPlaylist.ID.String())), "ko")
}

func TestAlbumNextPageFailure(t *testing.T) {
	var (
		client = testClient()
		album  = fullAlbum
	)
	album.Tracks.Next = "http://0.0.0.0"

	// monkey patching
	defer gomonkey.NewPatches().
		ApplyFunc(time.Sleep, func() {}).
		ApplyMethod(&spotify.Client{}, "GetAlbum", func() (*spotify.FullAlbum, error) {
			return album, nil
		}).
		Reset()

	// testing
	assert.True(t, errors.Is(util.ErrOnly(client.Album(fullAlbum.ID.String())), syscall.ECONNREFUSED))
}
