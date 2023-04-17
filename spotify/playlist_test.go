package spotify

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"syscall"
	"testing"
	"time"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/streambinder/spotitube/util"
	"github.com/stretchr/testify/assert"
	"github.com/zmb3/spotify/v2"
	spotifyauth "github.com/zmb3/spotify/v2/auth"
)

var fullPlaylist = &spotify.FullPlaylist{
	SimplePlaylist: spotify.SimplePlaylist{
		ID:    spotify.ID("0000000000000000000000"),
		Name:  "Playlist",
		Owner: spotify.User{ID: "User"},
	},
	Tracks: spotify.PlaylistTrackPage{
		Tracks: []spotify.PlaylistTrack{
			{Track: fullTrack},
		},
	},
}

func TestPlaylist(t *testing.T) {
	// monkey patching
	patchtimeSleep := gomonkey.ApplyFunc(time.Sleep, func(time.Duration) {})
	defer patchtimeSleep.Reset()
	patchspotifyClientGetPlaylist := gomonkey.ApplyMethod(reflect.TypeOf(&spotify.Client{}), "GetPlaylist",
		func(*spotify.Client, context.Context, spotify.ID, ...spotify.RequestOption) (*spotify.FullPlaylist, error) {
			return fullPlaylist, nil
		})
	defer patchspotifyClientGetPlaylist.Reset()

	// testing
	channel := make(chan interface{}, 1)
	defer close(channel)
	playlist, err := (&Client{}).Playlist(fullPlaylist.ID.String(), channel)
	assert.Nil(t, err)
	assert.Equal(t, fullPlaylist.ID.String(), playlist.ID)
	assert.Equal(t, fullPlaylist.Name, playlist.Name)
	assert.Equal(t, fullPlaylist.Owner.ID, playlist.Owner)
	assert.Equal(t, len(fullPlaylist.Tracks.Tracks), len(playlist.Tracks))
	assert.Equal(t, fullPlaylist.Tracks.Tracks[0].Track.ID.String(), playlist.Tracks[0].ID)
	assert.Equal(t, fullPlaylist.Tracks.Tracks[0].Track.Name, playlist.Tracks[0].Title)
	assert.Equal(t, playlist.Tracks[0], <-channel)
}

func TestPlaylistGetPlaylistFailure(t *testing.T) {
	// monkey patching
	patchtimeSleep := gomonkey.ApplyFunc(time.Sleep, func(time.Duration) {})
	defer patchtimeSleep.Reset()
	patchspotifyClientGetPlaylist := gomonkey.ApplyMethod(reflect.TypeOf(&spotify.Client{}), "GetPlaylist",
		func(*spotify.Client, context.Context, spotify.ID, ...spotify.RequestOption) (*spotify.FullPlaylist, error) {
			return nil, errors.New("failure")
		})
	defer patchspotifyClientGetPlaylist.Reset()

	// testing
	assert.EqualError(t, util.ErrOnly((&Client{}).Playlist(fullPlaylist.ID.String())), "failure")
}

func TestPlaylistNextPageFailure(t *testing.T) {
	var (
		client   = (&Client{spotify.New(http.DefaultClient), &spotifyauth.Authenticator{}, ""})
		playlist = fullPlaylist
	)
	playlist.Tracks.Next = "http://0.0.0.0"

	// monkey patching
	patchtimeSleep := gomonkey.ApplyFunc(time.Sleep, func(time.Duration) {})
	defer patchtimeSleep.Reset()
	patchspotifyClientGetPlaylist := gomonkey.ApplyMethod(reflect.TypeOf(&spotify.Client{}), "GetPlaylist",
		func(*spotify.Client, context.Context, spotify.ID, ...spotify.RequestOption) (*spotify.FullPlaylist, error) {
			return playlist, nil
		})
	defer patchspotifyClientGetPlaylist.Reset()

	// testing
	assert.True(t, errors.Is(util.ErrOnly(client.Playlist(fullPlaylist.ID.String())), syscall.ECONNREFUSED))
}
