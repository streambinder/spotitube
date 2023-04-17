package cmd

import (
	"context"
	"errors"
	"os"
	"reflect"
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
	err   = errors.New("ko")
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
	patchSleep := gomonkey.ApplyFunc(time.Sleep, func(time.Duration) {})
	defer patchSleep.Reset()
	patchcmdOpen := gomonkey.ApplyFunc(cmd.Open, func(string, ...string) error { return nil })
	defer patchcmdOpen.Reset()
	patchspotifyAuthenticate := gomonkey.ApplyFunc(spotify.Authenticate, func(...string) (*spotify.Client, error) { return &spotify.Client{}, nil })
	defer patchspotifyAuthenticate.Reset()
	patchspotifyClientLibrary := gomonkey.ApplyMethod(reflect.TypeOf(&spotify.Client{}), "Library",
		func(c *spotify.Client, ch ...chan interface{}) error {
			ch[0] <- track
			return nil
		})
	defer patchspotifyClientLibrary.Reset()
	patchspotifyClientPlaylist := gomonkey.ApplyMethod(reflect.TypeOf(&spotify.Client{}), "Playlist",
		func(c *spotify.Client, id string, ch ...chan interface{}) (*entity.Playlist, error) {
			ch[0] <- track
			return playlist, nil
		})
	defer patchspotifyClientPlaylist.Reset()
	patchspotifyClientAlbum := gomonkey.ApplyMethod(reflect.TypeOf(&spotify.Client{}), "Album",
		func(c *spotify.Client, id string, ch ...chan interface{}) (*entity.Album, error) {
			ch[0] <- track
			return album, nil
		})
	defer patchspotifyClientAlbum.Reset()
	patchspotifyClientTrack := gomonkey.ApplyMethod(reflect.TypeOf(&spotify.Client{}), "Track",
		func(c *spotify.Client, id string, ch ...chan interface{}) (*entity.Track, error) {
			ch[0] <- track
			return track, nil
		})
	defer patchspotifyClientTrack.Reset()
	patchproviderSearch := gomonkey.ApplyFunc(provider.Search, func(*entity.Track) ([]*provider.Match, error) {
		return []*provider.Match{{URL: "http://localhost/", Score: 0}}, nil
	})
	defer patchproviderSearch.Reset()
	patchdownloaderDownload := gomonkey.ApplyFunc(downloader.Download, func(url, path string, ch ...chan []byte) error {
		for _, c := range ch {
			c <- []byte{}
		}
		return nil
	})
	defer patchdownloaderDownload.Reset()
	patchlyricsSearch := gomonkey.ApplyFunc(lyrics.Search, func(*entity.Track) (string, error) {
		return "lyrics", nil
	})
	defer patchlyricsSearch.Reset()
	patchprocessorDo := gomonkey.ApplyFunc(processor.Do, func(*entity.Track) error {
		return nil
	})
	defer patchprocessorDo.Reset()

	// testing
	assert.Nil(t, util.ErrOnly(testExecute("sync")))
	library, err := cmdSync.Flags().GetBool("library")
	assert.Nil(t, err)
	assert.True(t, library)
	assert.Nil(t, util.ErrOnly(testExecute("sync", "-l", "-p", "123", "-a", "123", "-t", "123")))
}

func TestCmdSyncPathFailure(t *testing.T) {
	// monkey patching
	patchosChdir := gomonkey.ApplyFunc(os.Chdir, func(string) error { return err })
	defer patchosChdir.Reset()

	// testing
	assert.EqualError(t, util.ErrOnly(testExecute("sync")), err.Error())
}

