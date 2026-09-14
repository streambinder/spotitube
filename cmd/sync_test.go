package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/bogem/id3v2/v2"
	"github.com/bytedance/mockey"
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
	"github.com/streambinder/spotitube/sys/anchor"
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

func cloneTrack(track *entity.Track) *entity.Track {
	copied := *track
	return &copied
}

func TestCmdSync(t *testing.T) {
	t.Cleanup(cleanup)

	var (
		_track         = &entity.Track{ID: "TestCmdSync", Title: "Title", Artists: []string{"Artist"}, Artwork: entity.Artwork{URL: "http://localhost/"}}
		_trackNotFound = &entity.Track{ID: "TestCmdSyncNotFound", Title: "Title Not Found", Artists: []string{"Artist"}}
		_playlist      = &playlist.Playlist{Tracks: []*entity.Track{_track, _trackNotFound}}
		_album         = &entity.Album{Tracks: []*entity.Track{_track}}
	)

	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(cmd.ValidateEnvironment).Return(nil).Build()
	mockey.Mock(cmd.Open).Return(nil).Build()
	mockey.Mock(mockey.GetMethod(&index.Index{}, "BuildWithProgress")).Return(nil).Build()
	mockey.Mock(spotify.Authenticate).Return(&spotify.Client{}, nil).Build()
	mockey.Mock(mockey.GetMethod(&spotify.Client{}, "Library")).To(func(_ int, ch ...chan interface{}) error {
		for _, c := range ch {
			c <- cloneTrack(_track)
			c <- cloneTrack(_track) // to trigger duplicate check
		}
		return nil
	}).Build()
	mockey.Mock(mockey.GetMethod(&spotify.Client{}, "Playlist")).To(func(_ string, ch ...chan interface{}) (*playlist.Playlist, error) {
		ch[0] <- cloneTrack(_track)
		ch[0] <- cloneTrack(_trackNotFound) // to skip inclusion in playlist
		return _playlist, nil
	}).Build()
	mockey.Mock(mockey.GetMethod(&spotify.Client{}, "Album")).To(func(_ string, ch ...chan interface{}) (*entity.Album, error) {
		ch[0] <- cloneTrack(_track)
		return _album, nil
	}).Build()
	mockey.Mock(mockey.GetMethod(&spotify.Client{}, "Track")).To(func(_ string, ch ...chan interface{}) (*entity.Track, error) {
		ch[0] <- cloneTrack(_track)
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
	mockey.Mock(downloader.Download).To(func(_ context.Context, _, _ string, _ processor.Processor, ch ...chan []byte) error {
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
	mockey.Mock(mockey.GetMethod(&index.Index{}, "BuildWithProgress")).To(func(data *index.Index, _ string, indexed chan<- string, _ ...int) error {
		data.Set(_track, index.Offline)
		if indexed != nil {
			indexed <- "Artist - Title.mp3"
		}
		return nil
	}).Build()
	mockey.Mock(spotify.Authenticate).Return(&spotify.Client{}, nil).Build()
	mockey.Mock(mockey.GetMethod(&spotify.Client{}, "Library")).To(func(_ int, ch ...chan interface{}) error {
		ch[0] <- cloneTrack(_track)
		return nil
	}).Build()
	mockey.Mock(provider.Search).Return([]*provider.Match{{URL: "http://localhost/", Score: 0}}, nil).Build()
	mockey.Mock(downloader.Download).To(func(_ context.Context, _, _ string, _ processor.Processor, ch ...chan []byte) error {
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
	mockey.Mock(mockey.GetMethod(&index.Index{}, "BuildWithProgress")).Return(errors.New("ko")).Build()

	// testing
	assert.EqualError(t, sys.ErrOnly(testExecute(cmdSync(), "--plain")), "ko")
}

func TestCmdSyncAuthFailure(t *testing.T) {
	t.Cleanup(cleanup)

	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(cmd.ValidateEnvironment).Return(nil).Build()
	mockey.Mock(cmd.Open).Return(nil).Build()
	mockey.Mock(mockey.GetMethod(&index.Index{}, "BuildWithProgress")).Return(nil).Build()
	mockey.Mock(spotify.Authenticate).Return(&spotify.Client{}, errors.New("ko")).Build()

	// testing
	assert.EqualError(t, sys.ErrOnly(testExecute(cmdSync(), "--plain")), "ko")
}

func TestRoutineAuthUsername(t *testing.T) {
	stdout := os.Stdout
	oldTUI := tui
	oldRoutineSemaphores := routineSemaphores
	defer func() {
		os.Stdout = stdout
	}()
	t.Cleanup(func() {
		tui = oldTUI
		routineSemaphores = oldRoutineSemaphores
	})
	reader, writer, err := os.Pipe()
	assert.Nil(t, err)
	os.Stdout = writer

	tui = anchor.New(anchor.Red)
	tui.EnablePlainMode()
	cmd := cmdSync()
	cmd.PreRun(cmd, nil)
	errChannel := make(chan error, 1)
	t.Cleanup(func() {
		close(errChannel)
	})

	defer mockey.UnPatchAll()
	mockey.Mock(spotify.Authenticate).Return(&spotify.Client{}, nil).Build()
	mockey.Mock(mockey.GetMethod(&spotify.Client{}, "Username")).Return("alice", nil).Build()

	routineAuth(context.Background(), errChannel)

	assert.Nil(t, writer.Close())
	output, err := io.ReadAll(reader)
	assert.Nil(t, err)
	assert.Contains(t, string(output), "auth alice")
	assert.Empty(t, errChannel)
	ok, open := <-routineSemaphores[routineTypeAuth]
	assert.True(t, open)
	assert.True(t, ok)
}

func TestCmdSyncLibraryFailure(t *testing.T) {
	t.Cleanup(cleanup)

	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(cmd.ValidateEnvironment).Return(nil).Build()
	mockey.Mock(cmd.Open).Return(nil).Build()
	mockey.Mock(mockey.GetMethod(&index.Index{}, "BuildWithProgress")).Return(nil).Build()
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
	mockey.Mock(mockey.GetMethod(&index.Index{}, "BuildWithProgress")).Return(nil).Build()
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
	mockey.Mock(mockey.GetMethod(&index.Index{}, "BuildWithProgress")).Return(nil).Build()
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
	mockey.Mock(mockey.GetMethod(&index.Index{}, "BuildWithProgress")).Return(nil).Build()
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
	mockey.Mock(mockey.GetMethod(&index.Index{}, "BuildWithProgress")).Return(nil).Build()
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
	mockey.Mock(mockey.GetMethod(&index.Index{}, "BuildWithProgress")).Return(nil).Build()
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
	mockey.Mock(mockey.GetMethod(&index.Index{}, "BuildWithProgress")).Return(nil).Build()
	mockey.Mock(spotify.Authenticate).Return(&spotify.Client{}, nil).Build()
	mockey.Mock(id3.Open).Return(&id3.Tag{}, nil).Build()
	mockey.Mock(mockey.GetMethod(&id3.Tag{}, "userDefinedText")).Return("123").Build()
	mockey.Mock(mockey.GetMethod(&id3v2.Tag{}, "Close")).Return(errors.New("ko")).Build()

	// testing
	assert.EqualError(t, sys.ErrOnly(testExecute(cmdSync(), "--plain", "-f", "path")), "ko")
}

func TestCmdSyncDecideManual(t *testing.T) {
	t.Cleanup(cleanup)

	_trackURL := &entity.Track{ID: "TestCmdSyncDecideManualURL", Title: "URL", Artists: []string{"Artist"}}
	_trackEmpty := &entity.Track{ID: "TestCmdSyncDecideManualEmpty", Title: "Empty", Artists: []string{"Artist"}}
	_trackSkip := &entity.Track{ID: "TestCmdSyncDecideManualSkip", Title: "Skip", Artists: []string{"Artist"}}

	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(cmd.ValidateEnvironment).Return(nil).Build()
	mockey.Mock(cmd.Open).Return(nil).Build()
	mockey.Mock(mockey.GetMethod(&index.Index{}, "BuildWithProgress")).Return(nil).Build()
	mockey.Mock(spotify.Authenticate).Return(&spotify.Client{}, nil).Build()
	mockey.Mock(mockey.GetMethod(&spotify.Client{}, "Library")).To(func(_ int, ch ...chan interface{}) error {
		ch[0] <- cloneTrack(_trackURL)
		ch[0] <- cloneTrack(_trackEmpty)
		ch[0] <- cloneTrack(_trackSkip)
		return nil
	}).Build()
	mockey.Mock(mockey.GetMethod(&id3.Tag{}, "userDefinedText")).Return("123").Build()
	mockey.Mock(mockey.GetMethod(&id3.Tag{}, "Close")).Return(nil).Build()
	mockey.Mock(downloader.Download).To(func(_ context.Context, _, _ string, _ processor.Processor, ch ...chan []byte) error {
		for _, c := range ch {
			c <- []byte{}
		}
		return nil
	}).Build()
	mockey.Mock(lyrics.Search).Return("lyrics", nil).Build()
	mockey.Mock(processor.Do).Return(nil).Build()
	mockey.Mock(sys.FileMoveOrCopy).Return(nil).Build()
	mockey.Mock(mockey.GetMethod(&playlist.M3UEncoder{}, "Close")).Return(nil).Build()

	// stdin feeds the manual prompts: the first track gets a URL, the second hits EOF
	stdinReader, stdinWriter, err := os.Pipe()
	assert.Nil(t, err)
	_, err = stdinWriter.WriteString("http://localhost/\n")
	assert.Nil(t, err)
	assert.Nil(t, stdinWriter.Close())
	oldStdin := os.Stdin
	os.Stdin = stdinReader
	defer func() { os.Stdin = oldStdin }()

	// the skipped track is already synced
	indexData.Set(_trackSkip, index.Online)

	// testing: answered, empty and skipped tracks are decided without errors
	err = sys.ErrOnly(testExecute(cmdSync(), "--plain", "--manual"))
	assert.Nil(t, err)
}

func TestCmdSyncDecideManualCollision(t *testing.T) {
	t.Cleanup(cleanup)

	_trackCollision := &entity.Track{ID: "TestCmdSyncDecideManualCollision", Title: "Collision", Artists: []string{"Artist"}}
	_trackOccupant := &entity.Track{ID: "TestCmdSyncDecideManualOccupant", Title: "Collision", Artists: []string{"Artist"}}

	// the filename is already owned by another track
	indexData.Set(_trackOccupant, index.Offline)

	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(cmd.ValidateEnvironment).Return(nil).Build()
	mockey.Mock(cmd.Open).Return(nil).Build()
	mockey.Mock(mockey.GetMethod(&index.Index{}, "BuildWithProgress")).Return(nil).Build()
	mockey.Mock(spotify.Authenticate).Return(&spotify.Client{}, nil).Build()
	mockey.Mock(mockey.GetMethod(&spotify.Client{}, "Library")).To(func(_ int, ch ...chan interface{}) error {
		ch[0] <- cloneTrack(_trackCollision)
		return nil
	}).Build()

	// testing: the sync aborts early with a filename collision error
	err := sys.ErrOnly(testExecute(cmdSync(), "--plain", "--manual"))
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "filename collision")
}

func TestCmdSyncFatalCollisionUnwinds(t *testing.T) {
	t.Cleanup(cleanup)

	const total = 5000
	_existing := &entity.Track{ID: "TestCmdSyncFatalCollisionUnwindsExisting", Title: "Collision", Artists: []string{"Artist"}}
	_colliding := &entity.Track{ID: "TestCmdSyncFatalCollisionUnwindsColliding", Title: "Collision", Artists: []string{"Artist"}}

	// the filename is already owned by another track
	indexData.Set(_existing, index.Offline)

	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(cmd.ValidateEnvironment).Return(nil).Build()
	mockey.Mock(cmd.Open).Return(nil).Build()
	mockey.Mock(mockey.GetMethod(&index.Index{}, "BuildWithProgress")).Return(nil).Build()
	mockey.Mock(spotify.Authenticate).Return(&spotify.Client{}, nil).Build()
	mockey.Mock(mockey.GetMethod(&spotify.Client{}, "Library")).To(func(_ int, ch ...chan interface{}) error {
		// the collider goes first so the fatal fires while the
		// fetcher is still blocked on the full decide queue
		for i := -1; i < total; i++ {
			track := &entity.Track{ID: fmt.Sprintf("TestCmdSyncFatalCollisionUnwinds%d", i), Title: fmt.Sprintf("Title %d", i), Artists: []string{"Artist"}}
			if i == -1 {
				track = cloneTrack(_colliding)
			}
			for _, c := range ch {
				c <- track
			}
		}
		return nil
	}).Build()
	mockey.Mock(id3.Open).Return(&id3.Tag{}, nil).Build()
	mockey.Mock(mockey.GetMethod(&id3.Tag{}, "userDefinedText")).Return("123").Build()
	mockey.Mock(mockey.GetMethod(&id3.Tag{}, "Close")).Return(nil).Build()
	mockey.Mock(provider.Search).To(func(*entity.Track) ([]*provider.Match, error) {
		return []*provider.Match{{URL: "http://localhost/", Score: 0}}, nil
	}).Build()
	mockey.Mock(downloader.Download).To(func(_ context.Context, _, _ string, _ processor.Processor, ch ...chan []byte) error {
		for _, c := range ch {
			c <- []byte{}
		}
		return nil
	}).Build()
	mockey.Mock(lyrics.Search).Return("lyrics", nil).Build()
	mockey.Mock(processor.Do).Return(nil).Build()
	mockey.Mock(sys.FileMoveOrCopy).Return(nil).Build()
	mockey.Mock(mockey.GetMethod(&playlist.M3UEncoder{}, "Close")).Return(nil).Build()

	// testing: a fatal collision while the fetcher is blocked on a full
	// decide queue unwinds the sync with the collision error instead of
	// deadlocking
	err := sys.ErrOnly(testExecute(cmdSync(), "--plain"))
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "filename collision")
}

func TestDecideWorkerContextCancel(t *testing.T) {
	// an open, empty queue: the only ready select case is the cancelled context
	routineQueues = map[int](chan interface{}){
		routineTypeDecide: make(chan interface{}),
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	var consecutiveFailures atomic.Int32
	done := make(chan struct{})
	go func() {
		decideWorker(ctx, make(chan error, 1), &consecutiveFailures)
		close(done)
	}()

	// testing: the worker stops promptly on a cancelled context
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("decideWorker did not stop on cancelled context")
	}
}

func TestCollectWorkerContextCancel(t *testing.T) {
	// an open, empty queue: the only ready select case is the cancelled context
	routineQueues = map[int](chan interface{}){
		routineTypeCollect: make(chan interface{}),
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	done := make(chan struct{})
	go func() {
		collectWorker(ctx, false)
		close(done)
	}()

	// testing: the worker stops promptly on a cancelled context
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("collectWorker did not stop on cancelled context")
	}
}

func TestProcessWorkerContextCancel(t *testing.T) {
	// an open, empty queue: the only ready select case is the cancelled context
	routineQueues = map[int](chan interface{}){
		routineTypeProcess: make(chan interface{}),
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	done := make(chan struct{})
	go func() {
		processWorker(ctx, make(chan error, 1))
		close(done)
	}()

	// testing: the worker stops promptly on a cancelled context
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("processWorker did not stop on cancelled context")
	}
}

func TestRoutineProcessFanout(t *testing.T) {
	const tracks = 10

	routineQueues = map[int](chan interface{}){
		routineTypeProcess: make(chan interface{}, tracks),
		routineTypeInstall: make(chan interface{}, tracks),
	}

	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(processor.Do).Return(nil).Build()

	for i := 0; i < tracks; i++ {
		routineQueues[routineTypeProcess] <- &entity.Track{
			ID:      fmt.Sprintf("processfanout%d", i),
			Title:   "Title",
			Artists: []string{"Artist"},
		}
	}
	close(routineQueues[routineTypeProcess])

	// testing: every track is processed and forwarded, then the install queue closes
	routineProcess(context.Background(), make(chan error, 1))
	processed := 0
	for range routineQueues[routineTypeInstall] {
		processed++
	}
	assert.Equal(t, tracks, processed)
}

func TestRoutineProcessFailure(t *testing.T) {
	routineQueues = map[int](chan interface{}){
		routineTypeProcess: make(chan interface{}, 1),
		routineTypeInstall: make(chan interface{}, 1),
	}

	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(processor.Do).Return(errors.New("ko")).Build()

	routineQueues[routineTypeProcess] <- &entity.Track{ID: "processfail", Title: "Title", Artists: []string{"Artist"}}
	close(routineQueues[routineTypeProcess])

	// testing: a processing failure aborts the sync
	errCh := make(chan error, 1)
	routineProcess(context.Background(), errCh)
	select {
	case err := <-errCh:
		assert.Contains(t, err.Error(), "ko")
	case <-time.After(10 * time.Second):
		t.Fatal("routineProcess did not report the processing failure")
	}
}

func TestRoutineProcessLoudnessSkipped(t *testing.T) {
	routineQueues = map[int](chan interface{}){
		routineTypeProcess: make(chan interface{}, 1),
		routineTypeInstall: make(chan interface{}, 1),
	}

	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(processor.Do).Return(fmt.Errorf("%w: ko", processor.ErrLoudnessSkipped)).Build()

	routineQueues[routineTypeProcess] <- &entity.Track{ID: "processskip", Title: "Title", Artists: []string{"Artist"}}
	close(routineQueues[routineTypeProcess])

	// testing: a skipped normalization forwards the track without aborting the sync
	errCh := make(chan error, 1)
	done := make(chan struct{})
	go func() {
		routineProcess(context.Background(), errCh)
		close(done)
	}()
	select {
	case track := <-routineQueues[routineTypeInstall]:
		assert.Equal(t, "processskip", track.(*entity.Track).ID)
	case <-time.After(10 * time.Second):
		t.Fatal("track was not forwarded to the installer")
	}
	<-done
	select {
	case err := <-errCh:
		t.Fatalf("routineProcess aborted the sync: %s", err)
	default:
	}
}

func TestRoutineCollectFanout(t *testing.T) {
	const tracks = 10

	routineQueues = map[int](chan interface{}){
		routineTypeCollect: make(chan interface{}, tracks),
		routineTypeProcess: make(chan interface{}, tracks),
	}

	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(downloader.Download).To(func(_ context.Context, _, _ string, _ processor.Processor, ch ...chan []byte) error {
		for _, c := range ch {
			c <- []byte{}
		}
		return nil
	}).Build()
	mockey.Mock(lyrics.Search).Return("lyrics", nil).Build()

	for i := 0; i < tracks; i++ {
		routineQueues[routineTypeCollect] <- &entity.Track{
			ID:      fmt.Sprintf("fanout%d", i),
			Title:   "Title",
			Artists: []string{"Artist"},
			Artwork: entity.Artwork{URL: "http://localhost/cover.jpg"},
		}
	}
	close(routineQueues[routineTypeCollect])

	// testing: every track is collected and forwarded, then the process queue closes
	routineCollect(false)(context.Background(), make(chan error, 1))
	collected := 0
	for range routineQueues[routineTypeProcess] {
		collected++
	}
	assert.Equal(t, tracks, collected)
}

func TestCmdSyncDecideFailure(t *testing.T) {
	t.Cleanup(cleanup)

	_track := &entity.Track{ID: "TestCmdSyncDecideFailure", Title: "Title", Artists: []string{"Artist"}}

	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(cmd.ValidateEnvironment).Return(nil).Build()
	mockey.Mock(cmd.Open).Return(nil).Build()
	mockey.Mock(mockey.GetMethod(&index.Index{}, "BuildWithProgress")).Return(nil).Build()
	mockey.Mock(spotify.Authenticate).Return(&spotify.Client{}, nil).Build()
	mockey.Mock(mockey.GetMethod(&spotify.Client{}, "Library")).To(func(_ int, ch ...chan interface{}) error {
		ch[0] <- cloneTrack(_track)
		return nil
	}).Build()
	mockey.Mock(provider.Search).To(func(*entity.Track) ([]*provider.Match, error) {
		return nil, errors.New("ko")
	}).Build()

	// testing: search failure is non-fatal, track gets skipped
	assert.Nil(t, sys.ErrOnly(testExecute(cmdSync(), "--plain")))
}

func TestCmdSyncDecideCircuitBreaker(t *testing.T) {
	t.Cleanup(cleanup)

	// the breaker count is only deterministic with a single worker
	prevDecideWorkers := decideWorkers
	decideWorkers = 1
	defer func() { decideWorkers = prevDecideWorkers }()

	tracks := []*entity.Track{
		{ID: "cb1", Title: "Title1", Artists: []string{"Artist"}},
		{ID: "cb2", Title: "Title2", Artists: []string{"Artist"}},
		{ID: "cb3", Title: "Title3", Artists: []string{"Artist"}},
		{ID: "cb4", Title: "Title4", Artists: []string{"Artist"}},
	}

	// monkey patching
	searchCount := 0
	defer mockey.UnPatchAll()
	mockey.Mock(cmd.ValidateEnvironment).Return(nil).Build()
	mockey.Mock(cmd.Open).Return(nil).Build()
	mockey.Mock(mockey.GetMethod(&index.Index{}, "BuildWithProgress")).Return(nil).Build()
	mockey.Mock(spotify.Authenticate).Return(&spotify.Client{}, nil).Build()
	mockey.Mock(mockey.GetMethod(&spotify.Client{}, "Library")).To(func(_ int, ch ...chan interface{}) error {
		for _, track := range tracks {
			ch[0] <- track
		}
		return nil
	}).Build()
	mockey.Mock(provider.Search).To(func(*entity.Track) ([]*provider.Match, error) {
		searchCount++
		return nil, errors.New("ko")
	}).Build()

	// testing: 4th track should be skipped by circuit breaker (only 3 searches)
	assert.Nil(t, sys.ErrOnly(testExecute(cmdSync(), "--plain")))
	assert.Equal(t, 3, searchCount)
}

func TestCmdSyncDecideNotFound(t *testing.T) {
	t.Cleanup(cleanup)

	_track := &entity.Track{ID: "TestCmdSyncDecideNotFound", Title: "Title", Artists: []string{"Artist"}}

	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(cmd.ValidateEnvironment).Return(nil).Build()
	mockey.Mock(cmd.Open).Return(nil).Build()
	mockey.Mock(mockey.GetMethod(&index.Index{}, "BuildWithProgress")).Return(nil).Build()
	mockey.Mock(spotify.Authenticate).Return(&spotify.Client{}, nil).Build()
	mockey.Mock(mockey.GetMethod(&spotify.Client{}, "Library")).To(func(_ int, ch ...chan interface{}) error {
		ch[0] <- cloneTrack(_track)
		return nil
	}).Build()
	mockey.Mock(provider.Search).To(func(*entity.Track) ([]*provider.Match, error) {
		return []*provider.Match{}, nil
	}).Build()

	// testing
	assert.Nil(t, sys.ErrOnly(testExecute(cmdSync(), "--plain")))
}

func TestCmdSyncDecideParallelFanout(t *testing.T) {
	t.Cleanup(cleanup)

	var tracks []*entity.Track
	for i := 0; i < 10; i++ {
		tracks = append(tracks, &entity.Track{
			ID:      fmt.Sprintf("TestCmdSyncDecideParallelFanout%d", i),
			Title:   fmt.Sprintf("Title%d", i),
			Artists: []string{"Artist"},
		})
	}

	// monkey patching
	searched := map[string]int{}
	var mu sync.Mutex
	defer mockey.UnPatchAll()
	mockey.Mock(cmd.ValidateEnvironment).Return(nil).Build()
	mockey.Mock(cmd.Open).Return(nil).Build()
	mockey.Mock(mockey.GetMethod(&index.Index{}, "BuildWithProgress")).Return(nil).Build()
	mockey.Mock(spotify.Authenticate).Return(&spotify.Client{}, nil).Build()
	mockey.Mock(mockey.GetMethod(&spotify.Client{}, "Library")).To(func(_ int, ch ...chan interface{}) error {
		for _, track := range tracks {
			ch[0] <- track
		}
		return nil
	}).Build()
	mockey.Mock(provider.Search).To(func(track *entity.Track) ([]*provider.Match, error) {
		mu.Lock()
		searched[track.ID]++
		mu.Unlock()
		return []*provider.Match{}, nil
	}).Build()

	// testing: every track is searched exactly once, none is lost or duplicated
	assert.Nil(t, sys.ErrOnly(testExecute(cmdSync(), "--plain")))
	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, len(tracks), len(searched))
	for _, track := range tracks {
		assert.Equal(t, 1, searched[track.ID])
	}
}

func TestCmdSyncDecideFilenameCollision(t *testing.T) {
	t.Cleanup(cleanup)

	_existing := &entity.Track{ID: "TestCmdSyncDecideFilenameCollisionA", Title: "Title", Artists: []string{"Artist"}}
	_colliding := &entity.Track{ID: "TestCmdSyncDecideFilenameCollisionB", Title: "Title", Artists: []string{"Artist"}}

	// the filename is already owned by another track
	indexData.Set(_existing, index.Offline)

	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(cmd.ValidateEnvironment).Return(nil).Build()
	mockey.Mock(cmd.Open).Return(nil).Build()
	mockey.Mock(mockey.GetMethod(&index.Index{}, "BuildWithProgress")).Return(nil).Build()
	mockey.Mock(spotify.Authenticate).Return(&spotify.Client{}, nil).Build()
	mockey.Mock(mockey.GetMethod(&spotify.Client{}, "Library")).To(func(_ int, ch ...chan interface{}) error {
		ch[0] <- cloneTrack(_colliding)
		return nil
	}).Build()

	// testing: the sync aborts early with a filename collision error
	err := sys.ErrOnly(testExecute(cmdSync(), "--plain"))
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "filename collision")
}

func TestCmdSyncCollectFailure(t *testing.T) {
	t.Cleanup(cleanup)

	// the artwork download is the failing one: any URL other than
	// "http://localhost/" makes the mocked downloader return an error
	_track := &entity.Track{
		ID: "TestCmdSyncCollectFailure", Title: "Title", Artists: []string{"Artist"},
		Artwork: entity.Artwork{URL: "http://localhost/ko"},
	}

	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(cmd.ValidateEnvironment).Return(nil).Build()
	mockey.Mock(cmd.Open).Return(nil).Build()
	mockey.Mock(mockey.GetMethod(&index.Index{}, "BuildWithProgress")).Return(nil).Build()
	mockey.Mock(spotify.Authenticate).Return(&spotify.Client{}, nil).Build()
	mockey.Mock(mockey.GetMethod(&spotify.Client{}, "Library")).To(func(_ int, ch ...chan interface{}) error {
		ch[0] <- cloneTrack(_track)
		return nil
	}).Build()
	mockey.Mock(provider.Search).To(func(*entity.Track) ([]*provider.Match, error) {
		return []*provider.Match{{URL: "http://localhost/", Score: 0}}, nil
	}).Build()
	mockey.Mock(downloader.Download).To(func(_ context.Context, url string, _ string, _ processor.Processor, ch ...chan []byte) error {
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
	// testing: the failing track is skipped, the sync completes
	assert.Nil(t, sys.ErrOnly(testExecute(cmdSync(), "--plain")))
	assert.Equal(t, 0, indexData.Size(index.Installed))
}

func TestCmdSyncDownloadFailure(t *testing.T) {
	t.Cleanup(cleanup)

	_track := &entity.Track{ID: "TestCmdSyncDownloadFailure", Title: "Title", Artists: []string{"Artist"}}

	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(cmd.ValidateEnvironment).Return(nil).Build()
	mockey.Mock(cmd.Open).Return(nil).Build()
	mockey.Mock(mockey.GetMethod(&index.Index{}, "BuildWithProgress")).Return(nil).Build()
	mockey.Mock(spotify.Authenticate).Return(&spotify.Client{}, nil).Build()
	mockey.Mock(mockey.GetMethod(&spotify.Client{}, "Library")).To(func(_ int, ch ...chan interface{}) error {
		ch[0] <- cloneTrack(_track)
		return nil
	}).Build()
	mockey.Mock(provider.Search).To(func(*entity.Track) ([]*provider.Match, error) {
		return []*provider.Match{{URL: "http://localhost/", Score: 0}}, nil
	}).Build()
	mockey.Mock(downloader.Download).To(func(_ context.Context, _, _ string, _ processor.Processor, ch ...chan []byte) error {
		for _, c := range ch {
			c <- []byte{}
		}
		return errors.New("ko")
	}).Build()
	mockey.Mock(lyrics.Search).Return("", nil).Build()

	// testing
	// testing: the failing track is skipped, the sync completes
	assert.Nil(t, sys.ErrOnly(testExecute(cmdSync(), "--plain")))
	assert.Equal(t, 0, indexData.Size(index.Installed))
}

func TestCmdSyncLyricsFailure(t *testing.T) {
	t.Cleanup(cleanup)

	_track := &entity.Track{ID: "TestCmdSyncLyricsFailure", Title: "Title", Artists: []string{"Artist"}}

	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(cmd.ValidateEnvironment).Return(nil).Build()
	mockey.Mock(cmd.Open).Return(nil).Build()
	mockey.Mock(mockey.GetMethod(&index.Index{}, "BuildWithProgress")).Return(nil).Build()
	mockey.Mock(spotify.Authenticate).Return(&spotify.Client{}, nil).Build()
	mockey.Mock(mockey.GetMethod(&spotify.Client{}, "Library")).To(func(_ int, ch ...chan interface{}) error {
		ch[0] <- cloneTrack(_track)
		return nil
	}).Build()
	mockey.Mock(provider.Search).Return([]*provider.Match{{URL: "http://localhost/", Score: 0}}, nil).Build()
	mockey.Mock(downloader.Download).To(func(_ context.Context, _, _ string, _ processor.Processor, ch ...chan []byte) error {
		for _, c := range ch {
			c <- []byte{}
		}
		return nil
	}).Build()
	mockey.Mock(lyrics.Search).Return("", errors.New("ko")).Build()
	mockey.Mock(processor.Do).Return(nil).Build()
	mockey.Mock(sys.FileMoveOrCopy).Return(nil).Build()

	// testing: a lyrics failure degrades to a warning,
	// the track is installed anyway and the sync completes
	assert.Nil(t, sys.ErrOnly(testExecute(cmdSync(), "--plain")))
	assert.Equal(t, 1, indexData.Size(index.Installed))
}

func TestCmdSyncProcessorFailure(t *testing.T) {
	t.Cleanup(cleanup)

	_track := &entity.Track{ID: "TestCmdSyncProcessorFailure", Title: "Title", Artists: []string{"Artist"}}

	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(cmd.ValidateEnvironment).Return(nil).Build()
	mockey.Mock(cmd.Open).Return(nil).Build()
	mockey.Mock(mockey.GetMethod(&index.Index{}, "BuildWithProgress")).Return(nil).Build()
	mockey.Mock(spotify.Authenticate).Return(&spotify.Client{}, nil).Build()
	mockey.Mock(mockey.GetMethod(&spotify.Client{}, "Library")).To(func(_ int, ch ...chan interface{}) error {
		ch[0] <- cloneTrack(_track)
		return nil
	}).Build()
	mockey.Mock(provider.Search).Return([]*provider.Match{{URL: "http://localhost/", Score: 0}}, nil).Build()
	mockey.Mock(downloader.Download).To(func(_ context.Context, _, _ string, _ processor.Processor, ch ...chan []byte) error {
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
	mockey.Mock(mockey.GetMethod(&index.Index{}, "BuildWithProgress")).Return(nil).Build()
	mockey.Mock(spotify.Authenticate).Return(&spotify.Client{}, nil).Build()
	mockey.Mock(mockey.GetMethod(&spotify.Client{}, "Library")).To(func(_ int, ch ...chan interface{}) error {
		ch[0] <- cloneTrack(_track)
		return nil
	}).Build()
	mockey.Mock(provider.Search).Return([]*provider.Match{{URL: "http://localhost/", Score: 0}}, nil).Build()
	mockey.Mock(downloader.Download).To(func(_ context.Context, _, _ string, _ processor.Processor, ch ...chan []byte) error {
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
	mockey.Mock(mockey.GetMethod(&index.Index{}, "BuildWithProgress")).Return(nil).Build()
	mockey.Mock(spotify.Authenticate).Return(&spotify.Client{}, nil).Build()
	mockey.Mock(mockey.GetMethod(&spotify.Client{}, "Playlist")).Return(_playlist, nil).Build()
	mockey.Mock(provider.Search).Return([]*provider.Match{{URL: "http://localhost/", Score: 0}}, nil).Build()
	mockey.Mock(downloader.Download).To(func(_ context.Context, _, _ string, _ processor.Processor, ch ...chan []byte) error {
		for _, c := range ch {
			c <- []byte{}
		}
		return nil
	}).Build()
	mockey.Mock(lyrics.Search).Return("lyrics", nil).Build()
	mockey.Mock(processor.Do).Return(nil).Build()
	mockey.Mock(sys.FileMoveOrCopy).Return(nil).Build()
	mockey.Mock(mockey.GetMethod(playlist.Playlist{}, "Encoder")).Return(nil, errors.New("ko")).Build()

	// testing: the broken playlist is skipped, the run survives
	assert.Nil(t, sys.ErrOnly(testExecute(cmdSync(), "--plain", "-p", "123")))
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
	mockey.Mock(mockey.GetMethod(&index.Index{}, "BuildWithProgress")).Return(nil).Build()
	mockey.Mock(spotify.Authenticate).Return(&spotify.Client{}, nil).Build()
	mockey.Mock(mockey.GetMethod(&spotify.Client{}, "Playlist")).To(func(_ string, ch ...chan interface{}) (*playlist.Playlist, error) {
		ch[0] <- _playlist.Tracks[0]
		return _playlist, nil
	}).Build()
	mockey.Mock(provider.Search).Return([]*provider.Match{{URL: "http://localhost/", Score: 0}}, nil).Build()
	mockey.Mock(downloader.Download).To(func(_ context.Context, _, _ string, _ processor.Processor, ch ...chan []byte) error {
		for _, c := range ch {
			c <- []byte{}
		}
		return nil
	}).Build()
	mockey.Mock(lyrics.Search).Return("lyrics", nil).Build()
	mockey.Mock(processor.Do).Return(nil).Build()
	mockey.Mock(sys.FileMoveOrCopy).Return(nil).Build()
	mockey.Mock(mockey.GetMethod(&playlist.M3UEncoder{}, "Add")).Return(errors.New("ko")).Build()

	// testing: the failed add stops the playlist, the run survives
	assert.Nil(t, sys.ErrOnly(testExecute(cmdSync(), "--plain", "-p", "123")))
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
	mockey.Mock(mockey.GetMethod(&index.Index{}, "BuildWithProgress")).Return(nil).Build()
	mockey.Mock(spotify.Authenticate).Return(&spotify.Client{}, nil).Build()
	mockey.Mock(mockey.GetMethod(&spotify.Client{}, "Playlist")).To(func(_ string, ch ...chan interface{}) (*playlist.Playlist, error) {
		ch[0] <- _playlist.Tracks[0]
		return _playlist, nil
	}).Build()
	mockey.Mock(provider.Search).Return([]*provider.Match{{URL: "http://localhost/", Score: 0}}, nil).Build()
	mockey.Mock(downloader.Download).To(func(_ context.Context, _, _ string, _ processor.Processor, ch ...chan []byte) error {
		for _, c := range ch {
			c <- []byte{}
		}
		return nil
	}).Build()
	mockey.Mock(lyrics.Search).Return("lyrics", nil).Build()
	mockey.Mock(processor.Do).Return(nil).Build()
	mockey.Mock(sys.FileMoveOrCopy).Return(nil).Build()
	mockey.Mock(mockey.GetMethod(&playlist.M3UEncoder{}, "Close")).Return(errors.New("ko")).Build()

	// testing: the broken playlist is dropped, the run survives
	assert.Nil(t, sys.ErrOnly(testExecute(cmdSync(), "--plain", "-p", "123")))
}

func TestRoutineCollectArtworkEmptyURL(t *testing.T) {
	// an empty artwork URL must not deadlock waiting on the artwork channel
	track := &entity.Track{Title: "Title", Artists: []string{"Artist"}}

	done := make(chan struct{})
	go func() {
		defer close(done)
		routineCollectArtwork(track)(context.Background(), make(chan error, 1))
	}()

	select {
	case <-done:
		assert.Nil(t, track.Artwork.Data)
	case <-time.After(10 * time.Second):
		t.Fatal("routineCollectArtwork deadlocked on empty artwork URL")
	}
}

func TestCmdSyncFixRemoveFailure(t *testing.T) {
	t.Cleanup(cleanup)

	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(cmd.ValidateEnvironment).Return(nil).Build()
	mockey.Mock(cmd.Open).Return(nil).Build()
	mockey.Mock(mockey.GetMethod(&index.Index{}, "BuildWithProgress")).Return(nil).Build()
	mockey.Mock(spotify.Authenticate).Return(&spotify.Client{}, nil).Build()
	mockey.Mock(id3.Open).Return(&id3.Tag{}, nil).Build()
	mockey.Mock(mockey.GetMethod(&id3.Tag{}, "userDefinedText")).Return("123").Build()
	mockey.Mock(mockey.GetMethod(&id3.Tag{}, "Close")).Return(nil).Build()
	mockey.Mock(os.Remove).Return(errors.New("ko")).Build()

	// testing
	assert.EqualError(t, sys.ErrOnly(testExecute(cmdSync(), "--plain", "-f", "path")), "ko")
}
