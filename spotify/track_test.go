package spotify

import (
	"errors"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/streambinder/spotitube/sys"
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

func BenchmarkTrack(b *testing.B) {
	for i := 0; i < b.N; i++ {
		TestTrack(&testing.T{})
	}
}

func TestTrack(t *testing.T) {
	// monkey patching
	defer gomonkey.NewPatches().
		ApplyFunc(time.Sleep, func() {}).
		ApplyMethod(&spotify.Client{}, "GetTrack", func() (*spotify.FullTrack, error) {
			return &fullTrack, nil
		}).
		Reset()

	// testing
	track, err := testClient().Track(fullTrack.ID.String())
	assert.Nil(t, err)
	assert.Equal(t, fullTrack.ID.String(), track.ID)
	assert.Equal(t, fullTrack.Name, track.Title)
	assert.Equal(t, len(fullTrack.Artists), len(track.Artists))
	assert.Equal(t, len(fullTrack.Album.Name), len(track.Album))
	assert.Equal(t, int(fullTrack.Duration)/1000, track.Duration)
	assert.Equal(t, int(fullTrack.TrackNumber), track.Number)
	assert.Equal(t, fullTrack.Album.Images[0].URL, track.Artwork.URL)
	assert.True(t, strings.HasPrefix(fullTrack.Album.ReleaseDate, strconv.Itoa(track.Year)))
}

func TestTrackChannel(t *testing.T) {
	// monkey patching
	defer gomonkey.NewPatches().
		ApplyFunc(time.Sleep, func() {}).
		ApplyMethod(&spotify.Client{}, "GetTrack", func() (*spotify.FullTrack, error) {
			return &fullTrack, nil
		}).
		Reset()

	// testing
	channel := make(chan interface{}, 1)
	defer close(channel)
	track, err := testClient().Track(fullTrack.ID.String(), channel)
	assert.Nil(t, err)
	assert.Equal(t, track, <-channel)
}

func TestTrackGetTrackFailure(t *testing.T) {
	// monkey patching
	defer gomonkey.NewPatches().
		ApplyFunc(time.Sleep, func() {}).
		ApplyMethod(&spotify.Client{}, "GetTrack", func() (*spotify.FullTrack, error) {
			return nil, errors.New("ko")
		}).
		Reset()

	// testing
	assert.EqualError(t, sys.ErrOnly(testClient().Track(fullTrack.ID.String())), "ko")
}
