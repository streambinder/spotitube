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

var library = &spotify.SavedTrackPage{
	Tracks: []spotify.SavedTrack{
		{FullTrack: fullTrack},
	},
}

func TestLibrary(t *testing.T) {
	// monkey patching
	monkey.Patch(time.Sleep, func(time.Duration) {})
	defer monkey.Unpatch(time.Sleep)
	monkey.PatchInstanceMethod(reflect.TypeOf(&spotify.Client{}), "CurrentUsersTracksOpt",
		func(*spotify.Client, *spotify.Options) (*spotify.SavedTrackPage, error) {
			return library, nil
		})
	defer monkey.UnpatchInstanceMethod(reflect.TypeOf(&spotify.Client{}), "CurrentUsersTracksOpt")

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
	monkey.PatchInstanceMethod(reflect.TypeOf(&spotify.Client{}), "CurrentUsersTracksOpt",
		func(*spotify.Client, *spotify.Options) (*spotify.SavedTrackPage, error) {
			return nil, errors.New("failure")
		})
	defer monkey.UnpatchInstanceMethod(reflect.TypeOf(&spotify.Client{}), "CurrentUsersTracksOpt")

	// testing
	assert.EqualError(t, util.ErrOnly((&Client{}).Library()), "failure")
}

func TestLibraryNextPageFailure(t *testing.T) {
	var (
		client              = (&Client{spotify.NewClient(http.DefaultClient), spotify.NewAuthenticator(""), ""})
		libraryWithNextPage = library
	)
	libraryWithNextPage.Next = "http://0.0.0.0"

	// monkey patching
	monkey.Patch(time.Sleep, func(time.Duration) {})
	defer monkey.Unpatch(time.Sleep)
	monkey.PatchInstanceMethod(reflect.TypeOf(&spotify.Client{}), "CurrentUsersTracksOpt",
		func(*spotify.Client, *spotify.Options) (*spotify.SavedTrackPage, error) {
			return libraryWithNextPage, nil
		})
	defer monkey.UnpatchInstanceMethod(reflect.TypeOf(&spotify.Client{}), "CurrentUsersTracksOpt")

	// testing
	assert.True(t, errors.Is(util.ErrOnly(client.Library()), syscall.ECONNREFUSED))
}
