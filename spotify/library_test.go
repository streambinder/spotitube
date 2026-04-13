package spotify

import (
	"errors"
	"testing"

	"github.com/bytedance/mockey"
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

func BenchmarkLibrary(b *testing.B) {
	for i := 0; i < b.N; i++ {
		TestLibrary(&testing.T{})
	}
}

func TestLibrary(t *testing.T) {
	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(mockey.GetMethod(&spotify.Client{}, "CurrentUsersTracks")).Return(library, nil).Build()

	// testing
	assert.Nil(t, testClient().Library(0))
}

func TestLibraryChannel(t *testing.T) {
	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(mockey.GetMethod(&spotify.Client{}, "CurrentUsersTracks")).Return(library, nil).Build()

	// testing
	channel := make(chan interface{}, 1)
	defer close(channel)
	err := testClient().Library(1, channel)
	assert.Nil(t, err)
	assert.Equal(t, library.Tracks[0].Name, ((<-channel).(*entity.Track)).Title)
}

func TestLibraryFailure(t *testing.T) {
	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(mockey.GetMethod(&spotify.Client{}, "CurrentUsersTracks")).Return(nil, errors.New("ko")).Build()

	// testing
	assert.EqualError(t, sys.ErrOnly(testClient().Library(0)), "ko")
}

func TestLibraryNextPageFailure(t *testing.T) {
	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(mockey.GetMethod(&spotify.Client{}, "CurrentUsersTracks")).Return(library, nil).Build()
	mockey.Mock(mockey.GetMethod(&spotify.Client{}, "NextPage")).Return(errors.New("ko")).Build()

	// testing
	assert.EqualError(t, sys.ErrOnly(testClient().Library(0)), "ko")
}
