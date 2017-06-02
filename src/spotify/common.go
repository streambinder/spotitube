package spotify

import (
	"fmt"
	"log"
	"net/http"

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

	url := auth.AuthURL(state)
	fmt.Println("Auhtorize request to:", url)

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
