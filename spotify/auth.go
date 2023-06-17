package spotify

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"

	"github.com/arunsworld/nursery"
	"github.com/streambinder/spotitube/util"
	"github.com/streambinder/spotitube/util/cmd"
	"github.com/thanhpk/randstr"
	"github.com/zmb3/spotify/v2"
	spotifyauth "github.com/zmb3/spotify/v2/auth"
	"golang.org/x/oauth2"
)

const (
	TokenBasename = "session.json"
	closeTabHTML  = "<!DOCTYPE html><html><head><script>open(location, '_self').close();</script></head></html>"
)

var (
	port      = 65535
	tokenPath = util.CacheFile(TokenBasename)
)

type Client struct {
	*spotify.Client
	authenticator *spotifyauth.Authenticator
	state         string
	cache         map[string]interface{}
}

func Authenticate(urlProcessor func(string) error, callbacks ...string) (*Client, error) {
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

	authenticator := spotifyauth.New(
		spotifyauth.WithRedirectURL(fmt.Sprintf("http://%s:%d/callback", callback, port)),
		spotifyauth.WithScopes(
			spotifyauth.ScopeUserLibraryRead,
			spotifyauth.ScopeUserLibraryModify,
			spotifyauth.ScopePlaylistReadPrivate,
			spotifyauth.ScopePlaylistReadCollaborative,
			spotifyauth.ScopePlaylistModifyPublic,
			spotifyauth.ScopePlaylistModifyPrivate,
		),
		spotifyauth.WithClientID(os.Getenv("SPOTIFY_ID")),
		spotifyauth.WithClientSecret(os.Getenv("SPOTIFY_KEY")),
	)
	if client, err := Recover(authenticator, state); err == nil {
		return client, client.Persist()
	}

	serverMux.HandleFunc("/callback", func(writer http.ResponseWriter, request *http.Request) {
		fmt.Fprintln(writer, closeTabHTML)
		token, err := authenticator.Token(request.Context(), state, request)
		if err != nil {
			clientChannel <- nil
			errChannel <- errors.New(http.StatusText(http.StatusForbidden))
		} else if requestState := request.FormValue("state"); requestState != state {
			clientChannel <- nil
			errChannel <- errors.New(http.StatusText(http.StatusNotFound))
		} else {
			client := spotify.New(
				authenticator.Client(request.Context(), token),
				spotify.WithRetry(true),
			)
			clientChannel <- client
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
			if urlProcessor == nil {
				return
			}

			if err := urlProcessor(authenticator.AuthURL(state)); err != nil {
				ch <- err
			}
		},
		// wait to obtain a valid client from global channel
		func(ctx context.Context, ch chan error) {
			c, err := <-clientChannel, <-errChannel
			if err != nil {
				ch <- err
			} else {
				client = Client{c, authenticator, state, make(map[string]interface{})}
			}
			ch <- server.Shutdown(ctx)
		}); err != nil {
		return nil, err
	}

	return &client, client.Persist()
}

func Recover(authenticator *spotifyauth.Authenticator, state string) (*Client, error) {
	data, err := os.ReadFile(tokenPath)
	if err != nil {
		return nil, err
	}

	var token oauth2.Token
	if err := json.Unmarshal(data, &token); err != nil {
		return nil, err
	}

	return &Client{spotify.New(
		authenticator.Client(context.Background(), &token),
		spotify.WithRetry(true),
	), authenticator, state, make(map[string]interface{})}, nil
}

func (client *Client) Persist() error {
	token, err := client.Token()
	if err != nil {
		return err
	}

	file, err := os.OpenFile(tokenPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	defer file.Close()
	return json.NewEncoder(file).Encode(token)
}

func BrowserProcessor(url string) error {
	return cmd.Open(url)
}
