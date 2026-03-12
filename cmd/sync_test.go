package cmd

import (
	"errors"
	"os"
	"testing"

	"github.com/bytedance/mockey"
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
	"github.com/streambinder/spotitube/sys"
	"github.com/streambinder/spotitube/sys/cmd"
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
	defer mockey.UnPatchAll()
	mockey.Mock(cmd.ValidateEnvironment).Return(nil).Build()
	mockey.Mock(cmd.Open).Return(nil).Build()
	mockey.Mock(mockey.GetMethod(&index.Index{}, "Build")).Return(nil).Build()
	mockey.Mock(spotify.Authenticate).Return(&spotify.Client{}, nil).Build()
	mockey.Mock(mockey.GetMethod(&spotify.Client{}, "Library")).To(func(_ int, ch ...chan interface{}) error {
		for _, c := range ch {
			c <- _track
			c <- _track // to trigger duplicate check
		}
		return nil
	}).Build()
	mockey.Mock(mockey.GetMethod(&spotify.Client{}, "Playlist")).To(func(_ string, ch ...chan interface{}) (*playlist.Playlist, error) {
		ch[0] <- _track
		ch[0] <- _trackNotFound // to skip inclusion in playlist
		return _playlist, nil
	}).Build()
	mockey.Mock(mockey.GetMethod(&spotify.Client{}, "Album")).To(func(_ string, ch ...chan interface{}) (*entity.Album, error) {
		ch[0] <- _track
		return _album, nil
	}).Build()
	mockey.Mock(mockey.GetMethod(&spotify.Client{}, "Track")).To(func(_ string, ch ...chan interface{}) (*entity.Track, error) {
		ch[0] <- _track
		return _track, nil
	}).Build()
	mockey.Mock(id3.Open).Return(&id3.Tag{}, nil).Build()
	mockey.Mock(mockey.GetMethod(&id3.Tag{}, "userDefinedText")).Return("123").Build()
	mockey.Mock(mockey.GetMethod(&id3.Tag{}, "Close")).Return(nil).Build()
	mockey.Mock(provider.Search).To(func(track *entity.Track) ([]*provider.Match, error) {
		if track.ID == _trackNotFound.ID {
			return []*provider.Match{}, nil
		}
		return []*provider.Match{{URL: "http://localhost/", Score: 0}}, nil
	}).Build()
	mockey.Mock(downloader.Download).To(func(_, _ string, _ processor.Processor, ch ...chan []byte) error {
		for _, c := range ch {
			c <- []byte{}
		}
		return nil
	}).Build()
	mockey.Mock(lyrics.Search).Return("lyrics", nil).Build()
	mockey.Mock(processor.Do).Return(nil).Build()
	mockey.Mock(sys.FileMoveOrCopy).Return(nil).Build()
	mockey.Mock(mockey.GetMethod(&playlist.M3UEncoder{}, "Close")).Return(nil).Build()

	// testing
	cmd := cmdSync()
	assert.Nil(t, sys.ErrOnly(testExecute(cmd)))
	library, err := cmd.Flags().GetBool("library")
	assert.Nil(t, err)
	assert.True(t, library)
	assert.Nil(t, sys.ErrOnly(testExecute(cmdSync(), "--plain", "-l", "-p", "123", "-a", "123", "-t", "123", "-f", "path")))
}

func TestCmdSyncInvalidEnvironment(t *testing.T) {
	t.Cleanup(cleanup)

	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(cmd.ValidateEnvironment).Return(errors.New("ko")).Build()

	// testing
	assert.EqualError(t, sys.ErrOnly(testExecute(cmdSync(), "--plain")), "ko")
}

