package cmd

import (
	"errors"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/streambinder/spotitube/entity"
	"github.com/streambinder/spotitube/lyrics"
	"github.com/streambinder/spotitube/provider"
	"github.com/streambinder/spotitube/spotify"
	"github.com/streambinder/spotitube/util"
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
	defer gomonkey.NewPatches().
		ApplyFunc(spotify.Authenticate, func() (*spotify.Client, error) {
			return &spotify.Client{}, nil
		}).
		ApplyMethod(&spotify.Client{}, "Library", func(_ *spotify.Client, ch ...chan interface{}) error {
			ch[0] <- _track
			ch[1] <- _track
			return nil
		}).
		ApplyFunc(provider.Search, func() ([]*provider.Match, error) {
			return []*provider.Match{{URL: "http://localhost/", Score: 0}}, nil
		}).
		ApplyFunc(lyrics.Search, func() (string, error) {
			return "lyrics", nil
		}).
		Reset()

	// testing
	assert.Nil(t, util.ErrOnly(testExecute(cmdLookup(), "-l")))
}

func TestCmdLookupNoTrack(t *testing.T) {
	assert.Error(t, util.ErrOnly(testExecute(cmdLookup())), "no track has been issued")
}

func TestCmdLookupTrack(t *testing.T) {
	_track := &entity.Track{ID: "TestCmdLookupTrack", Title: "Title", Artists: []string{"Artist"}}

	// monkey patching
	defer gomonkey.NewPatches().
		ApplyFunc(spotify.Authenticate, func() (*spotify.Client, error) {
			return &spotify.Client{}, nil
		}).
		ApplyMethod(&spotify.Client{}, "Track", func(_ *spotify.Client, _ string, ch ...chan interface{}) (*entity.Track, error) {
			ch[0] <- _track
			ch[1] <- _track
			return _track, nil
		}).
		ApplyFunc(provider.Search, func() ([]*provider.Match, error) {
			return []*provider.Match{{URL: "http://localhost/", Score: 0}}, nil
		}).
		ApplyFunc(lyrics.Search, func() (string, error) {
			return "lyrics", nil
		}).
		Reset()

	// testing
	assert.Nil(t, util.ErrOnly(testExecute(cmdLookup(), "123")))
}

func TestCmdLookupRandom(t *testing.T) {
	// monkey patching
	defer gomonkey.NewPatches().
		ApplyFunc(spotify.Authenticate, func() (*spotify.Client, error) {
			return &spotify.Client{}, nil
		}).
		ApplyMethod(&spotify.Client{}, "Random", func() error {
			return nil
		}).
		ApplyFunc(provider.Search, func() ([]*provider.Match, error) {
			return []*provider.Match{{URL: "http://localhost/", Score: 0}}, nil
		}).
		ApplyFunc(lyrics.Search, func() (string, error) {
			return "lyrics", nil
		}).
		Reset()

	// testing
	assert.Nil(t, util.ErrOnly(testExecute(cmdLookup(), "-r")))
}

func TestCmdLookupAuthFailure(t *testing.T) {
	// monkey patching
	defer gomonkey.ApplyFunc(spotify.Authenticate, func() (*spotify.Client, error) {
		return nil, errors.New("ko")
	}).Reset()

	// testing
	assert.Error(t, util.ErrOnly(testExecute(cmdLookup(), "-l")), "ko")
}

func TestCmdLookupLibraryFailure(t *testing.T) {
	// monkey patching
	defer gomonkey.NewPatches().
		ApplyFunc(spotify.Authenticate, func() (*spotify.Client, error) {
			return &spotify.Client{}, nil
		}).
		ApplyMethod(&spotify.Client{}, "Library", func(_ *spotify.Client, ch ...chan interface{}) error {
			return errors.New("ko")
		}).
		Reset()

	// testing
	assert.Error(t, util.ErrOnly(testExecute(cmdLookup(), "-l")), "ko")
}

func TestCmdLookupTrackFailure(t *testing.T) {
	// monkey patching
	defer gomonkey.NewPatches().
		ApplyFunc(spotify.Authenticate, func() (*spotify.Client, error) {
			return &spotify.Client{}, nil
		}).
		ApplyMethod(&spotify.Client{}, "Track", func(_ *spotify.Client, _ string, ch ...chan interface{}) (*entity.Track, error) {
			return nil, errors.New("ko")
		}).Reset()

	// testing
	assert.Error(t, util.ErrOnly(testExecute(cmdLookup(), "123")), "ko")
}

