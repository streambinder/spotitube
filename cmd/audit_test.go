package cmd

import (
	"errors"
	"io"
	"os"
	"testing"

	"github.com/bytedance/mockey"
	"github.com/streambinder/spotitube/entity"
	"github.com/streambinder/spotitube/entity/id3"
	"github.com/streambinder/spotitube/entity/index"
	"github.com/streambinder/spotitube/entity/playlist"
	"github.com/streambinder/spotitube/spotify"
	"github.com/streambinder/spotitube/sys"
	"github.com/stretchr/testify/assert"
)

func TestCmdAudit(t *testing.T) {
	t.Cleanup(cleanup)

	_track := &entity.Track{ID: "TestCmdAudit", Title: "Title", Artists: []string{"Artist"}}
	indexData.Set(_track, index.Installed)

	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(mockey.GetMethod(&index.Index{}, "BuildWithProgress")).To(func(_ *index.Index, _ string, indexed chan<- string, _ ...int) error {
		if indexed != nil {
			indexed <- "Artist - Title.mp3"
		}
		return nil
	}).Build()
	mockey.Mock(spotify.Authenticate).Return(&spotify.Client{}, nil).Build()
	mockey.Mock(mockey.GetMethod(&spotify.Client{}, "Username")).Return("streambinder", nil).Build()
	mockey.Mock(mockey.GetMethod(&spotify.Client{}, "Library")).To(func(_ int, ch ...chan interface{}) error {
		for _, c := range ch {
			c <- cloneTrack(_track)
		}
		return nil
	}).Build()
	mockey.Mock(mockey.GetMethod(&spotify.Client{}, "Album")).To(func(_ string, ch ...chan interface{}) (*entity.Album, error) {
		for _, c := range ch {
			c <- cloneTrack(_track)
		}
		return &entity.Album{}, nil
	}).Build()
	mockey.Mock(mockey.GetMethod(&spotify.Client{}, "Track")).To(func(_ string, ch ...chan interface{}) (*entity.Track, error) {
		for _, c := range ch {
			c <- cloneTrack(_track)
		}
		return _track, nil
	}).Build()
	mockey.Mock(mockey.GetMethod(&spotify.Client{}, "Playlist")).To(func(_ string, ch ...chan interface{}) (*playlist.Playlist, error) {
		for _, c := range ch {
			c <- cloneTrack(_track)
		}
		return &playlist.Playlist{}, nil
	}).Build()

	// testing: no collision, every source selector exercised
	assert.Nil(t, sys.ErrOnly(testExecute(cmdAudit(), "-o", t.TempDir(), "-l", "-p", "123", "-a", "123", "-t", "123")))

	// testing: library auto-enabled when no collection is supplied
	cmd := cmdAudit()
	assert.Nil(t, sys.ErrOnly(testExecute(cmd, "-o", t.TempDir())))
	library, err := cmd.Flags().GetBool("library")
	assert.Nil(t, err)
	assert.True(t, library)
}

func TestCmdAuditPathFailure(t *testing.T) {
	t.Cleanup(cleanup)

	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(os.Chdir).Return(errors.New("ko")).Build()

	// testing
	assert.EqualError(t, sys.ErrOnly(testExecute(cmdAudit())), "ko")
}

func TestCmdAuditIndexFailure(t *testing.T) {
	t.Cleanup(cleanup)

	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(spotify.Authenticate).Return(&spotify.Client{}, nil).Build()
	mockey.Mock(mockey.GetMethod(&index.Index{}, "BuildWithProgress")).Return(errors.New("ko")).Build()

	// testing
	assert.EqualError(t, sys.ErrOnly(testExecute(cmdAudit(), "-o", t.TempDir(), "-l")), "ko")
}

func TestCmdAuditAuthFailure(t *testing.T) {
	t.Cleanup(cleanup)

	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(mockey.GetMethod(&index.Index{}, "BuildWithProgress")).Return(nil).Build()
	mockey.Mock(spotify.Authenticate).Return(&spotify.Client{}, errors.New("ko")).Build()

	// testing
	assert.EqualError(t, sys.ErrOnly(testExecute(cmdAudit(), "-o", t.TempDir(), "-l")), "ko")
}

func TestCmdAuditUsernameFailure(t *testing.T) {
	t.Cleanup(cleanup)

	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(mockey.GetMethod(&index.Index{}, "BuildWithProgress")).Return(nil).Build()
	mockey.Mock(spotify.Authenticate).Return(&spotify.Client{}, nil).Build()
	mockey.Mock(mockey.GetMethod(&spotify.Client{}, "Username")).Return("", errors.New("ko")).Build()
	mockey.Mock(mockey.GetMethod(&spotify.Client{}, "Library")).Return(nil).Build()

	// testing: an unresolvable username is reported but does not abort the audit
	assert.Nil(t, sys.ErrOnly(testExecute(cmdAudit(), "-o", t.TempDir(), "-l")))
}