func TestCmdSyncOfflineIndex(t *testing.T) {
	t.Cleanup(cleanup)

	_track := &entity.Track{ID: "TestCmdSyncOfflineIndex", Title: "Title", Artists: []string{"Artist"}}

	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(cmd.ValidateEnvironment).Return(nil).Build()
	mockey.Mock(cmd.Open).Return(nil).Build()
	mockey.Mock(mockey.GetMethod(&index.Index{}, "Build")).To(func(data *index.Index, _ string, _ ...int) error {
		data.Set(_track, index.Offline)
		return nil
	}).Build()
	mockey.Mock(spotify.Authenticate).Return(&spotify.Client{}, nil).Build()
	mockey.Mock(mockey.GetMethod(&spotify.Client{}, "Library")).To(func(_ int, ch ...chan interface{}) error {
		ch[0] <- _track
		return nil
	}).Build()
	mockey.Mock(provider.Search).Return([]*provider.Match{{URL: "http://localhost/", Score: 0}}, nil).Build()
	mockey.Mock(downloader.Download).To(func(_, _ string, _ processor.Processor, ch ...chan []byte) error {
		for _, c := range ch {
			c <- []byte{}
		}
		return nil
	}).Build()
	mockey.Mock(lyrics.Search).Return("lyrics", nil).Build()
	mockey.Mock(processor.Do).Return(nil).Build()
	mockey.Mock(sys.FileMoveOrCopy).Return(nil).Build()
	mockey.Mock(mockey.GetMethod(&playlist.M3UEncoder{}, "Close")).Return(nil).Build()

	// testing
	assert.Nil(t, sys.ErrOnly(testExecute(cmdSync(), "--plain")))
}

func TestCmdSyncPathFailure(t *testing.T) {
	t.Cleanup(cleanup)

	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(cmd.ValidateEnvironment).Return(nil).Build()
	mockey.Mock(os.Chdir).Return(errors.New("ko")).Build()

	// testing
	assert.EqualError(t, sys.ErrOnly(testExecute(cmdSync(), "--plain")), "ko")
}

func TestCmdSyncIndexFailure(t *testing.T) {
	t.Cleanup(cleanup)

	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(cmd.ValidateEnvironment).Return(nil).Build()
	mockey.Mock(spotify.Authenticate).Return(&spotify.Client{}, nil).Build()
	mockey.Mock(mockey.GetMethod(&index.Index{}, "Build")).Return(errors.New("ko")).Build()

	// testing
	assert.EqualError(t, sys.ErrOnly(testExecute(cmdSync(), "--plain")), "ko")
}

func TestCmdSyncAuthFailure(t *testing.T) {
	t.Cleanup(cleanup)

	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(cmd.ValidateEnvironment).Return(nil).Build()
	mockey.Mock(cmd.Open).Return(nil).Build()
	mockey.Mock(mockey.GetMethod(&index.Index{}, "Build")).Return(nil).Build()
	mockey.Mock(spotify.Authenticate).Return(&spotify.Client{}, errors.New("ko")).Build()

	// testing
	assert.EqualError(t, sys.ErrOnly(testExecute(cmdSync(), "--plain")), "ko")
}

func TestCmdSyncLibraryFailure(t *testing.T) {
	t.Cleanup(cleanup)

	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(cmd.ValidateEnvironment).Return(nil).Build()
	mockey.Mock(cmd.Open).Return(nil).Build()
	mockey.Mock(mockey.GetMethod(&index.Index{}, "Build")).Return(nil).Build()
	mockey.Mock(spotify.Authenticate).Return(&spotify.Client{}, nil).Build()
	mockey.Mock(mockey.GetMethod(&spotify.Client{}, "Library")).Return(errors.New("ko")).Build()

	// testing
	assert.EqualError(t, sys.ErrOnly(testExecute(cmdSync(), "--plain")), "ko")
}

