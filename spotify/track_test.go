package spotify

import (
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"bou.ke/monkey"
	"github.com/streambinder/spotitube/util"
	"github.com/stretchr/testify/assert"
	"github.com/zmb3/spotify"
)

var fullTrack = spotify.FullTrack{
	SimpleTrack: spotify.SimpleTrack{
		ID:          spotify.ID("123"),
		Name:        "Title",
		Artists:     []spotify.SimpleArtist{{Name: "Artist"}},
		Duration:    180000,
		TrackNumber: 1,
	},
	Album: spotify.SimpleAlbum{
		Name:        "Album",
		ReleaseDate: "1970",
		Images:      []spotify.Image{{URL: "http://ima.ge"}},
	},
}

func TestTrack(t *testing.T) {
	// monkey patching
	monkey.Patch(time.Sleep, func(time.Duration) {})
	defer monkey.Unpatch(time.Sleep)
	monkey.PatchInstanceMethod(reflect.TypeOf(&spotify.Client{}), "GetTrackOpt",
		func(*spotify.Client, spotify.ID, *spotify.Options) (*spotify.FullTrack, error) {
			return &fullTrack, nil
		})
	defer monkey.UnpatchInstanceMethod(reflect.TypeOf(&spotify.Client{}), "GetTrackOpt")

	// testing
	channel := make(chan interface{}, 1)
	defer close(channel)
	track, err := (&Client{}).Track(fullTrack.ID.String(), channel)
	assert.Nil(t, err)
	assert.Equal(t, fullTrack.ID.String(), track.ID)
	assert.Equal(t, fullTrack.Name, track.Title)
	assert.Equal(t, len(fullTrack.Artists), len(track.Artists))
	assert.Equal(t, len(fullTrack.Album.Name), len(track.Album))
	assert.Equal(t, fullTrack.Duration/1000, track.Duration)
	assert.Equal(t, fullTrack.TrackNumber, track.Number)
	assert.Equal(t, fullTrack.Album.Images[0].URL, track.Artwork.URL)
	assert.Equal(t, track, <-channel)
	assert.True(t, strings.HasPrefix(fullTrack.Album.ReleaseDate, track.Year))
}

func TestTrackGetTrackFailure(t *testing.T) {
	// monkey patching
	monkey.Patch(time.Sleep, func(time.Duration) {})
	defer monkey.Unpatch(time.Sleep)
	monkey.PatchInstanceMethod(reflect.TypeOf(&spotify.Client{}), "GetTrackOpt",
		func(*spotify.Client, spotify.ID, *spotify.Options) (*spotify.FullTrack, error) {
			return nil, errors.New("failure")
		})
	defer monkey.UnpatchInstanceMethod(reflect.TypeOf(&spotify.Client{}), "GetTrackOpt")

	// testing
	assert.EqualError(t, util.ErrOnly((&Client{}).Track(fullTrack.ID.String())), "failure")
}
