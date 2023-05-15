package spotify

import (
	"errors"
	"net/http"
	"syscall"
	"testing"
	"time"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/streambinder/spotitube/entity"
	"github.com/streambinder/spotitube/util"
	"github.com/stretchr/testify/assert"
	"github.com/zmb3/spotify/v2"
	spotifyauth "github.com/zmb3/spotify/v2/auth"
)

var library = &spotify.SavedTrackPage{
	Tracks: []spotify.SavedTrack{
		{FullTrack: fullTrack},
	},
}

func BenchmarkLibrary(b *testing.B) {
	for i := 0; i < b.N; i++ {
		TestLibrary(&testing.T{})
	}
}

func TestLibrary(t *testing.T) {
	// monkey patching
	defer gomonkey.NewPatches().
		ApplyFunc(time.Sleep, func() {}).
		ApplyMethod(&spotify.Client{}, "CurrentUsersTracks", func() (*spotify.SavedTrackPage, error) {
			return library, nil
		}).
		Reset()

	// testing
	assert.Nil(t, (&Client{}).Library())
}

func TestLibraryChannel(t *testing.T) {
	// monkey patching
	defer gomonkey.NewPatches().
		ApplyFunc(time.Sleep, func() {}).
		ApplyMethod(&spotify.Client{}, "CurrentUsersTracks", func() (*spotify.SavedTrackPage, error) {
			return library, nil
		}).
		Reset()

	// testing
	channel := make(chan interface{}, 1)
	defer close(channel)
	err := (&Client{}).Library(channel)
	assert.Nil(t, err)
	assert.Equal(t, library.Tracks[0].Name, ((<-channel).(*entity.Track)).Title)
}

func TestLibraryFailure(t *testing.T) {
	// monkey patching
	defer gomonkey.NewPatches().
		ApplyFunc(time.Sleep, func() {}).
		ApplyMethod(&spotify.Client{}, "CurrentUsersTracks", func() (*spotify.SavedTrackPage, error) {
			return nil, errors.New("ko")
		}).
		Reset()

	// testing
	assert.EqualError(t, util.ErrOnly((&Client{}).Library()), "ko")
}

func TestLibraryNextPageFailure(t *testing.T) {
	var (
		client              = (&Client{spotify.New(http.DefaultClient), &spotifyauth.Authenticator{}, ""})
		libraryWithNextPage = library
	)
	libraryWithNextPage.Next = "http://0.0.0.0"

	// monkey patching
	defer gomonkey.NewPatches().
		ApplyFunc(time.Sleep, func() {}).
		ApplyMethod(&spotify.Client{}, "CurrentUsersTracks", func() (*spotify.SavedTrackPage, error) {
			return libraryWithNextPage, nil
		}).
		Reset()

	// testing
	assert.True(t, errors.Is(util.ErrOnly(client.Library()), syscall.ECONNREFUSED))
}