func TestCmdAuditLibraryFailure(t *testing.T) {
	t.Cleanup(cleanup)

	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(mockey.GetMethod(&index.Index{}, "BuildWithProgress")).Return(nil).Build()
	mockey.Mock(spotify.Authenticate).Return(&spotify.Client{}, nil).Build()
	mockey.Mock(mockey.GetMethod(&spotify.Client{}, "Library")).Return(errors.New("ko")).Build()

	// testing
	assert.EqualError(t, sys.ErrOnly(testExecute(cmdAudit(), "-o", t.TempDir(), "-l")), "library: ko")
}

func TestCmdAuditAlbumFailure(t *testing.T) {
	t.Cleanup(cleanup)

	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(mockey.GetMethod(&index.Index{}, "BuildWithProgress")).Return(nil).Build()
	mockey.Mock(spotify.Authenticate).Return(&spotify.Client{}, nil).Build()
	mockey.Mock(mockey.GetMethod(&spotify.Client{}, "Album")).Return(&entity.Album{}, errors.New("ko")).Build()

	// testing
	assert.EqualError(t, sys.ErrOnly(testExecute(cmdAudit(), "-o", t.TempDir(), "-a", "123")), "album 123: ko")
}

func TestCmdAuditTrackFailure(t *testing.T) {
	t.Cleanup(cleanup)

	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(mockey.GetMethod(&index.Index{}, "BuildWithProgress")).Return(nil).Build()
	mockey.Mock(spotify.Authenticate).Return(&spotify.Client{}, nil).Build()
	mockey.Mock(mockey.GetMethod(&spotify.Client{}, "Track")).Return(&entity.Track{}, errors.New("ko")).Build()

	// testing
	assert.EqualError(t, sys.ErrOnly(testExecute(cmdAudit(), "-o", t.TempDir(), "-t", "123")), "track 123: ko")
}

func TestCmdAuditPlaylistFailure(t *testing.T) {
	t.Cleanup(cleanup)

	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(mockey.GetMethod(&index.Index{}, "BuildWithProgress")).Return(nil).Build()
	mockey.Mock(spotify.Authenticate).Return(&spotify.Client{}, nil).Build()
	mockey.Mock(mockey.GetMethod(&spotify.Client{}, "Playlist")).Return(&playlist.Playlist{}, errors.New("ko")).Build()

	// testing
	assert.EqualError(t, sys.ErrOnly(testExecute(cmdAudit(), "-o", t.TempDir(), "-p", "123")), "playlist 123: ko")
}

func TestCmdAuditFetchFailureNoCleanReport(t *testing.T) {
	t.Cleanup(cleanup)

	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(mockey.GetMethod(&index.Index{}, "BuildWithProgress")).Return(nil).Build()
	mockey.Mock(spotify.Authenticate).Return(&spotify.Client{}, nil).Build()
	mockey.Mock(mockey.GetMethod(&spotify.Client{}, "Library")).Return(errors.New("ko")).Build()

	// capture stdout to inspect the printed report
	stdout := os.Stdout
	reader, writer, err := os.Pipe()
	assert.Nil(t, err)
	os.Stdout = writer
	defer func() { os.Stdout = stdout }()

	// testing: the fetch error surfaces with its source, and no clean-audit
	// report is printed on the incomplete data
	assert.EqualError(t, sys.ErrOnly(testExecute(cmdAudit(), "-o", t.TempDir(), "-l")), "library: ko")
	assert.Nil(t, writer.Close())
	output, err := io.ReadAll(reader)
	assert.Nil(t, err)
	assert.NotContains(t, string(output), "no filename collisions found")
}