func TestCmdSyncPlaylistFailure(t *testing.T) {
	t.Cleanup(cleanup)

	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(cmd.ValidateEnvironment).Return(nil).Build()
	mockey.Mock(cmd.Open).Return(nil).Build()
	mockey.Mock(mockey.GetMethod(&index.Index{}, "Build")).Return(nil).Build()
	mockey.Mock(spotify.Authenticate).Return(&spotify.Client{}, nil).Build()
	mockey.Mock(mockey.GetMethod(&spotify.Client{}, "Playlist")).Return(nil, errors.New("ko")).Build()

	// testing
	assert.EqualError(t, sys.ErrOnly(testExecute(cmdSync(), "--plain", "-p", "123")), "ko")
}

func TestCmdSyncAlbumFailure(t *testing.T) {
	t.Cleanup(cleanup)

	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(cmd.ValidateEnvironment).Return(nil).Build()
	mockey.Mock(cmd.Open).Return(nil).Build()
	mockey.Mock(mockey.GetMethod(&index.Index{}, "Build")).Return(nil).Build()
	mockey.Mock(spotify.Authenticate).Return(&spotify.Client{}, nil).Build()
	mockey.Mock(mockey.GetMethod(&spotify.Client{}, "Album")).Return(nil, errors.New("ko")).Build()

	// testing
	assert.EqualError(t, sys.ErrOnly(testExecute(cmdSync(), "--plain", "-a", "123")), "ko")
}

func TestCmdSyncTrackFailure(t *testing.T) {
	t.Cleanup(cleanup)

	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(cmd.ValidateEnvironment).Return(nil).Build()
	mockey.Mock(cmd.Open).Return(nil).Build()
	mockey.Mock(mockey.GetMethod(&index.Index{}, "Build")).Return(nil).Build()
	mockey.Mock(spotify.Authenticate).Return(&spotify.Client{}, nil).Build()
	mockey.Mock(mockey.GetMethod(&spotify.Client{}, "Track")).Return(nil, errors.New("ko")).Build()

	// testing
	assert.EqualError(t, sys.ErrOnly(testExecute(cmdSync(), "--plain", "-t", "123")), "ko")
}

func TestCmdSyncFixOpenFailure(t *testing.T) {
	t.Cleanup(cleanup)

	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(cmd.ValidateEnvironment).Return(nil).Build()
	mockey.Mock(cmd.Open).Return(nil).Build()
	mockey.Mock(mockey.GetMethod(&index.Index{}, "Build")).Return(nil).Build()
	mockey.Mock(spotify.Authenticate).Return(&spotify.Client{}, nil).Build()
	mockey.Mock(id3.Open).Return(nil, errors.New("ko")).Build()

	// testing
	assert.EqualError(t, sys.ErrOnly(testExecute(cmdSync(), "--plain", "-f", "path")), "ko")
}

func TestCmdSyncFixSpotifyIDFailure(t *testing.T) {
	t.Cleanup(cleanup)

	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(cmd.ValidateEnvironment).Return(nil).Build()
	mockey.Mock(cmd.Open).Return(nil).Build()
	mockey.Mock(mockey.GetMethod(&index.Index{}, "Build")).Return(nil).Build()
	mockey.Mock(spotify.Authenticate).Return(&spotify.Client{}, nil).Build()
	mockey.Mock(id3.Open).Return(&id3.Tag{}, nil).Build()
	mockey.Mock(mockey.GetMethod(&id3.Tag{}, "userDefinedText")).Return("").Build()

	// testing
	assert.ErrorContains(t, sys.ErrOnly(testExecute(cmdSync(), "--plain", "-f", "path")), "does not have spotify ID metadata set")
}

func TestCmdSyncFixCloseFailure(t *testing.T) {
	t.Cleanup(cleanup)

	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(cmd.ValidateEnvironment).Return(nil).Build()
	mockey.Mock(cmd.Open).Return(nil).Build()
	mockey.Mock(mockey.GetMethod(&index.Index{}, "Build")).Return(nil).Build()
	mockey.Mock(spotify.Authenticate).Return(&spotify.Client{}, nil).Build()
	mockey.Mock(id3.Open).Return(&id3.Tag{}, nil).Build()
	mockey.Mock(mockey.GetMethod(&id3.Tag{}, "userDefinedText")).Return("123").Build()
	mockey.Mock(mockey.GetMethod(&id3v2.Tag{}, "Close")).Return(errors.New("ko")).Build()

	// testing
	assert.EqualError(t, sys.ErrOnly(testExecute(cmdSync(), "--plain", "-f", "path")), "ko")
}