func TestCmdSyncAuthFailure(t *testing.T) {
	// monkey patching
	patchtimeSleep := gomonkey.ApplyFunc(time.Sleep, func(time.Duration) {})
	defer patchtimeSleep.Reset()
	patchcmdOpen := gomonkey.ApplyFunc(cmd.Open, func(string, ...string) error { return nil })
	defer patchcmdOpen.Reset()
	patchspotifyAuthenticate := gomonkey.ApplyFunc(spotify.Authenticate, func(...string) (*spotify.Client, error) { return client, err })
	defer patchspotifyAuthenticate.Reset()
	patchspotifyClientLibrary := gomonkey.ApplyMethod(reflect.TypeOf(&spotify.Client{}), "Library",
		func(*spotify.Client, ...chan interface{}) error { return nil })
	defer patchspotifyClientLibrary.Reset()
	patchspotifyClientPlaylist := gomonkey.ApplyMethod(reflect.TypeOf(&spotify.Client{}), "Playlist",
		func(*spotify.Client, string, ...chan interface{}) (*entity.Playlist, error) { return playlist, nil })
	defer patchspotifyClientPlaylist.Reset()
	patchspotifyClientAlbum := gomonkey.ApplyMethod(reflect.TypeOf(&spotify.Client{}), "Album",
		func(*spotify.Client, string, ...chan interface{}) (*entity.Album, error) { return album, nil })
	defer patchspotifyClientAlbum.Reset()
	patchspotifyClientTrack := gomonkey.ApplyMethod(reflect.TypeOf(&spotify.Client{}), "Track",
		func(*spotify.Client, string, ...chan interface{}) (*entity.Track, error) { return track, nil })
	defer patchspotifyClientTrack.Reset()

	// testing
	assert.EqualError(t, util.ErrOnly(testExecute("sync")), err.Error())
}

func TestCmdSyncLibraryFailure(t *testing.T) {
	// monkey patching
	patchtimeSleep := gomonkey.ApplyFunc(time.Sleep, func(time.Duration) {})
	defer patchtimeSleep.Reset()
	patchcmdOpen := gomonkey.ApplyFunc(cmd.Open, func(string, ...string) error { return nil })
	defer patchcmdOpen.Reset()
	patchspotifyAuthenticate := gomonkey.ApplyFunc(spotify.Authenticate, func(...string) (*spotify.Client, error) { return &spotify.Client{}, nil })
	defer patchspotifyAuthenticate.Reset()
	patchspotifyClientLibrary := gomonkey.ApplyMethod(reflect.TypeOf(&spotify.Client{}), "Library",
		func(*spotify.Client, ...chan interface{}) error { return err })
	defer patchspotifyClientLibrary.Reset()
	patchspotifyClientPlaylist := gomonkey.ApplyMethod(reflect.TypeOf(&spotify.Client{}), "Playlist",
		func(*spotify.Client, string, ...chan interface{}) (*entity.Playlist, error) { return playlist, nil })
	defer patchspotifyClientPlaylist.Reset()
	patchspotifyClientAlbum := gomonkey.ApplyMethod(reflect.TypeOf(&spotify.Client{}), "Album",
		func(*spotify.Client, string, ...chan interface{}) (*entity.Album, error) { return album, nil })
	defer patchspotifyClientAlbum.Reset()
	patchspotifyClientTrack := gomonkey.ApplyMethod(reflect.TypeOf(&spotify.Client{}), "Track",
		func(*spotify.Client, string, ...chan interface{}) (*entity.Track, error) { return track, nil })
	defer patchspotifyClientTrack.Reset()

	// testing
	assert.EqualError(t, util.ErrOnly(testExecute("sync")), err.Error())
}

func TestCmdSyncPlaylistFailure(t *testing.T) {
	// monkey patching
	patchtimeSleep := gomonkey.ApplyFunc(time.Sleep, func(time.Duration) {})
	defer patchtimeSleep.Reset()
	patchcmdOpen := gomonkey.ApplyFunc(cmd.Open, func(string, ...string) error { return nil })
	defer patchcmdOpen.Reset()
	patchspotifyAuthenticate := gomonkey.ApplyFunc(spotify.Authenticate, func(...string) (*spotify.Client, error) { return &spotify.Client{}, nil })
	defer patchspotifyAuthenticate.Reset()
	patchspotifyClientLibrary := gomonkey.ApplyMethod(reflect.TypeOf(&spotify.Client{}), "Library",
		func(*spotify.Client, ...chan interface{}) error { return nil })
	defer patchspotifyClientLibrary.Reset()
	patchspotifyClientPlaylist := gomonkey.ApplyMethod(reflect.TypeOf(&spotify.Client{}), "Playlist",
		func(*spotify.Client, string, ...chan interface{}) (*entity.Playlist, error) { return nil, err })
	defer patchspotifyClientPlaylist.Reset()
	patchspotifyClientAlbum := gomonkey.ApplyMethod(reflect.TypeOf(&spotify.Client{}), "Album",
		func(*spotify.Client, string, ...chan interface{}) (*entity.Album, error) { return album, nil })
	defer patchspotifyClientAlbum.Reset()
	patchspotifyClientTrack := gomonkey.ApplyMethod(reflect.TypeOf(&spotify.Client{}), "Track",
		func(*spotify.Client, string, ...chan interface{}) (*entity.Track, error) { return track, nil })
	defer patchspotifyClientTrack.Reset()

	// testing
	assert.EqualError(t, util.ErrOnly(testExecute("sync", "-p", "123")), err.Error())
}

