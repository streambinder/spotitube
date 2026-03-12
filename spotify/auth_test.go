package spotify

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/arunsworld/nursery"
	"github.com/bytedance/mockey"
	"github.com/streambinder/spotitube/sys"
	"github.com/streambinder/spotitube/sys/cmd"
	"github.com/stretchr/testify/assert"
	"github.com/thanhpk/randstr"
	"github.com/zmb3/spotify/v2"
	spotifyauth "github.com/zmb3/spotify/v2/auth"
)

const (
	state   = "S7473"
	token   = `{"access_token":"access","token_type":"type","refresh_token":"refresh","expiry":"2023-04-15T12:52:29.143037+02:00"}`
	portMin = 49152
	portMax = 65535
)

var (
	ports = make(map[int]bool)
	lock  sync.RWMutex
)

func testClient() *Client {
	return &Client{spotify.New(http.DefaultClient), &spotifyauth.Authenticator{}, "", make(map[string]interface{})}
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

func BenchmarkAuth(b *testing.B) {
	for i := 0; i < b.N; i++ {
		TestAuthenticate(&testing.T{})
	}
}

func TestAuthenticate(t *testing.T) {
	t.Cleanup(resetPort)
	port = getPort()

	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(os.Getenv).Return("value").Build()
	mockey.Mock(Recover).Return(nil, errors.New("ko")).Build()
	mockey.Mock(mockey.GetMethod(&Client{}, "Persist")).Return(nil).Build()
	mockey.Mock(cmd.Open).Return(nil).Build()
	mockey.Mock(randstr.String).Return(state).Build()
	mockey.Mock(mockey.GetMethod(spotifyauth.Authenticator{}, "Token")).Return(nil, nil).Build()

	// testing
	assert.Nil(t, nursery.RunConcurrently(
		func(_ context.Context, ch chan error) {
			ch <- sys.ErrOnly(Authenticate(BrowserProcessor, "127.0.0.1"))
		},
		func(_ context.Context, _ chan error) {
			var (
				response *http.Response
				err      error
			)

			for {
				response, err = http.Get(fmt.Sprintf("http://127.0.0.1:%d/callback?code=C0D3&state=%s", port, state))
				if errors.Is(err, syscall.ECONNREFUSED) {
					time.Sleep(100 * time.Millisecond)
					continue
				}
				response.Body.Close()
				break
			}
			assert.Equal(t, http.StatusOK, response.StatusCode)
		},
	))
}

func TestAuthenticateNoClientID(t *testing.T) {
	t.Cleanup(resetPort)
	port = getPort()

	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(os.Getenv).To(func(env string) string {
		if env == "SPOTIFY_ID" {
			return ""
		}
		return "value"
	}).Build()

	// testing
	assert.Error(t, sys.ErrOnly(Authenticate(nil)))
}

func TestAuthenticateNoClientSecret(t *testing.T) {
	t.Cleanup(resetPort)
	port = getPort()

	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(os.Getenv).To(func(env string) string {
		if env == "SPOTIFY_KEY" {
			return ""
		}
		return "value"
	}).Build()

	// testing
	assert.Error(t, sys.ErrOnly(Authenticate(nil)))
}

func TestAuthenticateRecoverAndPersist(t *testing.T) {
	t.Cleanup(resetPort)
	port = getPort()

	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(os.Getenv).Return("value").Build()
	mockey.Mock(os.ReadFile).Return([]byte(token), nil).Build()
	mockey.Mock(os.OpenFile).Return(&os.File{}, nil).Build()
	mockey.Mock(mockey.GetMethod(&spotify.Client{}, "Token")).Return(nil, nil).Build()
	mockey.Mock(mockey.GetMethod(&json.Encoder{}, "Encode")).Return(nil).Build()

	// testing
	assert.Nil(t, sys.ErrOnly(Authenticate(nil)))
}

func TestAuthenticateRecoverOpenFailure(t *testing.T) {
	t.Cleanup(resetPort)
	port = getPort()

	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(os.Getenv).Return("value").Build()
	mockey.Mock(os.ReadFile).Return(nil, errors.New("ko")).Build()
	mockey.Mock(mockey.GetMethod(&Client{}, "Persist")).Return(nil).Build()
	mockey.Mock(randstr.String).Return(state).Build()
	mockey.Mock(mockey.GetMethod(spotifyauth.Authenticator{}, "Token")).Return(nil, nil).Build()

	// testing
	assert.Nil(t, nursery.RunConcurrently(
		func(_ context.Context, ch chan error) {
			ch <- sys.ErrOnly(Authenticate(nil))
		},
		func(_ context.Context, _ chan error) {
			var (
				response *http.Response
				err      error
			)

			for {
				response, err = http.Get(fmt.Sprintf("http://127.0.0.1:%d/callback?code=C0D3&state=%s", port, state))
				if errors.Is(err, syscall.ECONNREFUSED) {
					time.Sleep(100 * time.Millisecond)
					continue
				}
				response.Body.Close()
				break
			}
			assert.Equal(t, http.StatusOK, response.StatusCode)
		},
	))
}

func TestAuthenticateRecoverUnmarshalFailure(t *testing.T) {
	t.Cleanup(resetPort)
	port = getPort()

	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(os.Getenv).Return("value").Build()
	mockey.Mock(os.ReadFile).Return([]byte(token), nil).Build()
	mockey.Mock(json.Unmarshal).Return(errors.New("ko")).Build()
	mockey.Mock(mockey.GetMethod(&Client{}, "Persist")).Return(nil).Build()
	mockey.Mock(randstr.String).Return(state).Build()
	mockey.Mock(mockey.GetMethod(spotifyauth.Authenticator{}, "Token")).Return(nil, nil).Build()

	// testing
	assert.Nil(t, nursery.RunConcurrently(
		func(_ context.Context, ch chan error) {
			ch <- sys.ErrOnly(Authenticate(nil))
		},
		func(_ context.Context, _ chan error) {
			var (
				response *http.Response
				err      error
			)

			for {
				response, err = http.Get(fmt.Sprintf("http://127.0.0.1:%d/callback?code=C0D3&state=%s", port, state))
				if errors.Is(err, syscall.ECONNREFUSED) {
					time.Sleep(100 * time.Millisecond)
					continue
				}
				response.Body.Close()
				break
			}
			assert.Equal(t, http.StatusOK, response.StatusCode)
		},
	))
}

func TestAuthenticateRecoverAndPersistTokenFailure(t *testing.T) {
	t.Cleanup(resetPort)
	port = getPort()

	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(os.Getenv).Return("value").Build()
	mockey.Mock(os.ReadFile).Return([]byte(token), nil).Build()
	mockey.Mock(mockey.GetMethod(&spotify.Client{}, "Token")).Return(nil, errors.New("ko")).Build()

	// testing
	assert.EqualError(t, sys.ErrOnly(Authenticate(nil)), "ko")
}

func TestAuthenticateRecoverAndPersistMkdirFailure(t *testing.T) {
	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(os.Getenv).Return("value").Build()
	mockey.Mock(Recover).Return(nil, nil).Build()
	mockey.Mock(os.MkdirAll).Return(errors.New("ko")).Build()

	// testing
	assert.EqualError(t, sys.ErrOnly(Authenticate(nil)), "ko")
}

func TestAuthenticateRecoverAndPersistOpenFailure(t *testing.T) {
	t.Cleanup(resetPort)
	port = getPort()

	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(os.Getenv).Return("value").Build()
	mockey.Mock(os.ReadFile).Return([]byte(token), nil).Build()
	mockey.Mock(mockey.GetMethod(&spotify.Client{}, "Token")).Return(nil, nil).Build()
	mockey.Mock(os.OpenFile).Return(nil, errors.New("ko")).Build()

	// testing
	assert.EqualError(t, sys.ErrOnly(Authenticate(nil)), "ko")
}

func TestAuthenticateNotFound(t *testing.T) {
	t.Cleanup(resetPort)
	port = getPort()

	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(os.Getenv).Return("value").Build()
	mockey.Mock(Recover).Return(nil, errors.New("ko")).Build()
	mockey.Mock(mockey.GetMethod(&Client{}, "Persist")).Return(nil).Build()
	mockey.Mock(randstr.String).Return(state).Build()
	mockey.Mock(mockey.GetMethod(spotifyauth.Authenticator{}, "Token")).Return(nil, nil).Build()

	// testing
	assert.EqualError(t, nursery.RunConcurrently(
		func(_ context.Context, ch chan error) {
			ch <- sys.ErrOnly(Authenticate(nil))
		},
		func(_ context.Context, _ chan error) {
			var (
				response *http.Response
				err      error
			)

			for {
				response, err = http.Get(fmt.Sprintf("http://127.0.0.1:%d/callback?code=C0D3&state=null", port))
				if errors.Is(err, syscall.ECONNREFUSED) {
					time.Sleep(100 * time.Millisecond)
					continue
				}
				response.Body.Close()
				break
			}
			assert.Equal(t, http.StatusOK, response.StatusCode)
		},
	), http.StatusText(http.StatusNotFound))
}

func TestAuthenticateForbidden(t *testing.T) {
	t.Cleanup(resetPort)
	port = getPort()

	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(os.Getenv).Return("value").Build()
	mockey.Mock(Recover).Return(nil, errors.New("ko")).Build()
	mockey.Mock(mockey.GetMethod(&Client{}, "Persist")).Return(nil).Build()
	mockey.Mock(randstr.String).Return(state).Build()
	mockey.Mock(mockey.GetMethod(spotifyauth.Authenticator{}, "Token")).Return(nil, errors.New("ko")).Build()

	// testing
	assert.EqualError(t, nursery.RunConcurrently(
		func(_ context.Context, ch chan error) {
			ch <- sys.ErrOnly(Authenticate(nil))
		},
		func(_ context.Context, _ chan error) {
			var (
				response *http.Response
				err      error
			)

			for {
				response, err = http.Get(fmt.Sprintf("http://127.0.0.1:%d/callback", port))
				if errors.Is(err, syscall.ECONNREFUSED) {
					time.Sleep(100 * time.Millisecond)
					continue
				}
				response.Body.Close()
				break
			}
			assert.Equal(t, http.StatusOK, response.StatusCode)
		},
	), http.StatusText(http.StatusForbidden))
}

func TestAuthenticateProcessorFailure(t *testing.T) {
	t.Cleanup(resetPort)
	port = getPort()

	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(os.Getenv).Return("value").Build()
	mockey.Mock(Recover).Return(nil, errors.New("ko")).Build()
	mockey.Mock(mockey.GetMethod(&Client{}, "Persist")).Return(nil).Build()
	mockey.Mock(randstr.String).Return(state).Build()

	// testing
	assert.EqualError(t, nursery.RunConcurrently(
		func(_ context.Context, ch chan error) {
			ch <- sys.ErrOnly(Authenticate(func(_ string) error {
				return errors.New("ko")
			}))
		},
		func(_ context.Context, _ chan error) {
			var (
				response *http.Response
				err      error
			)

			for {
				response, err = http.Get(fmt.Sprintf("http://127.0.0.1:%d/callback?code=C0D3&state=%s", port, state))
				if errors.Is(err, syscall.ECONNREFUSED) {
					time.Sleep(100 * time.Millisecond)
					continue
				}
				response.Body.Close()
				break
			}
			assert.Equal(t, http.StatusOK, response.StatusCode)
		},
	), "ko")
}

func TestAuthenticateServerUnserving(t *testing.T) {
	t.Cleanup(resetPort)
	port = getPort()

	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(os.Getenv).Return("value").Build()
	mockey.Mock(Recover).Return(nil, errors.New("ko")).Build()
	mockey.Mock(mockey.GetMethod(&Client{}, "Persist")).Return(nil).Build()
	mockey.Mock(randstr.String).Return(state).Build()
	mockey.Mock(net.Listen).Return(nil, errors.New("ko")).Build()

	// testing
	assert.EqualError(t, sys.ErrOnly(Authenticate(nil)), "ko")
}
