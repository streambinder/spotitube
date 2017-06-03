package spotify

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"

	"github.com/zmb3/spotify"
)

var (
	auth  = spotify.NewAuthenticator(SPOTIFY_REDIRECT_URI, spotify.ScopeUserLibraryRead)
	ch    = make(chan *spotify.Client)
	state = "state"
)

func AuthAndTracks() []spotify.SavedTrack {
	http.HandleFunc("/favicon.ico", HttpFaviconHandler)
	http.HandleFunc("/callback", HttpCompleteAuthHandler)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Println("Got request for:", r.URL.String())
	})
	go http.ListenAndServe(":8080", nil)

	auth.SetAuthInfo(SPOTIFY_CLIENT_ID, SPOTIFY_CLIENT_SECRET)
	url := auth.AuthURL(state)
	command_cmd := "xdg-open"
	command_args := []string{url}
	_, err := exec.Command(command_cmd, command_args...).Output()
	if err != nil {
		fmt.Println("Something went wrong while executing trying to make the default browser open the authorization URL.")
		fmt.Println("Please, in order to authorize me to read your library, go to:\n" + url)
	}

	// wait for auth to complete
	client := <-ch

	times := 0
	opt_limit := 50
	var tracks []spotify.SavedTrack
	for true {
		opt_offset := times * opt_limit
		options := spotify.Options{
			Limit:  &opt_limit,
			Offset: &opt_offset,
		}
		chunk, _ := client.CurrentUsersTracksOpt(&options)
		tracks = append(tracks, chunk.Tracks...)
		if len(chunk.Tracks) < 20 {
			break
		}
		times++
	}

	return tracks
}

func HttpFaviconHandler(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, SPOTIFY_FAVICON_URL, 301)
}

func HttpCompleteAuthHandler(w http.ResponseWriter, r *http.Request) {
	tok, err := auth.Token(state, r)
	if err != nil {
		http.Error(w, HttpMessage("Couldn't get token", "none"), http.StatusForbidden)
	}
	if st := r.FormValue("state"); st != state {
		http.NotFound(w, r)
	}
	client := auth.NewClient(tok)
	fmt.Fprintf(w, HttpMessage("Login completed!", "Come back to the shell and enjoy the magic!"))
	ch <- &client
}

func HttpMessage(content_title string, content_subtitle string) string {
	return fmt.Sprintf(SPOTIFY_HTML_TEMPLATE, content_title, content_subtitle)
}
