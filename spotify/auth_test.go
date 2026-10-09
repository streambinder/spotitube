package spotify

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/streambinder/spotitube/sys"
	"github.com/stretchr/testify/assert"
	"github.com/zmb3/spotify/v2"
	spotifyauth "github.com/zmb3/spotify/v2/auth"
	"golang.org/x/oauth2"
)

const (
	portMin = 49152
	portMax = 65535
)

var (
	ports      = make(map[int]bool)
	lock       sync.RWMutex
	httpClient = &http.Client{Transport: &http.Transport{Proxy: nil}}
)

func testClient() *Client {
	return &Client{spotify.New(http.DefaultClient), &spotifyauth.Authenticator{}, "", make(map[string]interface{})}
}

// oauthClient returns a client backed by an OAuth2 transport holding a
// fresh token, like the ones Authenticate builds.
func oauthClient() *Client {
	authenticator := spotifyauth.New()
	token := &oauth2.Token{
		AccessToken:  "access",
		TokenType:    "Bearer",
		RefreshToken: "refresh",
		Expiry:       time.Now().Add(time.Hour),
	}
	return &Client{
		spotify.New(authenticator.Client(context.Background(), token), spotify.WithRetry(true)),
		authenticator, "", make(map[string]interface{}),
	}
}

func getPort() int {
	lock.Lock()
	defer lock.Unlock()

	port = sys.RandomInt(portMax, portMin)
	if _, ok := ports[port]; ok {
		return getPort()
	}

	ports[port] = true
	return port
}

func resetPort() {
	port = 65535
}

// setCredentials sets the Spotify application credentials Authenticate
// reads from the environment.
func setCredentials(t *testing.T) {
	t.Helper()
	t.Setenv("SPOTIFY_ID", "id")
	t.Setenv("SPOTIFY_KEY", "key")
}

// setTokenPath redirects the session token file to the given path for the
// duration of a test.
func setTokenPath(t *testing.T, path string) {
	t.Helper()
	previous := tokenPath
	tokenPath = path
	t.Cleanup(func() { tokenPath = previous })
}

