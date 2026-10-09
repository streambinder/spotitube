package cmd

import (
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/streambinder/spotitube/spotify"
	"github.com/streambinder/spotitube/sys"
	"github.com/stretchr/testify/assert"
)

func BenchmarkAuth(b *testing.B) {
	for i := 0; i < b.N; i++ {
		TestCmdAuth(&testing.T{})
	}
}

func TestCmdAuth(t *testing.T) {
	// a seeded session is recovered without any browser flow
	seedSession(t)
	installSpotify(t, nil)

	// testing
	assert.Nil(t, sys.ErrOnly(testExecute(cmdAuth())))
}

func TestCmdAuthBrowserFlow(t *testing.T) {
	// without a session, the command opens the browser at the auth URL:
	// the fake opener logs it, this test completes the OAuth2 callback
	// against the flow's loopback server like a browser would
	noSession(t)
	installSpotify(t, nil)
	binDir := fakeBinDir(t)

	callbackErr := make(chan error, 1)
	go func() {
		logPath := filepath.Join(binDir, "open.log")
		var authURL string
		for deadline := time.Now().Add(30 * time.Second); time.Now().Before(deadline); {
			if data, err := os.ReadFile(logPath); err == nil && len(data) > 0 {
				authURL = strings.TrimSpace(string(data))
				break
			}
			time.Sleep(10 * time.Millisecond)
		}
		if authURL == "" {
			callbackErr <- os.ErrNotExist
			return
		}
		parsed, err := url.Parse(authURL)
		if err != nil {
			callbackErr <- err
			return
		}
		query := parsed.Query()
		callback := query.Get("redirect_uri") + "?code=testcode&state=" + query.Get("state")
		// the callback must reach the real loopback server: use a client
		// with a real transport, not the scripted default one
		realClient := &http.Client{Transport: &http.Transport{Proxy: nil}}
		var response *http.Response
		for deadline := time.Now().Add(30 * time.Second); time.Now().Before(deadline); {
			response, err = realClient.Get(callback) //nolint:bodyclose // closed below, or nil on error
			if err == nil {
				break
			}
			time.Sleep(10 * time.Millisecond)
		}
		if err != nil {
			callbackErr <- err
			return
		}
		_ = response.Body.Close()
		callbackErr <- nil
	}()

	// testing
	assert.Nil(t, sys.ErrOnly(testExecute(cmdAuth())))
	assert.Nil(t, <-callbackErr)

	// the completed flow persisted a fresh session
	_, err := os.Stat(sys.CacheFile(spotify.TokenBasename))
	assert.Nil(t, err)
}

func TestCmdAuthFailure(t *testing.T) {
	// without a session, authentication falls through to the browser
	// flow, whose loopback server cannot bind the occupied port
	noSession(t)
	installSpotify(t, nil)
	noOpenerPath(t)
	occupyAuthPort(t)

	// testing
	assert.NotNil(t, sys.ErrOnly(testExecute(cmdAuth())))
}

func TestCmdAuthLogout(t *testing.T) {
	seedSession(t)
	installSpotify(t, nil)
	noOpenerPath(t)
	occupyAuthPort(t)

	// testing: the logout deletes the session file, then the fresh
	// authentication fails at the browser flow
	assert.NotNil(t, sys.ErrOnly(testExecute(cmdAuth(), "--logout")))
	_, err := os.Stat(sys.CacheFile(spotify.TokenBasename))
	assert.True(t, os.IsNotExist(err))
}

func TestCmdAuthLogoutFailure(t *testing.T) {
	// a session path occupied by a non-empty directory makes the
	// removal fail with a real filesystem error
	noSession(t)
	path := sys.CacheFile(spotify.TokenBasename)
	assert.Nil(t, os.MkdirAll(path, 0o755))
	assert.Nil(t, os.WriteFile(filepath.Join(path, "keep"), nil, 0o644))
	t.Cleanup(func() { _ = os.RemoveAll(path) })

	// testing
	assert.NotNil(t, sys.ErrOnly(testExecute(cmdAuth(), "--logout")))
}

func TestCmdAuthLogoutNotExists(t *testing.T) {
	// a missing session file is not an error for the logout: the flow
	// proceeds to the browser authentication, which fails here
	noSession(t)
	installSpotify(t, nil)
	noOpenerPath(t)
	occupyAuthPort(t)

	// testing
	assert.NotNil(t, sys.ErrOnly(testExecute(cmdAuth(), "--logout")))
}