func TestCmdSyncAlbumFailure(t *testing.T) {
	// monkey patching
	patchspotifyAuthenticate := gomonkey.ApplyFunc(spotify.Authenticate, func(...string) (*spotify.Client, error) { return &spotify.Client{}, nil })
	defer patchspotifyAuthenticate.Reset()
	patchcmdOpen := gomonkey.ApplyFunc(cmd.Open, func(string, ...string) error { return nil })
	defer patchcmdOpen.Reset()
	patchspotifyClientLibrary := gomonkey.ApplyMethod(reflect.TypeOf(&spotify.Client{}), "Library",
		func(*spotify.Client, ...chan interface{}) error { return nil })
	defer patchspotifyClientLibrary.Reset()
	patchspotifyClientPlaylist := gomonkey.ApplyMethod(reflect.TypeOf(&spotify.Client{}), "Playlist",
		func(*spotify.Client, string, ...chan interface{}) (*entity.Playlist, error) { return playlist, nil })
	defer patchspotifyClientPlaylist.Reset()
	patchspotifyClientAlbum := gomonkey.ApplyMethod(reflect.TypeOf(&spotify.Client{}), "Album",
		func(*spotify.Client, string, ...chan interface{}) (*entity.Album, error) { return nil, err })
	defer patchspotifyClientAlbum.Reset()
	patchspotifyClientTrack := gomonkey.ApplyMethod(reflect.TypeOf(&spotify.Client{}), "Track",
		func(*spotify.Client, string, ...chan interface{}) (*entity.Track, error) { return track, nil })
	defer patchspotifyClientTrack.Reset()

	// testing
	assert.EqualError(t, util.ErrOnly(testExecute("sync", "-a", "123")), err.Error())
}

func TestCmdSyncTrackFailure(t *testing.T) {
	// monkey patching
	patchtimeSleep := gomonkey.ApplyFunc(time.Sleep, func(time.Duration) {})
	defer patchtimeSleep.Reset()
	patchcmdOpen := gomonkey.ApplyFunc(cmd.Open, func(string, ...string) error { return nil })
	defer patchcmdOpen.Reset()
	patchspotifyAuthenticate := gomonkey.ApplyFunc(spotify.Authenticate, func(...string) (*spotify.Client, error) { return &spotify.Client{}, nil })
	defer patchspotifyAuthenticate.Reset()
	patchspotifyClientLibrary := gomonkey.ApplyMethod(reflect.TypeOf(&spotify.Client{}), "Library",
		func(*spotify.Client, ...chan interface{}) error { return nil })
	defer patchspotifyClientLibrary.Reset()
	patchspotifyClientPlaylist := gomonkey.ApplyMethod(reflect.TypeOf(&spotify.Client{}), "Playlist",
		func(*spotify.Client, string, ...chan interface{}) (*entity.Playlist, error) { return playlist, nil })
	defer patchspotifyClientPlaylist.Reset()
	patchspotifyClientAlbum := gomonkey.ApplyMethod(reflect.TypeOf(&spotify.Client{}), "Album",
		func(*spotify.Client, string, ...chan interface{}) (*entity.Album, error) { return album, nil })
	defer patchspotifyClientAlbum.Reset()
	patchspotifyClientTrack := gomonkey.ApplyMethod(reflect.TypeOf(&spotify.Client{}), "Track",
		func(*spotify.Client, string, ...chan interface{}) (*entity.Track, error) { return nil, err })
	defer patchspotifyClientTrack.Reset()

	// testing
	assert.EqualError(t, util.ErrOnly(testExecute("sync", "-t", "123")), err.Error())
}

