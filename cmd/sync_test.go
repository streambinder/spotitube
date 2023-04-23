package cmd

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/streambinder/spotitube/downloader"
	"github.com/streambinder/spotitube/entity"
	"github.com/streambinder/spotitube/lyrics"
	"github.com/streambinder/spotitube/processor"
	"github.com/streambinder/spotitube/provider"
	"github.com/streambinder/spotitube/spotify"
	"github.com/streambinder/spotitube/util"
	"github.com/streambinder/spotitube/util/cmd"
	"github.com/stretchr/testify/assert"
)

var (
	track = &entity.Track{
		ID:          "123",
		Title:       "Title",
		Artists:     []string{"Artist"},
		Album:       "Album",
		Artwork:     entity.Artwork{URL: "http://ima.ge"},
		Duration:    180,
		Lyrics:      "",
		Number:      1,
		Year:        "1970",
		UpstreamURL: "",
	}
	playlist = &entity.Playlist{
		ID:     "123",
		Name:   "Playlist",
		Owner:  "Owner",
		Tracks: []*entity.Track{track},
	}
	album = &entity.Album{
		ID:      "123",
		Name:    "Album",
		Artists: []string{"Artist"},
		Tracks:  []*entity.Track{track},
	}
)

func TestCmdSync(t *testing.T) {
	// monkey patching
	defer gomonkey.NewPatches().
		ApplyFunc(time.Sleep, func() {}).
		ApplyFunc(cmd.Open, func() error { return nil }).
		ApplyFunc(spotify.Authenticate, func() (*spotify.Client, error) {
			return &spotify.Client{}, nil
		}).
		ApplyMethod(&spotify.Client{}, "Library", func(_ *spotify.Client, ch ...chan interface{}) error {
			ch[0] <- track
			return nil
		}).
		ApplyMethod(&spotify.Client{}, "Playlist", func(_ *spotify.Client, _ string, ch ...chan interface{}) (*entity.Playlist, error) {
			ch[0] <- track
			return playlist, nil
		}).
		ApplyMethod(&spotify.Client{}, "Album", func(_ *spotify.Client, _ string, ch ...chan interface{}) (*entity.Album, error) {
			ch[0] <- track
			return album, nil
		}).
		ApplyMethod(&spotify.Client{}, "Track", func(_ *spotify.Client, _ string, ch ...chan interface{}) (*entity.Track, error) {
			ch[0] <- track
			return track, nil
		}).
		ApplyFunc(provider.Search, func() ([]*provider.Match, error) {
			return []*provider.Match{{URL: "http://localhost/", Score: 0}}, nil
		}).
		ApplyFunc(downloader.Download, func(_, _ string, _ processor.Processor, ch ...chan []byte) error {
			for _, c := range ch {
				c <- []byte{}
			}
			return nil
		}).
		ApplyFunc(lyrics.Search, func() (string, error) {
			return "lyrics", nil
		}).
		ApplyFunc(processor.Do, func() error {
			return nil
		}).
		Reset()

	// testing
	assert.Nil(t, util.ErrOnly(testExecute("sync")))
	library, err := cmdSync.Flags().GetBool("library")
	assert.Nil(t, err)
	assert.True(t, library)
	assert.Nil(t, util.ErrOnly(testExecute("sync", "-l", "-p", "123", "-a", "123", "-t", "123")))
}

func TestCmdSyncPathFailure(t *testing.T) {
	// monkey patching
	defer gomonkey.ApplyFunc(os.Chdir, func() error {
		return errors.New("ko")
	}).Reset()

	// testing
	assert.EqualError(t, util.ErrOnly(testExecute("sync")), "ko")
}

