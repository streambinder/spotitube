package spotify

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"reflect"
	"syscall"
	"testing"
	"time"

	"bou.ke/monkey"
	"github.com/arunsworld/nursery"
	"github.com/streambinder/spotitube/util"
	"github.com/streambinder/spotitube/util/cmd"
	"github.com/stretchr/testify/assert"
	"github.com/thanhpk/randstr"
	"github.com/zmb3/spotify"
	"golang.org/x/oauth2"
)

const (
	state = "S7473"
)

func TestAuthenticate(t *testing.T) {
	// monkey patching
	monkey.Patch(cmd.Open, func(string, ...string) error { return nil })
	defer monkey.Unpatch(cmd.Open)
	monkey.Patch(randstr.Hex, func(int) string { return state })
	defer monkey.Unpatch(randstr.Hex)
	monkey.PatchInstanceMethod(reflect.TypeOf(spotify.Authenticator{}), "Token",
		func(spotify.Authenticator, string, *http.Request) (*oauth2.Token, error) {
			return nil, nil
		})
	defer monkey.UnpatchInstanceMethod(reflect.TypeOf(spotify.Authenticator{}), "Token")

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
				response, err = http.Get(fmt.Sprintf("http://127.0.0.1:%d/callback?code=C0D3&state=S7473", port))
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

func TestAuthenticateNotFound(t *testing.T) {
	// monkey patching
	monkey.Patch(cmd.Open, func(string, ...string) error { return nil })
	defer monkey.Unpatch(cmd.Open)
	monkey.Patch(randstr.Hex, func(int) string { return state })
	defer monkey.Unpatch(randstr.Hex)
	monkey.PatchInstanceMethod(reflect.TypeOf(spotify.Authenticator{}), "Token",
		func(spotify.Authenticator, string, *http.Request) (*oauth2.Token, error) {
			return nil, nil
		})
	defer monkey.UnpatchInstanceMethod(reflect.TypeOf(spotify.Authenticator{}), "Token")

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
	monkey.Patch(cmd.Open, func(string, ...string) error { return nil })
	defer monkey.Unpatch(cmd.Open)
	monkey.Patch(randstr.Hex, func(int) string { return state })
	defer monkey.Unpatch(randstr.Hex)
	monkey.PatchInstanceMethod(reflect.TypeOf(spotify.Authenticator{}), "Token",
		func(spotify.Authenticator, string, *http.Request) (*oauth2.Token, error) {
			return nil, errors.New("failure")
		})
	defer monkey.UnpatchInstanceMethod(reflect.TypeOf(spotify.Authenticator{}), "Token")

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
	monkey.Patch(cmd.Open, func(string, ...string) error { return errors.New("failure") })
	defer monkey.Unpatch(cmd.Open)
	monkey.Patch(randstr.Hex, func(int) string { return state })
	defer monkey.Unpatch(randstr.Hex)

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
				response, err = http.Get(fmt.Sprintf("http://127.0.0.1:%d/callback?code=C0D3&state=S7473", port))
				if errors.Is(err, syscall.ECONNREFUSED) {
					time.Sleep(100 * time.Millisecond)
					continue
				}
				response.Body.Close()
				break
			}
			assert.Equal(t, http.StatusOK, response.StatusCode)
		},
	), "failure")
}

func TestAuthenticateServerUnserving(t *testing.T) {
	// monkey patching
	monkey.Patch(cmd.Open, func(string, ...string) error { return nil })
	defer monkey.Unpatch(cmd.Open)
	monkey.Patch(randstr.Hex, func(int) string { return state })
	defer monkey.Unpatch(randstr.Hex)
	monkey.Patch(net.Listen, func(string, string) (net.Listener, error) { return nil, errors.New("failure") })
	defer monkey.Unpatch(net.Listen)

	// testing
	assert.EqualError(t, util.ErrOnly(Authenticate()), "failure")
}