func TestCmdSyncDecideManual(t *testing.T) {
	t.Cleanup(cleanup)

	_track := &entity.Track{ID: "TestCmdSyncDecideManual", Title: "Title", Artists: []string{"Artist"}}

	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(cmd.ValidateEnvironment).Return(nil).Build()
	mockey.Mock(cmd.Open).Return(nil).Build()
	mockey.Mock(mockey.GetMethod(&index.Index{}, "Build")).Return(nil).Build()
	mockey.Mock(spotify.Authenticate).Return(&spotify.Client{}, nil).Build()
	mockey.Mock(mockey.GetMethod(&spotify.Client{}, "Library")).To(func(_ int, ch ...chan interface{}) error {
		ch[0] <- _track
		return nil
	}).Build()

	// testing
	assert.Nil(t, sys.ErrOnly(testExecute(cmdSync(), "--plain", "--manual")))
}

func TestCmdSyncDecideFailure(t *testing.T) {
	t.Cleanup(cleanup)

	_track := &entity.Track{ID: "TestCmdSyncDecideFailure", Title: "Title", Artists: []string{"Artist"}}

	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(cmd.ValidateEnvironment).Return(nil).Build()
	mockey.Mock(cmd.Open).Return(nil).Build()
	mockey.Mock(mockey.GetMethod(&index.Index{}, "Build")).Return(nil).Build()
	mockey.Mock(spotify.Authenticate).Return(&spotify.Client{}, nil).Build()
	mockey.Mock(mockey.GetMethod(&spotify.Client{}, "Library")).To(func(_ int, ch ...chan interface{}) error {
		ch[0] <- _track
		return nil
	}).Build()
	mockey.Mock(provider.Search).To(func(*entity.Track) ([]*provider.Match, error) {
		return nil, errors.New("ko")
	}).Build()

	// testing
	assert.EqualError(t, sys.ErrOnly(testExecute(cmdSync(), "--plain")), "ko")
}

func TestCmdSyncDecideNotFound(t *testing.T) {
	t.Cleanup(cleanup)

	_track := &entity.Track{ID: "TestCmdSyncDecideNotFound", Title: "Title", Artists: []string{"Artist"}}

	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(cmd.ValidateEnvironment).Return(nil).Build()
	mockey.Mock(cmd.Open).Return(nil).Build()
	mockey.Mock(mockey.GetMethod(&index.Index{}, "Build")).Return(nil).Build()
	mockey.Mock(spotify.Authenticate).Return(&spotify.Client{}, nil).Build()
	mockey.Mock(mockey.GetMethod(&spotify.Client{}, "Library")).To(func(_ int, ch ...chan interface{}) error {
		ch[0] <- _track
		return nil
	}).Build()
	mockey.Mock(provider.Search).To(func(*entity.Track) ([]*provider.Match, error) {
		return []*provider.Match{}, nil
	}).Build()

	// testing
	assert.Nil(t, sys.ErrOnly(testExecute(cmdSync(), "--plain")))
}