func TestCmdSyncDecideFailure(t *testing.T) {
	// monkey patching
	patchtimeSleep := gomonkey.ApplyFunc(time.Sleep, func(time.Duration) {})
	defer patchtimeSleep.Reset()
	patchcmdOpen := gomonkey.ApplyFunc(cmd.Open, func(string, ...string) error { return nil })
	defer patchcmdOpen.Reset()
	patchspotifyAuthenticate := gomonkey.ApplyFunc(spotify.Authenticate, func(...string) (*spotify.Client, error) { return &spotify.Client{}, nil })
	defer patchspotifyAuthenticate.Reset()
	patchspotifyClientLibrary := gomonkey.ApplyMethod(reflect.TypeOf(&spotify.Client{}), "Library",
		func(c *spotify.Client, ch ...chan interface{}) error {
			ch[0] <- track
			return nil
		})
	defer patchspotifyClientLibrary.Reset()
	patchspotifyClientPlaylist := gomonkey.ApplyMethod(reflect.TypeOf(&spotify.Client{}), "Playlist",
		func(*spotify.Client, string, ...chan interface{}) (*entity.Playlist, error) { return playlist, nil })
	defer patchspotifyClientPlaylist.Reset()
	patchspotifyClientAlbum := gomonkey.ApplyMethod(reflect.TypeOf(&spotify.Client{}), "Album",
		func(*spotify.Client, string, ...chan interface{}) (*entity.Album, error) { return album, nil })
	defer patchspotifyClientAlbum.Reset()
	patchspotifyClientTrack := gomonkey.ApplyMethod(reflect.TypeOf(&spotify.Client{}), "Track",
		func(*spotify.Client, string, ...chan interface{}) (*entity.Track, error) { return track, nil })
	defer patchspotifyClientTrack.Reset()
	patchproviderSearch := gomonkey.ApplyFunc(provider.Search, func(*entity.Track) ([]*provider.Match, error) {
		return nil, err
	})
	defer patchproviderSearch.Reset()

	// testing
	assert.EqualError(t, util.ErrOnly(testExecute("sync")), err.Error())
}

func TestCmdSyncCollectFailure(t *testing.T) {
	// monkey patching
	patchtimeSleep := gomonkey.ApplyFunc(time.Sleep, func(time.Duration) {})
	defer patchtimeSleep.Reset()
	patchcmdOpen := gomonkey.ApplyFunc(cmd.Open, func(string, ...string) error { return nil })
	defer patchcmdOpen.Reset()
	patchspotifyAuthenticate := gomonkey.ApplyFunc(spotify.Authenticate, func(...string) (*spotify.Client, error) { return &spotify.Client{}, nil })
	defer patchspotifyAuthenticate.Reset()
	patchspotifyClientLibrary := gomonkey.ApplyMethod(reflect.TypeOf(&spotify.Client{}), "Library",
		func(c *spotify.Client, ch ...chan interface{}) error {
			ch[0] <- track
			return nil
		})
	defer patchspotifyClientLibrary.Reset()
	patchspotifyClientPlaylist := gomonkey.ApplyMethod(reflect.TypeOf(&spotify.Client{}), "Playlist",
		func(*spotify.Client, string, ...chan interface{}) (*entity.Playlist, error) { return playlist, nil })
	defer patchspotifyClientPlaylist.Reset()
	patchspotifyClientAlbum := gomonkey.ApplyMethod(reflect.TypeOf(&spotify.Client{}), "Album",
		func(*spotify.Client, string, ...chan interface{}) (*entity.Album, error) { return album, nil })
	defer patchspotifyClientAlbum.Reset()
	patchspotifyClientTrack := gomonkey.ApplyMethod(reflect.TypeOf(&spotify.Client{}), "Track",
		func(*spotify.Client, string, ...chan interface{}) (*entity.Track, error) { return track, nil })
	defer patchspotifyClientTrack.Reset()
	patchPainter := gomonkey.ApplyFunc(painter, func(track *entity.Track) func(context.Context, chan error) {
		return func(ctx context.Context, ch chan error) {
			ch <- err
		}
	})
	defer patchPainter.Reset()
	patchdownloaderDownload := gomonkey.ApplyFunc(downloader.Download, func(url, path string, ch ...chan []byte) error {
		for _, c := range ch {
			c <- []byte{}
		}
		return nil
	})
	defer patchdownloaderDownload.Reset()
	patchlyricsSearch := gomonkey.ApplyFunc(lyrics.Search, func(*entity.Track) (string, error) {
		return "", nil
	})
	defer patchlyricsSearch.Reset()

	// testing
	assert.EqualError(t, util.ErrOnly(testExecute("sync")), err.Error())
}

