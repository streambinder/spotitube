package spotify

import (
	"context"
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
	"github.com/zmb3/spotify/v2"
	spotifyauth "github.com/zmb3/spotify/v2/auth"
)

var library = &spotify.SavedTrackPage{
	Tracks: []spotify.SavedTrack{
		{FullTrack: fullTrack},
	},
}

func TestLibrary(t *testing.T) {
	// monkey patching
	monkey.Patch(time.Sleep, func(time.Duration) {})
	defer monkey.Unpatch(time.Sleep)
	monkey.PatchInstanceMethod(reflect.TypeOf(&spotify.Client{}), "CurrentUsersTracks",
		func(*spotify.Client, context.Context, ...spotify.RequestOption) (*spotify.SavedTrackPage, error) {
			return library, nil
		})
	defer monkey.UnpatchInstanceMethod(reflect.TypeOf(&spotify.Client{}), "CurrentUsersTracks")

	// testing
	channel := make(chan interface{}, 1)
	defer close(channel)
	err := (&Client{}).Library(channel)
	assert.Nil(t, err)
	assert.Equal(t, library.Tracks[0].Name, ((<-channel).(*entity.Track)).Title)
}

func TestLibraryFailure(t *testing.T) {
	// monkey patching
	monkey.Patch(time.Sleep, func(time.Duration) {})
	defer monkey.Unpatch(time.Sleep)
	monkey.PatchInstanceMethod(reflect.TypeOf(&spotify.Client{}), "CurrentUsersTracks",
		func(*spotify.Client, context.Context, ...spotify.RequestOption) (*spotify.SavedTrackPage, error) {
			return nil, errors.New("failure")
		})
	defer monkey.UnpatchInstanceMethod(reflect.TypeOf(&spotify.Client{}), "CurrentUsersTracks")

	// testing
	assert.EqualError(t, util.ErrOnly((&Client{}).Library()), "failure")
}

func TestLibraryNextPageFailure(t *testing.T) {
	var (
		client              = (&Client{spotify.New(http.DefaultClient), &spotifyauth.Authenticator{}, ""})
		libraryWithNextPage = library
	)
	libraryWithNextPage.Next = "http://0.0.0.0"

	// monkey patching
	monkey.Patch(time.Sleep, func(time.Duration) {})
	defer monkey.Unpatch(time.Sleep)
	monkey.PatchInstanceMethod(reflect.TypeOf(&spotify.Client{}), "CurrentUsersTracks",
		func(*spotify.Client, context.Context, ...spotify.RequestOption) (*spotify.SavedTrackPage, error) {
			return libraryWithNextPage, nil
		})
	defer monkey.UnpatchInstanceMethod(reflect.TypeOf(&spotify.Client{}), "CurrentUsersTracks")

	// testing
	assert.True(t, errors.Is(util.ErrOnly(client.Library()), syscall.ECONNREFUSED))
}
