package cmd

import (
	"errors"
	"os"
	"testing"
	"time"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/streambinder/id3v2-sylt"
	"github.com/streambinder/spotitube/downloader"
	"github.com/streambinder/spotitube/entity"
	"github.com/streambinder/spotitube/entity/id3"
	"github.com/streambinder/spotitube/entity/index"
	"github.com/streambinder/spotitube/entity/playlist"
	"github.com/streambinder/spotitube/lyrics"
	"github.com/streambinder/spotitube/processor"
	"github.com/streambinder/spotitube/provider"
	"github.com/streambinder/spotitube/spotify"
	"github.com/streambinder/spotitube/util"
	"github.com/streambinder/spotitube/util/cmd"
	"github.com/stretchr/testify/assert"
)

func BenchmarkSync(b *testing.B) {
	for i := 0; i < b.N; i++ {
		TestCmdSync(&testing.T{})
	}
}

func cleanup() {
	indexData = index.New()
}

func TestCmdSync(t *testing.T) {
	t.Cleanup(cleanup)

	var (
		_track         = &entity.Track{ID: "TestCmdSync", Title: "Title", Artists: []string{"Artist"}}
		_trackNotFound = &entity.Track{ID: "TestCmdSyncNotFound", Title: "Title Not Found", Artists: []string{"Artist"}}
		_playlist      = &playlist.Playlist{Tracks: []*entity.Track{_track, _trackNotFound}}
		_album         = &entity.Album{Tracks: []*entity.Track{_track}}
	)

	// monkey patching
	defer gomonkey.NewPatches().
		ApplyFunc(cmd.ValidateEnvironment, func() error { return nil }).
		ApplyFunc(time.Sleep, func() {}).
		ApplyFunc(cmd.Open, func() error { return nil }).
		ApplyMethod(&index.Index{}, "Build", func() error {
			return nil
		}).
		ApplyFunc(spotify.Authenticate, func() (*spotify.Client, error) {
			return &spotify.Client{}, nil
		}).
		ApplyMethod(&spotify.Client{}, "Library", func(_ *spotify.Client, _ int, ch ...chan interface{}) error {
			for _, c := range ch {
				c <- _track
				c <- _track // to trigger duplicate check
			}
			return nil
		}).
		ApplyMethod(&spotify.Client{}, "Playlist", func(_ *spotify.Client, _ string, ch ...chan interface{}) (*playlist.Playlist, error) {
			ch[0] <- _track
			ch[0] <- _trackNotFound // to skip inclusion in playlist
			return _playlist, nil
		}).
		ApplyMethod(&spotify.Client{}, "Album", func(_ *spotify.Client, _ string, ch ...chan interface{}) (*entity.Album, error) {
			ch[0] <- _track
			return _album, nil
		}).
		ApplyMethod(&spotify.Client{}, "Track", func(_ *spotify.Client, _ string, ch ...chan interface{}) (*entity.Track, error) {
			ch[0] <- _track
			return _track, nil
		}).
		ApplyFunc(id3.Open, func() (*id3.Tag, error) {
			return &id3.Tag{}, nil
		}).
		ApplyPrivateMethod(&id3.Tag{}, "userDefinedText", func() string {
			return "123"
		}).
		ApplyMethod(&id3.Tag{}, "Close", func() error {
			return nil
		}).
		ApplyFunc(provider.Search, func(track *entity.Track) ([]*provider.Match, error) {
			if track.ID == _trackNotFound.ID {
				return []*provider.Match{}, nil
			}
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
		ApplyFunc(util.FileMoveOrCopy, func() error {
			return nil
		}).
		ApplyMethod(&playlist.M3UEncoder{}, "Close", func() error {
			return nil
		}).
		Reset()

	// testing
	cmd := cmdSync()
	assert.Nil(t, util.ErrOnly(testExecute(cmd)))
	library, err := cmd.Flags().GetBool("library")
	assert.Nil(t, err)
	assert.True(t, library)
	assert.Nil(t, util.ErrOnly(testExecute(cmdSync(), "--plain", "-l", "-p", "123", "-a", "123", "-t", "123", "-f", "path")))
}

func TestCmdSyncInvalidEnvironment(t *testing.T) {
	t.Cleanup(cleanup)

	// monkey patching
	defer gomonkey.ApplyFunc(cmd.ValidateEnvironment, func() error { return errors.New("ko") }).Reset()

	// testing
	assert.EqualError(t, util.ErrOnly(testExecute(cmdSync(), "--plain")), "ko")
}

func TestCmdSyncOfflineIndex(t *testing.T) {
	t.Cleanup(cleanup)

	_track := &entity.Track{ID: "TestCmdSyncOfflineIndex", Title: "Title", Artists: []string{"Artist"}}

	// monkey patching
	defer gomonkey.NewPatches().
		ApplyFunc(cmd.ValidateEnvironment, func() error { return nil }).
		ApplyFunc(time.Sleep, func() {}).
		ApplyFunc(cmd.Open, func() error { return nil }).
		ApplyMethod(&index.Index{}, "Build", func(data *index.Index) error {
			data.Set(_track, index.Offline)
			return nil
		}).
		ApplyFunc(spotify.Authenticate, func() (*spotify.Client, error) {
			return &spotify.Client{}, nil
		}).
		ApplyMethod(&spotify.Client{}, "Library", func(_ *spotify.Client, _ int, ch ...chan interface{}) error {
			ch[0] <- _track
			return nil
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
		ApplyFunc(util.FileMoveOrCopy, func() error {
			return nil
		}).
		ApplyMethod(&playlist.M3UEncoder{}, "Close", func() error {
			return nil
		}).
		Reset()

	// testing
	assert.Nil(t, util.ErrOnly(testExecute(cmdSync(), "--plain")))
}

func TestCmdSyncPathFailure(t *testing.T) {
	t.Cleanup(cleanup)

	// monkey patching
	defer gomonkey.
		ApplyFunc(cmd.ValidateEnvironment, func() error { return nil }).
		ApplyFunc(os.Chdir, func() error { return errors.New("ko") }).
		Reset()

	// testing
	assert.EqualError(t, util.ErrOnly(testExecute(cmdSync(), "--plain")), "ko")
}

func TestCmdSyncIndexFailure(t *testing.T) {
	t.Cleanup(cleanup)

	// monkey patching
	defer gomonkey.NewPatches().
		ApplyFunc(cmd.ValidateEnvironment, func() error { return nil }).
		ApplyFunc(spotify.Authenticate, func() (*spotify.Client, error) {
			return &spotify.Client{}, nil
		}).
		ApplyMethod(&index.Index{}, "Build", func() error {
			return errors.New("ko")
		}).
		Reset()

	// testing
	assert.EqualError(t, util.ErrOnly(testExecute(cmdSync(), "--plain")), "ko")
}

func TestCmdSyncAuthFailure(t *testing.T) {
	t.Cleanup(cleanup)

	// monkey patching
	defer gomonkey.NewPatches().
		ApplyFunc(cmd.ValidateEnvironment, func() error { return nil }).
		ApplyFunc(time.Sleep, func() {}).
		ApplyFunc(cmd.Open, func() error {
			return nil
		}).
		ApplyMethod(&index.Index{}, "Build", func() error {
			return nil
		}).
		ApplyFunc(spotify.Authenticate, func() (*spotify.Client, error) {
			return &spotify.Client{}, errors.New("ko")
		}).
		Reset()

	// testing
	assert.EqualError(t, util.ErrOnly(testExecute(cmdSync(), "--plain")), "ko")
}

func TestCmdSyncLibraryFailure(t *testing.T) {
	t.Cleanup(cleanup)

	// monkey patching
	defer gomonkey.NewPatches().
		ApplyFunc(cmd.ValidateEnvironment, func() error { return nil }).
		ApplyFunc(time.Sleep, func() {}).
		ApplyFunc(cmd.Open, func() error {
			return nil
		}).
		ApplyMethod(&index.Index{}, "Build", func() error {
			return nil
		}).
		ApplyFunc(spotify.Authenticate, func() (*spotify.Client, error) {
			return &spotify.Client{}, nil
		}).
		ApplyMethod(&spotify.Client{}, "Library", func() error {
			return errors.New("ko")
		}).
		Reset()

	// testing
	assert.EqualError(t, util.ErrOnly(testExecute(cmdSync(), "--plain")), "ko")
}

func TestCmdSyncPlaylistFailure(t *testing.T) {
	t.Cleanup(cleanup)

	// monkey patching
	defer gomonkey.NewPatches().
		ApplyFunc(cmd.ValidateEnvironment, func() error { return nil }).
		ApplyFunc(time.Sleep, func() {}).
		ApplyFunc(cmd.Open, func() error {
			return nil
		}).
		ApplyMethod(&index.Index{}, "Build", func() error {
			return nil
		}).
		ApplyFunc(spotify.Authenticate, func() (*spotify.Client, error) {
			return &spotify.Client{}, nil
		}).
		ApplyMethod(&spotify.Client{}, "Playlist", func() (*playlist.Playlist, error) {
			return nil, errors.New("ko")
		}).
		Reset()

	// testing
	assert.EqualError(t, util.ErrOnly(testExecute(cmdSync(), "--plain", "-p", "123")), "ko")
}

func TestCmdSyncAlbumFailure(t *testing.T) {
	t.Cleanup(cleanup)

	// monkey patching
	defer gomonkey.NewPatches().
		ApplyFunc(cmd.ValidateEnvironment, func() error { return nil }).
		ApplyFunc(time.Sleep, func() {}).
		ApplyFunc(cmd.Open, func() error {
			return nil
		}).
		ApplyMethod(&index.Index{}, "Build", func() error {
			return nil
		}).
		ApplyFunc(spotify.Authenticate, func() (*spotify.Client, error) {
			return &spotify.Client{}, nil
		}).
		ApplyMethod(&spotify.Client{}, "Album", func() (*entity.Album, error) {
			return nil, errors.New("ko")
		}).
		Reset()

	// testing
	assert.EqualError(t, util.ErrOnly(testExecute(cmdSync(), "--plain", "-a", "123")), "ko")
}

func TestCmdSyncTrackFailure(t *testing.T) {
	t.Cleanup(cleanup)

	// monkey patching
	defer gomonkey.NewPatches().
		ApplyFunc(cmd.ValidateEnvironment, func() error { return nil }).
		ApplyFunc(time.Sleep, func() {}).
		ApplyFunc(cmd.Open, func() error {
			return nil
		}).
		ApplyMethod(&index.Index{}, "Build", func() error {
			return nil
		}).
		ApplyFunc(spotify.Authenticate, func() (*spotify.Client, error) {
			return &spotify.Client{}, nil
		}).
		ApplyMethod(&spotify.Client{}, "Track", func() (*entity.Track, error) {
			return nil, errors.New("ko")
		}).
		Reset()

	// testing
	assert.EqualError(t, util.ErrOnly(testExecute(cmdSync(), "--plain", "-t", "123")), "ko")
}

func TestCmdSyncFixOpenFailure(t *testing.T) {
	t.Cleanup(cleanup)

	// monkey patching
	defer gomonkey.NewPatches().
		ApplyFunc(cmd.ValidateEnvironment, func() error { return nil }).
		ApplyFunc(time.Sleep, func() {}).
		ApplyFunc(cmd.Open, func() error {
			return nil
		}).
		ApplyMethod(&index.Index{}, "Build", func() error {
			return nil
		}).
		ApplyFunc(spotify.Authenticate, func() (*spotify.Client, error) {
			return &spotify.Client{}, nil
		}).
		ApplyFunc(id3.Open, func() (*id3.Tag, error) {
			return nil, errors.New("ko")
		}).
		Reset()

	// testing
	assert.EqualError(t, util.ErrOnly(testExecute(cmdSync(), "--plain", "-f", "path")), "ko")
}

func TestCmdSyncFixSpotifyIDFailure(t *testing.T) {
	t.Cleanup(cleanup)

	// monkey patching
	defer gomonkey.NewPatches().
		ApplyFunc(cmd.ValidateEnvironment, func() error { return nil }).
		ApplyFunc(time.Sleep, func() {}).
		ApplyFunc(cmd.Open, func() error {
			return nil
		}).
		ApplyMethod(&index.Index{}, "Build", func() error {
			return nil
		}).
		ApplyFunc(spotify.Authenticate, func() (*spotify.Client, error) {
			return &spotify.Client{}, nil
		}).
		ApplyFunc(id3.Open, func() (*id3.Tag, error) {
			return &id3.Tag{}, nil
		}).
		ApplyPrivateMethod(&id3.Tag{}, "userDefinedText", func() string {
			return ""
		}).
		Reset()

	// testing
	assert.ErrorContains(t, util.ErrOnly(testExecute(cmdSync(), "--plain", "-f", "path")), "does not have spotify ID metadata set")
}

func TestCmdSyncFixCloseFailure(t *testing.T) {
	t.Cleanup(cleanup)

	// monkey patching
	defer gomonkey.NewPatches().
		ApplyFunc(cmd.ValidateEnvironment, func() error { return nil }).
		ApplyFunc(time.Sleep, func() {}).
		ApplyFunc(cmd.Open, func() error {
			return nil
		}).
		ApplyMethod(&index.Index{}, "Build", func() error {
			return nil
		}).
		ApplyFunc(spotify.Authenticate, func() (*spotify.Client, error) {
			return &spotify.Client{}, nil
		}).
		ApplyFunc(id3.Open, func() (*id3.Tag, error) {
			return &id3.Tag{}, nil
		}).
		ApplyPrivateMethod(&id3.Tag{}, "userDefinedText", func() string {
			return "123"
		}).
		ApplyMethod(&id3v2.Tag{}, "Close", func() error {
			return errors.New("ko")
		}).
		Reset()

	// testing
	assert.EqualError(t, util.ErrOnly(testExecute(cmdSync(), "--plain", "-f", "path")), "ko")
}

func TestCmdSyncDecideManual(t *testing.T) {
	t.Cleanup(cleanup)

	_track := &entity.Track{ID: "TestCmdSyncDecideManual", Title: "Title", Artists: []string{"Artist"}}

	// monkey patching
	defer gomonkey.NewPatches().
		ApplyFunc(cmd.ValidateEnvironment, func() error { return nil }).
		ApplyFunc(time.Sleep, func() {}).
		ApplyFunc(cmd.Open, func() error {
			return nil
		}).
		ApplyMethod(&index.Index{}, "Build", func() error {
			return nil
		}).
		ApplyFunc(spotify.Authenticate, func() (*spotify.Client, error) {
			return &spotify.Client{}, nil
		}).
		ApplyMethod(&spotify.Client{}, "Library",
			func(_ *spotify.Client, _ int, ch ...chan interface{}) error {
				ch[0] <- _track
				return nil
			}).
		Reset()

	// testing
	assert.Nil(t, util.ErrOnly(testExecute(cmdSync(), "--plain", "--manual")))
}

func TestCmdSyncDecideFailure(t *testing.T) {
	t.Cleanup(cleanup)

	_track := &entity.Track{ID: "TestCmdSyncDecideFailure", Title: "Title", Artists: []string{"Artist"}}

	// monkey patching
	defer gomonkey.NewPatches().
		ApplyFunc(cmd.ValidateEnvironment, func() error { return nil }).
		ApplyFunc(time.Sleep, func() {}).
		ApplyFunc(cmd.Open, func() error {
			return nil
		}).
		ApplyMethod(&index.Index{}, "Build", func() error {
			return nil
		}).
		ApplyFunc(spotify.Authenticate, func() (*spotify.Client, error) {
			return &spotify.Client{}, nil
		}).
		ApplyMethod(&spotify.Client{}, "Library",
			func(_ *spotify.Client, _ int, ch ...chan interface{}) error {
				ch[0] <- _track
				return nil
			}).
		ApplyFunc(provider.Search, func(*entity.Track) ([]*provider.Match, error) {
			return nil, errors.New("ko")
		}).
		Reset()

	// testing
	assert.EqualError(t, util.ErrOnly(testExecute(cmdSync(), "--plain")), "ko")
}

func TestCmdSyncDecideNotFound(t *testing.T) {
	t.Cleanup(cleanup)

	_track := &entity.Track{ID: "TestCmdSyncDecideNotFound", Title: "Title", Artists: []string{"Artist"}}

	// monkey patching
	defer gomonkey.NewPatches().
		ApplyFunc(cmd.ValidateEnvironment, func() error { return nil }).
		ApplyFunc(time.Sleep, func() {}).
		ApplyFunc(cmd.Open, func() error {
			return nil
		}).
		ApplyMethod(&index.Index{}, "Build", func() error {
			return nil
		}).
		ApplyFunc(spotify.Authenticate, func() (*spotify.Client, error) {
			return &spotify.Client{}, nil
		}).
		ApplyMethod(&spotify.Client{}, "Library",
			func(_ *spotify.Client, _ int, ch ...chan interface{}) error {
				ch[0] <- _track
				return nil
			}).
		ApplyFunc(provider.Search, func(*entity.Track) ([]*provider.Match, error) {
			return []*provider.Match{}, nil
		}).
		Reset()

	// testing
	assert.Nil(t, util.ErrOnly(testExecute(cmdSync(), "--plain")))
}

func TestCmdSyncCollectFailure(t *testing.T) {
	t.Cleanup(cleanup)

	_track := &entity.Track{ID: "TestCmdSyncCollectFailure", Title: "Title", Artists: []string{"Artist"}}

	// monkey patching
	defer gomonkey.NewPatches().
		ApplyFunc(cmd.ValidateEnvironment, func() error { return nil }).
		ApplyFunc(time.Sleep, func() {}).
		ApplyFunc(cmd.Open, func() error {
			return nil
		}).
		ApplyMethod(&index.Index{}, "Build", func() error {
			return nil
		}).
		ApplyFunc(spotify.Authenticate, func() (*spotify.Client, error) {
			return &spotify.Client{}, nil
		}).
		ApplyMethod(&spotify.Client{}, "Library",
			func(_ *spotify.Client, _ int, ch ...chan interface{}) error {
				ch[0] <- _track
				return nil
			}).
		ApplyFunc(provider.Search, func(*entity.Track) ([]*provider.Match, error) {
			return []*provider.Match{{URL: "http://localhost/", Score: 0}}, nil
		}).
		ApplyFunc(downloader.Download, func(url string, _ string, _ processor.Processor, ch ...chan []byte) error {
			if url != "http://localhost/" {
				return errors.New("ko")
			}
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
	assert.EqualError(t, util.ErrOnly(testExecute(cmdSync(), "--plain")), "ko")
}

func TestCmdSyncDownloadFailure(t *testing.T) {
	t.Cleanup(cleanup)

	_track := &entity.Track{ID: "TestCmdSyncDownloadFailure", Title: "Title", Artists: []string{"Artist"}}

	// monkey patching
	defer gomonkey.NewPatches().
		ApplyFunc(cmd.ValidateEnvironment, func() error { return nil }).
		ApplyFunc(time.Sleep, func() {}).
		ApplyFunc(cmd.Open, func() error {
			return nil
		}).
		ApplyMethod(&index.Index{}, "Build", func() error {
			return nil
		}).
		ApplyFunc(spotify.Authenticate, func() (*spotify.Client, error) {
			return &spotify.Client{}, nil
		}).
		ApplyMethod(&spotify.Client{}, "Library", func(_ *spotify.Client, _ int, ch ...chan interface{}) error {
			ch[0] <- _track
			return nil
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
	assert.EqualError(t, util.ErrOnly(testExecute(cmdSync(), "--plain")), "ko")
}

func TestCmdSyncLyricsFailure(t *testing.T) {
	t.Cleanup(cleanup)

	_track := &entity.Track{ID: "TestCmdSyncLyricsFailure", Title: "Title", Artists: []string{"Artist"}}

	// monkey patching
	defer gomonkey.NewPatches().
		ApplyFunc(cmd.ValidateEnvironment, func() error { return nil }).
		ApplyFunc(time.Sleep, func() {}).
		ApplyFunc(cmd.Open, func() error {
			return nil
		}).
		ApplyMethod(&index.Index{}, "Build", func() error {
			return nil
		}).
		ApplyFunc(spotify.Authenticate, func() (*spotify.Client, error) {
			return &spotify.Client{}, nil
		}).
		ApplyMethod(&spotify.Client{}, "Library", func(_ *spotify.Client, _ int, ch ...chan interface{}) error {
			ch[0] <- _track
			return nil
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
	assert.EqualError(t, util.ErrOnly(testExecute(cmdSync(), "--plain")), "ko")
}

func TestCmdSyncProcessorFailure(t *testing.T) {
	t.Cleanup(cleanup)

	_track := &entity.Track{ID: "TestCmdSyncProcessorFailure", Title: "Title", Artists: []string{"Artist"}}

	// monkey patching
	defer gomonkey.NewPatches().
		ApplyFunc(cmd.ValidateEnvironment, func() error { return nil }).
		ApplyFunc(time.Sleep, func() {}).
		ApplyFunc(cmd.Open, func() error {
			return nil
		}).
		ApplyMethod(&index.Index{}, "Build", func() error {
			return nil
		}).
		ApplyFunc(spotify.Authenticate, func() (*spotify.Client, error) {
			return &spotify.Client{}, nil
		}).
		ApplyMethod(&spotify.Client{}, "Library", func(_ *spotify.Client, _ int, ch ...chan interface{}) error {
			ch[0] <- _track
			return nil
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
	assert.EqualError(t, util.ErrOnly(testExecute(cmdSync(), "--plain")), "ko")
}

func TestCmdSyncInstallerFailure(t *testing.T) {
	t.Cleanup(cleanup)

	_track := &entity.Track{ID: "TestCmdSyncInstallerFailure", Title: "Title", Artists: []string{"Artist"}}

	// monkey patching
	defer gomonkey.NewPatches().
		ApplyFunc(cmd.ValidateEnvironment, func() error { return nil }).
		ApplyFunc(time.Sleep, func() {}).
		ApplyFunc(cmd.Open, func() error {
			return nil
		}).
		ApplyMethod(&index.Index{}, "Build", func() error {
			return nil
		}).
		ApplyFunc(spotify.Authenticate, func() (*spotify.Client, error) {
			return &spotify.Client{}, nil
		}).
		ApplyMethod(&spotify.Client{}, "Library", func(_ *spotify.Client, _ int, ch ...chan interface{}) error {
			ch[0] <- _track
			return nil
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
		ApplyFunc(util.FileMoveOrCopy, func() error {
			return errors.New("ko")
		}).
		Reset()

	// testing
	assert.EqualError(t, util.ErrOnly(testExecute(cmdSync(), "--plain")), "ko")
}

func TestCmdSyncPlaylistEncoderFailure(t *testing.T) {
	t.Cleanup(cleanup)

	_playlist := &playlist.Playlist{Tracks: []*entity.Track{
		{ID: "TestCmdSyncPlaylistEncoderFailure", Title: "Title", Artists: []string{"Artist"}},
	}}

	// monkey patching
	defer gomonkey.NewPatches().
		ApplyFunc(cmd.ValidateEnvironment, func() error { return nil }).
		ApplyFunc(time.Sleep, func() {}).
		ApplyFunc(cmd.Open, func() error {
			return nil
		}).
		ApplyMethod(&index.Index{}, "Build", func() error {
			return nil
		}).
		ApplyFunc(spotify.Authenticate, func() (*spotify.Client, error) {
			return &spotify.Client{}, nil
		}).
		ApplyMethod(&spotify.Client{}, "Playlist", func() (*playlist.Playlist, error) {
			return _playlist, nil
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
		ApplyFunc(util.FileMoveOrCopy, func() error {
			return nil
		}).
		ApplyMethod(playlist.Playlist{}, "Encoder", func() (any, error) {
			return nil, errors.New("ko")
		}).
		Reset()

	// testing
	assert.EqualError(t, util.ErrOnly(testExecute(cmdSync(), "--plain", "-p", "123")), "ko")
}

func TestCmdSyncPlaylistEncoderAddFailure(t *testing.T) {
	t.Cleanup(cleanup)

	_playlist := &playlist.Playlist{Tracks: []*entity.Track{
		{ID: "TestCmdSyncPlaylistEncoderAddFailure", Title: "Title", Artists: []string{"Artist"}},
	}}

	// monkey patching
	defer gomonkey.NewPatches().
		ApplyFunc(cmd.ValidateEnvironment, func() error { return nil }).
		ApplyFunc(time.Sleep, func() {}).
		ApplyFunc(cmd.Open, func() error {
			return nil
		}).
		ApplyMethod(&index.Index{}, "Build", func() error {
			return nil
		}).
		ApplyFunc(spotify.Authenticate, func() (*spotify.Client, error) {
			return &spotify.Client{}, nil
		}).
		ApplyMethod(&spotify.Client{}, "Playlist", func(_ *spotify.Client, _ string, ch ...chan interface{}) (*playlist.Playlist, error) {
			ch[0] <- _playlist.Tracks[0]
			return _playlist, nil
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
		ApplyFunc(util.FileMoveOrCopy, func() error {
			return nil
		}).
		ApplyMethod(&playlist.M3UEncoder{}, "Add", func() error {
			return errors.New("ko")
		}).
		Reset()

	// testing
	assert.EqualError(t, util.ErrOnly(testExecute(cmdSync(), "--plain", "-p", "123")), "ko")
}

func TestCmdSyncPlaylistEncoderCloseFailure(t *testing.T) {
	t.Cleanup(cleanup)

	_playlist := &playlist.Playlist{Tracks: []*entity.Track{
		{ID: "TestCmdSyncPlaylistEncoderCloseFailure", Title: "Title", Artists: []string{"Artist"}},
	}}

	// monkey patching
	defer gomonkey.NewPatches().
		ApplyFunc(cmd.ValidateEnvironment, func() error { return nil }).
		ApplyFunc(time.Sleep, func() {}).
		ApplyFunc(cmd.Open, func() error {
			return nil
		}).
		ApplyMethod(&index.Index{}, "Build", func() error {
			return nil
		}).
		ApplyFunc(spotify.Authenticate, func() (*spotify.Client, error) {
			return &spotify.Client{}, nil
		}).
		ApplyMethod(&spotify.Client{}, "Playlist", func(_ *spotify.Client, _ string, ch ...chan interface{}) (*playlist.Playlist, error) {
			ch[0] <- _playlist.Tracks[0]
			return _playlist, nil
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
		ApplyFunc(util.FileMoveOrCopy, func() error {
			return nil
		}).
		ApplyMethod(&playlist.M3UEncoder{}, "Close", func() error {
			return errors.New("ko")
		}).
		Reset()

	// testing
	assert.EqualError(t, util.ErrOnly(testExecute(cmdSync(), "--plain", "-p", "123")), "ko")
}
