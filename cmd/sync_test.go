package cmd

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/bogem/id3v2/v2"
	"github.com/streambinder/spotitube/entity"
	"github.com/streambinder/spotitube/entity/id3"
	"github.com/streambinder/spotitube/entity/index"
	"github.com/streambinder/spotitube/sys"
	"github.com/stretchr/testify/assert"
)

func BenchmarkSync(b *testing.B) {
	for i := 0; i < b.N; i++ {
		TestCmdSync(&testing.T{})
	}
}

func cleanup() {
	indexData = index.New()
	ignoreCollisions = false
	collisions.Store(0)
}

// syncEnv prepares a full sync environment: fresh package state, a
// seeded session, fake tool binaries, scripted transports and cleared
// per-track caches. It returns the empty music directory to sync into
// and the fake binaries directory.
func syncEnv(t *testing.T, overrides map[string]func(*http.Request) (int, string), web webScript) (string, string) {
	t.Helper()
	resetState(t)
	seedSession(t)
	binDir := fakeBinDir(t)
	installWeb(t, overrides, web)
	clearTrackCache(t, "123", "456", "789", "999")
	return t.TempDir(), binDir
}

// writeTaggedFile creates a real audio file carrying the given Spotify
// ID in its tag.
func writeTaggedFile(t *testing.T, path, spotifyID string) {
	t.Helper()
	assert.Nil(t, os.WriteFile(path, []byte{}, 0o644))

	tag, err := id3.Open(path, id3v2.Options{Parse: true})
	assert.Nil(t, err)
	tag.SetTitle("Title")
	tag.SetArtist("Artist")
	if spotifyID != "" {
		tag.SetSpotifyID(spotifyID)
	}
	assert.Nil(t, tag.Save())
	assert.Nil(t, tag.Close())
}

// initRoutines initializes the pipeline queues and semaphores the way
// the sync command's PreRun does, for tests calling routines directly.
func initRoutines() {
	routineSemaphores = map[int](chan bool){
		routineTypeIndex:   make(chan bool, 1),
		routineTypeAuth:    make(chan bool, 1),
		routineTypeInstall: make(chan bool, 1),
	}
	routineQueues = map[int](chan interface{}){
		routineTypeDecide:  make(chan interface{}, pipelineBuffer),
		routineTypeCollect: make(chan interface{}, pipelineBuffer),
		routineTypeProcess: make(chan interface{}, pipelineBuffer),
		routineTypeInstall: make(chan interface{}, pipelineBuffer),
		routineTypeMix:     make(chan interface{}, pipelineBuffer),
	}
}

func TestCmdSync(t *testing.T) {
	musicDir, _ := syncEnv(t, map[string]func(*http.Request) (int, string){
		// the library carries the fixture track twice: the duplicate is
		// skipped by the decider
		"/v1/me/tracks": func(*http.Request) (int, string) { return 200, libraryBody("123", "123") },
	}, webScript{qobuzEmpty: true})

	// testing: with no collection supplied, the library is auto-enabled
	cmd := cmdSync()
	assert.Nil(t, sys.ErrOnly(testExecute(cmd, "--plain", "-o", musicDir)))
	library, err := cmd.Flags().GetBool("library")
	assert.Nil(t, err)
	assert.True(t, library)

	// testing: a full sync across every collection kind
	musicDir, _ = syncEnv(t, nil, webScript{qobuzEmpty: true})
	assert.Nil(t, sys.ErrOnly(testExecute(cmdSync(),
		"--plain", "-o", musicDir, "-l", "-p", "Mix", "-a", "album123", "-t", "123")))
	_, err = os.Stat(filepath.Join(musicDir, "Artist - Title.mp3"))
	assert.Nil(t, err)

	// the playlist file lists the installed track
	mix, err := os.ReadFile(filepath.Join(musicDir, "Mix.m3u"))
	assert.Nil(t, err)
	assert.Contains(t, string(mix), "Artist - Title.mp3")
}

func TestCmdSyncInvalidEnvironment(t *testing.T) {
	resetState(t)
	seedSession(t)
	// an empty PATH leaves ffmpeg and yt-dlp unreachable
	t.Setenv("PATH", t.TempDir())

	// testing
	assert.NotNil(t, sys.ErrOnly(testExecute(cmdSync(), "-o", t.TempDir())))
}

