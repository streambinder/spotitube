package spotify

import (
	"net/http"
	"testing"

	"github.com/streambinder/spotitube/entity"
	"github.com/streambinder/spotitube/sys"
	"github.com/stretchr/testify/assert"
	"github.com/zmb3/spotify/v2"
)

var searchResult = &spotify.SearchResult{
	Artists:   &spotify.FullArtistPage{},
	Albums:    &spotify.SimpleAlbumPage{},
	Playlists: &spotify.SimplePlaylistPage{},
	Tracks: &spotify.FullTrackPage{
		Tracks: []spotify.FullTrack{fullTrack},
	},
	Shows:    &spotify.SimpleShowPage{},
	Episodes: &spotify.SimpleEpisodePage{},
}

// searchJSON mirrors the searchResult fixture as a GET /v1/search payload
// whose tracks page carries the given next link (empty for a single page).
func searchJSON(next string) string {
	nextJSON := "null"
	if next != "" {
		nextJSON = `"` + next + `"`
	}
	return `{"tracks":{"items":[` + trackJSON + `],"next":` + nextJSON + `,"total":1}}`
}

func BenchmarkRandom(b *testing.B) {
	for i := 0; i < b.N; i++ {
		TestRandom(&testing.T{})
	}
}

func TestRandom(t *testing.T) {
	scriptTransport(t, func(_ *http.Request) (int, string) {
		return http.StatusOK, searchJSON("")
	})

	channel := make(chan interface{}, 1)
	defer close(channel)
	err := testClient().Random(TypeTrack, len(searchResult.Tracks.Tracks), channel)
	assert.Nil(t, err)
	assert.Equal(t, searchResult.Tracks.Tracks[0].ID.String(), (<-channel).(*entity.Track).ID)
}

func TestRandomFailure(t *testing.T) {
	scriptTransport(t, func(_ *http.Request) (int, string) {
		return apiError(http.StatusInternalServerError, "ko")
	})

	err := testClient().Random(TypeTrack, len(searchResult.Tracks.Tracks))
	assert.EqualError(t, err, "ko")
}

func TestRandomNextPageFailure(t *testing.T) {
	scriptTransport(t, func(request *http.Request) (int, string) {
		if request.URL.Query().Get("offset") != "" {
			return apiError(http.StatusInternalServerError, "ko")
		}
		return http.StatusOK, searchJSON("https://api.spotify.com/v1/search?query=x&type=track&offset=1&limit=50")
	})

	assert.EqualError(t, sys.ErrOnly(testClient().Random(TypeTrack, len(searchResult.Tracks.Tracks))), "ko")
}