func TestCmdSyncAuthFailure(t *testing.T) {
	// monkey patching
	defer gomonkey.NewPatches().
		ApplyFunc(time.Sleep, func() {}).
		ApplyFunc(cmd.Open, func() error {
			return nil
		}).
		ApplyFunc(spotify.Authenticate, func() (*spotify.Client, error) {
			return client, errors.New("ko")
		}).
		ApplyMethod(&spotify.Client{}, "Library", func() error {
			return nil
		}).
		ApplyMethod(&spotify.Client{}, "Playlist", func() (*entity.Playlist, error) {
			return playlist, nil
		}).
		ApplyMethod(&spotify.Client{}, "Album", func() (*entity.Album, error) {
			return album, nil
		}).
		ApplyMethod(&spotify.Client{}, "Track", func() (*entity.Track, error) {
			return track, nil
		}).
		Reset()

	// testing
	assert.EqualError(t, util.ErrOnly(testExecute("sync")), "ko")
}

func TestCmdSyncLibraryFailure(t *testing.T) {
	// monkey patching
	defer gomonkey.NewPatches().
		ApplyFunc(time.Sleep, func() {}).
		ApplyFunc(cmd.Open, func() error {
			return nil
		}).
		ApplyFunc(spotify.Authenticate, func() (*spotify.Client, error) {
			return &spotify.Client{}, nil
		}).
		ApplyMethod(&spotify.Client{}, "Library", func() error {
			return errors.New("ko")
		}).
		ApplyMethod(&spotify.Client{}, "Playlist", func() (*entity.Playlist, error) {
			return playlist, nil
		}).
		ApplyMethod(&spotify.Client{}, "Album", func() (*entity.Album, error) {
			return album, nil
		}).
		ApplyMethod(&spotify.Client{}, "Track", func() (*entity.Track, error) {
			return track, nil
		}).
		Reset()

	// testing
	assert.EqualError(t, util.ErrOnly(testExecute("sync")), "ko")
}

func TestCmdSyncPlaylistFailure(t *testing.T) {
	// monkey patching
	defer gomonkey.NewPatches().
		ApplyFunc(time.Sleep, func() {}).
		ApplyFunc(cmd.Open, func() error {
			return nil
		}).
		ApplyFunc(spotify.Authenticate, func() (*spotify.Client, error) {
			return &spotify.Client{}, nil
		}).
		ApplyMethod(&spotify.Client{}, "Library", func() error {
			return nil
		}).
		ApplyMethod(&spotify.Client{}, "Playlist", func() (*entity.Playlist, error) {
			return nil, errors.New("ko")
		}).
		ApplyMethod(&spotify.Client{}, "Album", func() (*entity.Album, error) {
			return album, nil
		}).
		ApplyMethod(&spotify.Client{}, "Track", func() (*entity.Track, error) {
			return track, nil
		}).
		Reset()

	// testing
	assert.EqualError(t, util.ErrOnly(testExecute("sync", "-p", "123")), "ko")
}

func TestCmdSyncAlbumFailure(t *testing.T) {
	// monkey patching
	defer gomonkey.NewPatches().
		ApplyFunc(time.Sleep, func() {}).
		ApplyFunc(cmd.Open, func() error {
			return nil
		}).
		ApplyFunc(spotify.Authenticate, func() (*spotify.Client, error) {
			return &spotify.Client{}, nil
		}).
		ApplyMethod(&spotify.Client{}, "Library", func() error {
			return nil
		}).
		ApplyMethod(&spotify.Client{}, "Playlist", func() (*entity.Playlist, error) {
			return playlist, nil
		}).
		ApplyMethod(&spotify.Client{}, "Album", func() (*entity.Album, error) {
			return nil, errors.New("ko")
		}).
		ApplyMethod(&spotify.Client{}, "Track", func() (*entity.Track, error) {
			return track, nil
		}).
		Reset()

	// testing
	assert.EqualError(t, util.ErrOnly(testExecute("sync", "-a", "123")), "ko")
}

func TestCmdSyncTrackFailure(t *testing.T) {
	// monkey patching
	defer gomonkey.NewPatches().
		ApplyFunc(time.Sleep, func() {}).
		ApplyFunc(cmd.Open, func() error {
			return nil
		}).
		ApplyFunc(spotify.Authenticate, func() (*spotify.Client, error) {
			return &spotify.Client{}, nil
		}).
		ApplyMethod(&spotify.Client{}, "Library", func() error {
			return nil
		}).
		ApplyMethod(&spotify.Client{}, "Playlist", func() (*entity.Playlist, error) {
			return playlist, nil
		}).
		ApplyMethod(&spotify.Client{}, "Album", func() (*entity.Album, error) {
			return album, nil
		}).
		ApplyMethod(&spotify.Client{}, "Track", func() (*entity.Track, error) {
			return nil, errors.New("ko")
		}).
		Reset()

	// testing
	assert.EqualError(t, util.ErrOnly(testExecute("sync", "-t", "123")), "ko")
}

