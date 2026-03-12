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

var fullPlaylist = &spotify.FullPlaylist{
	SimplePlaylist: spotify.SimplePlaylist{
		ID:    spotify.ID("123"),
		Name:  "Playlist",
		Owner: spotify.User{ID: "User"},
	},
	Tracks: spotify.PlaylistTrackPage{
		Tracks: []spotify.PlaylistTrack{
			{Track: fullTrack},
		},
	},
}

func BenchmarkPlaylist(b *testing.B) {
	for i := 0; i < b.N; i++ {
		TestPlaylist(&testing.T{})
	}
}

func TestPlaylist(t *testing.T) {
	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(mockey.GetMethod(&spotify.Client{}, "CurrentUsersPlaylists")).Return(&spotify.SimplePlaylistPage{
		Playlists: []spotify.SimplePlaylist{{ID: "123", Name: "Playlist"}},
	}, nil).Build()
	mockey.Mock(mockey.GetMethod(&spotify.Client{}, "GetPlaylist")).Return(fullPlaylist, nil).Build()

	// testing
	client := testClient()
	_, err := client.Playlist(fullPlaylist.Name)
	assert.Nil(t, err)
	playlist, err := client.Playlist(fullPlaylist.Name)
	assert.Nil(t, err)
	assert.Equal(t, fullPlaylist.ID.String(), playlist.ID)
	assert.Equal(t, fullPlaylist.Name, playlist.Name)
	assert.Equal(t, fullPlaylist.Owner.ID, playlist.Owner)
	assert.Equal(t, len(fullPlaylist.Tracks.Tracks), len(playlist.Tracks))
	assert.Equal(t, fullPlaylist.Tracks.Tracks[0].Track.ID.String(), playlist.Tracks[0].ID)
	assert.Equal(t, fullPlaylist.Tracks.Tracks[0].Track.Name, playlist.Tracks[0].Title)
}

func TestPlaylistChannel(t *testing.T) {
	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(mockey.GetMethod(&spotify.Client{}, "CurrentUsersPlaylists")).Return(&spotify.SimplePlaylistPage{}, nil).Build()
	mockey.Mock(mockey.GetMethod(&spotify.Client{}, "GetPlaylist")).Return(fullPlaylist, nil).Build()

	// testing
	channel := make(chan interface{}, 1)
	defer close(channel)
	playlist, err := testClient().Playlist(fullPlaylist.ID.String(), channel)
	assert.Nil(t, err)
	assert.Equal(t, playlist.Tracks[0], <-channel)
}

func TestPlaylistCurrentUsersPlaylistsFailure(t *testing.T) {
	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(mockey.GetMethod(&spotify.Client{}, "CurrentUsersPlaylists")).Return(nil, errors.New("ko")).Build()

	// testing
	assert.Error(t, sys.ErrOnly(testClient().Playlist(fullPlaylist.ID.String())))
}

func TestPlaylistCurrentUsersPlaylistsNextPageFailure(t *testing.T) {
	var (
		client       = testClient()
		playlistPage = &spotify.SimplePlaylistPage{}
	)
	playlistPage.Next = "http://0.0.0.0"

	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(mockey.GetMethod(&spotify.Client{}, "CurrentUsersPlaylists")).Return(playlistPage, nil).Build()

	// testing
	assert.True(t, errors.Is(sys.ErrOnly(client.Playlist(fullPlaylist.ID.String())), syscall.ECONNREFUSED))
}

func TestPlaylistGetPlaylistFailure(t *testing.T) {
	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(mockey.GetMethod(&spotify.Client{}, "CurrentUsersPlaylists")).Return(&spotify.SimplePlaylistPage{}, nil).Build()
	mockey.Mock(mockey.GetMethod(&spotify.Client{}, "GetPlaylist")).Return(nil, errors.New("ko")).Build()

	// testing
	assert.EqualError(t, sys.ErrOnly(testClient().Playlist(fullPlaylist.ID.String())), "ko")
}

func TestPlaylistGetPlaylistNextPageFailure(t *testing.T) {
	client := testClient()
	// shallow copy to avoid mutating package-level fixture
	playlistCopy := *fullPlaylist
	playlistCopy.Tracks.Next = "http://0.0.0.0"
	defer func() { playlistCopy.Tracks.Next = "" }()

	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(mockey.GetMethod(&spotify.Client{}, "CurrentUsersPlaylists")).Return(&spotify.SimplePlaylistPage{}, nil).Build()
	mockey.Mock(mockey.GetMethod(&spotify.Client{}, "GetPlaylist")).Return(&playlistCopy, nil).Build()

	// testing
	assert.True(t, errors.Is(sys.ErrOnly(client.Playlist(fullPlaylist.ID.String())), syscall.ECONNREFUSED))
}
