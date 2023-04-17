package spotify

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/streambinder/spotitube/util"
	"github.com/stretchr/testify/assert"
	"github.com/zmb3/spotify/v2"
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
	patchtimeSleep := gomonkey.ApplyFunc(time.Sleep, func(time.Duration) {})
	defer patchtimeSleep.Reset()
	patchspotifyClientGetTrack := gomonkey.ApplyMethod(reflect.TypeOf(&spotify.Client{}), "GetTrack",
		func(*spotify.Client, context.Context, spotify.ID, ...spotify.RequestOption) (*spotify.FullTrack, error) {
			return &fullTrack, nil
		})
	defer patchspotifyClientGetTrack.Reset()

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
	patchtimeSleep := gomonkey.ApplyFunc(time.Sleep, func(time.Duration) {})
	defer patchtimeSleep.Reset()
	patchspotifyClientGetTrack := gomonkey.ApplyMethod(reflect.TypeOf(&spotify.Client{}), "GetTrack",
		func(*spotify.Client, context.Context, spotify.ID, ...spotify.RequestOption) (*spotify.FullTrack, error) {
			return nil, errors.New("failure")
		})
	defer patchspotifyClientGetTrack.Reset()

	// testing
	assert.EqualError(t, util.ErrOnly((&Client{}).Track(fullTrack.ID.String())), "failure")
}
