package cmd

import (
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

var track = entity.Track{ID: "123"}

func init() {
	monkey.Patch(time.Sleep, func(time.Duration) {})
	monkey.Patch(cmd.Open, func(string, ...string) error { return nil })
	monkey.Patch(spotify.Authenticate, func(...string) (*spotify.Client, error) { return &spotify.Client{}, nil })
	monkey.PatchInstanceMethod(reflect.TypeOf(&spotify.Client{}), "Playlist",
		func(client *spotify.Client, id string, channels ...chan *entity.Track) (*entity.Playlist, error) {
			channels[0] <- &track
			return &entity.Playlist{}, nil
		})
}

func TestCmdSync(t *testing.T) {
	err := util.ErrOnly(testExecute("sync"))
	assert.Nil(t, err)
	library, err := cmdSync.Flags().GetBool("library")
	assert.Nil(t, err)
	assert.True(t, library)
}

func TestCmdSyncAuthFailure(t *testing.T) {
	monkey.PatchInstanceMethod(reflect.TypeOf(&spotify.Client{}), "Playlist",
		func(*spotify.Client, string, ...chan *entity.Track) (*entity.Playlist, error) {
			return nil, errors.New("failure")
		})
	assert.EqualError(t, util.ErrOnly(testExecute("sync")), "failure")
}

func TestCmdSyncPlaylistFailure(t *testing.T) {
	monkey.Patch(spotify.Authenticate, func(...string) (*spotify.Client, error) { return client, errors.New("failure") })
	assert.EqualError(t, util.ErrOnly(testExecute("sync")), "failure")
}
