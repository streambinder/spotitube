package spotify

import (
	"net/http"
	"testing"

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

// playlistJSON mirrors the fullPlaylist fixture as a GET /v1/playlists
// payload whose tracks page carries the given next link (empty for a
// single page).
func playlistJSON(next string) string {
	nextJSON := "null"
	if next != "" {
		nextJSON = `"` + next + `"`
	}
	return `{"id":"123","name":"Playlist","owner":{"id":"User"},"collaborative":false,` +
		`"tracks":{"items":[{"added_at":"2026-01-01T00:00:00Z","track":` + trackJSON +
		`}],"next":` + nextJSON + `,"total":1}}`
}

// personalPlaylistsJSON builds a GET /v1/me/playlists payload out of
// id/name pairs.
func personalPlaylistsJSON(pairs ...string) string {
	items := ""
	for i := 0; i < len(pairs); i += 2 {
		if items != "" {
			items += ","
		}
		items += `{"id":"` + pairs[i] + `","name":"` + pairs[i+1] + `","owner":{"id":"User"}}`
	}
	return `{"items":[` + items + `],"next":null,"total":1}`
}

func BenchmarkPlaylist(b *testing.B) {
	for i := 0; i < b.N; i++ {
		TestPlaylist(&testing.T{})
	}
}

func TestPlaylist(t *testing.T) {
	scriptTransport(t, func(request *http.Request) (int, string) {
		switch request.URL.Path {
		case "/v1/me/playlists":
			return http.StatusOK, personalPlaylistsJSON("123", "Playlist")
		default:
			return http.StatusOK, playlistJSON("")
		}
	})

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

func TestPersonalPlaylistExactNameMatch(t *testing.T) {
	var (
		popPlusID  = spotify.ID("1111111111111111111111")
		popMinusID = spotify.ID("2222222222222222222222")
	)

	scriptTransport(t, func(_ *http.Request) (int, string) {
		return http.StatusOK, personalPlaylistsJSON(
			popPlusID.String(), "pop+",
			popMinusID.String(), "pop-",
		)
	})

	client := testClient()
	id, err := client.personalPlaylistNameToID("pop+")
	assert.Nil(t, err)
	assert.Equal(t, popPlusID, id)
	id, err = client.personalPlaylistNameToID("pop-")
	assert.Nil(t, err)
	assert.Equal(t, popMinusID, id)
	// a name not matching exactly falls back to a direct ID/URI/URL target
	id, err = client.personalPlaylistNameToID("Pop+")
	assert.Nil(t, err)
	assert.Equal(t, spotify.ID("Pop+"), id)
}

func TestPlaylistChannel(t *testing.T) {
	scriptTransport(t, func(request *http.Request) (int, string) {
		switch request.URL.Path {
		case "/v1/me/playlists":
			return http.StatusOK, personalPlaylistsJSON()
		default:
			return http.StatusOK, playlistJSON("")
		}
	})

	channel := make(chan interface{}, 1)
	defer close(channel)
	playlist, err := testClient().Playlist(fullPlaylist.ID.String(), channel)
	assert.Nil(t, err)
	assert.Equal(t, playlist.Tracks[0], <-channel)
}

func TestPlaylistCurrentUsersPlaylistsFailure(t *testing.T) {
	scriptTransport(t, func(_ *http.Request) (int, string) {
		return apiError(http.StatusInternalServerError, "ko")
	})

	assert.Error(t, sys.ErrOnly(testClient().Playlist(fullPlaylist.ID.String())))
}

func TestPlaylistCurrentUsersPlaylistsNextPageFailure(t *testing.T) {
	scriptTransport(t, func(request *http.Request) (int, string) {
		if request.URL.Query().Get("offset") != "" {
			return apiError(http.StatusInternalServerError, "ko")
		}
		return http.StatusOK, `{"items":[],"next":"https://api.spotify.com/v1/me/playlists?offset=1&limit=50","total":1}`
	})

	assert.EqualError(t, sys.ErrOnly(testClient().Playlist(fullPlaylist.ID.String())), "ko")
}

func TestPlaylistGetPlaylistFailure(t *testing.T) {
	scriptTransport(t, func(request *http.Request) (int, string) {
		switch request.URL.Path {
		case "/v1/me/playlists":
			return http.StatusOK, personalPlaylistsJSON()
		default:
			return apiError(http.StatusInternalServerError, "ko")
		}
	})

	assert.EqualError(t, sys.ErrOnly(testClient().Playlist(fullPlaylist.ID.String())), "ko")
}

func TestPlaylistGetPlaylistNextPageFailure(t *testing.T) {
	// pre-seed personalPlaylists cache so personalPlaylists() is skipped,
	// ensuring the pagination failure fires only in Playlist's own loop
	client := &Client{
		testClient().Client,
		testClient().authenticator,
		testClient().state,
		map[string]interface{}{
			personalPlaylistsCacheID: map[string]string{},
		},
	}
	scriptTransport(t, func(request *http.Request) (int, string) {
		if request.URL.Query().Get("offset") != "" {
			return apiError(http.StatusInternalServerError, "ko")
		}
		return http.StatusOK, playlistJSON("https://api.spotify.com/v1/playlists/123/tracks?offset=1&limit=50")
	})

	assert.EqualError(t, sys.ErrOnly(client.Playlist(fullPlaylist.ID.String())), "ko")
}