func TestCmdSyncCollectFailure(t *testing.T) {
	t.Cleanup(cleanup)

	_track := &entity.Track{ID: "TestCmdSyncCollectFailure", Title: "Title", Artists: []string{"Artist"}}

	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(cmd.ValidateEnvironment).Return(nil).Build()
	mockey.Mock(cmd.Open).Return(nil).Build()
	mockey.Mock(mockey.GetMethod(&index.Index{}, "Build")).Return(nil).Build()
	mockey.Mock(spotify.Authenticate).Return(&spotify.Client{}, nil).Build()
	mockey.Mock(mockey.GetMethod(&spotify.Client{}, "Library")).To(func(_ int, ch ...chan interface{}) error {
		ch[0] <- _track
		return nil
	}).Build()
	mockey.Mock(provider.Search).To(func(*entity.Track) ([]*provider.Match, error) {
		return []*provider.Match{{URL: "http://localhost/", Score: 0}}, nil
	}).Build()
	mockey.Mock(downloader.Download).To(func(url string, _ string, _ processor.Processor, ch ...chan []byte) error {
		if url != "http://localhost/" {
			return errors.New("ko")
		}
		for _, c := range ch {
			c <- []byte{}
		}
		return nil
	}).Build()
	mockey.Mock(lyrics.Search).Return("", nil).Build()

	// testing
	assert.EqualError(t, sys.ErrOnly(testExecute(cmdSync(), "--plain")), "ko")
}

func TestCmdSyncDownloadFailure(t *testing.T) {
	t.Cleanup(cleanup)

	_track := &entity.Track{ID: "TestCmdSyncDownloadFailure", Title: "Title", Artists: []string{"Artist"}}

	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(cmd.ValidateEnvironment).Return(nil).Build()
	mockey.Mock(cmd.Open).Return(nil).Build()
	mockey.Mock(mockey.GetMethod(&index.Index{}, "Build")).Return(nil).Build()
	mockey.Mock(spotify.Authenticate).Return(&spotify.Client{}, nil).Build()
	mockey.Mock(mockey.GetMethod(&spotify.Client{}, "Library")).To(func(_ int, ch ...chan interface{}) error {
		ch[0] <- _track
		return nil
	}).Build()
	mockey.Mock(provider.Search).To(func(*entity.Track) ([]*provider.Match, error) {
		return []*provider.Match{{URL: "http://localhost/", Score: 0}}, nil
	}).Build()
	mockey.Mock(downloader.Download).To(func(_, _ string, _ processor.Processor, ch ...chan []byte) error {
		for _, c := range ch {
			c <- []byte{}
		}
		return errors.New("ko")
	}).Build()
	mockey.Mock(lyrics.Search).Return("", nil).Build()

	// testing
	assert.EqualError(t, sys.ErrOnly(testExecute(cmdSync(), "--plain")), "ko")
}

func TestCmdSyncLyricsFailure(t *testing.T) {
	t.Cleanup(cleanup)

	_track := &entity.Track{ID: "TestCmdSyncLyricsFailure", Title: "Title", Artists: []string{"Artist"}}

	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(cmd.ValidateEnvironment).Return(nil).Build()
	mockey.Mock(cmd.Open).Return(nil).Build()
	mockey.Mock(mockey.GetMethod(&index.Index{}, "Build")).Return(nil).Build()
	mockey.Mock(spotify.Authenticate).Return(&spotify.Client{}, nil).Build()
	mockey.Mock(mockey.GetMethod(&spotify.Client{}, "Library")).To(func(_ int, ch ...chan interface{}) error {
		ch[0] <- _track
		return nil
	}).Build()
	mockey.Mock(provider.Search).Return([]*provider.Match{{URL: "http://localhost/", Score: 0}}, nil).Build()
	mockey.Mock(downloader.Download).To(func(_, _ string, _ processor.Processor, ch ...chan []byte) error {
		for _, c := range ch {
			c <- []byte{}
		}
		return nil
	}).Build()
	mockey.Mock(lyrics.Search).Return("", errors.New("ko")).Build()

	// testing
	assert.EqualError(t, sys.ErrOnly(testExecute(cmdSync(), "--plain")), "ko")
}