func TestCmdSyncDownloadFailure(t *testing.T) {
	// monkey patching
	patchtimeSleep := gomonkey.ApplyFunc(time.Sleep, func(time.Duration) {})
	defer patchtimeSleep.Reset()
	patchcmdOpen := gomonkey.ApplyFunc(cmd.Open, func(string, ...string) error { return nil })
	defer patchcmdOpen.Reset()
	patchspotifyAuthenticate := gomonkey.ApplyFunc(spotify.Authenticate, func(...string) (*spotify.Client, error) { return &spotify.Client{}, nil })
	defer patchspotifyAuthenticate.Reset()
	patchspotifyClientLibrary := gomonkey.ApplyMethod(reflect.TypeOf(&spotify.Client{}), "Library",
		func(c *spotify.Client, ch ...chan interface{}) error {
			ch[0] <- track
			return nil
		})
	defer patchspotifyClientLibrary.Reset()
	patchspotifyClientPlaylist := gomonkey.ApplyMethod(reflect.TypeOf(&spotify.Client{}), "Playlist",
		func(*spotify.Client, string, ...chan interface{}) (*entity.Playlist, error) { return playlist, nil })
	defer patchspotifyClientPlaylist.Reset()
	patchspotifyClientAlbum := gomonkey.ApplyMethod(reflect.TypeOf(&spotify.Client{}), "Album",
		func(*spotify.Client, string, ...chan interface{}) (*entity.Album, error) { return album, nil })
	defer patchspotifyClientAlbum.Reset()
	patchspotifyClientTrack := gomonkey.ApplyMethod(reflect.TypeOf(&spotify.Client{}), "Track",
		func(*spotify.Client, string, ...chan interface{}) (*entity.Track, error) { return track, nil })
	defer patchspotifyClientTrack.Reset()
	patchproviderSearch := gomonkey.ApplyFunc(provider.Search, func(*entity.Track) ([]*provider.Match, error) {
		return []*provider.Match{{URL: "http://localhost/", Score: 0}}, nil
	})
	defer patchproviderSearch.Reset()
	patchdownloaderDownload := gomonkey.ApplyFunc(downloader.Download, func(url, path string, ch ...chan []byte) error {
		for _, c := range ch {
			c <- []byte{}
		}
		return err
	})
	defer patchdownloaderDownload.Reset()
	patchlyricsSearch := gomonkey.ApplyFunc(lyrics.Search, func(*entity.Track) (string, error) {
		return "", nil
	})
	defer patchlyricsSearch.Reset()
	// testing
	assert.EqualError(t, util.ErrOnly(testExecute("sync")), err.Error())
}

