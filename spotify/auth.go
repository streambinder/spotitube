package spotify

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"sync"

	"github.com/arunsworld/nursery"
	"github.com/streambinder/spotitube/util/cmd"
	"github.com/thanhpk/randstr"
	"github.com/zmb3/spotify"
)

const port = 8080

var register sync.Once

type Client struct {
	spotify.Client
	authenticator spotify.Authenticator
	state         string
}

func Authenticate(callbacks ...string) (*Client, error) {
	var (
		client   Client
		server   = &http.Server{Addr: fmt.Sprintf("0.0.0.0:%d", port)}
		state    = randstr.Hex(20)
		callback = "127.0.0.1"
		channel  = make(chan *spotify.Client, 1)
	)
	defer close(channel)

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
	authenticator.SetAuthInfo(
		os.Getenv("SPOTIFY_ID"),
		os.Getenv("SPOTIFY_KEY"),
	)

	register.Do(func() {
		http.HandleFunc("/callback", func(writer http.ResponseWriter, request *http.Request) {
			token, err := authenticator.Token(state, request)
			if err != nil {
				http.Error(writer, "could not get token: "+err.Error(), http.StatusForbidden)
				return
			} else if requestState := request.FormValue("state"); requestState != state {
				http.NotFound(writer, request)
				return
			}
			client := authenticator.NewClient(token)
			channel <- &client
			fmt.Fprintf(writer, "login completed")
		})
	})

	if err := nursery.RunConcurrently(
		// spawn web server to handle login redirection
		func(ctx context.Context, errChannel chan error) {
			if err := server.ListenAndServe(); err != http.ErrServerClosed {
				errChannel <- err
				channel <- nil
			}
		},
		// auto-launch web browser with authentication URL
		func(ctx context.Context, errChannel chan error) {
			errChannel <- cmd.Open(authenticator.AuthURL(state))
		},
		// wait to obtain a valid client from global channel
		func(ctx context.Context, errChannel chan error) {
			c := <-channel
			if c != nil {
				client = Client{
					*c,
					authenticator,
					state,
				}
			}
			errChannel <- server.Shutdown(ctx)
		}); err != nil {
		return nil, err
	}

	return &client, nil
}