func TestCmdSyncOfflineIndex(t *testing.T) {
	musicDir, _ := syncEnv(t, nil, webScript{qobuzEmpty: true})
	writeTaggedFile(t, filepath.Join(musicDir, "Artist - Title.mp3"), "123")
	before, err := os.ReadFile(filepath.Join(musicDir, "Artist - Title.mp3"))
	assert.Nil(t, err)

	// testing: the track is already in the library as offline, so the
	// sync skips it and leaves the file untouched
	assert.Nil(t, sys.ErrOnly(testExecute(cmdSync(), "--plain", "-o", musicDir, "-l")))
	after, err := os.ReadFile(filepath.Join(musicDir, "Artist - Title.mp3"))
	assert.Nil(t, err)
	assert.Equal(t, before, after)
}

func TestCmdSyncPathFailure(t *testing.T) {
	resetState(t)
	seedSession(t)
	fakeBinDir(t)
	installWeb(t, nil, webScript{})

	// testing: the output path does not exist, the chdir fails
	missing := filepath.Join(t.TempDir(), "missing")
	assert.NotNil(t, sys.ErrOnly(testExecute(cmdSync(), "-o", missing)))
}

func TestCmdSyncAuthFailure(t *testing.T) {
	resetState(t)
	noSession(t)
	fakeBinDir(t)
	installWeb(t, nil, webScript{})
	noOpenerPath(t)
	occupyAuthPort(t)

	// testing
	assert.NotNil(t, sys.ErrOnly(testExecute(cmdSync(), "-o", t.TempDir())))
}

func TestRoutineAuthUsername(t *testing.T) {
	resetState(t)
	initRoutines()
	seedSession(t)
	meCalls := 0
	installWeb(t, map[string]func(*http.Request) (int, string){
		"/v1/me": func(request *http.Request) (int, string) {
			if request.URL.Path != "/v1/me" {
				return 200, libraryBody("123")
			}
			meCalls++
			if meCalls == 1 {
				return 200, meJSON
			}
			return apiError(500, "ko")
		},
	}, webScript{})

	// testing: a username resolution failure is tolerated, the routine
	// still signals a successful authentication
	ch := make(chan error, 1)
	routineAuth(context.Background(), ch)
	assert.True(t, <-routineSemaphores[routineTypeAuth])
	assert.Empty(t, ch)
	assert.NotNil(t, spotifyClient)
}

func TestCmdSyncLibraryFailure(t *testing.T) {
	musicDir, _ := syncEnv(t, map[string]func(*http.Request) (int, string){
		"/v1/me/tracks": func(*http.Request) (int, string) { return apiError(500, "ko") },
	}, webScript{qobuzEmpty: true})

	// testing
	assert.EqualError(t, sys.ErrOnly(testExecute(cmdSync(), "--plain", "-o", musicDir, "-l")), "ko")
}

func TestCmdSyncPlaylistFailure(t *testing.T) {
	musicDir, _ := syncEnv(t, map[string]func(*http.Request) (int, string){
		"/v1/playlists/": func(*http.Request) (int, string) { return apiError(500, "ko") },
	}, webScript{qobuzEmpty: true})

	// testing
	assert.EqualError(t, sys.ErrOnly(testExecute(cmdSync(), "--plain", "-o", musicDir, "-p", "Mix")), "ko")
}

func TestCmdSyncAlbumFailure(t *testing.T) {
	musicDir, _ := syncEnv(t, map[string]func(*http.Request) (int, string){
		"/v1/albums/": func(*http.Request) (int, string) { return apiError(500, "ko") },
	}, webScript{qobuzEmpty: true})

	// testing
	assert.EqualError(t, sys.ErrOnly(testExecute(cmdSync(), "--plain", "-o", musicDir, "-a", "album123")), "ko")
}

func TestCmdSyncTrackFailure(t *testing.T) {
	musicDir, _ := syncEnv(t, map[string]func(*http.Request) (int, string){
		"/v1/tracks/": func(*http.Request) (int, string) { return apiError(500, "ko") },
	}, webScript{qobuzEmpty: true})

	// testing
	assert.EqualError(t, sys.ErrOnly(testExecute(cmdSync(), "--plain", "-o", musicDir, "-t", "123")), "ko")
}

