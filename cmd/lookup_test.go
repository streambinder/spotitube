package cmd

import (
	"errors"
	"testing"

	"github.com/bytedance/mockey"
	"github.com/streambinder/spotitube/entity"
	"github.com/streambinder/spotitube/lyrics"
	"github.com/streambinder/spotitube/provider"
	"github.com/streambinder/spotitube/spotify"
	"github.com/streambinder/spotitube/sys"
	"github.com/stretchr/testify/assert"
)

func BenchmarkLookup(b *testing.B) {
	for i := 0; i < b.N; i++ {
		TestCmdLookup(&testing.T{})
	}
}

func TestCmdLookup(t *testing.T) {
	_track := &entity.Track{ID: "TestCmdLookup", Title: "Title", Artists: []string{"Artist"}}

	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(spotify.Authenticate).Return(&spotify.Client{}, nil).Build()
	mockey.Mock(mockey.GetMethod(&spotify.Client{}, "Library")).To(func(_ int, ch ...chan interface{}) error {
		ch[0] <- _track
		ch[1] <- _track
		return nil
	}).Build()
	mockey.Mock(provider.Search).Return([]*provider.Match{{URL: "http://localhost/", Score: 0}}, nil).Build()
	mockey.Mock(lyrics.Search).Return("lyrics", nil).Build()

	// testing
	assert.Nil(t, sys.ErrOnly(testExecute(cmdLookup(), "-l")))
}

func TestCmdLookupNoTrack(t *testing.T) {
	assert.Error(t, sys.ErrOnly(testExecute(cmdLookup())), "no track has been issued")
}

func TestCmdLookupTrack(t *testing.T) {
	_track := &entity.Track{ID: "TestCmdLookupTrack", Title: "Title", Artists: []string{"Artist"}}

	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(spotify.Authenticate).Return(&spotify.Client{}, nil).Build()
	mockey.Mock(mockey.GetMethod(&spotify.Client{}, "Track")).To(func(_ string, ch ...chan interface{}) (*entity.Track, error) {
		ch[0] <- _track
		ch[1] <- _track
		return _track, nil
	}).Build()
	mockey.Mock(provider.Search).Return([]*provider.Match{{URL: "http://localhost/", Score: 0}}, nil).Build()
	mockey.Mock(lyrics.Search).Return("lyrics", nil).Build()

	// testing
	assert.Nil(t, sys.ErrOnly(testExecute(cmdLookup(), "123")))
}

func TestCmdLookupRandom(t *testing.T) {
	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(spotify.Authenticate).Return(&spotify.Client{}, nil).Build()
	mockey.Mock(mockey.GetMethod(&spotify.Client{}, "Random")).Return(nil).Build()
	mockey.Mock(provider.Search).Return([]*provider.Match{{URL: "http://localhost/", Score: 0}}, nil).Build()
	mockey.Mock(lyrics.Search).Return("lyrics", nil).Build()

	// testing
	assert.Nil(t, sys.ErrOnly(testExecute(cmdLookup(), "-r")))
}

func TestCmdLookupAuthFailure(t *testing.T) {
	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(spotify.Authenticate).Return(nil, errors.New("ko")).Build()

	// testing
	assert.EqualError(t, sys.ErrOnly(testExecute(cmdLookup(), "-l")), "ko")
}

func TestCmdLookupLibraryFailure(t *testing.T) {
	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(spotify.Authenticate).Return(&spotify.Client{}, nil).Build()
	mockey.Mock(mockey.GetMethod(&spotify.Client{}, "Library")).To(func(_ int, _ ...chan interface{}) error {
		return errors.New("ko")
	}).Build()

	// testing
	assert.EqualError(t, sys.ErrOnly(testExecute(cmdLookup(), "-l")), "ko")
}

