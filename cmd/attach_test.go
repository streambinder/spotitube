package cmd

import (
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/bogem/id3v2/v2"
	"github.com/streambinder/spotitube/entity/id3"
	"github.com/streambinder/spotitube/sys"
	"github.com/stretchr/testify/assert"
)

// writeAttachTrack creates the empty audio file an attach run writes to.
func writeAttachTrack(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "track.mp3")
	assert.Nil(t, os.WriteFile(path, []byte{}, 0o644))
	return path
}

func BenchmarkAttach(b *testing.B) {
	for i := 0; i < b.N; i++ {
		TestCmdAttach(&testing.T{})
	}
}

func TestCmdAttach(t *testing.T) {
	path := writeAttachTrack(t)
	seedSession(t)
	installWeb(t, nil, webScript{})
	clearTrackCache(t, "123")

	// testing
	assert.Nil(t, sys.ErrOnly(testExecute(cmdAttach(), path, "123")))

	// the track metadata landed in the file tag
	tag, err := id3.Open(path, id3v2.Options{Parse: true})
	assert.Nil(t, err)
	defer func() { _ = tag.Close() }()
	assert.Equal(t, "123", tag.SpotifyID())
	assert.Equal(t, "Title", tag.Title())
}

func TestCmdAttachOpenFailure(t *testing.T) {
	// testing: a missing file cannot be opened
	assert.NotNil(t, sys.ErrOnly(testExecute(cmdAttach(),
		filepath.Join(t.TempDir(), "missing.mp3"), "123")))
}

func TestCmdAttachAuthFailure(t *testing.T) {
	path := writeAttachTrack(t)
	noSession(t)
	installWeb(t, nil, webScript{})
	noOpenerPath(t)
	occupyAuthPort(t)

	// testing
	assert.NotNil(t, sys.ErrOnly(testExecute(cmdAttach(), path, "123")))
}

func TestCmdAttachTrackFailure(t *testing.T) {
	path := writeAttachTrack(t)
	seedSession(t)
	installWeb(t, map[string]func(*http.Request) (int, string){
		"/v1/tracks/": func(*http.Request) (int, string) { return apiError(500, "ko") },
	}, webScript{})
	clearTrackCache(t, "123")

	// testing
	assert.EqualError(t, sys.ErrOnly(testExecute(cmdAttach(), path, "123")), "ko")
}

func TestCmdAttachLyricsFailure(t *testing.T) {
	path := writeAttachTrack(t)
	seedSession(t)
	installWeb(t, nil, webScript{lyricsErr: true})
	clearTrackCache(t, "123")

	// testing
	assert.NotNil(t, sys.ErrOnly(testExecute(cmdAttach(), path, "123")))
}

func TestCmdAttacDownloadFailure(t *testing.T) {
	path := writeAttachTrack(t)
	seedSession(t)
	installWeb(t, nil, webScript{artworkErr: true})
	clearTrackCache(t, "123")

	// testing: the artwork URL answers 404, no downloader supports it
	assert.NotNil(t, sys.ErrOnly(testExecute(cmdAttach(), path, "123")))
}

func TestCmdAttachRename(t *testing.T) {
	path := writeAttachTrack(t)
	seedSession(t)
	installWeb(t, nil, webScript{})
	clearTrackCache(t, "123")

	// testing
	assert.Nil(t, sys.ErrOnly(testExecute(cmdAttach(), "--rename", path, "123")))
	_, err := os.Stat(path)
	assert.True(t, os.IsNotExist(err))
	_, err = os.Stat(filepath.Join(filepath.Dir(path), "Artist - Title.mp3"))
	assert.Nil(t, err)
}

func TestCmdAttachRenameFailure(t *testing.T) {
	path := writeAttachTrack(t)
	seedSession(t)
	installWeb(t, nil, webScript{})
	clearTrackCache(t, "123")

	// the rename destination already exists
	target := filepath.Join(filepath.Dir(path), "Artist - Title.mp3")
	assert.Nil(t, os.WriteFile(target, []byte("existing"), 0o644))

	// testing
	assert.NotNil(t, sys.ErrOnly(testExecute(cmdAttach(), "--rename", path, "123")))
}