func TestCmdSyncFix(t *testing.T) {
	musicDir, _ := syncEnv(t, nil, webScript{qobuzEmpty: true})
	stale := filepath.Join(musicDir, "old name.mp3")
	writeTaggedFile(t, stale, "123")

	// testing: the stale file is dropped and the track is re-downloaded
	// to its canonical filename
	assert.Nil(t, sys.ErrOnly(testExecute(cmdSync(), "--plain", "-o", musicDir, "-f", stale)))
	_, err := os.Stat(stale)
	assert.True(t, os.IsNotExist(err))
	_, err = os.Stat(filepath.Join(musicDir, "Artist - Title.mp3"))
	assert.Nil(t, err)
}

func TestCmdSyncFixOpenFailure(t *testing.T) {
	musicDir, _ := syncEnv(t, nil, webScript{qobuzEmpty: true})

	// testing: the file to fix does not exist
	missing := filepath.Join(musicDir, "missing.mp3")
	assert.NotNil(t, sys.ErrOnly(testExecute(cmdSync(), "--plain", "-o", musicDir, "-f", missing)))
}

func TestCmdSyncFixSpotifyIDFailure(t *testing.T) {
	musicDir, _ := syncEnv(t, nil, webScript{qobuzEmpty: true})
	untagged := filepath.Join(musicDir, "untagged.mp3")
	writeTaggedFile(t, untagged, "")

	// testing: the file carries no Spotify ID metadata
	err := sys.ErrOnly(testExecute(cmdSync(), "--plain", "-o", musicDir, "-f", untagged))
	assert.ErrorContains(t, err, "does not have spotify ID metadata set")
}

func TestCmdSyncDecideManual(t *testing.T) {
	musicDir, _ := syncEnv(t, nil, webScript{qobuzEmpty: true})
	feedStdin(t, "https://www.youtube.com/watch?v=abc123\n")

	// testing: the user-issued URL drives the download
	assert.Nil(t, sys.ErrOnly(testExecute(cmdSync(), "--plain", "-o", musicDir, "--manual", "-l")))
	_, err := os.Stat(filepath.Join(musicDir, "Artist - Title.mp3"))
	assert.Nil(t, err)
}

func TestCmdSyncDecideManualEmpty(t *testing.T) {
	musicDir, _ := syncEnv(t, nil, webScript{qobuzEmpty: true})
	feedStdin(t, "\n")

	// testing: an empty URL skips the track without failing the sync
	assert.Nil(t, sys.ErrOnly(testExecute(cmdSync(), "--plain", "-o", musicDir, "--manual", "-l")))
	_, err := os.Stat(filepath.Join(musicDir, "Artist - Title.mp3"))
	assert.True(t, os.IsNotExist(err))
}

func TestCmdSyncDecideManualCollision(t *testing.T) {
	musicDir, _ := syncEnv(t, nil, webScript{qobuzEmpty: true})
	writeTaggedFile(t, filepath.Join(musicDir, "Artist - Title.mp3"), "other-id")
	feedStdin(t, "\n")

	// testing: the collision is fatal in manual mode too
	err := sys.ErrOnly(testExecute(cmdSync(), "--plain", "-o", musicDir, "--manual", "-l"))
	assert.ErrorContains(t, err, "filename collision")
}

func TestCmdSyncDecideDuplicateInstalled(t *testing.T) {
	musicDir, _ := syncEnv(t, map[string]func(*http.Request) (int, string){
		"/v1/me/tracks": func(*http.Request) (int, string) { return 200, libraryBody("123", "123") },
	}, webScript{qobuzEmpty: true})

	// testing: the second copy of the track is skipped as a duplicate
	assert.Nil(t, sys.ErrOnly(testExecute(cmdSync(), "--plain", "-o", musicDir, "-l")))
	_, err := os.Stat(filepath.Join(musicDir, "Artist - Title.mp3"))
	assert.Nil(t, err)
}

func TestCmdSyncFatalCollisionUnwinds(t *testing.T) {
	musicDir, _ := syncEnv(t, nil, webScript{qobuzEmpty: true})
	writeTaggedFile(t, filepath.Join(musicDir, "Artist - Title.mp3"), "other-id")

	// testing: the collision error aborts the pipeline, which unwinds
	// instead of deadlocking on the queued routines
	err := sys.ErrOnly(testExecute(cmdSync(), "--plain", "-o", musicDir, "-l"))
	assert.ErrorContains(t, err, "filename collision")
}

