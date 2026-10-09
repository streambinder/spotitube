package spotify

import (
	"net/http"
	"testing"

	"github.com/streambinder/spotitube/entity"
	"github.com/streambinder/spotitube/sys"
	"github.com/stretchr/testify/assert"
	"github.com/zmb3/spotify/v2"
)

var library = &spotify.SavedTrackPage{
	Tracks: []spotify.SavedTrack{
		{FullTrack: fullTrack},
	},
}

// libraryJSON mirrors the library fixture as a GET /v1/me/tracks payload.
// Requests carrying an offset query fetch a subsequent page.
func libraryJSON(request *http.Request) (int, string) {
	if request.URL.Query().Get("offset") != "" {
		return http.StatusOK, `{"items":[],"next":null,"total":1}`
	}
	return http.StatusOK, `{"items":[{"added_at":"2026-01-01T00:00:00Z","track":` +
		trackJSON + `}],"next":null,"total":1}`
}

func BenchmarkLibrary(b *testing.B) {
	for i := 0; i < b.N; i++ {
		TestLibrary(&testing.T{})
	}
}

func TestLibrary(t *testing.T) {
	scriptTransport(t, libraryJSON)

	assert.Nil(t, testClient().Library(0))
}

func TestLibraryChannel(t *testing.T) {
	scriptTransport(t, libraryJSON)

	channel := make(chan interface{}, 1)
	defer close(channel)
	err := testClient().Library(1, channel)
	assert.Nil(t, err)
	assert.Equal(t, library.Tracks[0].Name, (<-channel).(*entity.Track).Title)
}

func TestLibraryFailure(t *testing.T) {
	scriptTransport(t, func(_ *http.Request) (int, string) {
		return apiError(http.StatusInternalServerError, "ko")
	})

	assert.EqualError(t, sys.ErrOnly(testClient().Library(0)), "ko")
}

func TestLibraryNextPageFailure(t *testing.T) {
	scriptTransport(t, func(request *http.Request) (int, string) {
		if request.URL.Query().Get("offset") != "" {
			return apiError(http.StatusInternalServerError, "ko")
		}
		return http.StatusOK, `{"items":[{"added_at":"2026-01-01T00:00:00Z","track":` +
			trackJSON + `}],"next":"https://api.spotify.com/v1/me/tracks?offset=1&limit=50","total":2}`
	})

	assert.EqualError(t, sys.ErrOnly(testClient().Library(0)), "ko")
}
