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

func init() {
	monkey.Patch(cmd.Open, func(string, ...string) error { return nil })
	monkey.Patch(randstr.Hex, func(int) string { return state })
}

func TestAuthenticate(t *testing.T) {
	assert.Nil(t, nursery.RunConcurrently(
		func(ctx context.Context, errChannel chan error) {
			errChannel <- util.ErrOnly(Authenticate("127.0.0.1"))
		},
		func(ctx context.Context, errChannel chan error) {
			var (
				response *http.Response
				err      error
			)

			// this loops is used to wait for the authentication
			// handler server to come up
			for {
				response, err = http.Get(fmt.Sprintf("http://127.0.0.1:%d/callback", port))
				if errors.Is(err, syscall.ECONNREFUSED) {
					time.Sleep(100 * time.Millisecond)
					continue
				}
				break
			}
			assert.Nil(t, err)
			assert.Equal(t, http.StatusForbidden, response.StatusCode)
			monkey.PatchInstanceMethod(reflect.TypeOf(spotify.Authenticator{}), "Token",
				func(spotify.Authenticator, string, *http.Request) (*oauth2.Token, error) {
					return nil, nil
				})

			response, err = http.Get(fmt.Sprintf("http://127.0.0.1:%d/callback?code=C0D3&state=S", port))
			assert.Nil(t, err)
			assert.Equal(t, http.StatusNotFound, response.StatusCode)

			response, err = http.Get(fmt.Sprintf("http://127.0.0.1:%d/callback?code=C0D3&state=S7473", port))
			assert.Nil(t, err)
			assert.Equal(t, http.StatusOK, response.StatusCode)
		},
	))
}

func TestAuthenticateServerUnserving(t *testing.T) {
	monkey.Patch(net.Listen, func(string, string) (net.Listener, error) { return nil, errors.New("failure") })
	assert.EqualError(t, util.ErrOnly(Authenticate()), "failure")
}
