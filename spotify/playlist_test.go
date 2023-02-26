package spotify

import (
	"errors"
	"net/http"
	"reflect"
	"syscall"
	"testing"
	"time"

	"bou.ke/monkey"
	"github.com/streambinder/spotitube/entity"
	"github.com/streambinder/spotitube/util"
	"github.com/stretchr/testify/assert"
	"github.com/zmb3/spotify"
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

func init() {
	monkey.Patch(time.Sleep, func(time.Duration) {})
	monkey.PatchInstanceMethod(reflect.TypeOf(&spotify.Client{}), "GetPlaylistOpt",
		func(*spotify.Client, spotify.ID, string) (*spotify.FullPlaylist, error) {
			return fullPlaylist, nil
		})
}

func TestPlaylist(t *testing.T) {
	channel := make(chan *entity.Track, 1)
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
	monkey.PatchInstanceMethod(reflect.TypeOf(&spotify.Client{}), "GetPlaylistOpt",
		func(*spotify.Client, spotify.ID, string) (*spotify.FullPlaylist, error) {
			return nil, errors.New("failure")
		})
	assert.EqualError(t, util.ErrOnly((&Client{}).Playlist(fullPlaylist.ID.String())), "failure")
}

func TestPlaylistNextPageFailure(t *testing.T) {
	var (
		client   = (&Client{spotify.NewClient(http.DefaultClient), spotify.NewAuthenticator(""), ""})
		playlist = fullPlaylist
	)
	playlist.Tracks.Next = "http://0.0.0.0"
	monkey.PatchInstanceMethod(reflect.TypeOf(&spotify.Client{}), "GetPlaylistOpt",
		func(*spotify.Client, spotify.ID, string) (*spotify.FullPlaylist, error) {
			return playlist, nil
		})
	assert.True(t, errors.Is(util.ErrOnly(client.Playlist(fullPlaylist.ID.String())), syscall.ECONNREFUSED))
}
