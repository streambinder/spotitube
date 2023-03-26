package spotify

import (
	"errors"
	"net/http"
	"reflect"
	"syscall"
	"testing"
	"time"

	"bou.ke/monkey"
	"github.com/streambinder/spotitube/util"
	"github.com/stretchr/testify/assert"
	"github.com/zmb3/spotify"
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

func TestAlbum(t *testing.T) {
	// monkey patching
	monkey.Patch(time.Sleep, func(time.Duration) {})
	defer monkey.Unpatch(time.Sleep)
	monkey.PatchInstanceMethod(reflect.TypeOf(&spotify.Client{}), "GetAlbumOpt",
		func(*spotify.Client, spotify.ID, *spotify.Options) (*spotify.FullAlbum, error) {
			return fullAlbum, nil
		})
	defer monkey.UnpatchInstanceMethod(reflect.TypeOf(&spotify.Client{}), "GetAlbumOpt")

	// testing
	channel := make(chan interface{}, 1)
	defer close(channel)
	album, err := (&Client{}).Album(fullAlbum.ID.String(), channel)
	assert.Nil(t, err)
	assert.Equal(t, fullAlbum.ID.String(), album.ID)
	assert.Equal(t, fullAlbum.Name, album.Name)
	assert.Equal(t, len(fullAlbum.Artists), len(album.Artists))
	assert.Equal(t, len(fullAlbum.Tracks.Tracks), len(album.Tracks))
	assert.Equal(t, album.Tracks[0], <-channel)
}

func TestPlaylistGetAlbumFailure(t *testing.T) {
	// monkey patching
	monkey.Patch(time.Sleep, func(time.Duration) {})
	defer monkey.Unpatch(time.Sleep)
	monkey.PatchInstanceMethod(reflect.TypeOf(&spotify.Client{}), "GetAlbumOpt",
		func(*spotify.Client, spotify.ID, *spotify.Options) (*spotify.FullAlbum, error) {
			return nil, errors.New("failure")
		})
	defer monkey.UnpatchInstanceMethod(reflect.TypeOf(&spotify.Client{}), "GetAlbumOpt")

	// testing
	assert.EqualError(t, util.ErrOnly((&Client{}).Album(fullPlaylist.ID.String())), "failure")
}

func TestAlbumNextPageFailure(t *testing.T) {
	var (
		client = (&Client{spotify.NewClient(http.DefaultClient), spotify.NewAuthenticator(""), ""})
		album  = fullAlbum
	)
	album.Tracks.Next = "http://0.0.0.0"

	// monkey patching
	monkey.Patch(time.Sleep, func(time.Duration) {})
	defer monkey.Unpatch(time.Sleep)
	monkey.PatchInstanceMethod(reflect.TypeOf(&spotify.Client{}), "GetAlbumOpt",
		func(*spotify.Client, spotify.ID, *spotify.Options) (*spotify.FullAlbum, error) {
			return album, nil
		})
	defer monkey.UnpatchInstanceMethod(reflect.TypeOf(&spotify.Client{}), "GetAlbumOpt")

	// testing
	assert.True(t, errors.Is(util.ErrOnly(client.Album(fullAlbum.ID.String())), syscall.ECONNREFUSED))
}
