package spotify

import (
	"errors"
	"syscall"
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
	client := testClient()
	// shallow copy to avoid mutating package-level fixture
	libraryCopy := *library
	libraryCopy.Next = "http://0.0.0.0"
	defer func() { libraryCopy.Next = "" }()

	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(mockey.GetMethod(&spotify.Client{}, "CurrentUsersTracks")).Return(&libraryCopy, nil).Build()

	// testing
	assert.True(t, errors.Is(sys.ErrOnly(client.Library(0)), syscall.ECONNREFUSED))
}
