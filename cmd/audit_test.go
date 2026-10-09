package cmd

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bogem/id3v2/v2"
	"github.com/streambinder/spotitube/entity"
	"github.com/streambinder/spotitube/entity/id3"
	"github.com/streambinder/spotitube/entity/index"
	"github.com/streambinder/spotitube/sys"
	"github.com/stretchr/testify/assert"
)

// writeAuditTrack creates a real tagged audio file in the music
// directory, named after the fixture track's final filename.
func writeAuditTrack(t *testing.T, dir, spotifyID string) {
	t.Helper()
	path := filepath.Join(dir, "Artist - Title.mp3")
	assert.Nil(t, os.WriteFile(path, []byte{}, 0o644))

	tag, err := id3.Open(path, id3v2.Options{Parse: true})
	assert.Nil(t, err)
	tag.SetTitle("Title")
	tag.SetArtist("Artist")
	tag.SetSpotifyID(spotifyID)
	assert.Nil(t, tag.Save())
	assert.Nil(t, tag.Close())
}

// libraryBody renders a library page carrying the given track ids.
func libraryBody(ids ...string) string {
	body := `{"items":[`
	for i, id := range ids {
		if i > 0 {
			body += ","
		}
		body += `{"added_at":"2026-01-01T00:00:00Z","track":` + trackJSON(id) + `}`
	}
	return body + `],"next":null,"total":` + string(rune('0'+len(ids))) + `}`
}

// sameTitleLibraryBody renders a library page whose tracks all carry
// the canonical title, so they resolve to the same final filename.
func sameTitleLibraryBody(ids ...string) string {
	body := `{"items":[`
	for i, id := range ids {
		if i > 0 {
			body += ","
		}
		track := strings.Replace(trackJSON("123"), `"id":"123"`, `"id":"`+id+`"`, 1)
		body += `{"added_at":"2026-01-01T00:00:00Z","track":` + track + `}`
	}
	return body + `],"next":null,"total":` + string(rune('0'+len(ids))) + `}`
}

func BenchmarkAudit(b *testing.B) {
	for i := 0; i < b.N; i++ {
		TestCmdAudit(&testing.T{})
	}
}

func TestCmdAudit(t *testing.T) {
	resetState(t)
	seedSession(t)
	installWeb(t, nil, webScript{})

	// testing: an empty library and one fetched track, no collisions
	assert.Nil(t, sys.ErrOnly(testExecute(cmdAudit(), "-o", t.TempDir())))
}

func TestCmdAuditPathFailure(t *testing.T) {
	resetState(t)

	// testing: the output path does not exist, the chdir fails first
	missing := filepath.Join(t.TempDir(), "missing")
	assert.NotNil(t, sys.ErrOnly(testExecute(cmdAudit(), "-o", missing)))
}

func TestCmdAuditAuthFailure(t *testing.T) {
	resetState(t)
	noSession(t)
	installWeb(t, nil, webScript{})
	noOpenerPath(t)
	occupyAuthPort(t)

	// testing
	assert.NotNil(t, sys.ErrOnly(testExecute(cmdAudit(), "-o", t.TempDir())))
}

func TestCmdAuditUsernameFailure(t *testing.T) {
	resetState(t)
	seedSession(t)
	meCalls := 0
	installWeb(t, map[string]func(*http.Request) (int, string){
		"/v1/me": func(request *http.Request) (int, string) {
			if request.URL.Path != "/v1/me" {
				// the library fetch shares the prefix: serve the fixture
				return 200, libraryBody("123")
			}
			meCalls++
			if meCalls == 1 {
				// the session recovery validation succeeds
				return 200, meJSON
			}
			// the audit's own username resolution fails, and is tolerated
			return apiError(500, "ko")
		},
	}, webScript{})

	// testing
	assert.Nil(t, sys.ErrOnly(testExecute(cmdAudit(), "-o", t.TempDir())))
}

func TestCmdAuditLibraryFailure(t *testing.T) {
	resetState(t)
	seedSession(t)
	installWeb(t, map[string]func(*http.Request) (int, string){
		"/v1/me/tracks": func(*http.Request) (int, string) { return apiError(500, "ko") },
	}, webScript{})

	// testing
	assert.EqualError(t, sys.ErrOnly(testExecute(cmdAudit(), "-o", t.TempDir())), "library: ko")
}

func TestCmdAuditAlbumFailure(t *testing.T) {
	resetState(t)
	seedSession(t)
	installWeb(t, map[string]func(*http.Request) (int, string){
		"/v1/albums/": func(*http.Request) (int, string) { return apiError(500, "ko") },
	}, webScript{})

	// testing
	assert.EqualError(t, sys.ErrOnly(testExecute(cmdAudit(), "-o", t.TempDir(), "-a", "album123")), "album album123: ko")
}