func TestCmdSyncDecideFailure(t *testing.T) {
	// monkey patching
	defer gomonkey.NewPatches().
		ApplyFunc(time.Sleep, func() {}).
		ApplyFunc(cmd.Open, func() error {
			return nil
		}).
		ApplyFunc(spotify.Authenticate, func() (*spotify.Client, error) {
			return &spotify.Client{}, nil
		}).
		ApplyMethod(&spotify.Client{}, "Library",
			func(_ *spotify.Client, ch ...chan interface{}) error {
				ch[0] <- track
				return nil
			}).
		ApplyMethod(&spotify.Client{}, "Playlist", func() (*entity.Playlist, error) {
			return playlist, nil
		}).
		ApplyMethod(&spotify.Client{}, "Album", func() (*entity.Album, error) {
			return album, nil
		}).
		ApplyMethod(&spotify.Client{}, "Track", func() (*entity.Track, error) {
			return track, nil
		}).
		ApplyFunc(provider.Search, func(*entity.Track) ([]*provider.Match, error) {
			return nil, errors.New("ko")
		}).
		Reset()

	// testing
	assert.EqualError(t, util.ErrOnly(testExecute("sync")), "ko")
}

func TestCmdSyncCollectFailure(t *testing.T) {
	// monkey patching
	defer gomonkey.NewPatches().
		ApplyFunc(time.Sleep, func() {}).
		ApplyFunc(cmd.Open, func() error {
			return nil
		}).
		ApplyFunc(spotify.Authenticate, func() (*spotify.Client, error) {
			return &spotify.Client{}, nil
		}).
		ApplyMethod(&spotify.Client{}, "Library",
			func(_ *spotify.Client, ch ...chan interface{}) error {
				ch[0] <- track
				return nil
			}).
		ApplyMethod(&spotify.Client{}, "Playlist", func() (*entity.Playlist, error) {
			return playlist, nil
		}).
		ApplyMethod(&spotify.Client{}, "Album", func() (*entity.Album, error) {
			return album, nil
		}).
		ApplyMethod(&spotify.Client{}, "Track", func() (*entity.Track, error) {
			return track, nil
		}).
		ApplyFunc(provider.Search, func(*entity.Track) ([]*provider.Match, error) {
			return []*provider.Match{{URL: "http://localhost/", Score: 0}}, nil
		}).
		ApplyFunc(painter, func(track *entity.Track) func(context.Context, chan error) {
			return func(_ context.Context, ch chan error) {
				ch <- errors.New("ko")
			}
		}).
		ApplyFunc(downloader.Download, func(_, _ string, _ processor.Processor, ch ...chan []byte) error {
			for _, c := range ch {
				c <- []byte{}
			}
			return nil
		}).
		ApplyFunc(lyrics.Search, func() (string, error) {
			return "", nil
		}).
		Reset()

	// testing
	assert.EqualError(t, util.ErrOnly(testExecute("sync")), "ko")
}

