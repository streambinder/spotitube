package spotify

import (
	"net/http"
	"testing"

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

// albumJSON mirrors the fullAlbum fixture as a GET /v1/albums payload whose
// tracks page carries the given next link (empty for a single page).
func albumJSON(next string) string {
	nextJSON := "null"
	if next != "" {
		nextJSON = `"` + next + `"`
	}
	return `{"id":"123","name":"Album","artists":[{"name":"Artist"}],` +
		`"tracks":{"items":[` + simpleTrackJSON + `],"next":` + nextJSON + `,"total":1}}`
}

func BenchmarkAlbum(b *testing.B) {
	for i := 0; i < b.N; i++ {
		TestAlbum(&testing.T{})
	}
}

func TestAlbum(t *testing.T) {
	scriptTransport(t, func(_ *http.Request) (int, string) {
		return http.StatusOK, albumJSON("")
	})

	album, err := testClient().Album(fullAlbum.ID.String())
	assert.Nil(t, err)
	assert.Equal(t, fullAlbum.ID.String(), album.ID)
	assert.Equal(t, fullAlbum.Name, album.Name)
	assert.Equal(t, len(fullAlbum.Artists), len(album.Artists))
	assert.Equal(t, len(fullAlbum.Tracks.Tracks), len(album.Tracks))
}

func TestAlbumChannel(t *testing.T) {
	scriptTransport(t, func(_ *http.Request) (int, string) {
		return http.StatusOK, albumJSON("")
	})

	channel := make(chan interface{}, 1)
	defer close(channel)
	album, err := testClient().Album(fullAlbum.ID.String(), channel)
	assert.Nil(t, err)
	assert.Equal(t, album.Tracks[0], <-channel)
}

func TestAlbumGetAlbumFailure(t *testing.T) {
	scriptTransport(t, func(_ *http.Request) (int, string) {
		return apiError(http.StatusInternalServerError, "ko")
	})

	assert.EqualError(t, sys.ErrOnly(testClient().Album(fullPlaylist.ID.String())), "ko")
}

func TestAlbumNextPageFailure(t *testing.T) {
	scriptTransport(t, func(request *http.Request) (int, string) {
		if request.URL.Path == "/v1/albums/123/tracks" {
			return apiError(http.StatusInternalServerError, "ko")
		}
		return http.StatusOK, albumJSON("https://api.spotify.com/v1/albums/123/tracks?offset=1&limit=50")
	})

	assert.EqualError(t, sys.ErrOnly(testClient().Album(fullAlbum.ID.String())), "ko")
}