func TestCmdSyncProcessorFailure(t *testing.T) {
	t.Cleanup(cleanup)

	_track := &entity.Track{ID: "TestCmdSyncProcessorFailure", Title: "Title", Artists: []string{"Artist"}}

	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(cmd.ValidateEnvironment).Return(nil).Build()
	mockey.Mock(cmd.Open).Return(nil).Build()
	mockey.Mock(mockey.GetMethod(&index.Index{}, "Build")).Return(nil).Build()
	mockey.Mock(spotify.Authenticate).Return(&spotify.Client{}, nil).Build()
	mockey.Mock(mockey.GetMethod(&spotify.Client{}, "Library")).To(func(_ int, ch ...chan interface{}) error {
		ch[0] <- _track
		return nil
	}).Build()
	mockey.Mock(provider.Search).Return([]*provider.Match{{URL: "http://localhost/", Score: 0}}, nil).Build()
	mockey.Mock(downloader.Download).To(func(_, _ string, _ processor.Processor, ch ...chan []byte) error {
		for _, c := range ch {
			c <- []byte{}
		}
		return nil
	}).Build()
	mockey.Mock(lyrics.Search).Return("lyrics", nil).Build()
	mockey.Mock(processor.Do).Return(errors.New("ko")).Build()

	// testing
	assert.EqualError(t, sys.ErrOnly(testExecute(cmdSync(), "--plain")), "ko")
}

func TestCmdSyncInstallerFailure(t *testing.T) {
	t.Cleanup(cleanup)

	_track := &entity.Track{ID: "TestCmdSyncInstallerFailure", Title: "Title", Artists: []string{"Artist"}}

	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(cmd.ValidateEnvironment).Return(nil).Build()
	mockey.Mock(cmd.Open).Return(nil).Build()
	mockey.Mock(mockey.GetMethod(&index.Index{}, "Build")).Return(nil).Build()
	mockey.Mock(spotify.Authenticate).Return(&spotify.Client{}, nil).Build()
	mockey.Mock(mockey.GetMethod(&spotify.Client{}, "Library")).To(func(_ int, ch ...chan interface{}) error {
		ch[0] <- _track
		return nil
	}).Build()
	mockey.Mock(provider.Search).Return([]*provider.Match{{URL: "http://localhost/", Score: 0}}, nil).Build()
	mockey.Mock(downloader.Download).To(func(_, _ string, _ processor.Processor, ch ...chan []byte) error {
		for _, c := range ch {
			c <- []byte{}
		}
		return nil
	}).Build()
	mockey.Mock(lyrics.Search).Return("lyrics", nil).Build()
	mockey.Mock(processor.Do).Return(nil).Build()
	mockey.Mock(sys.FileMoveOrCopy).Return(errors.New("ko")).Build()

	// testing
	assert.EqualError(t, sys.ErrOnly(testExecute(cmdSync(), "--plain")), "ko")
}

func TestCmdSyncPlaylistEncoderFailure(t *testing.T) {
	t.Cleanup(cleanup)

	_playlist := &playlist.Playlist{Tracks: []*entity.Track{
		{ID: "TestCmdSyncPlaylistEncoderFailure", Title: "Title", Artists: []string{"Artist"}},
	}}

	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(cmd.ValidateEnvironment).Return(nil).Build()
	mockey.Mock(cmd.Open).Return(nil).Build()
	mockey.Mock(mockey.GetMethod(&index.Index{}, "Build")).Return(nil).Build()
	mockey.Mock(spotify.Authenticate).Return(&spotify.Client{}, nil).Build()
	mockey.Mock(mockey.GetMethod(&spotify.Client{}, "Playlist")).Return(_playlist, nil).Build()
	mockey.Mock(provider.Search).Return([]*provider.Match{{URL: "http://localhost/", Score: 0}}, nil).Build()
	mockey.Mock(downloader.Download).To(func(_, _ string, _ processor.Processor, ch ...chan []byte) error {
		for _, c := range ch {
			c <- []byte{}
		}
		return nil
	}).Build()
	mockey.Mock(lyrics.Search).Return("lyrics", nil).Build()
	mockey.Mock(processor.Do).Return(nil).Build()
	mockey.Mock(sys.FileMoveOrCopy).Return(nil).Build()
	mockey.Mock(mockey.GetMethod(playlist.Playlist{}, "Encoder")).Return(nil, errors.New("ko")).Build()

	// testing
	assert.EqualError(t, sys.ErrOnly(testExecute(cmdSync(), "--plain", "-p", "123")), "ko")
}

