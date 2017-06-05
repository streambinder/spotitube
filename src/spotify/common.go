package spotify

import (
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"strconv"

	"github.com/zmb3/spotify"
	. "utils"
)

var (
	auth   = spotify.NewAuthenticator(SPOTIFY_REDIRECT_URI, spotify.ScopeUserLibraryRead)
	ch     = make(chan *spotify.Client)
	state  = "state"
	logger = NewLogger()
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
		logger.Log("Something went wrong while executing trying to make the default browser open the authorization URL.")
		logger.Log("Please, in order to authorize me to read your library, go to:\n" + url)
	}

	// wait for auth to complete
	logger.Log("Waiting for authentication process to complete.")
	client := <-ch

	logger.Log("Pulling out user library.")
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
		logger.Log(strconv.Itoa(times*opt_limit) + " songs taken.")
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
		logger.Fatal("Couldn't get token.")
	}
	if st := r.FormValue("state"); st != state {
		http.NotFound(w, r)
		logger.Fatal("\"state\" value not found.")
	}
	client := auth.NewClient(tok)
	fmt.Fprintf(w, HttpMessage("Login completed", "Come back to the shell and enjoy the magic!"))
	logger.Log("Login process completed.")
	ch <- &client
}

func HttpMessage(content_title string, content_subtitle string) string {
	return fmt.Sprintf(SPOTIFY_HTML_TEMPLATE, content_title, content_subtitle)
}
