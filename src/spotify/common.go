package spotify

import (
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"strconv"
	"strings"

	"github.com/zmb3/spotify"
	. "utils"
)

var (
	auth                  = spotify.NewAuthenticator(SPOTIFY_REDIRECT_URI, spotify.ScopeUserLibraryRead, spotify.ScopePlaylistReadPrivate, spotify.ScopePlaylistReadCollaborative)
	ch                    = make(chan *spotify.Client)
	state                 = "state"
	logger         Logger = NewLogger()
	playlist_id           = ""
	playlist_owner        = ""
)

func AuthAndTracks(parameters ...string) []spotify.FullTrack {
	if len(parameters) > 0 {
		playlist_owner = strings.Split(parameters[0], ":")[2]
		playlist_id = strings.Split(parameters[0], ":")[4]
	}

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

	var tracks []spotify.FullTrack
	if playlist_id == "" {
		logger.Log("Pulling out user library.")
	} else {
		logger.Log("Pulling out playlist \"" + playlist_id + "\".")
	}
	times := 0
	opt_limit := 50
	for true {
		opt_offset := times * opt_limit
		options := spotify.Options{
			Limit:  &opt_limit,
			Offset: &opt_offset,
		}
		if playlist_id == "" {
			chunk, err := client.CurrentUsersTracksOpt(&options)
			if err != nil {
				logger.Fatal("Something gone wrong while getting user library: " + err.Error() + ".")
			}
			for _, track := range chunk.Tracks {
				tracks = append(tracks, track.FullTrack)
			}
			if len(chunk.Tracks) < 50 {
				break
			}
		} else {
			chunk, err := client.GetPlaylistTracksOpt(playlist_owner, spotify.ID(playlist_id), &options, "")
			if err != nil {
				logger.Fatal("Something gone wrong while getting playlist \"" + playlist_id + "\" songs: " + err.Error() + ".")
			}
			for _, track := range chunk.Tracks {
				tracks = append(tracks, track.Track)
			}
			if len(chunk.Tracks) < 50 {
				break
			}
		}
		times++
	}
	logger.Log(strconv.Itoa(len(tracks)) + " songs taken.")
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