func TestCmdSyncLyricsFailure(t *testing.T) {
	// monkey patching
	patchtimeSleep := gomonkey.ApplyFunc(time.Sleep, func(time.Duration) {})
	defer patchtimeSleep.Reset()
	patchcmdOpen := gomonkey.ApplyFunc(cmd.Open, func(string, ...string) error { return nil })
	defer patchcmdOpen.Reset()
	patchspotifyAuthenticate := gomonkey.ApplyFunc(spotify.Authenticate, func(...string) (*spotify.Client, error) { return &spotify.Client{}, nil })
	defer patchspotifyAuthenticate.Reset()
	patchspotifyClientLibrary := gomonkey.ApplyMethod(reflect.TypeOf(&spotify.Client{}), "Library",
		func(c *spotify.Client, ch ...chan interface{}) error {
			ch[0] <- track
			return nil
		})
	defer patchspotifyClientLibrary.Reset()
	patchspotifyClientPlaylist := gomonkey.ApplyMethod(reflect.TypeOf(&spotify.Client{}), "Playlist",
		func(*spotify.Client, string, ...chan interface{}) (*entity.Playlist, error) { return playlist, nil })
	defer patchspotifyClientPlaylist.Reset()
	patchspotifyClientAlbum := gomonkey.ApplyMethod(reflect.TypeOf(&spotify.Client{}), "Album",
		func(*spotify.Client, string, ...chan interface{}) (*entity.Album, error) { return album, nil })
	defer patchspotifyClientAlbum.Reset()
	patchspotifyClientTrack := gomonkey.ApplyMethod(reflect.TypeOf(&spotify.Client{}), "Track",
		func(*spotify.Client, string, ...chan interface{}) (*entity.Track, error) { return track, nil })
	defer patchspotifyClientTrack.Reset()
	patchproviderSearch := gomonkey.ApplyFunc(provider.Search, func(*entity.Track) ([]*provider.Match, error) {
		return []*provider.Match{{URL: "http://localhost/", Score: 0}}, nil
	})
	defer patchproviderSearch.Reset()
	patchdownloaderDownload := gomonkey.ApplyFunc(downloader.Download, func(url, path string, ch ...chan []byte) error {
		for _, c := range ch {
			c <- []byte{}
		}
		return nil
	})
	defer patchdownloaderDownload.Reset()
	patchlyricsSearch := gomonkey.ApplyFunc(lyrics.Search, func(*entity.Track) (string, error) {
		return "", err
	})
	defer patchlyricsSearch.Reset()

	// testing
	assert.EqualError(t, util.ErrOnly(testExecute("sync")), err.Error())
}

func TestCmdSyncProcessorFailure(t *testing.T) {
	// monkey patching
	patchtimeSleep := gomonkey.ApplyFunc(time.Sleep, func(time.Duration) {})
	defer patchtimeSleep.Reset()
	patchcmdOpen := gomonkey.ApplyFunc(cmd.Open, func(string, ...string) error { return nil })
	defer patchcmdOpen.Reset()
	patchspotifyAuthenticate := gomonkey.ApplyFunc(spotify.Authenticate, func(...string) (*spotify.Client, error) { return &spotify.Client{}, nil })
	defer patchspotifyAuthenticate.Reset()
	patchspotifyClientLibrary := gomonkey.ApplyMethod(reflect.TypeOf(&spotify.Client{}), "Library",
		func(c *spotify.Client, ch ...chan interface{}) error {
			ch[0] <- track
			return nil
		})
	defer patchspotifyClientLibrary.Reset()
	patchspotifyClientPlaylist := gomonkey.ApplyMethod(reflect.TypeOf(&spotify.Client{}), "Playlist",
		func(*spotify.Client, string, ...chan interface{}) (*entity.Playlist, error) { return playlist, nil })
	defer patchspotifyClientPlaylist.Reset()
	patchspotifyClientAlbum := gomonkey.ApplyMethod(reflect.TypeOf(&spotify.Client{}), "Album",
		func(*spotify.Client, string, ...chan interface{}) (*entity.Album, error) { return album, nil })
	defer patchspotifyClientAlbum.Reset()
	patchspotifyClientTrack := gomonkey.ApplyMethod(reflect.TypeOf(&spotify.Client{}), "Track",
		func(*spotify.Client, string, ...chan interface{}) (*entity.Track, error) { return track, nil })
	defer patchspotifyClientTrack.Reset()
	patchproviderSearch := gomonkey.ApplyFunc(provider.Search, func(*entity.Track) ([]*provider.Match, error) {
		return []*provider.Match{{URL: "http://localhost/", Score: 0}}, nil
	})
	defer patchproviderSearch.Reset()
	patchdownloaderDownload := gomonkey.ApplyFunc(downloader.Download, func(url, path string, ch ...chan []byte) error {
		for _, c := range ch {
			c <- []byte{}
		}
		return nil
	})
	defer patchdownloaderDownload.Reset()
	patchlyricsSearch := gomonkey.ApplyFunc(lyrics.Search, func(*entity.Track) (string, error) {
		return "lyrics", nil
	})
	defer patchlyricsSearch.Reset()
	patchprocessorDo := gomonkey.ApplyFunc(processor.Do, func(*entity.Track) error {
		return err
	})
	defer patchprocessorDo.Reset()

	// testing
	assert.EqualError(t, util.ErrOnly(testExecute("sync")), err.Error())
}
