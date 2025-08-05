package spotify

import (
	"errors"
	"syscall"
	"testing"
	"time"

	"github.com/agiledragon/gomonkey/v2"
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

func BenchmarkRandom(b *testing.B) {
	for i := 0; i < b.N; i++ {
		TestRandom(&testing.T{})
	}
}

func TestRandom(t *testing.T) {
	// monkey patching
	defer gomonkey.NewPatches().
		ApplyFunc(time.Sleep, func() {}).
		ApplyMethod(&spotify.Client{}, "Search", func() (*spotify.SearchResult, error) {
			return searchResult, nil
		}).
		Reset()

	// testing
	channel := make(chan interface{}, 1)
	defer close(channel)
	err := testClient().Random(TypeTrack, len(searchResult.Tracks.Tracks), channel)
	assert.Nil(t, err)
	assert.Equal(t, searchResult.Tracks.Tracks[0].ID.String(), (<-channel).(*entity.Track).ID)
}

func TestRandomFailure(t *testing.T) {
	// monkey patching
	defer gomonkey.NewPatches().
		ApplyFunc(time.Sleep, func() {}).
		ApplyMethod(&spotify.Client{}, "Search", func() (*spotify.SearchResult, error) {
			return nil, errors.New("ko")
		}).
		Reset()

	// testing
	err := testClient().Random(TypeTrack, len(searchResult.Tracks.Tracks))
	assert.EqualError(t, err, "ko")
}

func TestRandomNextPageFailure(t *testing.T) {
	var (
		client = testClient()
		search = searchResult
	)
	search.Tracks.Next = "http://0.0.0.0"

	// monkey patching
	defer gomonkey.NewPatches().
		ApplyFunc(time.Sleep, func() {}).
		ApplyMethod(&spotify.Client{}, "Search", func() (*spotify.SearchResult, error) {
			return search, nil
		}).
		Reset()

	// testing
	assert.True(t, errors.Is(sys.ErrOnly(client.Random(TypeTrack, len(searchResult.Tracks.Tracks))), syscall.ECONNREFUSED))
}
