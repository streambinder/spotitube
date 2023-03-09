package cmd

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"bou.ke/monkey"
	"github.com/streambinder/spotitube/entity"
	"github.com/streambinder/spotitube/spotify"
	"github.com/streambinder/spotitube/util"
	"github.com/streambinder/spotitube/util/cmd"
	"github.com/stretchr/testify/assert"
)

var (
	err   = errors.New("ko")
	track = &entity.Track{
		ID:         "123",
		Title:      "Title",
		Artists:    []string{"Artist"},
		Album:      "Album",
		ArtworkURL: "http://ima.ge",
		Duration:   180,
		Number:     1,
		Year:       "1970",
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

func init() {
	monkey.Patch(time.Sleep, func(time.Duration) {})
	monkey.Patch(cmd.Open, func(string, ...string) error { return nil })
}

func TestCmdSync(t *testing.T) {
	// monkey patching
	monkey.Patch(spotify.Authenticate, func(...string) (*spotify.Client, error) { return &spotify.Client{}, nil })
	monkey.PatchInstanceMethod(reflect.TypeOf(&spotify.Client{}), "Library",
		func(c *spotify.Client, ch ...chan interface{}) error {
			ch[0] <- track
			return nil
		})
	monkey.PatchInstanceMethod(reflect.TypeOf(&spotify.Client{}), "Playlist",
		func(c *spotify.Client, id string, ch ...chan interface{}) (*entity.Playlist, error) {
			ch[0] <- track
			return playlist, nil
		})
	monkey.PatchInstanceMethod(reflect.TypeOf(&spotify.Client{}), "Album",
		func(c *spotify.Client, id string, ch ...chan interface{}) (*entity.Album, error) {
			ch[0] <- track
			return album, nil
		})
	monkey.PatchInstanceMethod(reflect.TypeOf(&spotify.Client{}), "Track",
		func(c *spotify.Client, id string, ch ...chan interface{}) (*entity.Track, error) {
			ch[0] <- track
			return track, nil
		})

	// testing
	assert.Nil(t, util.ErrOnly(testExecute("sync")))
	library, err := cmdSync.Flags().GetBool("library")
	assert.Nil(t, err)
	assert.True(t, library)
	assert.Nil(t, util.ErrOnly(testExecute("sync", "-l", "-p", "123", "-a", "123", "-t", "123")))
}

func TestCmdSyncAuthFailure(t *testing.T) {
	// monkey patching
	monkey.Patch(spotify.Authenticate, func(...string) (*spotify.Client, error) { return client, err })
	monkey.PatchInstanceMethod(reflect.TypeOf(&spotify.Client{}), "Library",
		func(*spotify.Client, ...chan interface{}) error { return nil })
	monkey.PatchInstanceMethod(reflect.TypeOf(&spotify.Client{}), "Playlist",
		func(*spotify.Client, string, ...chan interface{}) (*entity.Playlist, error) { return playlist, nil })
	monkey.PatchInstanceMethod(reflect.TypeOf(&spotify.Client{}), "Album",
		func(*spotify.Client, string, ...chan interface{}) (*entity.Album, error) { return album, nil })
	monkey.PatchInstanceMethod(reflect.TypeOf(&spotify.Client{}), "Track",
		func(*spotify.Client, string, ...chan interface{}) (*entity.Track, error) { return track, nil })

	// testing
	assert.EqualError(t, util.ErrOnly(testExecute("sync")), err.Error())
}

func TestCmdSyncLibraryFailure(t *testing.T) {
	// monkey patching
	monkey.Patch(spotify.Authenticate, func(...string) (*spotify.Client, error) { return &spotify.Client{}, nil })
	monkey.PatchInstanceMethod(reflect.TypeOf(&spotify.Client{}), "Library",
		func(*spotify.Client, ...chan interface{}) error { return err })
	monkey.PatchInstanceMethod(reflect.TypeOf(&spotify.Client{}), "Playlist",
		func(*spotify.Client, string, ...chan interface{}) (*entity.Playlist, error) { return playlist, nil })
	monkey.PatchInstanceMethod(reflect.TypeOf(&spotify.Client{}), "Album",
		func(*spotify.Client, string, ...chan interface{}) (*entity.Album, error) { return album, nil })
	monkey.PatchInstanceMethod(reflect.TypeOf(&spotify.Client{}), "Track",
		func(*spotify.Client, string, ...chan interface{}) (*entity.Track, error) { return track, nil })

	// testing
	assert.EqualError(t, util.ErrOnly(testExecute("sync")), err.Error())
}

func TestCmdSyncPlaylistFailure(t *testing.T) {
	// monkey patching
	monkey.Patch(spotify.Authenticate, func(...string) (*spotify.Client, error) { return &spotify.Client{}, nil })
	monkey.PatchInstanceMethod(reflect.TypeOf(&spotify.Client{}), "Library",
		func(*spotify.Client, ...chan interface{}) error { return nil })
	monkey.PatchInstanceMethod(reflect.TypeOf(&spotify.Client{}), "Playlist",
		func(*spotify.Client, string, ...chan interface{}) (*entity.Playlist, error) { return nil, err })
	monkey.PatchInstanceMethod(reflect.TypeOf(&spotify.Client{}), "Album",
		func(*spotify.Client, string, ...chan interface{}) (*entity.Album, error) { return album, nil })
	monkey.PatchInstanceMethod(reflect.TypeOf(&spotify.Client{}), "Track",
		func(*spotify.Client, string, ...chan interface{}) (*entity.Track, error) { return track, nil })

	// testing
	assert.EqualError(t, util.ErrOnly(testExecute("sync", "-p", "123")), err.Error())
}

func TestCmdSyncAlbumFailure(t *testing.T) {
	// monkey patching
	monkey.Patch(spotify.Authenticate, func(...string) (*spotify.Client, error) { return &spotify.Client{}, nil })
	monkey.PatchInstanceMethod(reflect.TypeOf(&spotify.Client{}), "Library",
		func(*spotify.Client, ...chan interface{}) error { return nil })
	monkey.PatchInstanceMethod(reflect.TypeOf(&spotify.Client{}), "Playlist",
		func(*spotify.Client, string, ...chan interface{}) (*entity.Playlist, error) { return playlist, nil })
	monkey.PatchInstanceMethod(reflect.TypeOf(&spotify.Client{}), "Album",
		func(*spotify.Client, string, ...chan interface{}) (*entity.Album, error) { return nil, err })
	monkey.PatchInstanceMethod(reflect.TypeOf(&spotify.Client{}), "Track",
		func(*spotify.Client, string, ...chan interface{}) (*entity.Track, error) { return track, nil })

	// testing
	assert.EqualError(t, util.ErrOnly(testExecute("sync", "-a", "123")), err.Error())
}

func TestCmdSyncTrackFailure(t *testing.T) {
	// monkey patching
	monkey.Patch(spotify.Authenticate, func(...string) (*spotify.Client, error) { return &spotify.Client{}, nil })
	monkey.PatchInstanceMethod(reflect.TypeOf(&spotify.Client{}), "Library",
		func(*spotify.Client, ...chan interface{}) error { return nil })
	monkey.PatchInstanceMethod(reflect.TypeOf(&spotify.Client{}), "Playlist",
		func(*spotify.Client, string, ...chan interface{}) (*entity.Playlist, error) { return playlist, nil })
	monkey.PatchInstanceMethod(reflect.TypeOf(&spotify.Client{}), "Album",
		func(*spotify.Client, string, ...chan interface{}) (*entity.Album, error) { return album, nil })
	monkey.PatchInstanceMethod(reflect.TypeOf(&spotify.Client{}), "Track",
		func(*spotify.Client, string, ...chan interface{}) (*entity.Track, error) { return nil, err })

	// testing
	assert.EqualError(t, util.ErrOnly(testExecute("sync", "-t", "123")), err.Error())
}

func TestCmdSyncCollectFailure(t *testing.T) {
	// monkey patching
	monkey.Patch(spotify.Authenticate, func(...string) (*spotify.Client, error) { return &spotify.Client{}, nil })
	monkey.PatchInstanceMethod(reflect.TypeOf(&spotify.Client{}), "Library",
		func(c *spotify.Client, ch ...chan interface{}) error {
			ch[0] <- track
			return nil
		})
	monkey.PatchInstanceMethod(reflect.TypeOf(&spotify.Client{}), "Playlist",
		func(*spotify.Client, string, ...chan interface{}) (*entity.Playlist, error) { return playlist, nil })
	monkey.PatchInstanceMethod(reflect.TypeOf(&spotify.Client{}), "Album",
		func(*spotify.Client, string, ...chan interface{}) (*entity.Album, error) { return album, nil })
	monkey.PatchInstanceMethod(reflect.TypeOf(&spotify.Client{}), "Track",
		func(*spotify.Client, string, ...chan interface{}) (*entity.Track, error) { return track, nil })
	monkey.Patch(painter, func(track *entity.Track) func(context.Context, chan error) {
		return func(ctx context.Context, ch chan error) {
			ch <- err
		}
	})

	// testing
	assert.EqualError(t, util.ErrOnly(testExecute("sync")), err.Error())
}