func TestCmdSyncDecideFilenameCollision(t *testing.T) {
	musicDir, _ := syncEnv(t, nil, webScript{qobuzEmpty: true})
	writeTaggedFile(t, filepath.Join(musicDir, "Artist - Title.mp3"), "other-id")

	// testing
	err := sys.ErrOnly(testExecute(cmdSync(), "--plain", "-o", musicDir, "-l"))
	assert.ErrorContains(t, err, "filename collision")
}

func TestCmdSyncIgnoreCollisionsCompletes(t *testing.T) {
	musicDir, _ := syncEnv(t, nil, webScript{qobuzEmpty: true})
	existing := filepath.Join(musicDir, "Artist - Title.mp3")
	writeTaggedFile(t, existing, "other-id")
	before, err := os.ReadFile(existing)
	assert.Nil(t, err)

	// testing: the colliding track is skipped and counted, the other
	// track completes, and the existing file is left untouched
	assert.Nil(t, sys.ErrOnly(testExecute(cmdSync(),
		"--plain", "-o", musicDir, "-l", "-t", "999", "--ignore-collisions")))
	assert.Equal(t, int64(1), collisions.Load())
	_, err = os.Stat(filepath.Join(musicDir, "Artist - Title 999.mp3"))
	assert.Nil(t, err)
	after, err := os.ReadFile(existing)
	assert.Nil(t, err)
	assert.Equal(t, before, after)
}

func TestDecideWorkerContextCancel(t *testing.T) {
	resetState(t)
	initRoutines()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	done := make(chan struct{})
	go func() {
		defer close(done)
		var failures atomic.Int32
		decideWorker(ctx, make(chan error, 1), &failures)
	}()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("decide worker did not stop on context cancellation")
	}
}

func TestCollectWorkerContextCancel(t *testing.T) {
	resetState(t)
	initRoutines()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	done := make(chan struct{})
	go func() {
		defer close(done)
		collectWorker(ctx, false)
	}()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("collect worker did not stop on context cancellation")
	}
}

func TestProcessWorkerContextCancel(t *testing.T) {
	resetState(t)
	initRoutines()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	done := make(chan struct{})
	go func() {
		defer close(done)
		processWorker(ctx, make(chan error, 1))
	}()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("process worker did not stop on context cancellation")
	}
}

func TestRoutineProcessFanout(t *testing.T) {
	resetState(t)
	initRoutines()
	fakeBinDir(t)
	clearTrackCache(t, "123", "456")

	// two tracks with downloaded blobs and artwork data are processed
	// in parallel and both handed to the installer
	for _, id := range []string{"123", "456"} {
		track := &entity.Track{
			ID: id, Title: "Title " + id, Artists: []string{"Artist"},
			Artwork: entity.Artwork{URL: "http://ima.ge", Data: jpegBytes(t)},
		}
		if id == "123" {
			track.Title = "Title"
		}
		assert.Nil(t, os.MkdirAll(filepath.Dir(track.Path().Download()), 0o755))
		assert.Nil(t, os.WriteFile(track.Path().Download(), []byte{}, 0o644))
		routineQueues[routineTypeProcess] <- track
	}
	close(routineQueues[routineTypeProcess])

	ch := make(chan error, 1)
	routineProcess(context.Background(), ch)
	assert.Empty(t, ch)
	assert.Equal(t, 2, len(routineQueues[routineTypeInstall]))
}

func TestRoutineProcessFailure(t *testing.T) {
	resetState(t)
	initRoutines()
	fakeBinDir(t)
	clearTrackCache(t, "123")

	// the downloaded blob carries a truncated ID3 header: the encoder
	// cannot even open it as a tag
	track := &entity.Track{ID: "123", Title: "Title", Artists: []string{"Artist"}}
	assert.Nil(t, os.MkdirAll(filepath.Dir(track.Path().Download()), 0o755))
	assert.Nil(t, os.WriteFile(track.Path().Download(), []byte("ID3"), 0o644))
	routineQueues[routineTypeProcess] <- track
	close(routineQueues[routineTypeProcess])

	// testing: the failure is reported on the routine channel
	ch := make(chan error, 1)
	go processWorker(context.Background(), ch)
	select {
	case err := <-ch:
		assert.NotNil(t, err)
	case <-time.After(10 * time.Second):
		t.Fatal("process worker did not report the failure")
	}
}