func TestCmdAuditCollisionLibrary(t *testing.T) {
	t.Cleanup(cleanup)

	_existing := &entity.Track{ID: "TestCmdAuditCollisionLibraryExisting", Title: "Title", Artists: []string{"Artist"}}
	_colliding := &entity.Track{ID: "TestCmdAuditCollisionLibraryColliding", Title: "Title", Artists: []string{"Artist"}}

	// the filename is already owned by a library file
	indexData.Set(_existing, index.Offline)

	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(mockey.GetMethod(&index.Index{}, "BuildWithProgress")).Return(nil).Build()
	mockey.Mock(spotify.Authenticate).Return(&spotify.Client{}, nil).Build()
	mockey.Mock(mockey.GetMethod(&spotify.Client{}, "Library")).To(func(_ int, ch ...chan interface{}) error {
		for _, c := range ch {
			c <- cloneTrack(_colliding)
		}
		return nil
	}).Build()
	mockey.Mock(id3.Open).Return(&id3.Tag{}, nil).Build()
	mockey.Mock(mockey.GetMethod(&id3.Tag{}, "SpotifyID")).Return(_existing.ID).Build()
	mockey.Mock(mockey.GetMethod(&id3.Tag{}, "Title")).Return(_existing.Title).Build()
	mockey.Mock(mockey.GetMethod(&id3.Tag{}, "Artist")).Return("Artist").Build()
	mockey.Mock(mockey.GetMethod(&id3.Tag{}, "Close")).Return(nil).Build()

	// testing: the audit reports the collision with both sides and their sources
	err := sys.ErrOnly(testExecute(cmdAudit(), "-o", t.TempDir(), "-l"))
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "audit failed: 1 filename collisions found")
}

func TestCmdAuditCollisionTagFailure(t *testing.T) {
	t.Cleanup(cleanup)

	_existing := &entity.Track{ID: "TestCmdAuditCollisionTagFailureExisting", Title: "Title", Artists: []string{"Artist"}}
	_colliding := &entity.Track{ID: "TestCmdAuditCollisionTagFailureColliding", Title: "Title", Artists: []string{"Artist"}}

	// the filename is already owned by a library file
	indexData.Set(_existing, index.Offline)

	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(mockey.GetMethod(&index.Index{}, "BuildWithProgress")).Return(nil).Build()
	mockey.Mock(spotify.Authenticate).Return(&spotify.Client{}, nil).Build()
	mockey.Mock(mockey.GetMethod(&spotify.Client{}, "Library")).To(func(_ int, ch ...chan interface{}) error {
		for _, c := range ch {
			c <- cloneTrack(_colliding)
		}
		return nil
	}).Build()
	mockey.Mock(id3.Open).Return(&id3.Tag{}, errors.New("ko")).Build()

	// testing: an unreadable tag still reports the collision with a fallback entry
	err := sys.ErrOnly(testExecute(cmdAudit(), "-o", t.TempDir(), "-l"))
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "audit failed: 1 filename collisions found")
}

func TestCmdAuditCollisionFetched(t *testing.T) {
	t.Cleanup(cleanup)

	_first := &entity.Track{ID: "TestCmdAuditCollisionFetchedFirst", Title: "Title", Artists: []string{"Artist"}}
	_second := &entity.Track{ID: "TestCmdAuditCollisionFetchedSecond", Title: "Title", Artists: []string{"Artist"}}

	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(mockey.GetMethod(&index.Index{}, "BuildWithProgress")).Return(nil).Build()
	mockey.Mock(spotify.Authenticate).Return(&spotify.Client{}, nil).Build()
	mockey.Mock(mockey.GetMethod(&spotify.Client{}, "Album")).To(func(_ string, ch ...chan interface{}) (*entity.Album, error) {
		for _, c := range ch {
			c <- cloneTrack(_first)
		}
		return &entity.Album{}, nil
	}).Build()
	mockey.Mock(mockey.GetMethod(&spotify.Client{}, "Playlist")).To(func(_ string, ch ...chan interface{}) (*playlist.Playlist, error) {
		for _, c := range ch {
			c <- cloneTrack(_second)
		}
		return &playlist.Playlist{}, nil
	}).Build()

	// testing: the second same-filename track collides with the first fetched one,
	// attributing each side to its own source selector
	err := sys.ErrOnly(testExecute(cmdAudit(), "-o", t.TempDir(), "-a", "somealbum", "-p", "someplaylist"))
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "audit failed: 1 filename collisions found")
}

