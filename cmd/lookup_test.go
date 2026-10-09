package cmd

import (
	"net/http"
	"testing"

	"github.com/streambinder/spotitube/sys"
	"github.com/stretchr/testify/assert"
)

func BenchmarkLookup(b *testing.B) {
	for i := 0; i < b.N; i++ {
		TestCmdLookup(&testing.T{})
	}
}

func TestCmdLookup(t *testing.T) {
	seedSession(t)
	installWeb(t, nil, webScript{})
	clearTrackCache(t, "123")

	// testing
	assert.Nil(t, sys.ErrOnly(testExecute(cmdLookup(), "-l")))
}

func TestCmdLookupNoTrack(t *testing.T) {
	// testing
	assert.EqualError(t, sys.ErrOnly(testExecute(cmdLookup())), "no track has been issued")
}

func TestCmdLookupTrack(t *testing.T) {
	seedSession(t)
	installWeb(t, nil, webScript{})
	clearTrackCache(t, "123")

	// testing
	assert.Nil(t, sys.ErrOnly(testExecute(cmdLookup(), "123")))
}

func TestCmdLookupRandom(t *testing.T) {
	seedSession(t)
	installWeb(t, nil, webScript{})
	clearTrackCache(t, "123")

	// testing
	assert.Nil(t, sys.ErrOnly(testExecute(cmdLookup(), "-r")))
}

func TestCmdLookupAuthFailure(t *testing.T) {
	noSession(t)
	installWeb(t, nil, webScript{})
	noOpenerPath(t)
	occupyAuthPort(t)

	// testing
	assert.NotNil(t, sys.ErrOnly(testExecute(cmdLookup(), "-l")))
}

func TestCmdLookupLibraryFailure(t *testing.T) {
	seedSession(t)
	installWeb(t, map[string]func(*http.Request) (int, string){
		"/v1/me/tracks": func(*http.Request) (int, string) { return apiError(500, "ko") },
	}, webScript{})

	// testing
	assert.EqualError(t, sys.ErrOnly(testExecute(cmdLookup(), "-l")), "ko")
}

func TestCmdLookupTrackFailure(t *testing.T) {
	seedSession(t)
	installWeb(t, map[string]func(*http.Request) (int, string){
		"/v1/tracks/": func(*http.Request) (int, string) { return apiError(500, "ko") },
	}, webScript{})

	// testing
	assert.EqualError(t, sys.ErrOnly(testExecute(cmdLookup(), "123")), "ko")
}

func TestCmdLookupRandomFailure(t *testing.T) {
	seedSession(t)
	installWeb(t, map[string]func(*http.Request) (int, string){
		"/v1/search": func(*http.Request) (int, string) { return apiError(500, "ko") },
	}, webScript{})

	// testing
	assert.EqualError(t, sys.ErrOnly(testExecute(cmdLookup(), "-r")), "ko")
}

func TestCmdLookupSearchFailure(t *testing.T) {
	seedSession(t)
	installWeb(t, nil, webScript{providerErr: true})
	clearTrackCache(t, "123")

	// testing: provider failures are reported, not propagated
	assert.Nil(t, sys.ErrOnly(testExecute(cmdLookup(), "-l")))
}

func TestCmdLookupSearchNotFound(t *testing.T) {
	seedSession(t)
	installWeb(t, nil, webScript{noResults: true})
	clearTrackCache(t, "123")

	// testing
	assert.Nil(t, sys.ErrOnly(testExecute(cmdLookup(), "-l")))
}

func TestCmdLookupLyricsFailure(t *testing.T) {
	seedSession(t)
	installWeb(t, nil, webScript{lyricsErr: true})
	clearTrackCache(t, "123")

	// testing: lyrics failures are reported, not propagated
	assert.Nil(t, sys.ErrOnly(testExecute(cmdLookup(), "-l")))
}

func TestCmdLookupLyricsNotFound(t *testing.T) {
	seedSession(t)
	installWeb(t, nil, webScript{noLyrics: true})
	clearTrackCache(t, "123")

	// testing
	assert.Nil(t, sys.ErrOnly(testExecute(cmdLookup(), "-l")))
}
