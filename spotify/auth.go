package spotify

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/arunsworld/nursery"
	"github.com/streambinder/spotitube/util/cmd"
	"github.com/thanhpk/randstr"
	"github.com/zmb3/spotify"
)

const port = 8080

type Client struct {
	spotify.Client
	authenticator spotify.Authenticator
	state         string
	cache         userCache
}

type userCache struct {
	username    string
	displayName string
}

func Authenticate(callbacks ...string) (*Client, error) {
	var (
		client   Client
		server   = http.Server{Addr: fmt.Sprintf("0.0.0.0:%d", port)}
		state    = randstr.Hex(20)
		callback = "127.0.0.1"
		channel  = make(chan spotify.Client, 1)
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

	http.HandleFunc("/callback", func(writer http.ResponseWriter, request *http.Request) {
		token, err := authenticator.Token(state, request)
		if err != nil {
			http.Error(writer, "could not get token: "+err.Error(), http.StatusForbidden)
			return
		} else if requestState := request.FormValue("state"); requestState != state {
			http.NotFound(writer, request)
			return
		}
		channel <- authenticator.NewClient(token)
		fmt.Fprintf(writer, "login completed")
	})

	if err := nursery.RunConcurrently(
		func(ctx context.Context, errChannel chan error) {
			if err := server.ListenAndServe(); err != http.ErrServerClosed {
				errChannel <- err
			}
		},
		func(ctx context.Context, errChannel chan error) {
			if err := cmd.Open(authenticator.AuthURL(state)); err != nil {
				errChannel <- err
			}
		},
		func(ctx context.Context, errChannel chan error) {
			client = Client{
				<-channel,
				authenticator,
				state,
				userCache{},
			}
			errChannel <- server.Shutdown(context.Background())
		}); err != nil {
		return nil, err
	}

	if err := client.precache(); err != nil {
		return nil, err
	}
	return &client, nil
}

func (client *Client) precache() error {
	user, err := client.CurrentUser()
	if err != nil {
		return err
	}

	client.cache.username = user.ID
	client.cache.displayName = user.DisplayName
	return nil
}

func (client *Client) Username() (username string, err error) {
	if len(client.cache.username) == 0 {
		err = client.precache()
	}
	return client.cache.username, err
}

func (client *Client) DisplayName() (displayName string, err error) {
	if len(client.cache.displayName) == 0 {
		err = client.precache()
	}
	return client.cache.displayName, err
}
