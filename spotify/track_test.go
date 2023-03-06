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

func init() {
	monkey.Patch(time.Sleep, func(time.Duration) {})
	monkey.PatchInstanceMethod(reflect.TypeOf(&spotify.Client{}), "GetTrackOpt",
		func(*spotify.Client, spotify.ID, *spotify.Options) (*spotify.FullTrack, error) {
			return &fullTrack, nil
		})
}

func TestTrack(t *testing.T) {
	channel := make(chan interface{}, 1)
	defer close(channel)

	track, err := (&Client{}).Track(fullTrack.ID.String(), channel)
	assert.Nil(t, err)
	assert.Equal(t, fullTrack.ID.String(), track.ID)
	assert.Equal(t, fullTrack.Name, track.Title)
	assert.Equal(t, len(fullTrack.Artists), len(track.Artists))
	assert.Equal(t, len(fullTrack.Album.Name), len(track.Album))
	assert.Equal(t, fullTrack.Duration/1000, track.Duration) // TODO find safer conversion approach
	assert.Equal(t, fullTrack.TrackNumber, track.Number)
	assert.Equal(t, fullTrack.Album.Images[0].URL, track.ArtworkURL)
	assert.Equal(t, track, <-channel)
	assert.True(t, strings.HasPrefix(fullTrack.Album.ReleaseDate, track.Year))
}

func TestTrackGetTrackFailure(t *testing.T) {
	monkey.PatchInstanceMethod(reflect.TypeOf(&spotify.Client{}), "GetTrackOpt",
		func(*spotify.Client, spotify.ID, *spotify.Options) (*spotify.FullTrack, error) {
			return nil, errors.New("failure")
		})
	assert.EqualError(t, util.ErrOnly((&Client{}).Track(fullTrack.ID.String())), "failure")
}