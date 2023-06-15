package spotify

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/arunsworld/nursery"
	"github.com/streambinder/spotitube/util"
	"github.com/streambinder/spotitube/util/cmd"
	"github.com/stretchr/testify/assert"
	"github.com/thanhpk/randstr"
	"github.com/zmb3/spotify/v2"
	spotifyauth "github.com/zmb3/spotify/v2/auth"
	"golang.org/x/oauth2"
)

const (
	state = "S7473"
	token = `{"access_token":"access","token_type":"type","refresh_token":"refresh","expiry":"2023-04-15T12:52:29.143037+02:00"}`
)

func testClient() *Client {
	return &Client{spotify.New(http.DefaultClient), &spotifyauth.Authenticator{}, "", make(map[string]interface{})}
}

func BenchmarkAuth(b *testing.B) {
	for i := 0; i < b.N; i++ {
		TestAuthenticate(&testing.T{})
	}
}

func TestAuthenticate(t *testing.T) {
	// monkey patching
	defer gomonkey.NewPatches().
		ApplyFunc(Recover, func() (*Client, error) {
			return nil, errors.New("ko")
		}).
		ApplyMethod(&Client{}, "Persist", func() error {
			return nil
		}).
		ApplyFunc(cmd.Open, func() error {
			return nil
		}).
		ApplyFunc(randstr.String, func() string {
			return state
		}).
		ApplyMethod(spotifyauth.Authenticator{}, "Token", func() (*oauth2.Token, error) {
			return nil, nil
		}).
		Reset()

	// testing
	assert.Nil(t, nursery.RunConcurrently(
		func(ctx context.Context, ch chan error) {
			ch <- util.ErrOnly(Authenticate("127.0.0.1"))
		},
		func(ctx context.Context, ch chan error) {
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

func TestAuthenticateRecoverAndPersist(t *testing.T) {
	// monkey patching
	defer gomonkey.NewPatches().
		ApplyFunc(os.ReadFile, func() ([]byte, error) {
			return []byte(token), nil
		}).
		ApplyFunc(os.OpenFile, func() (*os.File, error) {
			return &os.File{}, nil
		}).
		ApplyMethod(&spotify.Client{}, "Token", func() (*oauth2.Token, error) {
			return nil, nil
		}).
		ApplyMethod(&json.Encoder{}, "Encode", func() error {
			return nil
		}).
		Reset()

	// testing
	assert.Nil(t, util.ErrOnly(Authenticate("127.0.0.1")))
}

func TestAuthenticateRecoverOpenFailure(t *testing.T) {
	// monkey patching
	defer gomonkey.NewPatches().
		ApplyFunc(os.ReadFile, func() ([]byte, error) {
			return nil, errors.New("ko")
		}).
		ApplyMethod(&Client{}, "Persist", func() error {
			return nil
		}).
		ApplyFunc(cmd.Open, func() error {
			return nil
		}).
		ApplyFunc(randstr.String, func() string {
			return state
		}).
		ApplyMethod(spotifyauth.Authenticator{}, "Token", func() (*oauth2.Token, error) {
			return nil, nil
		}).
		Reset()

	// testing
	assert.Nil(t, nursery.RunConcurrently(
		func(ctx context.Context, ch chan error) {
			ch <- util.ErrOnly(Authenticate("127.0.0.1"))
		},
		func(ctx context.Context, ch chan error) {
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
	// monkey patching
	defer gomonkey.NewPatches().
		ApplyFunc(os.ReadFile, func() ([]byte, error) {
			return []byte(token), nil
		}).
		ApplyFunc(json.Unmarshal, func() error {
			return errors.New("ko")
		}).
		ApplyMethod(&Client{}, "Persist", func() error {
			return nil
		}).
		ApplyFunc(cmd.Open, func() error {
			return nil
		}).
		ApplyFunc(randstr.String, func() string {
			return state
		}).
		ApplyMethod(spotifyauth.Authenticator{}, "Token", func() (*oauth2.Token, error) {
			return nil, nil
		}).
		Reset()

	// testing
	assert.Nil(t, nursery.RunConcurrently(
		func(ctx context.Context, ch chan error) {
			ch <- util.ErrOnly(Authenticate("127.0.0.1"))
		},
		func(ctx context.Context, ch chan error) {
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
	// monkey patching
	defer gomonkey.NewPatches().
		ApplyFunc(os.ReadFile, func() ([]byte, error) {
			return []byte(token), nil
		}).
		ApplyMethod(&spotify.Client{}, "Token", func() (*oauth2.Token, error) {
			return nil, errors.New("ko")
		}).
		Reset()

	// testing
	assert.Error(t, util.ErrOnly(Authenticate("127.0.0.1")), "ko")
}

func TestAuthenticateRecoverAndPersistOpenFailure(t *testing.T) {
	// monkey patching
	defer gomonkey.NewPatches().
		ApplyFunc(os.ReadFile, func() ([]byte, error) {
			return []byte(token), nil
		}).
		ApplyMethod(&spotify.Client{}, "Token", func() (*oauth2.Token, error) {
			return nil, nil
		}).
		ApplyFunc(os.OpenFile, func() (*os.File, error) {
			return nil, errors.New("ko")
		}).
		Reset()

	// testing
	assert.Error(t, util.ErrOnly(Authenticate("127.0.0.1")), "ko")
}

func TestAuthenticateNotFound(t *testing.T) {
	// monkey patching
	defer gomonkey.NewPatches().
		ApplyFunc(Recover, func() (*Client, error) {
			return nil, errors.New("ko")
		}).
		ApplyMethod(&Client{}, "Persist", func() error {
			return nil
		}).
		ApplyFunc(cmd.Open, func() error {
			return nil
		}).
		ApplyFunc(randstr.String, func() string {
			return state
		}).
		ApplyMethod(spotifyauth.Authenticator{}, "Token", func() (*oauth2.Token, error) {
			return nil, nil
		}).
		Reset()

	// testing
	assert.EqualError(t, nursery.RunConcurrently(
		func(ctx context.Context, ch chan error) {
			ch <- util.ErrOnly(Authenticate("127.0.0.1"))
		},
		func(ctx context.Context, ch chan error) {
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
	// monkey patching
	defer gomonkey.NewPatches().
		ApplyFunc(Recover, func() (*Client, error) {
			return nil, errors.New("ko")
		}).
		ApplyMethod(&Client{}, "Persist", func() error {
			return nil
		}).
		ApplyFunc(cmd.Open, func() error {
			return nil
		}).
		ApplyFunc(randstr.String, func() string {
			return state
		}).
		ApplyMethod(spotifyauth.Authenticator{}, "Token", func() (*oauth2.Token, error) {
			return nil, errors.New("ko")
		}).
		Reset()

	// testing
	assert.EqualError(t, nursery.RunConcurrently(
		func(ctx context.Context, ch chan error) {
			ch <- util.ErrOnly(Authenticate("127.0.0.1"))
		},
		func(ctx context.Context, ch chan error) {
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

func TestAuthenticateOpenFailure(t *testing.T) {
	// monkey patching
	defer gomonkey.NewPatches().
		ApplyFunc(Recover, func() (*Client, error) {
			return nil, errors.New("ko")
		}).
		ApplyMethod(&Client{}, "Persist", func() error {
			return nil
		}).
		ApplyFunc(cmd.Open, func() error {
			return errors.New("ko")
		}).
		ApplyFunc(randstr.String, func() string {
			return state
		}).
		Reset()

	// testing
	assert.EqualError(t, nursery.RunConcurrently(
		func(ctx context.Context, ch chan error) {
			ch <- util.ErrOnly(Authenticate("127.0.0.1"))
		},
		func(ctx context.Context, ch chan error) {
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
	// monkey patching
	defer gomonkey.NewPatches().
		ApplyFunc(Recover, func() (*Client, error) {
			return nil, errors.New("ko")
		}).
		ApplyMethod(&Client{}, "Persist", func() error {
			return nil
		}).
		ApplyFunc(cmd.Open, func() error {
			return nil
		}).
		ApplyFunc(randstr.String, func() string {
			return state
		}).
		ApplyFunc(net.Listen, func() (net.Listener, error) {
			return nil, errors.New("ko")
		}).
		Reset()

	// testing
	assert.EqualError(t, util.ErrOnly(Authenticate()), "ko")
}