func TestCmdAuditCollisionMultiple(t *testing.T) {
	t.Cleanup(cleanup)

	// pushed out of order on purpose: the report must sort them by filename
	_b1 := &entity.Track{ID: "TestCmdAuditCollisionMultipleB1", Title: "TitleB", Artists: []string{"Artist"}}
	_b2 := &entity.Track{ID: "TestCmdAuditCollisionMultipleB2", Title: "TitleB", Artists: []string{"Artist"}}
	_a1 := &entity.Track{ID: "TestCmdAuditCollisionMultipleA1", Title: "TitleA", Artists: []string{"Artist"}}
	_a2 := &entity.Track{ID: "TestCmdAuditCollisionMultipleA2", Title: "TitleA", Artists: []string{"Artist"}}

	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(mockey.GetMethod(&index.Index{}, "BuildWithProgress")).Return(nil).Build()
	mockey.Mock(spotify.Authenticate).Return(&spotify.Client{}, nil).Build()
	mockey.Mock(mockey.GetMethod(&spotify.Client{}, "Library")).To(func(_ int, ch ...chan interface{}) error {
		for _, c := range ch {
			for _, track := range []*entity.Track{_b1, _b2, _a1, _a2} {
				c <- cloneTrack(track)
			}
		}
		return nil
	}).Build()

	// testing: every collision is reported, sorted by filename
	err := sys.ErrOnly(testExecute(cmdAudit(), "-o", t.TempDir(), "-l"))
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "audit failed: 2 filename collisions found")
}

func TestCmdAuditClassify(t *testing.T) {
	t.Cleanup(cleanup)

	// track without ID and unknown path: claims the filename
	_noID := &entity.Track{Title: "NoID", Artists: []string{"Artist"}}
	collides, claims := auditClassify(_noID)
	assert.False(t, collides)
	assert.True(t, claims)

	// already synced track: skipped
	_installed := &entity.Track{ID: "TestCmdAuditClassifyInstalled", Title: "Installed", Artists: []string{"Artist"}}
	indexData.Set(_installed, index.Installed)
	collides, claims = auditClassify(_installed)
	assert.False(t, collides)
	assert.False(t, claims)

	// flushed track: would be reinstalled, claims the filename
	_flushed := &entity.Track{ID: "TestCmdAuditClassifyFlushed", Title: "Flushed", Artists: []string{"Artist"}}
	indexData.SetID(_flushed.ID, index.Flush)
	collides, claims = auditClassify(_flushed)
	assert.False(t, collides)
	assert.True(t, claims)

	// unknown ID on a known path: collision
	_existing := &entity.Track{ID: "TestCmdAuditClassifyExisting", Title: "Existing", Artists: []string{"Artist"}}
	_colliding := &entity.Track{ID: "TestCmdAuditClassifyColliding", Title: "Existing", Artists: []string{"Artist"}}
	indexData.Set(_existing, index.Offline)
	collides, claims = auditClassify(_colliding)
	assert.True(t, collides)
	assert.False(t, claims)
}

func TestCmdAuditPlaylistTracksSource(t *testing.T) {
	t.Cleanup(cleanup)

	_first := &entity.Track{ID: "TestCmdAuditPlaylistTracksSourceFirst", Title: "Title", Artists: []string{"Artist"}}
	_second := &entity.Track{ID: "TestCmdAuditPlaylistTracksSourceSecond", Title: "Title", Artists: []string{"Artist"}}

	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(mockey.GetMethod(&index.Index{}, "BuildWithProgress")).Return(nil).Build()
	mockey.Mock(spotify.Authenticate).Return(&spotify.Client{}, nil).Build()
	mockey.Mock(mockey.GetMethod(&spotify.Client{}, "Playlist")).To(func(_ string, ch ...chan interface{}) (*playlist.Playlist, error) {
		for _, c := range ch {
			c <- cloneTrack(_first)
			c <- cloneTrack(_second)
		}
		return &playlist.Playlist{}, nil
	}).Build()

	// capture stdout to inspect the printed report
	stdout := os.Stdout
	reader, writer, err := os.Pipe()
	assert.Nil(t, err)
	os.Stdout = writer
	defer func() { os.Stdout = stdout }()

	// testing: the playlist-tracks selector is attributed distinctly from playlist
	err = sys.ErrOnly(testExecute(cmdAudit(), "-o", t.TempDir(), "--playlist-tracks", "sometracks"))
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "audit failed: 1 filename collisions found")
	assert.Nil(t, writer.Close())
	output, readErr := io.ReadAll(reader)
	assert.Nil(t, readErr)
	assert.Contains(t, string(output), "from playlist-tracks sometracks")
}

func TestCmdAuditPlaylistTracksFailure(t *testing.T) {
	t.Cleanup(cleanup)

	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(mockey.GetMethod(&index.Index{}, "BuildWithProgress")).Return(nil).Build()
	mockey.Mock(spotify.Authenticate).Return(&spotify.Client{}, nil).Build()
	mockey.Mock(mockey.GetMethod(&spotify.Client{}, "Playlist")).Return(&playlist.Playlist{}, errors.New("ko")).Build()

	// testing
	assert.EqualError(t, sys.ErrOnly(testExecute(cmdAudit(), "-o", t.TempDir(), "--playlist-tracks", "123")), "playlist-tracks 123: ko")
}