func TestCmdLookupTrackFailure(t *testing.T) {
	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(spotify.Authenticate).Return(&spotify.Client{}, nil).Build()
	mockey.Mock(mockey.GetMethod(&spotify.Client{}, "Track")).To(func(_ string, _ ...chan interface{}) (*entity.Track, error) {
		return nil, errors.New("ko")
	}).Build()

	// testing
	assert.EqualError(t, sys.ErrOnly(testExecute(cmdLookup(), "123")), "ko")
}

func TestCmdLookupRandomFailure(t *testing.T) {
	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(spotify.Authenticate).Return(&spotify.Client{}, nil).Build()
	mockey.Mock(mockey.GetMethod(&spotify.Client{}, "Random")).Return(errors.New("ko")).Build()

	// testing
	assert.EqualError(t, sys.ErrOnly(testExecute(cmdLookup(), "-r")), "ko")
}

func TestCmdLookupSearchFailure(t *testing.T) {
	_track := &entity.Track{ID: "TestCmdLookupSearchFailure", Title: "Title", Artists: []string{"Artist"}}

	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(spotify.Authenticate).Return(&spotify.Client{}, nil).Build()
	mockey.Mock(mockey.GetMethod(&spotify.Client{}, "Library")).To(func(_ int, ch ...chan interface{}) error {
		ch[0] <- _track
		ch[1] <- _track
		return nil
	}).Build()
	mockey.Mock(provider.Search).Return(nil, errors.New("ko")).Build()
	mockey.Mock(lyrics.Search).Return("lyrics", nil).Build()

	// testing
	assert.Nil(t, sys.ErrOnly(testExecute(cmdLookup(), "-l")))
}

func TestCmdLookupSearchNotFound(t *testing.T) {
	_track := &entity.Track{ID: "TestCmdLookupSearchNotFound", Title: "Title", Artists: []string{"Artist"}}

	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(spotify.Authenticate).Return(&spotify.Client{}, nil).Build()
	mockey.Mock(mockey.GetMethod(&spotify.Client{}, "Library")).To(func(_ int, ch ...chan interface{}) error {
		ch[0] <- _track
		ch[1] <- _track
		return nil
	}).Build()
	mockey.Mock(provider.Search).Return([]*provider.Match{}, nil).Build()
	mockey.Mock(lyrics.Search).Return("lyrics", nil).Build()

	// testing
	assert.Nil(t, sys.ErrOnly(testExecute(cmdLookup(), "-l")))
}

func TestCmdLookupLyricsFailure(t *testing.T) {
	_track := &entity.Track{ID: "TestCmdLookupLyricsFailure", Title: "Title", Artists: []string{"Artist"}}

	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(spotify.Authenticate).Return(&spotify.Client{}, nil).Build()
	mockey.Mock(mockey.GetMethod(&spotify.Client{}, "Library")).To(func(_ int, ch ...chan interface{}) error {
		ch[0] <- _track
		ch[1] <- _track
		return nil
	}).Build()
	mockey.Mock(provider.Search).Return([]*provider.Match{{URL: "http://localhost/", Score: 0}}, nil).Build()
	mockey.Mock(lyrics.Search).Return("", errors.New("ko")).Build()

	// testing
	assert.Nil(t, sys.ErrOnly(testExecute(cmdLookup(), "-l")))
}

func TestCmdLookupLyricsNotFound(t *testing.T) {
	_track := &entity.Track{ID: "TestCmdLookupLyricsNotFound", Title: "Title", Artists: []string{"Artist"}}

	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(spotify.Authenticate).Return(&spotify.Client{}, nil).Build()
	mockey.Mock(mockey.GetMethod(&spotify.Client{}, "Library")).To(func(_ int, ch ...chan interface{}) error {
		ch[0] <- _track
		ch[1] <- _track
		return nil
	}).Build()
	mockey.Mock(provider.Search).Return([]*provider.Match{{URL: "http://localhost/", Score: 0}}, nil).Build()
	mockey.Mock(lyrics.Search).Return("", nil).Build()

	// testing
	assert.Nil(t, sys.ErrOnly(testExecute(cmdLookup(), "-l")))
}