func TestCmdSyncPlaylistEncoderAddFailure(t *testing.T) {
	t.Cleanup(cleanup)

	_playlist := &playlist.Playlist{Tracks: []*entity.Track{
		{ID: "TestCmdSyncPlaylistEncoderAddFailure", Title: "Title", Artists: []string{"Artist"}},
	}}

	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(cmd.ValidateEnvironment).Return(nil).Build()
	mockey.Mock(cmd.Open).Return(nil).Build()
	mockey.Mock(mockey.GetMethod(&index.Index{}, "Build")).Return(nil).Build()
	mockey.Mock(spotify.Authenticate).Return(&spotify.Client{}, nil).Build()
	mockey.Mock(mockey.GetMethod(&spotify.Client{}, "Playlist")).To(func(_ string, ch ...chan interface{}) (*playlist.Playlist, error) {
		ch[0] <- _playlist.Tracks[0]
		return _playlist, nil
	}).Build()
	mockey.Mock(provider.Search).Return([]*provider.Match{{URL: "http://localhost/", Score: 0}}, nil).Build()
	mockey.Mock(downloader.Download).To(func(_, _ string, _ processor.Processor, ch ...chan []byte) error {
		for _, c := range ch {
			c <- []byte{}
		}
		return nil
	}).Build()
	mockey.Mock(lyrics.Search).Return("lyrics", nil).Build()
	mockey.Mock(processor.Do).Return(nil).Build()
	mockey.Mock(sys.FileMoveOrCopy).Return(nil).Build()
	mockey.Mock(mockey.GetMethod(&playlist.M3UEncoder{}, "Add")).Return(errors.New("ko")).Build()

	// testing
	assert.EqualError(t, sys.ErrOnly(testExecute(cmdSync(), "--plain", "-p", "123")), "ko")
}

func TestCmdSyncPlaylistEncoderCloseFailure(t *testing.T) {
	t.Cleanup(cleanup)

	_playlist := &playlist.Playlist{Tracks: []*entity.Track{
		{ID: "TestCmdSyncPlaylistEncoderCloseFailure", Title: "Title", Artists: []string{"Artist"}},
	}}

	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(cmd.ValidateEnvironment).Return(nil).Build()
	mockey.Mock(cmd.Open).Return(nil).Build()
	mockey.Mock(mockey.GetMethod(&index.Index{}, "Build")).Return(nil).Build()
	mockey.Mock(spotify.Authenticate).Return(&spotify.Client{}, nil).Build()
	mockey.Mock(mockey.GetMethod(&spotify.Client{}, "Playlist")).To(func(_ string, ch ...chan interface{}) (*playlist.Playlist, error) {
		ch[0] <- _playlist.Tracks[0]
		return _playlist, nil
	}).Build()
	mockey.Mock(provider.Search).Return([]*provider.Match{{URL: "http://localhost/", Score: 0}}, nil).Build()
	mockey.Mock(downloader.Download).To(func(_, _ string, _ processor.Processor, ch ...chan []byte) error {
		for _, c := range ch {
			c <- []byte{}
		}
		return nil
	}).Build()
	mockey.Mock(lyrics.Search).Return("lyrics", nil).Build()
	mockey.Mock(processor.Do).Return(nil).Build()
	mockey.Mock(sys.FileMoveOrCopy).Return(nil).Build()
	mockey.Mock(mockey.GetMethod(&playlist.M3UEncoder{}, "Close")).Return(errors.New("ko")).Build()

	// testing
	assert.EqualError(t, sys.ErrOnly(testExecute(cmdSync(), "--plain", "-p", "123")), "ko")
}