func TestCmdSyncDownloadFailure(t *testing.T) {
	// monkey patching
	defer gomonkey.NewPatches().
		ApplyFunc(time.Sleep, func() {}).
		ApplyFunc(cmd.Open, func() error {
			return nil
		}).
		ApplyFunc(spotify.Authenticate, func() (*spotify.Client, error) {
			return &spotify.Client{}, nil
		}).
		ApplyMethod(&spotify.Client{}, "Library", func(_ *spotify.Client, ch ...chan interface{}) error {
			ch[0] <- track
			return nil
		}).
		ApplyMethod(&spotify.Client{}, "Playlist", func() (*entity.Playlist, error) {
			return playlist, nil
		}).
		ApplyMethod(&spotify.Client{}, "Album", func() (*entity.Album, error) {
			return album, nil
		}).
		ApplyMethod(&spotify.Client{}, "Track", func() (*entity.Track, error) {
			return track, nil
		}).
		ApplyFunc(provider.Search, func(*entity.Track) ([]*provider.Match, error) {
			return []*provider.Match{{URL: "http://localhost/", Score: 0}}, nil
		}).
		ApplyFunc(downloader.Download, func(_, _ string, _ processor.Processor, ch ...chan []byte) error {
			for _, c := range ch {
				c <- []byte{}
			}
			return errors.New("ko")
		}).
		ApplyFunc(lyrics.Search, func() (string, error) {
			return "", nil
		}).
		Reset()

	// testing
	assert.EqualError(t, util.ErrOnly(testExecute("sync")), "ko")
}

func TestCmdSyncLyricsFailure(t *testing.T) {
	// monkey patching
	defer gomonkey.NewPatches().
		ApplyFunc(time.Sleep, func() {}).
		ApplyFunc(cmd.Open, func() error {
			return nil
		}).
		ApplyFunc(spotify.Authenticate, func() (*spotify.Client, error) {
			return &spotify.Client{}, nil
		}).
		ApplyMethod(&spotify.Client{}, "Library", func(_ *spotify.Client, ch ...chan interface{}) error {
			ch[0] <- track
			return nil
		}).
		ApplyMethod(&spotify.Client{}, "Playlist", func() (*entity.Playlist, error) {
			return playlist, nil
		}).
		ApplyMethod(&spotify.Client{}, "Album", func() (*entity.Album, error) {
			return album, nil
		}).
		ApplyMethod(&spotify.Client{}, "Track", func() (*entity.Track, error) {
			return track, nil
		}).
		ApplyFunc(provider.Search, func() ([]*provider.Match, error) {
			return []*provider.Match{{URL: "http://localhost/", Score: 0}}, nil
		}).
		ApplyFunc(downloader.Download, func(_, _ string, _ processor.Processor, ch ...chan []byte) error {
			for _, c := range ch {
				c <- []byte{}
			}
			return nil
		}).
		ApplyFunc(lyrics.Search, func() (string, error) {
			return "", errors.New("ko")
		}).
		Reset()

	// testing
	assert.EqualError(t, util.ErrOnly(testExecute("sync")), "ko")
}

func TestCmdSyncProcessorFailure(t *testing.T) {
	// monkey patching
	defer gomonkey.NewPatches().
		ApplyFunc(time.Sleep, func() {}).
		ApplyFunc(cmd.Open, func() error {
			return nil
		}).
		ApplyFunc(spotify.Authenticate, func() (*spotify.Client, error) {
			return &spotify.Client{}, nil
		}).
		ApplyMethod(&spotify.Client{}, "Library", func(_ *spotify.Client, ch ...chan interface{}) error {
			ch[0] <- track
			return nil
		}).
		ApplyMethod(&spotify.Client{}, "Playlist", func() (*entity.Playlist, error) {
			return playlist, nil
		}).
		ApplyMethod(&spotify.Client{}, "Album", func() (*entity.Album, error) {
			return album, nil
		}).
		ApplyMethod(&spotify.Client{}, "Track", func() (*entity.Track, error) {
			return track, nil
		}).
		ApplyFunc(provider.Search, func() ([]*provider.Match, error) {
			return []*provider.Match{{URL: "http://localhost/", Score: 0}}, nil
		}).
		ApplyFunc(downloader.Download, func(_, _ string, _ processor.Processor, ch ...chan []byte) error {
			for _, c := range ch {
				c <- []byte{}
			}
			return nil
		}).
		ApplyFunc(lyrics.Search, func() (string, error) {
			return "lyrics", nil
		}).
		ApplyFunc(processor.Do, func() error {
			return errors.New("ko")
		}).
		Reset()

	// testing
	assert.EqualError(t, util.ErrOnly(testExecute("sync")), "ko")
}