func TestCmdAuditTrackFailure(t *testing.T) {
	resetState(t)
	seedSession(t)
	installWeb(t, map[string]func(*http.Request) (int, string){
		"/v1/tracks/": func(*http.Request) (int, string) { return apiError(500, "ko") },
	}, webScript{})

	// testing
	assert.EqualError(t, sys.ErrOnly(testExecute(cmdAudit(), "-o", t.TempDir(), "-t", "123")), "track 123: ko")
}

func TestCmdAuditPlaylistFailure(t *testing.T) {
	resetState(t)
	seedSession(t)
	installWeb(t, map[string]func(*http.Request) (int, string){
		"/v1/playlists/": func(*http.Request) (int, string) { return apiError(500, "ko") },
	}, webScript{})

	// testing
	assert.EqualError(t, sys.ErrOnly(testExecute(cmdAudit(), "-o", t.TempDir(), "-p", "Mix")), "playlist Mix: ko")
}

func TestCmdAuditPlaylistTracksFailure(t *testing.T) {
	resetState(t)
	seedSession(t)
	installWeb(t, map[string]func(*http.Request) (int, string){
		"/v1/playlists/": func(*http.Request) (int, string) { return apiError(500, "ko") },
	}, webScript{})

	// testing
	assert.EqualError(t, sys.ErrOnly(testExecute(cmdAudit(), "-o", t.TempDir(), "--playlist-tracks", "Mix")), "playlist-tracks Mix: ko")
}

func TestCmdAuditPlaylistTracksSource(t *testing.T) {
	resetState(t)
	seedSession(t)
	installWeb(t, nil, webScript{})

	// testing: playlist tracks are fetched without the playlist file
	assert.Nil(t, sys.ErrOnly(testExecute(cmdAudit(), "-o", t.TempDir(), "--playlist-tracks", "Mix")))
}

func TestCmdAuditCollision(t *testing.T) {
	resetState(t)
	seedSession(t)
	installWeb(t, nil, webScript{})

	musicDir := t.TempDir()
	writeAuditTrack(t, musicDir, "other-id")

	// testing: the fetched track resolves to the filename of a library
	// file owned by a different Spotify track
	err := sys.ErrOnly(testExecute(cmdAudit(), "-o", musicDir))
	assert.EqualError(t, err, "audit failed: 1 filename collisions found")
}

func TestCmdAuditCollisionMultiple(t *testing.T) {
	resetState(t)
	seedSession(t)
	installWeb(t, map[string]func(*http.Request) (int, string){
		"/v1/me/tracks": func(*http.Request) (int, string) { return 200, sameTitleLibraryBody("123", "456") },
	}, webScript{})

	musicDir := t.TempDir()
	writeAuditTrack(t, musicDir, "other-id")

	// testing: both fetched tracks collide with the same library file
	err := sys.ErrOnly(testExecute(cmdAudit(), "-o", musicDir))
	assert.EqualError(t, err, "audit failed: 2 filename collisions found")
}

func TestCmdAuditCollisionFetched(t *testing.T) {
	resetState(t)
	seedSession(t)
	installWeb(t, map[string]func(*http.Request) (int, string){
		"/v1/me/tracks": func(*http.Request) (int, string) { return 200, sameTitleLibraryBody("123", "456") },
	}, webScript{})

	// testing: the second fetched track collides with the filename the
	// first one claimed during the same audit
	err := sys.ErrOnly(testExecute(cmdAudit(), "-o", t.TempDir()))
	assert.EqualError(t, err, "audit failed: 1 filename collisions found")
}

func TestAuditClassify(t *testing.T) {
	resetState(t)

	track := &entity.Track{ID: "123", Title: "Title", Artists: []string{"Artist"}}

	// an unknown track with a free filename proceeds and claims it
	collides, claims := auditClassify(track)
	assert.False(t, collides)
	assert.True(t, claims)

	// a second unknown track on the claimed filename collides
	other := &entity.Track{ID: "456", Title: "Title", Artists: []string{"Artist"}}
	collides, claims = auditClassify(other)
	assert.True(t, collides)
	assert.False(t, claims)

	// a known track whose index entry is not pending flush neither
	// collides nor claims
	collides, claims = auditClassify(track)
	assert.False(t, collides)
	assert.False(t, claims)

	// a known track pending flush claims its filename again
	indexData.Set(track, index.Flush)
	collides, claims = auditClassify(track)
	assert.False(t, collides)
	assert.True(t, claims)
}