func TestCmdSyncProcessorFailure(t *testing.T) {
	musicDir, binDir := syncEnv(t, nil, webScript{qobuzEmpty: true})
	// the track measures far from the loudness target and the
	// normalization encode fails: a hard processing failure
	assert.Nil(t, os.WriteFile(filepath.Join(binDir, "ffmpeg.far"), nil, 0o644))
	assert.Nil(t, os.WriteFile(filepath.Join(binDir, "ffmpeg.normfail"), nil, 0o644))

	// testing: a processing failure aborts the sync
	assert.NotNil(t, sys.ErrOnly(testExecute(cmdSync(), "--plain", "-o", musicDir, "-l")))
}

func TestRoutineProcessLoudnessSkipped(t *testing.T) {
	musicDir, binDir := syncEnv(t, nil, webScript{qobuzEmpty: true})
	// ffmpeg answers without the loudnorm JSON summary: the loudness
	// measurement is skipped and the sync still completes
	assert.Nil(t, os.WriteFile(filepath.Join(binDir, "ffmpeg.badjson"), nil, 0o644))

	// testing
	assert.Nil(t, sys.ErrOnly(testExecute(cmdSync(), "--plain", "-o", musicDir, "-l")))
	_, err := os.Stat(filepath.Join(musicDir, "Artist - Title.mp3"))
	assert.Nil(t, err)
}

func TestRoutineCollectFanout(t *testing.T) {
	resetState(t)
	initRoutines()
	fakeBinDir(t)
	installWeb(t, nil, webScript{})
	clearTrackCache(t, "123")

	track := &entity.Track{
		ID: "123", Title: "Title", Artists: []string{"Artist"},
		UpstreamURL: "https://www.youtube.com/watch?v=abc123",
		Artwork:     entity.Artwork{URL: "http://ima.ge"},
	}

	// testing: asset, lyrics and artwork are collected and the track is
	// handed to the processor
	collectTrack(false, track)
	assert.Equal(t, "some lyrics", track.Lyrics)
	assert.NotEmpty(t, track.Artwork.Data)
	assert.Equal(t, 1, len(routineQueues[routineTypeProcess]))
}

func TestRoutineCollectArtworkEmptyURL(t *testing.T) {
	resetState(t)
	initRoutines()
	fakeBinDir(t)
	installWeb(t, nil, webScript{})
	clearTrackCache(t, "123")

	track := &entity.Track{
		ID: "123", Title: "Title", Artists: []string{"Artist"},
		UpstreamURL: "https://www.youtube.com/watch?v=abc123",
	}

	// testing: a track without an artwork URL skips the painter without
	// deadlocking the collection
	collectTrack(false, track)
	assert.Empty(t, track.Artwork.Data)
	assert.Equal(t, 1, len(routineQueues[routineTypeProcess]))
}

func TestCmdSyncDecideFailure(t *testing.T) {
	musicDir, _ := syncEnv(t, nil, webScript{providerErr: true})

	// testing: a failing search drops the track, not the sync
	assert.Nil(t, sys.ErrOnly(testExecute(cmdSync(), "--plain", "-o", musicDir, "-l")))
	_, err := os.Stat(filepath.Join(musicDir, "Artist - Title.mp3"))
	assert.True(t, os.IsNotExist(err))
}

func TestCmdSyncDecideCircuitBreaker(t *testing.T) {
	musicDir, _ := syncEnv(t, map[string]func(*http.Request) (int, string){
		"/v1/me/tracks": func(*http.Request) (int, string) { return 200, libraryBody("123", "456", "789", "999") },
	}, webScript{providerErr: true})

	// testing: repeated search failures trip the circuit breaker and
	// the remaining tracks are skipped, without failing the sync
	assert.Nil(t, sys.ErrOnly(testExecute(cmdSync(), "--plain", "-o", musicDir, "-l")))
	entries, err := os.ReadDir(musicDir)
	assert.Nil(t, err)
	assert.Empty(t, entries)
}

func TestCmdSyncDecideNotFound(t *testing.T) {
	musicDir, _ := syncEnv(t, nil, webScript{noResults: true})

	// testing
	assert.Nil(t, sys.ErrOnly(testExecute(cmdSync(), "--plain", "-o", musicDir, "-l")))
	_, err := os.Stat(filepath.Join(musicDir, "Artist - Title.mp3"))
	assert.True(t, os.IsNotExist(err))
}

