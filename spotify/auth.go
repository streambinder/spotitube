package spotify

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"

	"github.com/arunsworld/nursery"
	"github.com/streambinder/spotitube/util/cmd"
	"github.com/thanhpk/randstr"
	"github.com/zmb3/spotify"
)

const (
	port         = 8080
	closeTabHTML = "<!DOCTYPE html><html><head><script>open(location, '_self').close();</script></head></html>"
)

type Client struct {
	spotify.Client
	authenticator spotify.Authenticator
	state         string
}

func Authenticate(callbacks ...string) (*Client, error) {
	var (
		client        Client
		serverMux     = http.NewServeMux()
		server        = &http.Server{Addr: fmt.Sprintf("0.0.0.0:%d", port), Handler: serverMux}
		state         = randstr.Hex(20)
		callback      = "127.0.0.1"
		clientChannel = make(chan *spotify.Client, 1)
		errChannel    = make(chan error, 1)
	)
	defer close(clientChannel)
	defer close(errChannel)

	if len(callbacks) > 0 {
		callback = callbacks[0]
	}

	authenticator := spotify.NewAuthenticator(
		fmt.Sprintf("http://%s:%d/callback", callback, port),
		spotify.ScopeUserLibraryRead,
		spotify.ScopeUserLibraryModify,
		spotify.ScopePlaylistReadPrivate,
		spotify.ScopePlaylistReadCollaborative,
		spotify.ScopePlaylistModifyPublic,
		spotify.ScopePlaylistModifyPrivate,
	)
	authenticator.SetAuthInfo(os.Getenv("SPOTIFY_ID"), os.Getenv("SPOTIFY_KEY"))

	serverMux.HandleFunc("/callback", func(writer http.ResponseWriter, request *http.Request) {
		fmt.Fprintln(writer, closeTabHTML)
		token, err := authenticator.Token(state, request)
		if err != nil {
			clientChannel <- nil
			errChannel <- errors.New(http.StatusText(http.StatusForbidden))
		} else if requestState := request.FormValue("state"); requestState != state {
			clientChannel <- nil
			errChannel <- errors.New(http.StatusText(http.StatusNotFound))
		} else {
			client := authenticator.NewClient(token)
			clientChannel <- &client
			errChannel <- nil
		}
	})

	if err := nursery.RunConcurrently(
		// spawn web server to handle login redirection
		func(ctx context.Context, ch chan error) {
			if err := server.ListenAndServe(); err != http.ErrServerClosed {
				ch <- err
				clientChannel <- nil
				errChannel <- err
			}
		},
		// auto-launch web browser with authentication URL
		func(ctx context.Context, ch chan error) {
			if err := cmd.Open(authenticator.AuthURL(state)); err != nil {
				ch <- err
			}
		},
		// wait to obtain a valid client from global channel
		func(ctx context.Context, ch chan error) {
			c, err := <-clientChannel, <-errChannel
			if err != nil {
				ch <- err
			} else {
				client = Client{*c, authenticator, state}
			}
			ch <- server.Shutdown(ctx)
		}); err != nil {
		return nil, err
	}

	return &client, nil
}
