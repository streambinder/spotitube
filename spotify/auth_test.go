package spotify

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"os"
	"reflect"
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

func TestAuthenticate(t *testing.T) {
	// monkey patching
	patchRecover := gomonkey.ApplyFunc(Recover, func(*spotifyauth.Authenticator, string) (*Client, error) {
		return nil, errors.New("failure")
	})
	defer patchRecover.Reset()
	patchClientPersist := gomonkey.ApplyMethod(reflect.TypeOf(&Client{}), "Persist", func(*Client) error {
		return nil
	})
	defer patchClientPersist.Reset()
	patchcmdOpen := gomonkey.ApplyFunc(cmd.Open, func(string, ...string) error { return nil })
	defer patchcmdOpen.Reset()
	patchrandstrHex := gomonkey.ApplyFunc(randstr.Hex, func(int) string { return state })
	defer patchrandstrHex.Reset()
	patchspotifyauthAuthenticatorToken := gomonkey.ApplyMethod(reflect.TypeOf(spotifyauth.Authenticator{}), "Token",
		func(spotifyauth.Authenticator, context.Context, string, *http.Request, ...oauth2.AuthCodeOption) (*oauth2.Token, error) {
			return nil, nil
		})
	defer patchspotifyauthAuthenticatorToken.Reset()

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

func TestAuthenticateRecoverAndPersist(t *testing.T) {
	// monkey patching
	patchosReadFile := gomonkey.ApplyFunc(os.ReadFile, func(string) ([]byte, error) {
		return []byte(token), nil
	})
	defer patchosReadFile.Reset()
	patchosOpenFile := gomonkey.ApplyFunc(os.OpenFile, func(string, int, fs.FileMode) (*os.File, error) {
		return &os.File{}, nil
	})
	defer patchosOpenFile.Reset()
	patchspotifyClientToken := gomonkey.ApplyMethod(reflect.TypeOf(&spotify.Client{}), "Token",
		func(*spotify.Client) (*oauth2.Token, error) {
			return nil, nil
		})
	defer patchspotifyClientToken.Reset()
	patchjsonEncoderEncode := gomonkey.ApplyMethod(reflect.TypeOf(&json.Encoder{}), "Encode",
		func(*json.Encoder, any) error {
			return nil
		})
	defer patchjsonEncoderEncode.Reset()

	// testing
	assert.Nil(t, util.ErrOnly(Authenticate("127.0.0.1")))
}

func TestAuthenticateRecoverOpenFailure(t *testing.T) {
	// monkey patching
	patchosReadFile := gomonkey.ApplyFunc(os.ReadFile, func(string) ([]byte, error) {
		return nil, errors.New("failure")
	})
	defer patchosReadFile.Reset()
	patchClientPersist := gomonkey.ApplyMethod(reflect.TypeOf(&Client{}), "Persist", func(*Client) error {
		return nil
	})
	defer patchClientPersist.Reset()
	patchcmdOpen := gomonkey.ApplyFunc(cmd.Open, func(string, ...string) error { return nil })
	defer patchcmdOpen.Reset()
	patchrandstrHex := gomonkey.ApplyFunc(randstr.Hex, func(int) string { return state })
	defer patchrandstrHex.Reset()
	patchspotifyauthAuthenticatorToken := gomonkey.ApplyMethod(reflect.TypeOf(spotifyauth.Authenticator{}), "Token",
		func(spotifyauth.Authenticator, context.Context, string, *http.Request, ...oauth2.AuthCodeOption) (*oauth2.Token, error) {
			return nil, nil
		})
	defer patchspotifyauthAuthenticatorToken.Reset()

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

func TestAuthenticateRecoverUnmarshalFailure(t *testing.T) {
	// monkey patching
	patchosReadFile := gomonkey.ApplyFunc(os.ReadFile, func(string) ([]byte, error) {
		return []byte(token), nil
	})
	defer patchosReadFile.Reset()
	patchjsonUnmarshal := gomonkey.ApplyFunc(json.Unmarshal, func([]byte, any) error {
		return errors.New("failure")
	})
	defer patchjsonUnmarshal.Reset()
	patchClientPersist := gomonkey.ApplyMethod(reflect.TypeOf(&Client{}), "Persist", func(*Client) error {
		return nil
	})
	defer patchClientPersist.Reset()
	patchcmdOpen := gomonkey.ApplyFunc(cmd.Open, func(string, ...string) error { return nil })
	defer patchcmdOpen.Reset()
	patchrandstrHex := gomonkey.ApplyFunc(randstr.Hex, func(int) string { return state })
	defer patchrandstrHex.Reset()
	patchspotifyauthAuthenticatorToken := gomonkey.ApplyMethod(reflect.TypeOf(spotifyauth.Authenticator{}), "Token",
		func(spotifyauth.Authenticator, context.Context, string, *http.Request, ...oauth2.AuthCodeOption) (*oauth2.Token, error) {
			return nil, nil
		})
	defer patchspotifyauthAuthenticatorToken.Reset()

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

func TestAuthenticateRecoverAndPersistTokenFailure(t *testing.T) {
	// monkey patching
	patchosReadFile := gomonkey.ApplyFunc(os.ReadFile, func(string) ([]byte, error) {
		return []byte(token), nil
	})
	defer patchosReadFile.Reset()
	patchspotifyClientToken := gomonkey.ApplyMethod(reflect.TypeOf(&spotify.Client{}), "Token",
		func(*spotify.Client) (*oauth2.Token, error) {
			return nil, errors.New("failure")
		})
	defer patchspotifyClientToken.Reset()

	// testing
	assert.Error(t, util.ErrOnly(Authenticate("127.0.0.1")), "failure")
}

func TestAuthenticateRecoverAndPersistOpenFailure(t *testing.T) {
	// monkey patching
	patchosReadFile := gomonkey.ApplyFunc(os.ReadFile, func(string) ([]byte, error) {
		return []byte(token), nil
	})
	defer patchosReadFile.Reset()
	patchspotifyClientToken := gomonkey.ApplyMethod(reflect.TypeOf(&spotify.Client{}), "Token",
		func(*spotify.Client) (*oauth2.Token, error) {
			return nil, nil
		})
	defer patchspotifyClientToken.Reset()
	patchosOpenFile := gomonkey.ApplyFunc(os.OpenFile, func(string, int, fs.FileMode) (*os.File, error) {
		return nil, errors.New("failure")
	})
	defer patchosOpenFile.Reset()

	// testing
	assert.Error(t, util.ErrOnly(Authenticate("127.0.0.1")), "failure")
}

func TestAuthenticateNotFound(t *testing.T) {
	// monkey patching
	patchRecover := gomonkey.ApplyFunc(Recover, func(*spotifyauth.Authenticator, string) (*Client, error) {
		return nil, errors.New("failure")
	})
	defer patchRecover.Reset()
	patchClientPersist := gomonkey.ApplyMethod(reflect.TypeOf(&Client{}), "Persist", func(*Client) error {
		return nil
	})
	defer patchClientPersist.Reset()
	patchcmdOpen := gomonkey.ApplyFunc(cmd.Open, func(string, ...string) error { return nil })
	defer patchcmdOpen.Reset()
	patchrandstrHex := gomonkey.ApplyFunc(randstr.Hex, func(int) string { return state })
	defer patchrandstrHex.Reset()
	patchspotifyauthAuthenticatorToken := gomonkey.ApplyMethod(reflect.TypeOf(spotifyauth.Authenticator{}), "Token",
		func(spotifyauth.Authenticator, context.Context, string, *http.Request, ...oauth2.AuthCodeOption) (*oauth2.Token, error) {
			return nil, nil
		})
	defer patchspotifyauthAuthenticatorToken.Reset()

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
	patchRecover := gomonkey.ApplyFunc(Recover, func(*spotifyauth.Authenticator, string) (*Client, error) {
		return nil, errors.New("failure")
	})
	defer patchRecover.Reset()
	patchClientPersist := gomonkey.ApplyMethod(reflect.TypeOf(&Client{}), "Persist", func(*Client) error {
		return nil
	})
	defer patchClientPersist.Reset()
	patchcmdOpen := gomonkey.ApplyFunc(cmd.Open, func(string, ...string) error { return nil })
	defer patchcmdOpen.Reset()
	patchrandstrHex := gomonkey.ApplyFunc(randstr.Hex, func(int) string { return state })
	defer patchrandstrHex.Reset()
	patchspotifyauthAuthenticatorToken := gomonkey.ApplyMethod(reflect.TypeOf(spotifyauth.Authenticator{}), "Token",
		func(spotifyauth.Authenticator, context.Context, string, *http.Request, ...oauth2.AuthCodeOption) (*oauth2.Token, error) {
			return nil, errors.New("failure")
		})
	defer patchspotifyauthAuthenticatorToken.Reset()

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
	patchRecover := gomonkey.ApplyFunc(Recover, func(*spotifyauth.Authenticator, string) (*Client, error) {
		return nil, errors.New("failure")
	})
	defer patchRecover.Reset()
	patchClientPersist := gomonkey.ApplyMethod(reflect.TypeOf(&Client{}), "Persist", func(*Client) error {
		return nil
	})
	defer patchClientPersist.Reset()
	patchcmdOpen := gomonkey.ApplyFunc(cmd.Open, func(string, ...string) error { return errors.New("failure") })
	defer patchcmdOpen.Reset()
	patchrandstrHex := gomonkey.ApplyFunc(randstr.Hex, func(int) string { return state })
	defer patchrandstrHex.Reset()

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
	patchRecover := gomonkey.ApplyFunc(Recover, func(*spotifyauth.Authenticator, string) (*Client, error) {
		return nil, errors.New("failure")
	})
	defer patchRecover.Reset()
	patchClientPersist := gomonkey.ApplyMethod(reflect.TypeOf(&Client{}), "Persist", func(*Client) error {
		return nil
	})
	defer patchClientPersist.Reset()
	patchcmdOpen := gomonkey.ApplyFunc(cmd.Open, func(string, ...string) error { return nil })
	defer patchcmdOpen.Reset()
	patchrandstrHex := gomonkey.ApplyFunc(randstr.Hex, func(int) string { return state })
	defer patchrandstrHex.Reset()
	patchnetListen := gomonkey.ApplyFunc(net.Listen, func(string, string) (net.Listener, error) { return nil, errors.New("failure") })
	defer patchnetListen.Reset()

	// testing
	assert.EqualError(t, util.ErrOnly(Authenticate()), "failure")
}