func TestCmdLookupRandomFailure(t *testing.T) {
	// monkey patching
	defer gomonkey.NewPatches().
		ApplyFunc(spotify.Authenticate, func() (*spotify.Client, error) {
			return &spotify.Client{}, nil
		}).
		ApplyMethod(&spotify.Client{}, "Random", func() error {
			return errors.New("ko")
		}).Reset()

	// testing
	assert.Error(t, util.ErrOnly(testExecute(cmdLookup(), "-r")), "ko")
}

func TestCmdLookupSearchFailure(t *testing.T) {
	_track := &entity.Track{ID: "TestCmdLookupSearchFailure", Title: "Title", Artists: []string{"Artist"}}

	// monkey patching
	defer gomonkey.NewPatches().
		ApplyFunc(spotify.Authenticate, func() (*spotify.Client, error) {
			return &spotify.Client{}, nil
		}).
		ApplyMethod(&spotify.Client{}, "Library", func(_ *spotify.Client, ch ...chan interface{}) error {
			ch[0] <- _track
			ch[1] <- _track
			return nil
		}).
		ApplyFunc(provider.Search, func() ([]*provider.Match, error) {
			return nil, errors.New("ko")
		}).
		ApplyFunc(lyrics.Search, func() (string, error) {
			return "lyrics", nil
		}).
		Reset()

	// testing
	assert.Nil(t, util.ErrOnly(testExecute(cmdLookup(), "-l")))
}

func TestCmdLookupSearchNotFound(t *testing.T) {
	_track := &entity.Track{ID: "TestCmdLookupSearchNotFound", Title: "Title", Artists: []string{"Artist"}}

	// monkey patching
	defer gomonkey.NewPatches().
		ApplyFunc(spotify.Authenticate, func() (*spotify.Client, error) {
			return &spotify.Client{}, nil
		}).
		ApplyMethod(&spotify.Client{}, "Library", func(_ *spotify.Client, ch ...chan interface{}) error {
			ch[0] <- _track
			ch[1] <- _track
			return nil
		}).
		ApplyFunc(provider.Search, func() ([]*provider.Match, error) {
			return []*provider.Match{}, nil
		}).
		ApplyFunc(lyrics.Search, func() (string, error) {
			return "lyrics", nil
		}).
		Reset()

	// testing
	assert.Nil(t, util.ErrOnly(testExecute(cmdLookup(), "-l")))
}

func TestCmdLookupLyricsFailure(t *testing.T) {
	_track := &entity.Track{ID: "TestCmdLookupLyricsFailure", Title: "Title", Artists: []string{"Artist"}}

	// monkey patching
	defer gomonkey.NewPatches().
		ApplyFunc(spotify.Authenticate, func() (*spotify.Client, error) {
			return &spotify.Client{}, nil
		}).
		ApplyMethod(&spotify.Client{}, "Library", func(_ *spotify.Client, ch ...chan interface{}) error {
			ch[0] <- _track
			ch[1] <- _track
			return nil
		}).
		ApplyFunc(provider.Search, func() ([]*provider.Match, error) {
			return []*provider.Match{{URL: "http://localhost/", Score: 0}}, nil
		}).
		ApplyFunc(lyrics.Search, func() (string, error) {
			return "", errors.New("ko")
		}).
		Reset()

	// testing
	assert.Nil(t, util.ErrOnly(testExecute(cmdLookup(), "-l")))
}

func TestCmdLookupLyricsNotFound(t *testing.T) {
	_track := &entity.Track{ID: "TestCmdLookupLyricsNotFound", Title: "Title", Artists: []string{"Artist"}}

	// monkey patching
	defer gomonkey.NewPatches().
		ApplyFunc(spotify.Authenticate, func() (*spotify.Client, error) {
			return &spotify.Client{}, nil
		}).
		ApplyMethod(&spotify.Client{}, "Library", func(_ *spotify.Client, ch ...chan interface{}) error {
			ch[0] <- _track
			ch[1] <- _track
			return nil
		}).
		ApplyFunc(provider.Search, func() ([]*provider.Match, error) {
			return []*provider.Match{{URL: "http://localhost/", Score: 0}}, nil
		}).
		ApplyFunc(lyrics.Search, func() (string, error) {
			return "", nil
		}).
		Reset()

	// testing
	assert.Nil(t, util.ErrOnly(testExecute(cmdLookup(), "-l")))
}
