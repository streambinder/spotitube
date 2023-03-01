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

func init() {
	monkey.Patch(time.Sleep, func(time.Duration) {})
	monkey.PatchInstanceMethod(reflect.TypeOf(&spotify.Client{}), "CurrentUsersTracksOpt",
		func(*spotify.Client, *spotify.Options) (*spotify.SavedTrackPage, error) {
			return library, nil
		})
}

func TestLibrary(t *testing.T) {
	channel := make(chan *entity.Track, 1)
	defer close(channel)

	err := (&Client{}).Library(channel)
	assert.Nil(t, err)
	assert.Equal(t, library.Tracks[0].Name, (<-channel).Title)
}

func TestLibraryFailure(t *testing.T) {
	monkey.PatchInstanceMethod(reflect.TypeOf(&spotify.Client{}), "CurrentUsersTracksOpt",
		func(*spotify.Client, *spotify.Options) (*spotify.SavedTrackPage, error) {
			return nil, errors.New("failure")
		})
	assert.EqualError(t, util.ErrOnly((&Client{}).Library()), "failure")
}

func TestLibraryNextPageFailure(t *testing.T) {
	var (
		client              = (&Client{spotify.NewClient(http.DefaultClient), spotify.NewAuthenticator(""), ""})
		libraryWithNextPage = library
	)
	libraryWithNextPage.Next = "http://0.0.0.0"
	monkey.PatchInstanceMethod(reflect.TypeOf(&spotify.Client{}), "CurrentUsersTracksOpt",
		func(*spotify.Client, *spotify.Options) (*spotify.SavedTrackPage, error) {
			return libraryWithNextPage, nil
		})
	assert.True(t, errors.Is(util.ErrOnly(client.Library()), syscall.ECONNREFUSED))
}