func TestCmdSyncDecideParallelFanout(t *testing.T) {
	musicDir, _ := syncEnv(t, map[string]func(*http.Request) (int, string){
		"/v1/me/tracks": func(*http.Request) (int, string) { return 200, libraryBody("123", "456", "789") },
	}, webScript{qobuzEmpty: true})

	// testing: parallel deciders install every fetched track
	assert.Nil(t, sys.ErrOnly(testExecute(cmdSync(), "--plain", "-o", musicDir, "-l")))
	for _, name := range []string{"Artist - Title.mp3", "Artist - Title 456.mp3", "Artist - Title 789.mp3"} {
		_, err := os.Stat(filepath.Join(musicDir, name))
		assert.Nil(t, err)
	}
}

func TestCmdSyncCollectFailure(t *testing.T) {
	musicDir, _ := syncEnv(t, nil, webScript{qobuzEmpty: true, artworkErr: true})

	// testing: an artwork download failure drops the track during
	// collection, without failing the sync
	assert.Nil(t, sys.ErrOnly(testExecute(cmdSync(), "--plain", "-o", musicDir, "-l")))
	_, err := os.Stat(filepath.Join(musicDir, "Artist - Title.mp3"))
	assert.True(t, os.IsNotExist(err))
}

func TestCmdSyncDownloadFailure(t *testing.T) {
	musicDir, binDir := syncEnv(t, nil, webScript{qobuzEmpty: true})
	assert.Nil(t, os.WriteFile(filepath.Join(binDir, "ytdlp.fail"), nil, 0o644))

	// testing: a failing download drops the track, not the sync
	assert.Nil(t, sys.ErrOnly(testExecute(cmdSync(), "--plain", "-o", musicDir, "-l")))
	_, err := os.Stat(filepath.Join(musicDir, "Artist - Title.mp3"))
	assert.True(t, os.IsNotExist(err))
}

func TestCmdSyncLyricsFailure(t *testing.T) {
	musicDir, _ := syncEnv(t, nil, webScript{qobuzEmpty: true, lyricsErr: true})

	// testing: lyrics are a nice-to-have, the track still installs
	assert.Nil(t, sys.ErrOnly(testExecute(cmdSync(), "--plain", "-o", musicDir, "-l")))
	_, err := os.Stat(filepath.Join(musicDir, "Artist - Title.mp3"))
	assert.Nil(t, err)
}

func TestCmdSyncInstallerFailure(t *testing.T) {
	resetState(t)
	initRoutines()
	clearTrackCache(t, "123")

	// the downloaded blob does not exist, so the move to the final
	// destination fails
	track := &entity.Track{ID: "123", Title: "Title", Artists: []string{"Artist"}}
	routineQueues[routineTypeInstall] <- track
	close(routineQueues[routineTypeInstall])

	// testing: the failure is reported on the routine channel
	ch := make(chan error, 1)
	routineInstall(context.Background(), ch)
	select {
	case err := <-ch:
		assert.NotNil(t, err)
	case <-time.After(10 * time.Second):
		t.Fatal("install routine did not report the failure")
	}
}

func TestCmdSyncPlaylistEncoderFailure(t *testing.T) {
	musicDir, _ := syncEnv(t, nil, webScript{qobuzEmpty: true})

	// testing: an unknown playlist encoding skips the mix without
	// failing the sync
	assert.Nil(t, sys.ErrOnly(testExecute(cmdSync(),
		"--plain", "-o", musicDir, "-p", "Mix", "--playlist-encoding", "bogus")))
	_, err := os.Stat(filepath.Join(musicDir, "Artist - Title.mp3"))
	assert.Nil(t, err)
	entries, err := os.ReadDir(musicDir)
	assert.Nil(t, err)
	for _, entry := range entries {
		assert.NotEqual(t, "Mix.m3u", entry.Name())
		assert.NotEqual(t, "Mix.pls", entry.Name())
	}
}

func TestCmdSyncPlaylistEncoderCloseFailure(t *testing.T) {
	musicDir, _ := syncEnv(t, nil, webScript{qobuzEmpty: true})

	// the playlist target path is occupied by a directory, so the
	// encoder cannot write the playlist file on close
	assert.Nil(t, os.MkdirAll(filepath.Join(musicDir, "Mix.m3u"), 0o755))

	// testing: the failure is reported and skipped, the sync completes
	assert.Nil(t, sys.ErrOnly(testExecute(cmdSync(), "--plain", "-o", musicDir, "-p", "Mix")))
	_, err := os.Stat(filepath.Join(musicDir, "Artist - Title.mp3"))
	assert.Nil(t, err)
}