// seedTokenPath points the session token file at a fresh file holding a
// valid, unexpired OAuth2 token, and returns its path.
func seedTokenPath(t *testing.T) string {
	t.Helper()
	data, err := json.Marshal(oauth2.Token{
		AccessToken:  "access",
		TokenType:    "Bearer",
		RefreshToken: "refresh",
		Expiry:       time.Now().Add(time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), TokenBasename)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	setTokenPath(t, path)
	return path
}

// authenticateViaCallback runs Authenticate in a goroutine. The URL
// processor handed to Authenticate records the authorization URL and then
// delegates to processor (nil for a plain recording); once the URL is
// known, callback performs the loopback callback request against
// Authenticate's real server, and the error returned by Authenticate is
// returned to the caller.
func authenticateViaCallback(t *testing.T, processor func(string) error, callback func(authURL string)) error {
	t.Helper()
	urls := make(chan string, 1)
	errCh := make(chan error, 1)
	go func() {
		errCh <- sys.ErrOnly(Authenticate(func(authURL string) error {
			urls <- authURL
			if processor != nil {
				return processor(authURL)
			}
			return nil
		}))
	}()

	var authURL string
	select {
	case authURL = <-urls:
	case <-time.After(10 * time.Second):
		t.Fatal("authorization URL never delivered")
	}
	callback(authURL)
	return <-errCh
}

// stateOf extracts the state parameter out of an authorization URL.
func stateOf(t *testing.T, authURL string) string {
	t.Helper()
	parsed, err := url.Parse(authURL)
	if err != nil {
		t.Fatal(err)
	}
	return parsed.Query().Get("state")
}

// callbackTarget builds the loopback callback URL for the current port.
func callbackTarget(query string) string {
	if query == "" {
		return fmt.Sprintf("http://127.0.0.1:%d/callback", port)
	}
	return fmt.Sprintf("http://127.0.0.1:%d/callback?%s", port, query)
}

// tryRequest issues a request against Authenticate's loopback server,
// retrying while the server is not accepting connections yet, and reports
// whether the request eventually went through.
func tryRequest(request func() (*http.Response, error)) bool {
	for attempt := 0; attempt < 200; attempt++ {
		response, err := request()
		if err == nil {
			_, copyErr := io.Copy(io.Discard, response.Body)
			response.Body.Close()
			if copyErr != nil {
				return false
			}
			return response.StatusCode == http.StatusOK
		}
		time.Sleep(10 * time.Millisecond)
	}
	return false
}

// getCallback performs a successful authorization callback for the given
// authorization URL.
func getCallback(t *testing.T, authURL string) {
	t.Helper()
	target := callbackTarget("code=C0D3&state=" + stateOf(t, authURL))
	if !tryRequest(func() (*http.Response, error) { return httpClient.Get(target) }) {
		t.Fatal("callback request never succeeded")
	}
}

func BenchmarkAuth(b *testing.B) {
	for i := 0; i < b.N; i++ {
		TestAuthenticate(&testing.T{})
	}
}

func TestAuthenticate(t *testing.T) {
	t.Cleanup(resetPort)
	port = getPort()
	setCredentials(t)
	tokenFile := filepath.Join(t.TempDir(), TokenBasename)
	setTokenPath(t, tokenFile)
	scriptAuthTransport(t, func(_ *http.Request) (int, string) {
		return apiError(http.StatusNotFound, "unexpected")
	})

	err := authenticateViaCallback(t, nil, func(authURL string) {
		getCallback(t, authURL)
	})

	assert.Nil(t, err)
	// a successful authentication persists the session token
	_, statErr := os.Stat(tokenFile)
	assert.Nil(t, statErr)
}

func TestAuthenticateNoClientID(t *testing.T) {
	setCredentials(t)
	t.Setenv("SPOTIFY_ID", "")

	assert.EqualError(t, sys.ErrOnly(Authenticate(nil)), "SPOTIFY_ID not set")
}

func TestAuthenticateNoClientSecret(t *testing.T) {
	setCredentials(t)
	t.Setenv("SPOTIFY_KEY", "")

	assert.EqualError(t, sys.ErrOnly(Authenticate(nil)), "SPOTIFY_KEY not set")
}

func TestAuthenticateRecoverAndPersist(t *testing.T) {
	setCredentials(t)
	seedTokenPath(t)
	scriptAuthTransport(t, func(request *http.Request) (int, string) {
		if request.URL.Path == "/v1/me" {
			return http.StatusOK, meJSON
		}
		return apiError(http.StatusNotFound, "unexpected")
	})

	assert.Nil(t, sys.ErrOnly(Authenticate(nil)))
}

func TestAuthenticateRecoverOpenFailure(t *testing.T) {
	t.Cleanup(resetPort)
	port = getPort()
	setCredentials(t)
	// the token file does not exist: Recover fails and the flow falls
	// through to the browser-based authorization
	setTokenPath(t, filepath.Join(t.TempDir(), TokenBasename))
	scriptAuthTransport(t, func(_ *http.Request) (int, string) {
		return apiError(http.StatusNotFound, "unexpected")
	})

	err := authenticateViaCallback(t, nil, func(authURL string) {
		getCallback(t, authURL)
	})

	assert.Nil(t, err)
}

func TestAuthenticateRecoverUnmarshalFailure(t *testing.T) {
	t.Cleanup(resetPort)
	port = getPort()
	setCredentials(t)
	// the token file holds garbage: Recover cannot unmarshal it and the
	// flow falls through to the browser-based authorization
	path := filepath.Join(t.TempDir(), TokenBasename)
	if err := os.WriteFile(path, []byte("not a token"), 0o600); err != nil {
		t.Fatal(err)
	}
	setTokenPath(t, path)
	scriptAuthTransport(t, func(_ *http.Request) (int, string) {
		return apiError(http.StatusNotFound, "unexpected")
	})

	err := authenticateViaCallback(t, nil, func(authURL string) {
		getCallback(t, authURL)
	})

	assert.Nil(t, err)
}

func TestAuthenticateRecoverExpiredSession(t *testing.T) {
	t.Cleanup(resetPort)
	port = getPort()
	setCredentials(t)
	tokenFile := seedTokenPath(t)
	scriptAuthTransport(t, func(request *http.Request) (int, string) {
		if request.URL.Path == "/v1/me" {
			return apiError(http.StatusUnauthorized, "The access token expired")
		}
		return apiError(http.StatusNotFound, "unexpected")
	})

	// the stored session is rejected: Recover drops it and the flow falls
	// through to the browser-based authorization, which persists anew
	err := authenticateViaCallback(t, nil, func(authURL string) {
		getCallback(t, authURL)
	})

	assert.Nil(t, err)
	_, statErr := os.Stat(tokenFile)
	assert.Nil(t, statErr)
}

func TestAuthenticateRecoverAndPersistMkdirFailure(t *testing.T) {
	t.Cleanup(resetPort)
	port = getPort()
	setCredentials(t)
	// the token path lives below a regular file: Recover cannot read it,
	// and Persist cannot create its parent directory
	blocker := filepath.Join(t.TempDir(), "blocker")
	if err := os.WriteFile(blocker, []byte{}, 0o644); err != nil {
		t.Fatal(err)
	}
	setTokenPath(t, filepath.Join(blocker, TokenBasename))
	scriptAuthTransport(t, func(_ *http.Request) (int, string) {
		return apiError(http.StatusNotFound, "unexpected")
	})

	err := authenticateViaCallback(t, nil, func(authURL string) {
		getCallback(t, authURL)
	})

	assert.Error(t, err)
}

func TestAuthenticateRecoverAndPersistOpenFailure(t *testing.T) {
	t.Cleanup(resetPort)
	port = getPort()
	setCredentials(t)
	// the token path is an existing directory: Recover cannot read it,
	// and Persist cannot open it for writing
	tokenDir := filepath.Join(t.TempDir(), TokenBasename)
	if err := os.Mkdir(tokenDir, 0o755); err != nil {
		t.Fatal(err)
	}
	setTokenPath(t, tokenDir)
	scriptAuthTransport(t, func(_ *http.Request) (int, string) {
		return apiError(http.StatusNotFound, "unexpected")
	})

	err := authenticateViaCallback(t, nil, func(authURL string) {
		getCallback(t, authURL)
	})

	assert.Error(t, err)
}

func TestAuthenticateRecoverAndPersistWriteFailure(t *testing.T) {
	t.Cleanup(resetPort)
	port = getPort()
	setCredentials(t)
	// the token file does not exist, so Recover fails; while the flow is
	// in flight it is replaced by a symlink to /dev/full, which accepts
	// opens but fails every write with ENOSPC, so Persist fails writing
	// the session out
	tokenFile := filepath.Join(t.TempDir(), TokenBasename)
	setTokenPath(t, tokenFile)
	scriptAuthTransport(t, func(_ *http.Request) (int, string) {
		return apiError(http.StatusNotFound, "unexpected")
	})

	err := authenticateViaCallback(t, nil, func(authURL string) {
		if err := os.Symlink("/dev/full", tokenFile); err != nil {
			t.Fatal(err)
		}
		getCallback(t, authURL)
	})

	assert.Error(t, err)
}

func TestPersistTokenFailure(t *testing.T) {
	// a client not backed by an OAuth2 transport has no token to persist
	setTokenPath(t, filepath.Join(t.TempDir(), TokenBasename))

	assert.EqualError(t, testClient().Persist(), "spotify: client not backed by oauth2 transport")
}

func TestUsername(t *testing.T) {
	scriptTransport(t, func(_ *http.Request) (int, string) {
		return http.StatusOK, meJSON
	})

	username, err := testClient().Username()

	assert.Nil(t, err)
	assert.Equal(t, "alice", username)
}

func TestUsernameCached(t *testing.T) {
	username, err := (&Client{
		cache: map[string]interface{}{
			currentUserCacheID: &spotify.PrivateUser{
				User: spotify.User{ID: "alice"},
			},
		},
	}).Username()

	assert.Nil(t, err)
	assert.Equal(t, "alice", username)
}

func TestUsernameCurrentUserFailure(t *testing.T) {
	scriptTransport(t, func(_ *http.Request) (int, string) {
		return apiError(http.StatusInternalServerError, "ko")
	})

	username, err := testClient().Username()

	assert.Empty(t, username)
	assert.EqualError(t, err, "ko")
}

func TestUsernameUninitializedClient(t *testing.T) {
	username, err := (&Client{}).Username()

	assert.Empty(t, username)
	assert.EqualError(t, err, "spotify client not initialized")
}

func TestCloseNilClient(t *testing.T) {
	assert.Nil(t, (*Client)(nil).Close())
	assert.Nil(t, (&Client{}).Close())
}

func TestClose(t *testing.T) {
	tokenFile := filepath.Join(t.TempDir(), TokenBasename)
	setTokenPath(t, tokenFile)

	assert.Nil(t, oauthClient().Close())

	_, err := os.Stat(tokenFile)
	assert.Nil(t, err)
}

func TestBrowserProcessor(t *testing.T) {
	binDir := t.TempDir()
	script := filepath.Join(binDir, "xdg-open")
	if err := os.WriteFile(script, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir)

	assert.Nil(t, BrowserProcessor("http://ima.ge"))

	// without an opener on PATH, starting it fails
	t.Setenv("PATH", t.TempDir())
	assert.Error(t, BrowserProcessor("http://ima.ge"))
}

func TestAuthenticateNotFound(t *testing.T) {
	t.Cleanup(resetPort)
	port = getPort()
	setCredentials(t)
	setTokenPath(t, filepath.Join(t.TempDir(), TokenBasename))
	scriptAuthTransport(t, func(_ *http.Request) (int, string) {
		return apiError(http.StatusNotFound, "unexpected")
	})

	// the token exchange validates the state carried by the URL query,
	// while the callback handler re-reads it through FormValue, where
	// POST body parameters win over query ones: a state forged in the
	// body passes the exchange and fails the handler's own check
	err := authenticateViaCallback(t, nil, func(authURL string) {
		target := callbackTarget("code=C0D3&state=" + stateOf(t, authURL))
		if !tryRequest(func() (*http.Response, error) {
			request, err := http.NewRequest(http.MethodPost, target, strings.NewReader("state=forged"))
			if err != nil {
				return nil, err
			}
			request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			return httpClient.Do(request)
		}) {
			t.Fatal("callback request never succeeded")
		}
	})

	assert.EqualError(t, err, http.StatusText(http.StatusNotFound))
}

func TestAuthenticateForbidden(t *testing.T) {
	t.Cleanup(resetPort)
	port = getPort()
	setCredentials(t)
	setTokenPath(t, filepath.Join(t.TempDir(), TokenBasename))
	scriptAuthTransport(t, func(_ *http.Request) (int, string) {
		return apiError(http.StatusNotFound, "unexpected")
	})

	// a callback without an authorization code fails the token exchange
	err := authenticateViaCallback(t, nil, func(_ string) {
		if !tryRequest(func() (*http.Response, error) {
			return httpClient.Get(callbackTarget(""))
		}) {
			t.Fatal("callback request never succeeded")
		}
	})

	assert.EqualError(t, err, http.StatusText(http.StatusForbidden))
}

func TestAuthenticateProcessorFailure(t *testing.T) {
	t.Cleanup(resetPort)
	port = getPort()
	setCredentials(t)
	setTokenPath(t, filepath.Join(t.TempDir(), TokenBasename))
	scriptAuthTransport(t, func(_ *http.Request) (int, string) {
		return apiError(http.StatusNotFound, "unexpected")
	})

	err := authenticateViaCallback(t, func(string) error {
		return errors.New("ko")
	}, func(authURL string) {
		getCallback(t, authURL)
	})

	assert.EqualError(t, err, "ko")
}

func TestAuthenticateServerUnserving(t *testing.T) {
	t.Cleanup(resetPort)
	port = getPort()
	setCredentials(t)
	setTokenPath(t, filepath.Join(t.TempDir(), TokenBasename))

	// occupy the port so Authenticate's server cannot bind it
	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()

	assert.ErrorContains(t, sys.ErrOnly(Authenticate(nil)), "address already in use")
}
